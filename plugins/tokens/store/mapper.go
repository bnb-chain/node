package store

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common"
	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/utils"
	"github.com/binance-chain/node/wire"
)

type Tokens []types.Token
type MiniTokens []types.MiniToken
type ITokens []types.IToken

const miniTokenKeyPrefix = "mini:"

func (t Tokens) GetSymbols() *[]string {
	var symbols []string
	for _, token := range t {
		symbols = append(symbols, token.Symbol)
	}
	return &symbols
}

type Mapper interface {
	NewToken(ctx sdk.Context, token types.IToken) error
	Exists(ctx sdk.Context, symbol string) bool
	ExistsCC(ctx context.CLIContext, symbol string) bool
	GetTokenList(ctx sdk.Context, showZeroSupplyTokens bool, isMini bool) ITokens
	GetToken(ctx sdk.Context, symbol string) (types.IToken, error)
	// we do not provide the updateToken method
	UpdateTotalSupply(ctx sdk.Context, symbol string, supply int64) error
	UpdateMiniTokenURI(ctx sdk.Context, symbol string, uri string) error
}

var _ Mapper = mapper{}

type mapper struct {
	key sdk.StoreKey
	cdc *wire.Codec
}

func NewMapper(cdc *wire.Codec, key sdk.StoreKey) mapper {
	return mapper{
		key: key,
		cdc: cdc,
	}
}

func (m mapper) GetToken(ctx sdk.Context, symbol string) (types.IToken, error) {
	store := ctx.KVStore(m.key)
	var key []byte
	if types.IsMiniTokenSymbol(symbol) {
		key = m.calcMiniTokenKey(strings.ToUpper(symbol))
	} else {
		key = []byte(strings.ToUpper(symbol))
	}

	bz := store.Get(key)
	if bz != nil {
		return m.decodeIToken(bz), nil
	}

	return nil, fmt.Errorf("token(%v) not found", symbol)
}

func (m mapper) GetTokenList(ctx sdk.Context, showZeroSupplyTokens bool, isMini bool) ITokens {
	var res ITokens
	store := ctx.KVStore(m.key)
	iter := store.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		isValid := isMini == bytes.HasPrefix(iter.Key(), []byte(miniTokenKeyPrefix))
		if !isValid {
			continue
		}
		token := m.decodeIToken(iter.Value())
		if !showZeroSupplyTokens && token.GetTotalSupply().ToInt64() == 0 {
			continue
		}
		res = append(res, token)
	}
	return res
}

func (m mapper) Exists(ctx sdk.Context, symbol string) bool {
	store := ctx.KVStore(m.key)
	var key []byte
	if types.IsMiniTokenSymbol(symbol) {
		key = m.calcMiniTokenKey(strings.ToUpper(symbol))
	} else {
		key = []byte(strings.ToUpper(symbol))
	}
	return store.Has(key)
}

func (m mapper) ExistsCC(ctx context.CLIContext, symbol string) bool {
	var key []byte
	if types.IsMiniTokenSymbol(symbol) {
		key = m.calcMiniTokenKey(strings.ToUpper(symbol))
	} else {
		key = []byte(strings.ToUpper(symbol))
	}
	bz, err := ctx.QueryStore(key, common.TokenStoreName)
	if err != nil {
		return false
	}
	if bz != nil {
		return true
	}
	return false
}

func (m mapper) NewToken(ctx sdk.Context, token types.IToken) error {
	symbol := token.GetSymbol()
	var key []byte
	if types.IsMiniTokenSymbol(symbol) {
		if err := types.ValidateMiniToken(token); err != nil {
			return err
		}
		key = m.calcMiniTokenKey(strings.ToUpper(symbol))
	} else {
		if err := types.ValidateToken(token); err != nil {
			return err
		}
		key = []byte(strings.ToUpper(symbol))
	}

	store := ctx.KVStore(m.key)
	value := m.encodeIToken(token)
	store.Set(key, value)
	return nil
}

func (m mapper) UpdateTotalSupply(ctx sdk.Context, symbol string, supply int64) error {
	if len(symbol) == 0 {
		return errors.New("symbol cannot be empty")
	}
	var key []byte
	if types.IsMiniTokenSymbol(symbol) {
		key = m.calcMiniTokenKey(strings.ToUpper(symbol))
	} else {
		key = []byte(strings.ToUpper(symbol))
	}
	store := ctx.KVStore(m.key)
	bz := store.Get(key)
	if bz == nil {
		return errors.New("token does not exist")
	}

	toBeUpdated := m.decodeIToken(bz)

	if toBeUpdated.GetTotalSupply().ToInt64() != supply {
		toBeUpdated.SetTotalSupply(utils.Fixed8(supply))
		store.Set(key, m.encodeIToken(toBeUpdated))
	}
	return nil
}

func (m mapper) encodeIToken(token types.IToken) []byte {
	bz, err := m.cdc.MarshalBinaryBare(token)
	if err != nil {
		panic(err)
	}
	return bz
}

func (m mapper) decodeIToken(bz []byte) types.IToken {
	var token types.IToken
	err := m.cdc.UnmarshalBinaryBare(bz, &token)
	if err != nil {
		panic(err)
	}
	return token
}
