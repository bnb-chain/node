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
	ExistsBEP2(ctx sdk.Context, symbol string) bool
	ExistsMini(ctx sdk.Context, symbol string) bool
	ExistsCC(ctx context.CLIContext, symbol string) bool
	GetTokenList(ctx sdk.Context, showZeroSupplyTokens bool, isMini bool) ITokens
	GetToken(ctx sdk.Context, symbol string) (types.IToken, error)
	// we do not provide the updateToken method
	UpdateTotalSupply(ctx sdk.Context, symbol string, supply int64) error
	UpdateBind(ctx sdk.Context, symbol string, contractAddress string, decimals int8) error
	UpdateMiniTokenURI(ctx sdk.Context, symbol string, uri string) error
	UpdateOwner(ctx sdk.Context, symbol string, newOwner sdk.AccAddress) error
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
		return m.decodeToken(bz), nil
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
		token := m.decodeToken(iter.Value())
		if !showZeroSupplyTokens && token.GetTotalSupply().ToInt64() == 0 {
			continue
		}
		res = append(res, token)
	}
	return res
}

func (m mapper) ExistsBEP2(ctx sdk.Context, symbol string) bool {
	return m.exists(ctx, symbol, false)
}

func (m mapper) ExistsMini(ctx sdk.Context, symbol string) bool {
	return m.exists(ctx, symbol, true)
}

func (m mapper) exists(ctx sdk.Context, symbol string, isMini bool) bool {
	store := ctx.KVStore(m.key)
	var key []byte
	if isMini {
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
		key = m.calcMiniTokenKey(strings.ToUpper(symbol))
	} else {
		if err := types.ValidateTokenSymbol(token.GetSymbol()); err != nil {
			return err
		}
		if err := types.ValidateIssueSymbol(token.GetOrigSymbol()); err != nil {
			return err
		}
		key = []byte(strings.ToUpper(symbol))
	}

	store := ctx.KVStore(m.key)
	value := m.encodeToken(token)
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

	toBeUpdated := m.decodeToken(bz)

	if toBeUpdated.GetTotalSupply().ToInt64() != supply {
		toBeUpdated.SetTotalSupply(utils.Fixed8(supply))
		store.Set(key, m.encodeToken(toBeUpdated))
	}
	return nil
}

func (m mapper) UpdateBind(ctx sdk.Context, symbol string, contractAddress string, decimals int8) error {
	if len(symbol) == 0 {
		return errors.New("symbol cannot be empty")
	}

	key := []byte(strings.ToUpper(symbol))
	store := ctx.KVStore(m.key)
	bz := store.Get(key)
	if bz == nil {
		return errors.New("token does not exist")
	}

	toBeUpdated := m.decodeToken(bz)
	toBeUpdated.ContractDecimals = decimals
	toBeUpdated.ContractAddress = contractAddress

	store.Set(key, m.encodeToken(toBeUpdated))
	return nil
}

func (m mapper) UpdateOwner(ctx sdk.Context, symbol string, newOwner sdk.AccAddress) error {
	if len(symbol) == 0 {
		return errors.New("symbol cannot be empty")
	}

	if len(newOwner) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(fmt.Sprintf("Invalid newOwner, expected address length is %d, actual length is %d", sdk.AddrLen, len(newOwner)))
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

	toBeUpdated := m.decodeToken(bz)
	toBeUpdated.SetOwner(newOwner)

	store.Set(key, m.encodeToken(toBeUpdated))
	return nil
}

func (m mapper) encodeToken(token types.IToken) []byte {
	bz, err := m.cdc.MarshalBinaryBare(token)
	if err != nil {
		panic(err)
	}
	return bz
}

func (m mapper) decodeToken(bz []byte) types.IToken {
	var token types.IToken
	err := m.cdc.UnmarshalBinaryBare(bz, &token)
	if err != nil {
		panic(err)
	}
	return token
}
