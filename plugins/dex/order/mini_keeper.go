package order

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	bnclog "github.com/binance-chain/node/common/log"
	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/utils"
	me "github.com/binance-chain/node/plugins/dex/matcheng"
	"github.com/binance-chain/node/plugins/dex/store"
	dexTypes "github.com/binance-chain/node/plugins/dex/types"
	"github.com/binance-chain/node/wire"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	defaultMiniBlockMatchInterval = 16
	defaultActiveMiniSymbolCount  = 8
)

//order keeper for mini-token
type MiniKeeper struct {
	Keeper //use dex order keeper as base keeper
}

var _ DexOrderKeeper = &MiniKeeper{}

// NewKeeper - Returns the MiniToken Keeper
func NewMiniKeeper(dexMiniKey sdk.StoreKey, miniPairMapper store.TradingPairMapper, codespace sdk.CodespaceType,
	concurrency uint, cdc *wire.Codec, globalKeeper *GlobalKeeper) *MiniKeeper {
	logger := bnclog.With("module", "dexMiniKeeper")
	return &MiniKeeper{
		Keeper{PairMapper: miniPairMapper,
			storeKey:         dexMiniKey,
			codespace:        codespace,
			engines:          make(map[string]*me.MatchEng),
			recentPrices:     make(map[string]*utils.FixedSizeRing, 256),
			allOrders:        make(map[string]map[string]*OrderInfo, 256), // need to init the nested map when a new symbol added.
			OrderChangesMtx:  &sync.Mutex{},
			OrderChanges:     make(OrderChanges, 0),
			OrderInfosForPub: make(OrderInfoForPublish),
			roundOrders:      make(map[string][]string, 256),
			roundIOCOrders:   make(map[string][]string, 256),
			poolSize:         concurrency,
			cdc:              cdc,
			logger:           logger,
			symbolSelector:   &MiniSymbolSelector{make(map[string]uint32, 256), make([]string, 0, 256)},
			GlobalKeeper:     globalKeeper,
		},
	}
}

// override
func (kp *MiniKeeper) AddEngine(pair dexTypes.TradingPair) *me.MatchEng {
	eng := kp.Keeper.AddEngine(pair)
	symbol := strings.ToUpper(pair.GetSymbol())
	kp.symbolSelector.AddSymbolHash(symbol)
	return eng
}

// used by state sync to clear memory order book after we synced latest breathe block
//TODO check usage
func (kp *MiniKeeper) ClearOrders() {
	kp.Keeper.ClearOrders()
	emptyRoundMatchSymbols := make([]string, 0, 256)
	kp.symbolSelector.SetRoundMatchSymbol(emptyRoundMatchSymbols)
}

//override
func (kp *MiniKeeper) CanListTradingPair(ctx sdk.Context, baseAsset, quoteAsset string) error {
	// trading pair against native token should exist if quote token is not native token
	baseAsset = strings.ToUpper(baseAsset)
	quoteAsset = strings.ToUpper(quoteAsset)

	if baseAsset == quoteAsset {
		return fmt.Errorf("base asset symbol should not be identical to quote asset symbol")
	}

	if kp.PairMapper.Exists(ctx, baseAsset, quoteAsset) || kp.PairMapper.Exists(ctx, quoteAsset, baseAsset) {
		return errors.New("trading pair exists")
	}

	if types.NativeTokenSymbol != quoteAsset { //todo permit BUSD
		return errors.New("quote token is not valid: " + quoteAsset)
	}

	return nil
}

//override TODO check
func (kp *MiniKeeper) CanDelistTradingPair(ctx sdk.Context, baseAsset, quoteAsset string) error {
	// trading pair against native token should not be delisted if there is any other trading pair exist
	baseAsset = strings.ToUpper(baseAsset)
	quoteAsset = strings.ToUpper(quoteAsset)

	if baseAsset == quoteAsset {
		return fmt.Errorf("base asset symbol should not be identical to quote asset symbol")
	}

	if !kp.PairMapper.Exists(ctx, baseAsset, quoteAsset) {
		return fmt.Errorf("trading pair %s_%s does not exist", baseAsset, quoteAsset)
	}

	return nil
}

// override
func (kp *MiniKeeper) validateOrder(ctx sdk.Context, acc sdk.Account, msg NewOrderMsg) error {

	err := kp.Keeper.validateOrder(ctx, acc, msg)
	if err != nil {
		return err
	}
	coins := acc.GetCoins()
	symbol := strings.ToUpper(msg.Symbol)
	var quantityBigEnough bool
	if msg.Side == Side.BUY {
		quantityBigEnough = msg.Quantity >= types.MiniTokenMinTotalSupply
	} else if msg.Side == Side.SELL {
		quantityBigEnough = (msg.Quantity >= types.MiniTokenMinTotalSupply) || coins.AmountOf(symbol) == msg.Quantity
	}
	if !quantityBigEnough {
		return fmt.Errorf("quantity is too small, the min quantity is %d or total free balance of the mini token",
			types.MiniTokenMinTotalSupply)
	}
	return nil
}

// override
func (kp *MiniKeeper) LoadOrderBookSnapshot(ctx sdk.Context, latestBlockHeight int64, timeOfLatestBlock time.Time, blockInterval, daysBack int) (int64, error) {
	return kp.Keeper.LoadOrderBookSnapshot(ctx, latestBlockHeight, timeOfLatestBlock, blockInterval, daysBack)
}
