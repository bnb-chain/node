# Changelog
## Develop

FEATURES
* [\#606](https://github.com/binance-chain/node/pull/606) [Config] add upgrade config for bep19  
* [\#598](https://github.com/binance-chain/node/pull/598) [CLI] don't broadcast time lock related txs to blockchain by default
* [\#595](https://github.com/binance-chain/node/pull/595) [Pub] publish trade status
* [\#588](https://github.com/binance-chain/node/pull/588) [CLI] add offline option to all commands which are used to send transactions
* [\#577](https://github.com/binance-chain/node/pull/577) [Token] add time lock feature
* [\#575](https://github.com/binance-chain/node/pull/575) [Gov] add delist feature


IMPROVEMENTS
* [\#611](https://github.com/binance-chain/node/pull/611) [Match] add LastMatchHeight in match engine
* [\#610](https://github.com/binance-chain/node/pull/610) [Tools] add timelock store in tools
* [\#607](https://github.com/binance-chain/node/pull/607) [Pub] publish trade and single order update fee for match engine revise
* [\#600](https://github.com/binance-chain/node/pull/600) [Match] fee calculation change for revised match engine
* [\#593](https://github.com/binance-chain/node/pull/593) [Config] add store and msg types upgrade config
* [\#586](https://github.com/binance-chain/node/pull/586) [Deps] upgrade tendermint and fix compile error
* [\#576](https://github.com/binance-chain/node/pull/576) [Param] apply strict feeparam change proposal check
* [\#574](https://github.com/binance-chain/node/pull/574) [Deps] remove ledger tags from bnbchaind
* [\#573](https://github.com/binance-chain/node/pull/573) [Deps] remove indirect dependency btcd in gopkg.toml
* [\#571](https://github.com/binance-chain/node/pull/571) [CLI] make bnbcli to support ledger
* [\#568](https://github.com/binance-chain/node/pull/568) [StateSync] parity warp-like state sync

BUG FIXES
* [\#609](https://github.com/binance-chain/node/pull/609) [StateSync] fix for state sync snapshot command
* [\#605](https://github.com/binance-chain/node/pull/605) [Match] bugfix for fee calculation
* [\#604](https://github.com/binance-chain/node/pull/604) [Dex] remove concurrency when delist pair
* [\#603](https://github.com/binance-chain/node/pull/603) [Dex] hotfix statesync for paramhub change in breathe block is not loaded
* [\#584](https://github.com/binance-chain/node/pull/584) [Match] fixes for match engine

DEPENDENCIES

* [\#87](https://github.com/binance-chain/bnc-tendermint/pull/87) [tendermint] upgrade from 0301 to 0315