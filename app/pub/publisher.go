package pub

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	tmlog "github.com/tendermint/tendermint/libs/log"

	"github.com/binance-chain/node/app/config"
	"github.com/binance-chain/node/app/pub/sub"
	orderPkg "github.com/binance-chain/node/plugins/dex/order"
)

const (
	// TODO(#66): revisit the setting / whole thread model here,
	// do we need better way to make main thread less possibility to block
	TransferCollectionChannelSize = 4000
	ToRemoveOrderIdChannelSize    = 1000
	MaxOrderBookLevel             = 100
)

type OrderSymbolId struct {
	Symbol string
	Id     string
}

var (
	Logger            tmlog.Logger
	Cfg               *config.PublicationConfig
	ToPublishCh       chan BlockInfoToPublish
	ToRemoveOrderIdCh chan OrderSymbolId // order symbol and ids to remove from keeper.OrderInfoForPublish
	IsLive            bool

	ToPublishEventCh chan *sub.ToPublishEvent
)

type MarketDataPublisher interface {
	publish(msg AvroOrJsonMsg, tpe msgType, height int64, timestamp int64)
	Stop()
}

func PublishEvent(
	publisher MarketDataPublisher,
	Logger tmlog.Logger,
	cfg *config.PublicationConfig,
	ToPublishEventCh <-chan *sub.ToPublishEvent) {
	for toPublish := range ToPublishEventCh {
		eventData := toPublish.EventData
		Logger.Debug("publisher queue status", "size", len(ToPublishCh))
		if cfg.PublishStaking {
			var msgNum int
			var validators []*Validator
			var removedValidators map[string][]sdk.ValAddress
			var delegationsMap map[string][]*Delegation
			var ubdsMap map[string][]*UnbondingDelegation
			var redsMap map[string][]*ReDelegation
			var completedUBDsMap map[string][]*CompletedUnbondingDelegation
			var completedREDsMap map[string][]*CompletedReDelegation
			if eventData.StakeData != nil {
				if len(eventData.StakeData.Validators) > 0 {
					validators = make([]*Validator, len(eventData.StakeData.Validators), len(eventData.StakeData.Validators))
					msgNum += len(eventData.StakeData.Validators)
					var i int
					for _, val := range eventData.StakeData.Validators {
						v := Validator(val)
						validators[i] = &v
						i++
					}
				}
				if len(eventData.StakeData.RemovedValidators) > 0 {
					removedValidators = make(map[string][]sdk.ValAddress)
					for chainId, removedVals := range eventData.StakeData.RemovedValidators {
						vals := make([]sdk.ValAddress, len(removedVals), len(removedVals))
						msgNum += len(removedVals)
						var i int
						for _, val := range removedVals {
							vals[i] = val
							i++
						}
						removedValidators[chainId] = vals
					}
				}
				if len(eventData.StakeData.Delegations) > 0 || len(eventData.StakeData.RemovedDelegations) > 0 {
					delegationsMap = make(map[string][]*Delegation)
					for chainId, dels := range eventData.StakeData.Delegations {
						delegations := make([]*Delegation, len(dels), len(dels))
						msgNum += len(dels)
						var i int
						for _, del := range dels {
							d := Delegation(del)
							delegations[i] = &d
							i++
						}
						delegationsMap[chainId] = delegations
					}

					for chainId, removedDels := range eventData.StakeData.RemovedDelegations {
						if delegationsMap[chainId] == nil {
							delegationsMap[chainId] = make([]*Delegation, 0)
						}
						msgNum += len(removedDels)
						for _, dvPair := range removedDels {
							d := Delegation{
								DelegatorAddr: dvPair.DelegatorAddr,
								ValidatorAddr: dvPair.ValidatorAddr,
								Shares:        sdk.ZeroDec(),
							}
							delegationsMap[chainId] = append(delegationsMap[chainId], &d)
						}

					}
				}
				if len(eventData.StakeData.UnbondingDelegations) > 0 {
					ubdsMap = make(map[string][]*UnbondingDelegation)
					for chainId, ubds := range eventData.StakeData.UnbondingDelegations {
						unbondingDelegations := make([]*UnbondingDelegation, len(ubds), len(ubds))
						msgNum += len(ubds)
						var i int
						for _, ubd := range ubds {
							u := UnbondingDelegation(ubd)
							unbondingDelegations[i] = &u
							i++
						}
						ubdsMap[chainId] = unbondingDelegations
					}
				}
				if len(eventData.StakeData.ReDelegations) > 0 {
					redsMap = make(map[string][]*ReDelegation)
					for chainId, reds := range eventData.StakeData.ReDelegations {
						redelgations := make([]*ReDelegation, len(reds), len(reds))
						msgNum += len(reds)
						var i int
						for _, red := range reds {
							r := ReDelegation(red)
							redelgations[i] = &r
							i++
						}
						redsMap[chainId] = redelgations
					}
				}
				if len(eventData.StakeData.CompletedUBDs) > 0 {
					completedUBDsMap = make(map[string][]*CompletedUnbondingDelegation)
					for chainId, ubds := range eventData.StakeData.CompletedUBDs {
						comUBDs := make([]*CompletedUnbondingDelegation, len(ubds), len(ubds))
						msgNum += len(ubds)
						for i, ubd := range ubds {
							comUBDs[i] = &CompletedUnbondingDelegation{
								Validator: ubd.Validator,
								Delegator: ubd.Delegator,
								Amount:    Coin{Denom: ubd.Amount.Denom, Amount: ubd.Amount.Amount},
							}
						}
						completedUBDsMap[chainId] = comUBDs
					}
				}
				if len(eventData.StakeData.CompletedREDs) > 0 {
					completedREDsMap = make(map[string][]*CompletedReDelegation)
					for chainId, reds := range eventData.StakeData.CompletedREDs {
						comREDs := make([]*CompletedReDelegation, len(reds), len(reds))
						msgNum += len(reds)
						for i, red := range reds {
							comREDs[i] = &CompletedReDelegation{
								Delegator:    red.DelegatorAddr,
								ValidatorSrc: red.ValidatorSrcAddr,
								ValidatorDst: red.ValidatorDstAddr,
							}
						}
						completedREDsMap[chainId] = comREDs
					}
				}
				if msgNum > 0 {
					msg := StakingMsg{
						NumOfMsgs: msgNum,
						Height:    toPublish.Height,
						Timestamp: toPublish.Timestamp.Unix(),

						Validators:           validators,
						RemovedValidators:    removedValidators,
						Delegations:          delegationsMap,
						UnbondingDelegations: ubdsMap,
						ReDelegations:        redsMap,
						CompletedUBDs:        completedUBDsMap,
						CompletedREDs:        completedREDsMap,
					}
					publisher.publish(&msg, stakingTpe, toPublish.Height, toPublish.Timestamp.UnixNano())
				}
			}

		}

		if cfg.PublishDistributeReward {
			var msgNum int
			distributions := make(map[string][]*Distribution)
			for chainId, disData := range eventData.StakeData.Distribution {
				dis := make([]*Distribution, len(disData), len(disData))
				for i, disData := range disData {
					rewards := make([]*Reward, len(disData.Rewards), len(disData.Rewards))
					for i, reward := range disData.Rewards {
						delegatorTokens, err := sdk.MulQuoDec(disData.ValTokens, reward.Shares, disData.ValShares)
						if err != nil {
							Logger.Error("error convert shares to tokens, delegator: %s", reward.AccAddr)
							continue
						}
						rewardMsg := &Reward{
							Delegator: reward.AccAddr,
							Amount:    reward.Amount,
							Tokens:    delegatorTokens.RawInt(),
						}
						rewards[i] = rewardMsg
					}
					dis[i] = &Distribution{
						Validator:     disData.Validator,
						SelfDelegator: disData.SelfDelegator,
						ValTokens:     disData.ValTokens.RawInt(),
						TotalReward:   disData.TotalReward.RawInt(),
						Commission:    disData.Commission.RawInt(),
						Rewards:       rewards,
					}
					msgNum += len(disData.Rewards)
				}
				distributions[chainId] = dis
			}

			if len(distributions) > 0 {
				distributionMsg := DistributionMsg{
					NumOfMsgs:     msgNum,
					Height:        toPublish.Height,
					Timestamp:     toPublish.Timestamp.UnixNano(),
					Distributions: distributions,
				}
				publisher.publish(&distributionMsg, distributionTpe, toPublish.Height, toPublish.Timestamp.UnixNano())
			}
		}

		if cfg.PublishSlashing {
			var msgNum int
			slashData := make(map[string][]*Slash)
			for chainId, slashes := range eventData.SlashData {
				slashDataPerChain := make([]*Slash, len(slashes), len(slashes))
				for i, slash := range slashes {
					slashDataPerChain[i] = &Slash{
						Validator:        slash.Validator,
						InfractionType:   slash.InfractionType,
						InfractionHeight: slash.InfractionHeight,
						JailUtil:         slash.JailUtil.Unix(),
						SlashAmount:      slash.SlashAmount,
						Submitter:        slash.Submitter,
						SubmitterReward:  slash.SubmitterReward,
					}
					msgNum++
				}
				slashData[chainId] = slashDataPerChain
			}
			if msgNum > 0 {
				slashMsg := SlashMsg{
					NumOfMsgs: msgNum,
					Height:    toPublish.Height,
					Timestamp: toPublish.Timestamp.Unix(),
					SlashData: slashData,
				}
				publisher.publish(&slashMsg, slashingTpe, toPublish.Height, toPublish.Timestamp.UnixNano())
			}
		}
	}
}

