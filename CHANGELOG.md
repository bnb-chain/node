# Changelog

## 0.5.4

IMPROVEMENTS

* [\#518](https://github.com/binance-chain/node/pull/518) [Gov] Adapt to changes in cosmos

BUG FIXES

* [\#502](https://github.com/binance-chain/node/pull/502) [MatchEngine] Fix order sequence in price level
* [\#500](https://github.com/binance-chain/node/pull/500) [Publish] Failed blocking should not be regarded as closed order
* [\#495](https://github.com/binance-chain/node/pull/495) [MatchEngine] Fully fill order might not be correctly removed in orderbook when two continuous orders fully filled.

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
