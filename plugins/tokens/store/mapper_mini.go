package store

import (
	"bytes"
	"errors"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/types"
)


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

	decodedToken := m.decodeIToken(bz)

	toBeUpdated, ok := decodedToken.(*types.MiniToken)
	if !ok {
		return errors.New("token cannot be converted to MiniToken")
	}

	if toBeUpdated.TokenURI != uri {
		toBeUpdated.TokenURI = uri
		store.Set(key, m.encodeIToken(toBeUpdated))
	}
	return nil
}

func (m mapper) calcMiniTokenKey(symbol string) []byte {
	var buf bytes.Buffer
	buf.WriteString(miniTokenKeyPrefix)
	buf.WriteString(symbol)
	return buf.Bytes()
}