func Publish(
	publisher MarketDataPublisher,
	metrics *Metrics,
	Logger tmlog.Logger,
	cfg *config.PublicationConfig,
	ToPublishCh <-chan BlockInfoToPublish) {
	var lastPublishedTime time.Time
	for marketData := range ToPublishCh {
		Logger.Debug("publisher queue status", "size", len(ToPublishCh))
		if metrics != nil {
			metrics.PublicationQueueSize.Set(float64(len(ToPublishCh)))
		}

		publishTotalTime := Timer(Logger, fmt.Sprintf("publish market data, height=%d", marketData.height), func() {
			// Implementation note: publication order are important here,
			// DEX query service team relies on the fact that we publish orders before trades so that
			// they can assign buyer/seller address into trade before persist into DB
			var opensToPublish []*Order
			var closedToPublish []*Order
			var feeToPublish map[string]string

			opensToPublish, closedToPublish, feeToPublish = collectOrdersToPublish(
				marketData.tradesToPublish,
				marketData.orderChanges,
				marketData.orderInfos,
				marketData.feeHolder,
				marketData.timestamp)
			addClosedOrder(closedToPublish, ToRemoveOrderIdCh)

			// ToRemoveOrderIdCh would be only used in production code
			// will be nil in mock (pressure testing, local publisher) and test code
			if ToRemoveOrderIdCh != nil {
				close(ToRemoveOrderIdCh)
			}

			ordersToPublish := append(opensToPublish, closedToPublish...)

			if cfg.PublishOrderUpdates {
				duration := Timer(Logger, "publish all orders", func() {
					publishExecutionResult(
						publisher,
						marketData.height,
						marketData.timestamp,
						ordersToPublish,
						marketData.tradesToPublish,
						marketData.proposalsToPublish,
						marketData.stakeUpdates)
				})

				if metrics != nil {
					metrics.NumTrade.Set(float64(len(marketData.tradesToPublish)))
					metrics.NumOrder.Set(float64(len(ordersToPublish)))
					metrics.PublishTradeAndOrderTimeMs.Set(float64(duration))
				}
			}

			if cfg.PublishAccountBalance {
				duration := Timer(Logger, "publish all changed accounts", func() {
					publishAccount(publisher, marketData.height, marketData.timestamp, marketData.accounts, feeToPublish)
				})

				if metrics != nil {
					metrics.NumAccounts.Set(float64(len(marketData.accounts)))
					metrics.PublishAccountTimeMs.Set(float64(duration))
				}
			}

			if cfg.PublishOrderBook {
				var changedPrices = make(orderPkg.ChangedPriceLevelsMap)
				duration := Timer(Logger, "prepare order books to publish", func() {
					changedPrices = filterChangedOrderBooksByOrders(ordersToPublish, marketData.latestPricesLevels)
				})
				if metrics != nil {
					numOfChangedPrices := 0
					for _, changedPrice := range changedPrices {
						numOfChangedPrices += len(changedPrice.Buys)
						numOfChangedPrices += len(changedPrice.Sells)
					}
					metrics.NumOrderBook.Set(float64(numOfChangedPrices))
					metrics.CollectOrderBookTimeMs.Set(float64(duration))
				}

				duration = Timer(Logger, "publish changed order books", func() {
					publishOrderBookDelta(publisher, marketData.height, marketData.timestamp, changedPrices)
				})

				if metrics != nil {
					metrics.PublishOrderbookTimeMs.Set(float64(duration))
				}
			}

			if cfg.PublishBlockFee {
				duration := Timer(Logger, "publish blockfee", func() {
					publishBlockFee(publisher, marketData.height, marketData.timestamp, marketData.blockFee)
				})

				if metrics != nil {
					metrics.PublishBlockfeeTimeMs.Set(float64(duration))
				}
			}

			if cfg.PublishTransfer {
				duration := Timer(Logger, "publish transfers", func() {
					publishTransfers(publisher, marketData.height, marketData.timestamp, marketData.transfers)
				})
				if metrics != nil {
					metrics.NumTransfers.Set(float64(len(marketData.transfers.Transfers)))
					metrics.PublishTransfersTimeMs.Set(float64(duration))
				}
			}

			if cfg.PublishBlock {
				duration := Timer(Logger, "publish block", func() {
					publishBlock(publisher, marketData.height, marketData.timestamp, marketData.block)
				})
				if metrics != nil {
					metrics.PublishBlockTimeMs.Set(float64(duration))
				}
			}

			if metrics != nil {
				metrics.PublicationHeight.Set(float64(marketData.height))
				blockInterval := time.Since(lastPublishedTime)
				lastPublishedTime = time.Now()
				metrics.PublicationBlockIntervalMs.Set(float64(blockInterval.Nanoseconds() / int64(time.Millisecond)))
			}
		})

		if metrics != nil {
			metrics.PublishTotalTimeMs.Set(float64(publishTotalTime))
		}
	}
}

