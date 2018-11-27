# Local node start script

This script is used for E2E testing wallets and other external features. You need a working go v1.11 installation for this to work.
The script will clone the repository to your `GOPATH`, build it, and start a working single-node blockchain and `api-server`.

The bash script will set the following variables, which can be used if you use `source ./networks/local/start_local_node.sh`:
* `alice_addr` - the address holding the BNB coins.
* `alice_secret` - the mnemonic for Alice's wallet (wallet 1).
* `bob_addr` - an address that can be used to receive coins, make trades, etc.
* `bob_secret` - the mnemonic for Bob's wallet (wallet 2).

The following coins are issued and active on the blockchain:
* `BNB` - the native chain token for paying fees.
* `NNB` - a test token for transfers and making trades.
