package store

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cmn "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/utils"
	"github.com/binance-chain/node/plugins/dex/types"
	dexUtils "github.com/binance-chain/node/plugins/dex/utils"
	"github.com/binance-chain/node/wire"
)

var recentPricesKeyPrefix = "recentPrices"

type TradingPairMapper interface {
	AddTradingPair(ctx sdk.Context, pair types.TradingPair) error
	Exists(ctx sdk.Context, baseAsset, quoteAsset string) bool
	GetTradingPair(ctx sdk.Context, baseAsset, quoteAsset string) (types.TradingPair, error)
	DeleteTradingPair(ctx sdk.Context, baseAsset, quoteAsset string) error
	ListAllTradingPairs(ctx sdk.Context) []types.TradingPair
	ListAllBEP2TradingPairs(ctx sdk.Context) []types.TradingPair
	ListAllMiniTradingPairs(ctx sdk.Context) []types.TradingPair
	UpdateRecentPrices(ctx sdk.Context, pricesStoreEvery, numPricesStored int64, lastTradePrices map[string]int64)
	GetRecentPrices(ctx sdk.Context, pricesStoreEvery, numPricesStored int64) map[string]*utils.FixedSizeRing
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
	if err := cmn.ValidateMapperTokenSymbol(baseAsset); err != nil {
		return err
	}
	quoteAsset := pair.QuoteAssetSymbol
	if err := cmn.ValidateMapperTokenSymbol(quoteAsset); err != nil {
		return err
	}

	tradeSymbol := dexUtils.Assets2TradingPair(strings.ToUpper(baseAsset), strings.ToUpper(quoteAsset))
	key := []byte(tradeSymbol)
	store := ctx.KVStore(m.key)
	value := m.encodeTradingPair(pair)
	store.Set(key, value)
	ctx.Logger().Info("Added trading pair", "pair", tradeSymbol)
	return nil
}

func (m mapper) AddMiniTokenTradingPair(ctx sdk.Context, pair types.TradingPair) error {
	baseAsset := pair.BaseAssetSymbol
	if err := cmn.ValidateMapperTokenSymbol(baseAsset); err != nil {
		return err
	}
	quoteAsset := pair.QuoteAssetSymbol
	if err := cmn.ValidateMapperTokenSymbol(quoteAsset); err != nil {
		return err
	}

	tradeSymbol := dexUtils.Assets2TradingPair(strings.ToUpper(baseAsset), strings.ToUpper(quoteAsset))
	key := []byte(tradeSymbol)
	store := ctx.KVStore(m.key)
	value := m.encodeTradingPair(pair)
	store.Set(key, value)
	ctx.Logger().Info("Added trading pair", "pair", tradeSymbol)
	return nil
}

func (m mapper) DeleteTradingPair(ctx sdk.Context, baseAsset, quoteAsset string) error {
	symbol := dexUtils.Assets2TradingPair(strings.ToUpper(baseAsset), strings.ToUpper(quoteAsset))
	key := []byte(symbol)
	store := ctx.KVStore(m.key)

	bz := store.Get(key)
	if bz == nil {
		return fmt.Errorf("trading pair %s does not exist", symbol)
	}

	store.Delete(key)
	ctx.Logger().Info("delete trading pair", "pair", symbol)
	return nil
}

func (m mapper) Exists(ctx sdk.Context, baseAsset, quoteAsset string) bool {
	store := ctx.KVStore(m.key)

	symbol := dexUtils.Assets2TradingPair(strings.ToUpper(baseAsset), strings.ToUpper(quoteAsset))
	return store.Has([]byte(symbol))
}

func (m mapper) GetTradingPair(ctx sdk.Context, baseAsset, quoteAsset string) (types.TradingPair, error) {
	store := ctx.KVStore(m.key)
	symbol := dexUtils.Assets2TradingPair(strings.ToUpper(baseAsset), strings.ToUpper(quoteAsset))
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
		// TODO: temp solution, will add prefix to the trading pair key and use prefix iterator instead.
		if bytes.HasPrefix(iter.Key(), []byte(recentPricesKeyPrefix)) {
			continue
		}
		pair := m.decodeTradingPair(iter.Value())
		res = append(res, pair)
	}

	return res
}

