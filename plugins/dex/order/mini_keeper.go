package order

import (
	"errors"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"hash/crc32"
	"strings"
	"sync"

	"github.com/binance-chain/node/common/fees"
	bnclog "github.com/binance-chain/node/common/log"
	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/utils"
	me "github.com/binance-chain/node/plugins/dex/matcheng"
	"github.com/binance-chain/node/plugins/dex/store"
	dexTypes "github.com/binance-chain/node/plugins/dex/types"
	"github.com/binance-chain/node/wire"
)

const (
	defaultMiniBlockMatchInterval = 16
	defaultActiveMiniSymbolCount  = 8
)

//order keeper for mini-token
type MiniKeeper struct {
	Keeper                               //use dex order keeper as base keeper
	matchedMiniSymbols []string          //mini token pairs matched in this round
	miniSymbolsHash    map[string]uint32 //mini token pairs -> hash value for Round-Robin
}

var _ DexOrderKeeper = &MiniKeeper{}

// NewKeeper - Returns the MiniToken Keeper
func NewMiniKeeper(dexMiniKey sdk.StoreKey, am auth.AccountKeeper, miniPairMapper store.TradingPairMapper, codespace sdk.CodespaceType,
	concurrency uint, cdc *wire.Codec, collectOrderInfoForPublish bool) *MiniKeeper {
	logger := bnclog.With("module", "dexkeeper")
	return &MiniKeeper{
		Keeper{PairMapper: miniPairMapper,
			am:                         am,
			storeKey:                   dexMiniKey,
			codespace:                  codespace,
			engines:                    make(map[string]*me.MatchEng),
			recentPrices:               make(map[string]*utils.FixedSizeRing, 256),
			allOrders:                  make(map[string]map[string]*OrderInfo, 256), // need to init the nested map when a new symbol added.
			OrderChangesMtx:            &sync.Mutex{},
			OrderChanges:               make(OrderChanges, 0),
			OrderInfosForPub:           make(OrderInfoForPublish),
			roundOrders:                make(map[string][]string, 256),
			roundIOCOrders:             make(map[string][]string, 256),
			RoundOrderFees:             make(map[string]*types.Fee, 256),
			poolSize:                   concurrency,
			cdc:                        cdc,
			FeeManager:                 NewFeeManager(cdc, dexMiniKey, logger),
			CollectOrderInfoForPublish: collectOrderInfoForPublish,
			logger:                     logger},
		make([]string, 0, 256),
		make(map[string]uint32, 256),
	}
}

// override
func (kp *MiniKeeper) AddEngine(pair dexTypes.TradingPair) *me.MatchEng {
	eng := kp.Keeper.AddEngine(pair)
	symbol := strings.ToUpper(pair.GetSymbol())
	kp.miniSymbolsHash[symbol] = crc32.ChecksumIEEE([]byte(symbol))
	return eng
}

// override
// please note if distributeTrade this method will work in async mode, otherwise in sync mode.
func (kp *MiniKeeper) matchAndDistributeTrades(distributeTrade bool, height, timestamp int64, matchAllMiniSymbols bool) ([]chan Transfer) {
	size := len(kp.roundOrders)
	// size is the number of pairs that have new orders, i.e. it should call match()
	if size == 0 {
		kp.logger.Info("No new orders for any pair, give up matching")
		return nil
	}

	concurrency := 1 << kp.poolSize
	tradeOuts := make([]chan Transfer, concurrency)

	if matchAllMiniSymbols {
		for symbol := range kp.roundOrders {
			kp.matchedMiniSymbols = append(kp.matchedMiniSymbols, symbol)
		}
	} else {
		kp.selectMiniSymbolsToMatch(height, func(miniSymbols map[string]struct{}) {
			for symbol := range miniSymbols {
				kp.matchedMiniSymbols = append(kp.matchedMiniSymbols, symbol)
			}
		})
	}

	if distributeTrade {
		ordNum := 0
		for _, perSymbol := range kp.matchedMiniSymbols {
			ordNum += len(perSymbol)
		}
		for i := range tradeOuts {
			//assume every new order would have 2 trades and generate 4 transfer
			tradeOuts[i] = make(chan Transfer, ordNum*4/concurrency)
		}
	}

	symbolCh := make(chan string, concurrency)
	producer := func() {
		for _, symbol := range kp.matchedMiniSymbols {
			symbolCh <- symbol
		}
		close(symbolCh)
	}
	matchWorker := func() {
		i := 0
		for symbol := range symbolCh {
			i++
			kp.matchAndDistributeTradesForSymbol(symbol, height, timestamp, kp.allOrders[symbol], distributeTrade, tradeOuts)
		}
	}

	if distributeTrade {
		utils.ConcurrentExecuteAsync(concurrency, producer, matchWorker, func() {
			for _, tradeOut := range tradeOuts {
				close(tradeOut)
			}
		})
	} else {
		utils.ConcurrentExecuteSync(concurrency, producer, matchWorker)
	}
	return tradeOuts
}

