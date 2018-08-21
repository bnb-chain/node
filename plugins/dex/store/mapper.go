package store

import (
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"fmt"

	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/plugins/dex/types"
	dexUtils "github.com/BiJie/BinanceChain/plugins/dex/utils"
	"github.com/BiJie/BinanceChain/wire"
)

type TradingPairMapper interface {
	AddTradingPair(ctx sdk.Context, pair types.TradingPair) error
	Exists(ctx sdk.Context, tradeAsset, quoteAsset string) bool
	GetTradingPair(ctx sdk.Context, tradeAsset, quoteAsset string) (types.TradingPair, error)
	ListAllTradingPairs(ctx sdk.Context) []types.TradingPair
	UpdateTickSizeAndLotSize(ctx sdk.Context, pair types.TradingPair, price int64)
	ValidateOrder(ctx sdk.Context, symbol string, price int64, quantity int64) error
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

	key := []byte(utils.Ccy2TradeSymbol(tradeAsset, quoteAsset))
	store := ctx.KVStore(m.key)
	value := m.encodeTradingPair(pair)
	store.Set(key, value)
	return nil
}

func (m mapper) Exists(ctx sdk.Context, tradeAsset, quoteAsset string) bool {
	store := ctx.KVStore(m.key)

	symbol := utils.Ccy2TradeSymbol(tradeAsset, quoteAsset)
	return store.Has([]byte(symbol))
}

func (m mapper) GetTradingPair(ctx sdk.Context, tradeAsset, quoteAsset string) (types.TradingPair, error) {
	store := ctx.KVStore(m.key)
	symbol := utils.Ccy2TradeSymbol(tradeAsset, quoteAsset)
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

func (m mapper) UpdateTickSizeAndLotSize(ctx sdk.Context, pair types.TradingPair, price int64) {
	tickSize, lotSize := dexUtils.CalcTickSizeAndLotSize(price)

	if tickSize != pair.TickSize || lotSize != pair.LotSize {
		pair.TickSize = tickSize
		pair.LotSize = lotSize

		m.AddTradingPair(ctx, pair)
	}
}

func (m mapper) ValidateOrder(ctx sdk.Context, symbol string, price int64, quantity int64) error {
	tradeAsset, quoteAsset, err := utils.TradeSymbol2Ccy(symbol)
	if err != nil {
		return err
	}

	pair, err := m.GetTradingPair(ctx, tradeAsset, quoteAsset)
	if err != nil {
		return err
	}

	if quantity <= 0 || quantity%pair.LotSize != 0 {
		return errors.New(fmt.Sprintf("minimum quantity should be larger than %v and order's quantity is %v", pair.LotSize, quantity))
	}

	if price <= 0 || price%pair.TickSize != 0 {
		return errors.New(fmt.Sprintf("minimum price should be larger than %v and order's price is %v", pair.TickSize, price))
	}

	if utils.IsExceedMaxNotional(price, quantity) {
		return errors.New(fmt.Sprintf("the product of price(%v) and quantity(%v) should less than MaxInt64", price, quantity))
	}

	return nil
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
