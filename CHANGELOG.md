# Changelog

## 0.5.10
BUG FIXES

* [\#602](https://github.com/binance-chain/node/pull/602) [StateSync] Fix paramhub change in breathe block is not loaded

## 0.5.9

IMPROVEMENTS
* [rpc] Updated tendermint to make process of websocket request async.

BUG FIXES

* [\#584](https://github.com/binance-chain/node/pull/584) [MatchEngine] Fix minor issues in allocation

## 0.5.7

* [\#560](https://github.com/binance-chain/node/pull/560) [Publish] Change published empty msg to error level
* [\#559](https://github.com/binance-chain/node/pull/559) [Tool] Add Snapshot tool
* [\#558](https://github.com/binance-chain/node/pull/558) [Testnet] Fix the output from testnet cmd 

## 0.5.6

IMPROVEMENTS
* [\#466](https://github.com/binance-chain/node/pull/466)  Recover from last running mode when restarts 
* [\#546](https://github.com/binance-chain/node/pull/546)[Upgrade] Set UpgradeConfig before all other initializations
* [\#545](https://github.com/binance-chain/node/pull/545)[Publish] Change order creatation time and lastupdate time to nanosecond
* [\#540](https://github.com/binance-chain/node/pull/540)[Validator] Modify validator query interface
* [\#535](https://github.com/binance-chain/node/pull/535) [Validator] Upgrade logic for splitting validator address
* [\#533](https://github.com/binance-chain/node/pull/533) [Publish] Include txhash in published transfers


## 0.5.5

IMPROVEMENTS

* [\#518](https://github.com/binance-chain/node/pull/518) [Gov] Adapt to changes in cosmos
* [\#521](https://github.com/binance-chain/node/pull/521) [List] Add check for list proposal hook.
* [\#517](https://github.com/binance-chain/node/pull/517) [Validator] Split fee address and operator address
* [\#516](https://github.com/binance-chain/node/pull/516) [Publish] IocNoFill semantic correct
* [\#514](https://github.com/binance-chain/node/pull/514) [Upgrade] Support config for upgrade height
* [\#509](https://github.com/binance-chain/node/pull/509) [MatchEngine] Make the lot size reasonable for low price
* [\#498](https://github.com/binance-chain/node/pull/498) [MatchEngine] Rename price of TradingPair to list_price
* [\#497](https://github.com/binance-chain/node/pull/497) [Build] Support `build-windows`
* [\#496](https://github.com/binance-chain/node/pull/476) [StateSync] Cache latest snapshot in memory
* [\#526](https://github.com/binance-chain/node/pull/518) [ApiServer] Add gov queries in api server


BUG FIXES

* [\#508](https://github.com/binance-chain/node/pull/508) [\#511](https://github.com/binance-chain/node/pull/511) [\#501](https://github.com/binance-chain/node/pull/501) [Dex] Fix all potential int64 overflows, remove all use of float64, and optimize some calculation
* [\#478](https://github.com/binance-chain/node/pull/478) [Publish] Dump order ids for large expire message.

## 0.5.4

IMPROVEMENTS

BUG FIXES

* [\#502](https://github.com/binance-chain/node/pull/502) [MatchEngine] Fix order sequence in price level
* [\#500](https://github.com/binance-chain/node/pull/500) [Publish] Failed blocking should not be regarded as closed order
* [\#495](https://github.com/binance-chain/node/pull/500) [MatchEngine] Fully fill order might not be correctly removed in orderbook when two continuous orders fully filled.


## 0.5.1

BREAKING CHANGES

FEATURES

IMPROVEMENTS

* [\#489](https://github.com/binance-chain/node/pull/489) Check the length of signer addresses

BUG FIXES

* [\#485](https://github.com/binance-chain/node/pull/485) Fix reporting error log when an order partially canceled 
* [\#486](https://github.com/binance-chain/node/pull/486) Fix publication fee error when there is trade and expire (IOC) for same address in same block
* [\#479](https://github.com/binance-chain/node/pull/479) Recover recent price to make sure tick and lot size calculation is consistent after state sync 
* [\#487](https://github.com/binance-chain/node/pull/487) Fix error log in order handler, and hide the internal context from the response


## 0.5.0

BREAKING CHANGES

FEATURES

IMPROVEMENTS

* [\#460](https://github.com/binance-chain/node/issues/460) Return explicit error msgs when listing trading pair.

BUG FIXES