func (m mapper) ListAllBEP2TradingPairs(ctx sdk.Context) []types.TradingPair {
	//todo "implement me"
	return nil
}

func (m mapper) ListAllMiniTradingPairs(ctx sdk.Context) []types.TradingPair {
	//todo "implement me"
	return nil
}

func (m mapper) getRecentPricesSeq(height, pricesStoreEvery, numPricesStored int64) int64 {
	return (height/pricesStoreEvery - 1) % numPricesStored
}

func (m mapper) calcRecentPricesKey(seq int64) []byte {
	return []byte(fmt.Sprintf("%s:%v", recentPricesKeyPrefix, seq))
}

func (m mapper) UpdateRecentPrices(ctx sdk.Context, pricesStoreEvery, numPricesStored int64, lastTradePrices map[string]int64) {
	store := ctx.KVStore(m.key)
	seq := m.getRecentPricesSeq(ctx.BlockHeight(), pricesStoreEvery, numPricesStored)
	key := m.calcRecentPricesKey(seq)
	bz := m.encodeRecentPrices(lastTradePrices)
	store.Set(key, bz)
	ctx.Logger().Debug("Updated recentPrices", "key", string(key), "lastTradePrices", lastTradePrices)
}

func (m mapper) GetRecentPrices(ctx sdk.Context, pricesStoreEvery, numPricesStored int64) map[string]*utils.FixedSizeRing {
	recentPrices := make(map[string]*utils.FixedSizeRing, 256)
	height := ctx.BlockHeight()
	if height == 0 {
		return recentPrices
	}

	store := ctx.KVStore(m.key)
	recordStarted := false
	lastSeq := m.getRecentPricesSeq(height, pricesStoreEvery, numPricesStored)
	var i int64 = 1
	for ; i <= numPricesStored; i++ {
		key := m.calcRecentPricesKey((lastSeq + i) % numPricesStored)
		bz := store.Get(key)
		if bz == nil {
			if recordStarted {
				// we have sequential keys
				panic(fmt.Errorf("BUG!!! should not be here, key: %s", string(key)))
			} else {
				continue
			}
		} else {
			recordStarted = true
		}
		prices := m.decodeRecentPrices(bz, numPricesStored)
		numSymbol := len(prices.Pair)
		for i := 0; i < numSymbol; i++ {
			symbol := prices.Pair[i]
			if _, ok := recentPrices[symbol]; !ok {
				recentPrices[symbol] = utils.NewFixedSizedRing(numPricesStored)
			}
			recentPrices[symbol].Push(prices.Price[i])
		}
	}

	ctx.Logger().Debug("Got recentPrices", "lastSeq", lastSeq, "recentPrices", recentPrices)
	return recentPrices
}

func (m mapper) encodeRecentPrices(recentPrices map[string]int64) []byte {
	value := RecentPrice{}
	numSymbol := len(recentPrices)
	symbols := make([]string, numSymbol)
	i := 0
	for symbol, _ := range recentPrices {
		symbols[i] = symbol
		i++
	}
	// must sort to make it deterministic
	sort.Strings(symbols)
	if numSymbol != 0 {
		value.Pair = make([]string, numSymbol)
		value.Price = make([]int64, numSymbol)
	}
	for i, symbol := range symbols {
		value.Pair[i] = symbol
		value.Price[i] = recentPrices[symbol]
	}
	bz := m.cdc.MustMarshalBinaryBare(value)
	return bz
}

func (m mapper) decodeRecentPrices(bz []byte, numPricesStored int64) *RecentPrice {
	value := RecentPrice{}
	m.cdc.MustUnmarshalBinaryBare(bz, &value)
	return &value
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
