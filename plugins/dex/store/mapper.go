package store

import (
	"encoding/json"
	"errors"
	"strings"

	types2 "github.com/binance-chain/node/common/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/utils"
	"github.com/binance-chain/node/plugins/dex/types"
	dexUtils "github.com/binance-chain/node/plugins/dex/utils"
	"github.com/binance-chain/node/wire"
)

var recentPricesKey = []byte("recentPrices")

type TradingPairMapper interface {
	AddTradingPair(ctx sdk.Context, pair types.TradingPair) error
	Exists(ctx sdk.Context, baseAsset, quoteAsset string) bool
	GetTradingPair(ctx sdk.Context, baseAsset, quoteAsset string) (types.TradingPair, error)
	ListAllTradingPairs(ctx sdk.Context) []types.TradingPair
	UpdateTickSizeAndLotSize(ctx sdk.Context, pair types.TradingPair, price int64) (tickSize, lotSize int64)
	UpdateRecentPrices(ctx sdk.Context, recentPrices map[string]*utils.FixedSizeRing)
	GetRecentPrices(ctx sdk.Context) map[string]*utils.FixedSizeRing
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
	baseAsset := pair.BaseAssetSymbol
	if err := types2.ValidateMapperTokenSymbol(baseAsset); err != nil {
		return err
	}
	quoteAsset := pair.QuoteAssetSymbol
	if err := types2.ValidateMapperTokenSymbol(quoteAsset); err != nil {
		return err
	}

	tradeSymbol := utils.Assets2TradingPair(strings.ToUpper(baseAsset), strings.ToUpper(quoteAsset))
	key := []byte(tradeSymbol)
	store := ctx.KVStore(m.key)
	value := m.encodeTradingPair(pair)
	store.Set(key, value)
	ctx.Logger().Info("Added trading pair", "pair", tradeSymbol)
	return nil
}

func (m mapper) Exists(ctx sdk.Context, baseAsset, quoteAsset string) bool {
	store := ctx.KVStore(m.key)

	symbol := utils.Assets2TradingPair(strings.ToUpper(baseAsset), strings.ToUpper(quoteAsset))
	return store.Has([]byte(symbol))
}

func (m mapper) GetTradingPair(ctx sdk.Context, baseAsset, quoteAsset string) (types.TradingPair, error) {
	store := ctx.KVStore(m.key)
	symbol := utils.Assets2TradingPair(strings.ToUpper(baseAsset), strings.ToUpper(quoteAsset))
	bz := store.Get([]byte(symbol))

	if bz == nil {
		return types.TradingPair{}, errors.New("trading pair not found: " + symbol)
	}

	return m.decodeTradingPair(bz), nil
}

func (m mapper) ListAllTradingPairs(ctx sdk.Context) (res []types.TradingPair) {
	store := ctx.KVStore(m.key)
	iter := store.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		pair := m.decodeTradingPair(iter.Value())
		res = append(res, pair)
	}

	return res
}

func (m mapper) UpdateTickSizeAndLotSize(ctx sdk.Context, pair types.TradingPair, price int64) (tickSize, lotSize int64) {
	tickSize, lotSize = dexUtils.CalcTickSizeAndLotSize(price)

	if tickSize != pair.TickSize.ToInt64() ||
		lotSize != pair.LotSize.ToInt64() {
		ctx.Logger().Info("Updating tick/lotsize",
			"pair", pair.GetSymbol(), "old_ticksize", pair.TickSize, "new_ticksize", tickSize,
			"old_lotsize", pair.LotSize, "new_lotsize", lotSize)
		pair.TickSize = utils.Fixed8(tickSize)
		pair.LotSize = utils.Fixed8(lotSize)

		m.AddTradingPair(ctx, pair)
	}
	return tickSize, lotSize
}

func (m mapper) UpdateRecentPrices(ctx sdk.Context, recentPrices map[string]*utils.FixedSizeRing) {
	store := ctx.KVStore(m.key)
	value, err :=json.Marshal(recentPrices)
	if err != nil {
		panic(err)
	}

	store.Set(recentPricesKey, value)
	ctx.Logger().Debug("Updated recentPrices", "recentPrices", recentPrices)
}

func (m mapper) GetRecentPrices(ctx sdk.Context) map[string]*utils.FixedSizeRing {
	store := ctx.KVStore(m.key)
	bz := store.Get(recentPricesKey)
	var res map[string]*utils.FixedSizeRing
	if bz != nil {
		err :=json.Unmarshal(bz, &res)
		if err != nil {
			panic(err)
		}
	}
	return res
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
