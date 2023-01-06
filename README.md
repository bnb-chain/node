BNB Beacon Chain
------------

BNB Beacon Chain is a blockchain with a flexible set of native assets and pluggable modules. It uses [tendermint](https://tendermint.com) for consensus and app logic is written in golang. It targets fast block times, a native dApp layer and multi-token support with no smart contract VM.

[![Reference](
https://camo.githubusercontent.com/915b7be44ada53c290eb157634330494ebe3e30a/68747470733a2f2f676f646f632e6f72672f6769746875622e636f6d2f676f6c616e672f6764646f3f7374617475732e737667
)](https://docs.bnbchain.world/docs/learn/beaconIntro)
[![Discord](https://img.shields.io/badge/discord-join%20chat-blue.svg)](https://discord.gg/z2VpC455eU)

Beacon Chain has the basic features of most blockchains:
- Sending and receiving BNB and digital assets
- Issuing new digital assets (we have a standard called BEP-2)
- Mint/burn, freeze/unfreeze, lock/unlock of digital assets

Besides, it has many other rich features:
- Staking/governance for both Beacon Chain and BNB Smart Chain.
- Cross chain communication.
- Atomic swap support.
- Support hot sync and state sync.

## Overview

* This uses BFT consensus so up to 1/3 of all validator nodes can be rogue or bad.
* Validator nodes are part of the "validator set" so they are known, trusted and controlled by the network.
* Full nodes are not validator nodes, but anyone can get a copy of the whole blockchain and validate it.
* No PoW means block times are very fast.
* UXTO/account does not matter as we just use the [cosmos](https://github.com/cosmos/cosmos-sdk/tree/master/x/bank) bank.
* Features like the DEX (deprecated now) will run directly on the node as apps written in golang.
  [Read](https://tendermint.readthedocs.io/en/master/introduction.html) [more](https://blog.cosmos.network/tendermint-explained-bringing-bft-based-pos-to-the-public-blockchain-domain-f22e274a0fdb) about Tendermint and ABCI.

## Getting Started

### Environment setup

#### Requirement
Go version above 1.17 is required.

Please [install it](https://go.dev/doc/install) or use brew on macOS: `brew install go`.

#### Build from Source

```bash
$ git clone git@github.com:bnb-chain/node.git 
$ cd node && make build
```


To test that installation worked, try to run the cli tool:

```bash
$./build/bnbchaind version
```

### Start the blockchain

This command will generate a keypair for your node and create the genesis block config:

```bash
$ ./build/bnbchaind init --moniker testnode
$ cat ~/.bnbchaind/config/genesis.json
```

You may want to check the [Issuing assets](#issuing-assets) section below before you start, but this is how to start the node and begin generating blocks:

```bash
$ ./build/bnbchaind start --moniker testnode
```

If everything worked you will see blocks being generated around every 1s in your console, like
```shell
I[2023-01-05|20:39:46.960] Starting ABCI with Tendermint                module=main 
I[2023-01-05|20:39:47.016] Loading order book snapshot from last breathe block module=main blockHeight=0
I[2023-01-05|20:39:47.016] No breathe block is ever saved. just created match engines for all the pairs. module=main 
I[2023-01-05|20:39:47.017] get last breathe block height                module=main height=0
I[2023-01-05|20:39:48.194] Executed block                               module=state height=1 validTxs=0 invalidTxs=0
I[2023-01-05|20:39:48.200] Committed state                              module=state height=1 txs=0 appHash=45AE480E42446C584BEFF2162941F4A76C542E96E44F878B21546DCC7E79DCE5
I[2023-01-05|20:39:49.194] Executed block                               module=state height=2 validTxs=0 invalidTxs=0
I[2023-01-05|20:39:49.198] Committed state                              module=state height=2 txs=0 appHash=6903AA785839B393C7E252C74185F42297E38ADD9804D8EE0AF08A3EF9D99080
I[2023-01-05|20:39:50.215] Executed block                               module=state height=3 validTxs=0 invalidTxs=0
```

### Reset

When you make a change you probably want to reset your chain, remember to kill the node first.

```bash
$ ./build/bnbchaind unsafe-reset-all
```

### Join mainnet/testnet

Please refer to the document for joining [mainnet](https://docs.bnbchain.world/docs/beaconchain/develop/node/join-mainnet) or [testnnet](https://docs.bnbchain.world/docs/beaconchain/develop/node/join-testnet).

## Assets

### Issuing assets

Assets may be issued through `bnbcli` while the blockchain is running; see here for an example:

```bash
$ chainId=`cat ~/.bnbchaind/config/genesis.json | jq .chain_id --raw-output`
# Input password "12345678"
$ ./build/bnbcli token issue --trust-node --symbol FBTC --token-name FunBitCoin  --total-supply  10000000000  --from testnode  --chain-id ${chainId}
```

This will post a transaction with an `IssueMsg` to the blockchain, which contains the data needed for token issuance.

### Checking a balance

Start your node, then list your keys as below:

```bash
$ ./build/bnbcli keys list 

NAME:   TYPE:   ADDRESS:                                                PUBKEY:
testnode        local   bnb1elavnu4uyzt43hw380vhw0m2zl7djz0xeac60t      bnbp1addwnpepqva5fmn4r4hc66fpqafwwdf20nq8xjpr3kezkclpmluufxdk5x4gw9xsln5
```

Check a balance with this command, e.g.:

```bash
$  ./build/bnbcli account bnb1elavnu4uyzt43hw380vhw0m2zl7djz0xeac60t --chain-id ${chainId} | jq
```

### Sending assets

You have to send a transaction to send assets to another address, which is possible with the cli tool:

Make sure `chain-id` is set correctly; you can find it in your `genesis.json`.

```bash
$ ./build/bnbcli send --to bnb1u5mvgkqt9rmj4fut60rnpqfv0a865pwnn90v9q --amount 100:FBTC-4C9  --from testnode  --chain-id ${chainId}

Password to sign with 'you': 12345678
Committed at block 833 (tx hash: 40C2B27B056A63DFE5BCE32709F160F53633C0EBEBBD05E1AC26419D35303765, response: {Code:0 Data:[] Log:Msg 0:  Info: GasWanted:0 GasUsed:0 Events:[{Type: Attributes:[{Key:[115 101 110 100 101 114] Value:[98 110 98 49 101 108 97 118 110 117 52 117 121 122 116 52 51 104 119 51 56 48 118 104 119 48 109 50 122 108 55 100 106 122 48 120 101 97 99 54 48 116] XXX_NoUnkeyedLiteral:{} XXX_unrecognized:[] XXX_sizecache:0} {Key:[114 101 99 105 112 105 101 110 116] Value:[98 110 98 49 117 53 109 118 103 107 113 116 57 114 109 106 52 102 117 116 54 48 114 110 112 113 102 118 48 97 56 54 53 112 119 110 110 57 48 118 57 113] XXX_NoUnkeyedLiteral:{} XXX_unrecognized:[] XXX_sizecache:0} {Key:[97 99 116 105 111 110] Value:[115 101 110 100] XXX_NoUnkeyedLiteral:{} XXX_unrecognized:[] XXX_sizecache:0}] XXX_NoUnkeyedLiteral:{} XXX_unrecognized:[] XXX_sizecache:0}] Codespace: XXX_NoUnkeyedLiteral:{} XXX_unrecognized:[] XXX_sizecache:0})
```

You can look at the contents of the tx, use the tx hash above:

```bash
$  ./build/bnbcli tx 40C2B27B056A63DFE5BCE32709F160F53633C0EBEBBD05E1AC26419D35303765 --chain-id ${chainId} 
```

Then you can check the balance of pepe's key to see that he now has 100 satoshi units of `FBTC-4C9`:

```bash
$ ./build/bnbcli account bnb1u5mvgkqt9rmj4fut60rnpqfv0a865pwnn90v9q  --chain-id ${chainId} | jq
{
  "type": "bnbchain/Account",
  "value": {
    "base": {
      "address": "bnb1u5mvgkqt9rmj4fut60rnpqfv0a865pwnn90v9q",
      "coins": [
        {
          "denom": "FBTC-4C9",
          "amount": "100"
        }
      ],
      ...
    },
    ...
  }
}
```

Amounts are represented as ints, and all coins have a fixed scale of 8. This means that if a balance of `100000000` were to be shown here, that would represent a balance of `1` coin.

## Contribution
It is welcomed to contribute to this repo from everyone. If you'd like to contribute, please fork, fix, commit and submit a pull request to review and merge into the main code base. Please make sure your contributions adhere to our coding guidelines:

- Code must adhere to the official Go formatting guidelines (i.e. please use gofmt tool).
- Code must be documented adhering to the official Go commentary guidelines.
- Pull requests need to be based on and opened against the master branch.