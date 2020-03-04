package store

import (
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
	UpdateBind(ctx sdk.Context, symbol string, contractAddress string, decimal int) error
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
	store := ctx.KVStore(m.key)
	key := []byte(strings.ToUpper(symbol))

	bz := store.Get(key)
	if bz != nil {
		return m.decodeToken(bz), nil
	}

	return types.Token{}, fmt.Errorf("token(%v) not found", symbol)
}

func (m mapper) GetTokenCC(ctx context.CLIContext, symbol string) (types.Token, error) {
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
	key := []byte(strings.ToUpper(symbol))
	return store.Has(key)
}

func (m mapper) ExistsCC(ctx context.CLIContext, symbol string) bool {
	key := []byte(strings.ToUpper(symbol))
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

func (m mapper) UpdateBind(ctx sdk.Context, symbol string, contractAddress string, decimal int) error {
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
	toBeUpdated.ContractDecimal = decimal
	toBeUpdated.ContractAddress = contractAddress

	store.Set(key, m.encodeToken(toBeUpdated))
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
