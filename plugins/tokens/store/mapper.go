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

const miniTokenKeyPrefix = "mini"

func (t Tokens) GetSymbols() *[]string {
	var symbols []string
	for _, token := range t {
		symbols = append(symbols, token.Symbol)
	}
	return &symbols
}

type Mapper interface {
	NewToken(ctx sdk.Context, token types.Token) error
	Exists(ctx sdk.Context, symbol string) bool
	ExistsCC(ctx context.CLIContext, symbol string) bool
	GetTokenList(ctx sdk.Context, showZeroSupplyTokens bool) Tokens
	GetToken(ctx sdk.Context, symbol string) (types.Token, error)
	// we do not provide the updateToken method
	UpdateTotalSupply(ctx sdk.Context, symbol string, supply int64) error
	NewMiniToken(ctx sdk.Context, token types.MiniToken) error
	GetMiniTokenList(ctx sdk.Context, showZeroSupplyMiniTokens bool) MiniTokens
	GetMiniToken(ctx sdk.Context, symbol string) (types.MiniToken, error)
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

func (m mapper) GetToken(ctx sdk.Context, symbol string) (types.Token, error) {
	if strings.HasPrefix(symbol, miniTokenKeyPrefix) {
		//Mini token is not allowed to query by this method
		return types.Token{}, fmt.Errorf("token(%v) not found", symbol)
	}
	store := ctx.KVStore(m.key)
	key := []byte(strings.ToUpper(symbol))

	bz := store.Get(key)
	if bz != nil {
		return m.decodeToken(bz), nil
	}

	return types.Token{}, fmt.Errorf("token(%v) not found", symbol)
}

func (m mapper) GetTokenCC(ctx context.CLIContext, symbol string) (types.Token, error) {
	if strings.HasPrefix(symbol, miniTokenKeyPrefix) {
		//Mini token is not allowed to query by this method
		return types.Token{}, fmt.Errorf("token(%v) not found", symbol)
	}
	key := []byte(strings.ToUpper(symbol))
	bz, err := ctx.QueryStore(key, common.TokenStoreName)
	if err != nil {
		return types.Token{}, err
	}
	if bz != nil {
		return m.decodeToken(bz), nil
	}
	return types.Token{}, fmt.Errorf("token(%v) not found", symbol)
}

func (m mapper) GetTokenList(ctx sdk.Context, showZeroSupplyTokens bool) Tokens {
	var res Tokens
	store := ctx.KVStore(m.key)
	iter := store.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		if bytes.HasPrefix(iter.Key(), []byte(miniTokenKeyPrefix)) {
			continue
		}
		token := m.decodeToken(iter.Value())
		if !showZeroSupplyTokens && token.TotalSupply.ToInt64() == 0 {
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
	}else{
		key = []byte(strings.ToUpper(symbol))
	}
	return store.Has(key)
}

func (m mapper) ExistsCC(ctx context.CLIContext, symbol string) bool {
	var key []byte
	if types.IsMiniTokenSymbol(symbol) {
		key = m.calcMiniTokenKey(strings.ToUpper(symbol))
	}else{
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

func (m mapper) NewToken(ctx sdk.Context, token types.Token) error {
	symbol := token.Symbol
	if err := types.ValidateToken(token); err != nil {
		return err
	}
	key := []byte(strings.ToUpper(symbol))

	store := ctx.KVStore(m.key)
	value := m.encodeToken(token)
	store.Set(key, value)
	return nil
}

func (m mapper) UpdateTotalSupply(ctx sdk.Context, symbol string, supply int64) error {
	if len(symbol) == 0 {
		return errors.New("symbol cannot be empty")
	}

	if types.IsMiniTokenSymbol(symbol) {
		return m.updateMiniTotalSupply(ctx, symbol, supply)
	}

	key := []byte(strings.ToUpper(symbol))
	store := ctx.KVStore(m.key)
	bz := store.Get(key)
	if bz == nil {
		return errors.New("token does not exist")
	}

	toBeUpdated := m.decodeToken(bz)

	if toBeUpdated.TotalSupply.ToInt64() != supply {
		toBeUpdated.TotalSupply = utils.Fixed8(supply)
		store.Set(key, m.encodeToken(toBeUpdated))
	}
	return nil
}

func (m mapper) encodeToken(token types.Token) []byte {
	bz, err := m.cdc.MarshalBinaryBare(token)
	if err != nil {
		panic(err)
	}
	return bz
}

func (m mapper) decodeToken(bz []byte) (token types.Token) {
	err := m.cdc.UnmarshalBinaryBare(bz, &token)
	if err != nil {
		panic(err)
	}
	return
}