func addClosedOrder(closedToPublish []*Order, toRemoveOrderIdCh chan OrderSymbolId) {
	if toRemoveOrderIdCh != nil {
		for _, o := range closedToPublish {
			Logger.Debug(
				"going to delete order from order changes map",
				"orderId", o.OrderId, "status", o.Status)
			toRemoveOrderIdCh <- OrderSymbolId{o.Symbol, o.OrderId}
		}
	}
}

func Stop(publisher MarketDataPublisher) {
	if IsLive == false {
		Logger.Error("publication module has already been stopped")
		return
	}

	IsLive = false

	close(ToPublishCh)
	if ToRemoveOrderIdCh != nil {
		close(ToRemoveOrderIdCh)
	}

	publisher.Stop()
}

func publishExecutionResult(publisher MarketDataPublisher, height int64, timestamp int64, os []*Order, tradesToPublish []*Trade, proposalsToPublish *Proposals, stakeUpdates *StakeUpdates) {
	numOfOrders := len(os)
	numOfTrades := len(tradesToPublish)
	numOfProposals := proposalsToPublish.NumOfMsgs
	numOfStakeUpdatedAccounts := stakeUpdates.NumOfMsgs
	executionResultsMsg := ExecutionResults{Height: height, Timestamp: timestamp, NumOfMsgs: numOfTrades + numOfOrders + numOfProposals + numOfStakeUpdatedAccounts}
	if numOfOrders > 0 {
		executionResultsMsg.Orders = Orders{numOfOrders, os}
	}
	if numOfTrades > 0 {
		executionResultsMsg.Trades = trades{numOfTrades, tradesToPublish}
	}
	if numOfProposals > 0 {
		executionResultsMsg.Proposals = *proposalsToPublish
	}
	if numOfStakeUpdatedAccounts > 0 {
		executionResultsMsg.StakeUpdates = *stakeUpdates
	}

	publisher.publish(&executionResultsMsg, executionResultTpe, height, timestamp)
}