// override
func (kp *MiniKeeper) clearAfterMatch() {
	for _, symbol := range kp.matchedMiniSymbols {
		delete(kp.roundOrders, symbol)
		delete(kp.roundIOCOrders, symbol)
	}
	kp.matchedMiniSymbols = make([]string, 0, 256)
}

// MatchAndAllocateAll() is concurrently matching and allocating across
// all the symbols' order books, among all the clients
// Return whether match has been done in this height
func (kp *MiniKeeper) MatchAndAllocateAll(ctx sdk.Context, postAlloTransHandler TransferHandler, matchAllSymbols bool) {
	kp.logger.Debug("Start Matching for all...", "height", ctx.BlockHeader().Height, "symbolNum", len(kp.roundOrders))
	timestamp := ctx.BlockHeader().Time.UnixNano()
	tradeOuts := kp.matchAndDistributeTrades(true, ctx.BlockHeader().Height, timestamp, matchAllSymbols)
	if tradeOuts == nil {
		kp.logger.Info("No order comes in for the block")
	}

	totalFee := kp.allocateAndCalcFee(ctx, tradeOuts, postAlloTransHandler)
	fees.Pool.AddAndCommitFee("MATCH", totalFee)
	kp.clearAfterMatch()
}

// used by state sync to clear memory order book after we synced latest breathe block
//TODO check usage
func (kp *MiniKeeper) ClearOrders() {
	kp.Keeper.ClearOrders()
	kp.matchedMiniSymbols = make([]string, 0, 256)
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

func (kp *MiniKeeper) selectMiniSymbolsToMatch(height int64, postSelect func(map[string]struct{})) {
	symbolsToMatch := make(map[string]struct{}, 256)
	selectActiveMiniSymbols(&symbolsToMatch, &kp.roundOrders, defaultActiveMiniSymbolCount)
	selectMiniSymbolsRoundRobin(&symbolsToMatch, &kp.miniSymbolsHash, height)
	postSelect(symbolsToMatch)
}

func selectActiveMiniSymbols(symbolsToMatch *map[string]struct{}, roundOrdersMini *map[string][]string, k int) {
	//use quick select to select top k symbols
	symbolOrderNumsSlice := make([]*SymbolWithOrderNumber, 0, len(*roundOrdersMini))
	i := 0
	for symbol, orders := range *roundOrdersMini {
		symbolOrderNumsSlice[i] = &SymbolWithOrderNumber{symbol, len(orders)}
	}
	topKSymbolOrderNums := findTopKLargest(symbolOrderNumsSlice, k)

	for _, selected := range topKSymbolOrderNums {
		(*symbolsToMatch)[selected.symbol] = struct{}{}
	}
}

func selectMiniSymbolsRoundRobin(symbolsToMatch *map[string]struct{}, miniSymbolsHash *map[string]uint32, height int64) {
	m := height % defaultMiniBlockMatchInterval
	for symbol, symbolHash := range *miniSymbolsHash {
		if int64(symbolHash%defaultMiniBlockMatchInterval) == m {
			(*symbolsToMatch)[symbol] = struct{}{}
		}
	}
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
