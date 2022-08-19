# Changelog

## 0.10.1
IMPROVEMENTS
* [\#882](https://github.com/bnb-chain/node/pull/882) [DEX] Add BEP151 Mainnet Height

## 0.10.0
IMPROVEMENTS
* [\#875](https://github.com/bnb-chain/node/pull/875) [DEX] Implement BEP151

## 0.9.2
IMPROVEMENTS
* [\#865](https://github.com/bnb-chain/node/pull/865) [CI] Build state recover tool in release job
* [\#869](https://github.com/bnb-chain/node/pull/869) [Deps] Upgrade cosmos-sdk to v0.25.0 binance.28

## 0.9.1
IMPROVEMENTS
* [\#839](https://github.com/bnb-chain/node/pull/839) [CLIENT] Crypto-level random for client
* [\#840](https://github.com/bnb-chain/node/pull/840) [Code] Remove testnet demo deploy script
* [\#841](https://github.com/bnb-chain/node/pull/841) [CLI] Add cli param to set DefaultKeyPass
* [\#842](https://github.com/bnb-chain/node/pull/842) [Deps] Upgrade go to 1.1.7
* [\#843](https://github.com/bnb-chain/node/pull/843) [Code] Remove useless code
* [\#844](https://github.com/bnb-chain/node/pull/844) [Build] Add go compiler build flags
* [\#845](https://github.com/bnb-chain/node/pull/845) [Code] Tidy todo comments
* [\#846](https://github.com/bnb-chain/node/pull/846) [Code] Change wording binance to bnb
* [\#847](https://github.com/bnb-chain/node/pull/847) [Code] Rename import path binance-chain with bnb-chain
* [\#852](https://github.com/bnb-chain/node/pull/852) [R4R] Replace ioutil with io/os
* [\#853](https://github.com/bnb-chain/node/pull/853) [R4R] Change file permissions
* [\#857](https://github.com/bnb-chain/node/pull/857) [CI] Add workflows to check tests and build release
* [\#858](https://github.com/bnb-chain/node/pull/858) [CI] Add linter workflow
* [\#859](https://github.com/bnb-chain/node/pull/859) [Deps] Fix checksum mismatch issue of btcd
* [\#860](https://github.com/bnb-chain/node/pull/860) [Deps] Upgrade tendermint to v0.32.3-binance.7; upgrade cosmos-sdk to v0.25.0 binance.27

## 0.9.0
IMPROVEMENTS
* [\#835](https://github.com/bnb-chain/node/pull/835) [Staking] Implement BEP128

## 0.8.3
IMPROVEMENTS
* [\#836](https://github.com/bnb-chain/node/pull/836) [Deps] Upgrade tendermint to v0.32.3-binance.6; upgrade cosmos-sdk to v0.25.0 binance.25


## 0.8.2
BUG FIXES
* [\#823](https://github.com/bnb-chain/node/pull/823) [Pub] return error when executing mirror or mirror sync request failed

## 0.8.1
FEATURES
* [\#809](https://github.com/bnb-chain/node/pull/809) [Token] Implement BEP84
* [\#810](https://github.com/bnb-chain/node/pull/810) [Token] Transfer ownership of BEP2/BEP8 token
* [\#811](https://github.com/bnb-chain/node/pull/811) [Token] change the symbol minimum length to 2
* [\#815](https://github.com/bnb-chain/node/pull/815) [Token] burn transaction sender token fix


## 0.8.0-hf.2
BUG FIXES
[sync] fix memory leak issue in hotsync

## 0.8.0-hf.1
BUG FIXES
[CLI] bnbcli API Server get token issue

## 0.8.0
FEATURES
[Stake] import stake module for side chain
[Slashing] import slashing module for side chain
[Token] support cross chain transfer

IMPROVEMENTS
[Pub] import pubsub server for publishing message

## 0.7.2-hf.1
BUG FIXES
* [\#766](https://github.com/bnb-chain/node/pull/766) [Dex] remove orderInfo from orderInfoForPub when publish anything

## 0.7.2
BUG FIXES
* [\#753](https://github.com/bnb-chain/node/pull/753) [\#760](https://github.com/bnb-chain/node/pull/760) [Dex] Delete recent price from db when delisting
* [\#758](https://github.com/bnb-chain/node/pull/758) [Dex] Force match all BEP2 symbols on BEP8 upgrade height to update last match height
* [\#762](https://github.com/bnb-chain/node/pull/762) [Dex] Fix mini msg
## 0.7.0
FEATURES
* [\#725](https://github.com/bnb-chain/node/pull/725) [Token] [Dex] BEP8 - Mini-BEP2 token features
* [\#710](https://github.com/bnb-chain/node/pull/710) [DEX] BEP70 - Support busd pair listing and trading

IMPROVEMENTS
* [\#704](https://github.com/bnb-chain/node/pull/704) [DEX] BEP67 Price-based Order Expiration
* [\#714](https://github.com/bnb-chain/node/pull/714) [DEX] Add pendingMatch flag to orderbook query response

## 0.6.3-hf.1

BUG FIXES
* [\#693](https://github.com/bnb-chain/node/pull/693) [Deps] hot fix for hard fork in stdTx getSigner

## 0.6.3

BUG FIXES
* [\#677](https://github.com/bnb-chain/node/pull/677) [Dex] fix account may have currency with zero balance

IMPROVEMENTS
* [\#672](https://github.com/bnb-chain/node/pull/672) [DEX] Change listing rule
* [\#666](https://github.com/bnb-chain/node/pull/666) [Deps] Upgrade tendermint to 0.32.3
* [\#667](https://github.com/bnb-chain/node/pull/667) [Pub] publish block info for audit
* [\#686](https://github.com/bnb-chain/node/pull/686) [Pub] expose kafka version in publisher setting

## 0.6.2-hf.1

BUG FIXES
Bump Tendermint version to v0.31.5-binance.3 to address p2p panic errors.

## 0.6.2

FEATURES
* [\#634](https://github.com/bnb-chain/node/pull/634) [Token] BEP3 - Atomic swap

IMPROVEMENTS
* [\#638](https://github.com/bnb-chain/node/pull/638) [Pub] BEP39 - add memo to transfer kafka message
* [\#639](https://github.com/bnb-chain/node/pull/639) [ABCI] add levels parameter to depth ABCI query
* [\#643](https://github.com/bnb-chain/node/pull/643) [TOOL] tools: state_recover add index height rollback
* [\#637](https://github.com/bnb-chain/node/pull/637) [CLI] add account flag check for enable command and disable command

BUG FIXES
* [\#641](https://github.com/bnb-chain/node/pull/641) [Dex] add max lock time in time lock plugin
* [\#633]( https://github.com/bnb-chain/node/pull/633) [CLI] fix offline mode issue for sending order
* [\#651]( https://github.com/bnb-chain/node/pull/651) [API] add account flag in api-server account query response

## 0.6.1-hf.3
BUG FIXES
* [\#654](https://github.com/bnb-chain/node/pull/654) [Dex] fix can't bring bnbchaind back when there is an order whose symbol is lower case

## 0.6.1-hf.2
BUG FIXES
* [\#641](https://github.com/bnb-chain/node/pull/641) [Dex] Add max lock time in time lock plugin

IMPROVEMENTS
* [\#638](https://github.com/bnb-chain/node/pull/638) [Pub] BEP39 - add memo to transfer kafka message
* [\#639](https://github.com/bnb-chain/node/pull/639) [ABCI] add levels parameter to depth ABCI query

## 0.6.1-hf.1
BUG FIXES
* [\#635](https://github.com/bnb-chain/node/pull/635) fix panic in pre-check is not recovered

## 0.6.1
FEATURES
* [\#605](https://github.com/bnb-chain/node/pull/605) [Account] accounts can set flags to turn on memo validation

## 0.6.0
FEATURES
* [\#598](https://github.com/bnb-chain/node/pull/598) [CLI] don't broadcast time lock related txs to blockchain by default
* [\#595](https://github.com/bnb-chain/node/pull/595) [Pub] publish trade status
* [\#588](https://github.com/bnb-chain/node/pull/588) [CLI] add offline option to all commands which are used to send transactions
* [\#577](https://github.com/bnb-chain/node/pull/577) [Token] add time lock feature
* [\#575](https://github.com/bnb-chain/node/pull/575) [Gov] add delist feature
* [\#580](https://github.com/bnb-chain/node/pull/580) [\#606](https://github.com/bnb-chain/node/pull/580) [Match] match engine revision

IMPROVEMENTS
* [\#611](https://github.com/bnb-chain/node/pull/611) [Match] add LastMatchHeight in match engine
* [\#610](https://github.com/bnb-chain/node/pull/610) [Tools] add timelock store in tools
* [\#607](https://github.com/bnb-chain/node/pull/607) [Pub] publish trade and single order update fee for match engine revise
* [\#600](https://github.com/bnb-chain/node/pull/600) [Match] fee calculation change for revised match engine
* [\#593](https://github.com/bnb-chain/node/pull/593) [Config] add store and msg types upgrade config
* [\#586](https://github.com/bnb-chain/node/pull/586) [Deps] upgrade tendermint
* [\#576](https://github.com/bnb-chain/node/pull/576) [Param] apply strict feeparam change proposal check
* [\#574](https://github.com/bnb-chain/node/pull/574) [Deps] remove ledger tags from bnbchaind
* [\#573](https://github.com/bnb-chain/node/pull/573) [Deps] remove indirect dependency btcd in gopkg.toml
* [\#571](https://github.com/bnb-chain/node/pull/571) [CLI] make bnbcli to support ledger
* [\#568](https://github.com/bnb-chain/node/pull/568) [StateSync] parity warp-like state sync

BUG FIXES
* [\#609](https://github.com/bnb-chain/node/pull/609) [StateSync] fix for state sync snapshot command
* [\#603](https://github.com/bnb-chain/node/pull/603) [Dex] hotfix statesync for paramhub change in breathe block is not loaded

DEPENDENCIES

* [\#87](https://github.com/bnb-chain/bnc-tendermint/pull/87) [tendermint] upgrade from v0.30.1 to 0.31.5

## 0.5.10
BUG FIXES

* [\#602](https://github.com/bnb-chain/node/pull/602) [StateSync] Fix paramhub change in breathe block is not loaded

## 0.5.9

IMPROVEMENTS
* [rpc] Updated tendermint to make process of websocket request async.

BUG FIXES

* [\#584](https://github.com/bnb-chain/node/pull/584) [MatchEngine] Fix minor issues in allocation

## 0.5.7

* [\#560](https://github.com/bnb-chain/node/pull/560) [Publish] Change published empty msg to error level
* [\#559](https://github.com/bnb-chain/node/pull/559) [Tool] Add Snapshot tool
* [\#558](https://github.com/bnb-chain/node/pull/558) [Testnet] Fix the output from testnet cmd

## 0.5.6

IMPROVEMENTS
* [\#466](https://github.com/bnb-chain/node/pull/466)  Recover from last running mode when restarts
* [\#546](https://github.com/bnb-chain/node/pull/546) [Upgrade] Set UpgradeConfig before all other initializations
* [\#545](https://github.com/bnb-chain/node/pull/545) [Publish] Change order creatation time and lastupdate time to nanosecond
* [\#540](https://github.com/bnb-chain/node/pull/540) [Validator] Modify validator query interface
* [\#535](https://github.com/bnb-chain/node/pull/535) [Validator] Upgrade logic for splitting validator address
* [\#533](https://github.com/bnb-chain/node/pull/533) [Publish] Include txhash in published transfers


## 0.5.5

IMPROVEMENTS

* [\#518](https://github.com/bnb-chain/node/pull/518) [Gov] Adapt to changes in cosmos
* [\#521](https://github.com/bnb-chain/node/pull/521) [List] Add check for list proposal hook.
* [\#517](https://github.com/bnb-chain/node/pull/517) [Validator] Split fee address and operator address
* [\#516](https://github.com/bnb-chain/node/pull/516) [Publish] IocNoFill semantic correct
* [\#514](https://github.com/bnb-chain/node/pull/514) [Upgrade] Support config for upgrade height
* [\#509](https://github.com/bnb-chain/node/pull/509) [MatchEngine] Make the lot size reasonable for low price
* [\#498](https://github.com/bnb-chain/node/pull/498) [MatchEngine] Rename price of TradingPair to list_price
* [\#497](https://github.com/bnb-chain/node/pull/497) [Build] Support `build-windows`
* [\#496](https://github.com/bnb-chain/node/pull/476) [StateSync] Cache latest snapshot in memory
* [\#526](https://github.com/bnb-chain/node/pull/518) [ApiServer] Add gov queries in api server


BUG FIXES

* [\#508](https://github.com/bnb-chain/node/pull/508) [\#511](https://github.com/bnb-chain/node/pull/511) [\#501](https://github.com/bnb-chain/node/pull/501) [Dex] Fix all potential int64 overflows, remove all use of float64, and optimize some calculation
* [\#478](https://github.com/bnb-chain/node/pull/478) [Publish] Dump order ids for large expire message.

## 0.5.4

IMPROVEMENTS

BUG FIXES

* [\#502](https://github.com/bnb-chain/node/pull/502) [MatchEngine] Fix order sequence in price level
* [\#500](https://github.com/bnb-chain/node/pull/500) [Publish] Failed blocking should not be regarded as closed order
* [\#495](https://github.com/bnb-chain/node/pull/495) [MatchEngine] Fully fill order might not be correctly removed in orderbook when two continuous orders fully filled.

## 0.5.1

BREAKING CHANGES

FEATURES

IMPROVEMENTS

* [\#489](https://github.com/bnb-chain/node/pull/489) Check the length of signer addresses

BUG FIXES

* [\#485](https://github.com/bnb-chain/node/pull/485) Fix reporting error log when an order partially canceled
* [\#486](https://github.com/bnb-chain/node/pull/486) Fix publication fee error when there is trade and expire (IOC) for same address in same block
* [\#479](https://github.com/bnb-chain/node/pull/479) Recover recent price to make sure tick and lot size calculation is consistent after state sync
* [\#487](https://github.com/bnb-chain/node/pull/487) Fix error log in order handler, and hide the internal context from the response


## 0.5.0

BREAKING CHANGES

FEATURES

IMPROVEMENTS

* [\#460](https://github.com/bnb-chain/node/issues/460) Return explicit error msgs when listing trading pair.

BUG FIXES