func publishAccount(publisher MarketDataPublisher, height int64, timestamp int64, accountsToPublish map[string]Account, feeToPublish map[string]string) {
	numOfMsgs := len(accountsToPublish)

	idx := 0
	accs := make([]Account, numOfMsgs, numOfMsgs)
	for _, acc := range accountsToPublish {
		if fee, ok := feeToPublish[acc.Owner]; ok {
			acc.Fee = fee
		}
		accs[idx] = acc
		idx++
	}
	accountsMsg := Accounts{height, numOfMsgs, accs}

	publisher.publish(&accountsMsg, accountsTpe, height, timestamp)
}

func publishOrderBookDelta(publisher MarketDataPublisher, height int64, timestamp int64, changedPriceLevels orderPkg.ChangedPriceLevelsMap) {
	var deltas []OrderBookDelta
	for pair, pls := range changedPriceLevels {
		buys := make([]PriceLevel, len(pls.Buys), len(pls.Buys))
		sells := make([]PriceLevel, len(pls.Sells), len(pls.Sells))
		idx := 0
		for price, qty := range pls.Buys {
			buys[idx] = PriceLevel{price, qty}
			idx++
		}
		idx = 0
		for price, qty := range pls.Sells {
			sells[idx] = PriceLevel{price, qty}
			idx++
		}
		deltas = append(deltas, OrderBookDelta{pair, buys, sells})
	}

	books := Books{height, timestamp, len(deltas), deltas}

	publisher.publish(&books, booksTpe, height, timestamp)
}

func publishBlockFee(publisher MarketDataPublisher, height, timestamp int64, blockFee BlockFee) {
	publisher.publish(blockFee, blockFeeTpe, height, timestamp)
}

func publishTransfers(publisher MarketDataPublisher, height, timestamp int64, transfers *Transfers) {
	if transfers != nil {
		publisher.publish(transfers, transferTpe, height, timestamp)
	}
}

func publishBlock(publisher MarketDataPublisher, height, timestamp int64, block *Block) {
	if block != nil {
		publisher.publish(block, blockTpe, height, timestamp)
	}
}

func Timer(logger tmlog.Logger, description string, op func()) (durationMs int64) {
	start := time.Now()
	op()
	durationMs = time.Since(start).Nanoseconds() / int64(time.Millisecond)
	logger.Debug(description, "durationMs", durationMs)
	return durationMs
}
