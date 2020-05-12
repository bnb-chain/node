package store

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/utils"
)

func (m mapper) GetMiniToken(ctx sdk.Context, symbol string) (types.MiniToken, error) {
	store := ctx.KVStore(m.key)
	key := m.calcMiniTokenKey(strings.ToUpper(symbol))

	bz := store.Get(key)
	if bz != nil {
		return m.decodeMiniToken(bz), nil
	}

	return types.MiniToken{}, fmt.Errorf("mini-token(%v) not found", symbol)
}
func (m mapper) GetMiniTokenList(ctx sdk.Context, showZeroSupplyMiniTokens bool) MiniTokens {
	var res MiniTokens
	store := ctx.KVStore(m.key)
	iter := store.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		if !bytes.HasPrefix(iter.Key(), []byte(miniTokenKeyPrefix)) {
			continue
		}
		token := m.decodeMiniToken(iter.Value())
		if !showZeroSupplyMiniTokens && token.TotalSupply.ToInt64() == 0 {
			continue
		}
		res = append(res, token)
	}
	return res
}

func (m mapper) NewMiniToken(ctx sdk.Context, token types.MiniToken) error {
	symbol := token.Symbol
	if err := types.ValidateMiniToken(token); err != nil {
		return err
	}
	key := m.calcMiniTokenKey(strings.ToUpper(symbol))
	store := ctx.KVStore(m.key)
	value := m.encodeMiniToken(token)
	store.Set(key, value)
	return nil
}

func (m mapper) updateMiniTotalSupply(ctx sdk.Context, symbol string, supply int64) error {
	key := []byte(strings.ToUpper(symbol))
	store := ctx.KVStore(m.key)
	bz := store.Get(key)
	if bz == nil {
		return errors.New("mini token does not exist")
	}

	toBeUpdated := m.decodeMiniToken(bz)

	if toBeUpdated.TotalSupply.ToInt64() != supply {
		toBeUpdated.TotalSupply = utils.Fixed8(supply)
		store.Set(key, m.encodeMiniToken(toBeUpdated))
	}
	return nil
}

func (m mapper) UpdateMiniTokenURI(ctx sdk.Context, symbol string, uri string) error {
	if len(symbol) == 0 {
		return errors.New("symbol cannot be empty")
	}

	if len(uri) == 0 {
		return errors.New("uri cannot be empty")
	}

	if len(uri) > 2048 {
		return errors.New("uri length cannot be larger than 2048")
	}

	key := []byte(strings.ToUpper(symbol))
	store := ctx.KVStore(m.key)
	bz := store.Get(key)
	if bz == nil {
		return errors.New("token does not exist")
	}

	toBeUpdated := m.decodeMiniToken(bz)

	if toBeUpdated.TokenURI != uri {
		toBeUpdated.TokenURI = uri
		store.Set(key, m.encodeMiniToken(toBeUpdated))
	}
	return nil
}

func (m mapper) encodeMiniToken(token types.MiniToken) []byte {
	bz, err := m.cdc.MarshalBinaryBare(token)
	if err != nil {
		panic(err)
	}
	return bz
}

func (m mapper) decodeMiniToken(bz []byte) (token types.MiniToken) {
	err := m.cdc.UnmarshalBinaryBare(bz, &token)
	if err != nil {
		panic(err)
	}
	return
}

func (m mapper) calcMiniTokenKey(symbol string) []byte {
	var buf bytes.Buffer
	buf.WriteString(miniTokenKeyPrefix)
	buf.WriteString(":")
	buf.WriteString(symbol)
	return buf.Bytes()
}

