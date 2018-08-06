package store

import (
	"errors"

	"github.com/BiJie/BinanceChain/wire"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/plugins/dex/types"
)

type TradingPairMapper interface {
	AddTradingPair(ctx sdk.Context, pair types.TradingPair) error
	Exists(ctx sdk.Context, tradeAsset, quoteAsset string) bool
	GetTradingPair(ctx sdk.Context, tradeAsset, quoteAsset string) types.TradingPair
}

var _ TradingPairMapper = mapper{}

type mapper struct {
	key sdk.StoreKey
	cdc *wire.Codec
}

func NewTradingPairMapper(cdc *wire.Codec, key sdk.StoreKey) TradingPairMapper {
	return mapper{
		key: key,
		cdc: cdc,
	}
}

func (m mapper) AddTradingPair(ctx sdk.Context, pair types.TradingPair) error {
	tradeAsset := pair.TradeAsset
	if len(tradeAsset) == 0 {
		return errors.New("TradeAsset cannot be empty")
	}

	quoteAsset := pair.QuoteAsset
	if len(quoteAsset) == 0 {
		return errors.New("QuoteAsset cannot be empty")
	}

	key := []byte(types.GetPairLabel(tradeAsset, quoteAsset))
	store := ctx.KVStore(m.key)
	value := m.encodeTradingPair(pair)
	store.Set(key, value)
	return nil
}

func (m mapper) Exists(ctx sdk.Context, tradeAsset, quoteAsset string) bool {
	store := ctx.KVStore(m.key)

	label := types.GetPairLabel(tradeAsset, quoteAsset)
	return store.Has([]byte(label))
}

func (m mapper) GetTradingPair(ctx sdk.Context, tradeAsset, quoteAsset string) types.TradingPair {
	store := ctx.KVStore(m.key)
	label := types.GetPairLabel(tradeAsset, quoteAsset)
	bz := store.Get([]byte(label))
	return m.decodeTradingPair(bz)
}

func (m mapper) encodeTradingPair(pair types.TradingPair) []byte {
	bz, err := m.cdc.MarshalBinaryBare(pair)
	if err != nil {
		panic(err)
	}

	return bz
}

func (m mapper) decodeTradingPair(bz []byte) (pair types.TradingPair) {
	err := m.cdc.UnmarshalBinaryBare(bz, &pair)
	if err != nil {
		panic(err)
	}

	return
}
