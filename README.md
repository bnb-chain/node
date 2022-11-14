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

It has DEX and trading-specific functionality:
- Propose exchange listing for trading pairs
- Creating maker/taker orders for traders
- Listing assets from other chains using atomic swaps (BEP-3)

## Overview

* This uses BFT consensus so up to 1/3 of all validator nodes can be rogue or bad.
* Validator nodes are part of the "validator set" so they are known, trusted and controlled by the network.
* Full nodes are not validator nodes, but anyone can get a copy of the whole blockchain and validate it.
* No PoW means block times are very fast.
* UXTO/account does not matter as we just use the [cosmos](https://github.com/cosmos/cosmos-sdk/tree/master/x/bank) bank.
* Features like the DEX will run directly on the node as apps written in golang.

[Read](https://tendermint.readthedocs.io/en/master/introduction.html) [more](https://blog.cosmos.network/tendermint-explained-bringing-bft-based-pos-to-the-public-blockchain-domain-f22e274a0fdb) about Tendermint and ABCI.

## Getting Started

### Environment setup

If you do not have golang yet, please [install it](https://golang.org/dl) or use brew on macOS: `brew install go` and `brew install dep`.


**Mac & Linux**

```bash
$ export GOPATH=~/go
$ export PATH=~/go/bin:$PATH
$ export BNBCHAINPATH=~/go/src/github.com/bnb-chain/node
$ mkdir -p $BNBCHAINPATH
$ git clone git@github.com:bnb-chain/node.git $BNBCHAINPATH
$ cd $BNBCHAINPATH
$ make install
```

**Windows**

If you are working on windows, `GOPATH` and `PATH` should already be set when you install golang.
You may need add BNBCHAINPATH to the environment variables.

```bat
> md %BNBCHAINPATH%
> git clone git@github.com:bnb-chain/node.git %BNBCHAINPATH%
> cd %BNBCHAINPATH%
> make install
```

To test that installation worked, try to run the cli tool:

```bash
$ bnbcli
```

### Start the blockchain

This command will generate a keypair for your node and create the genesis block config:

```bash
$ bnbchaind init
$ cat ~/.bnbchaind/config/genesis.json
```

You may want to check the [Issuing assets](#issuing-assets) section below before you start, but this is how to start the node and begin generating blocks:

```bash
$ bnbchaind start --moniker ${YOURNAME}
```

If everything worked you will see blocks being generated around every 1s in your console.

### Reset

When you make a change you probably want to reset your chain, remember to kill the node first.

```bash
$ bnbchaind unsafe_reset_all
```

### Join mainnet/testnet

Please refer to the document for joining [mainnet](https://docs.bnbchain.world/docs/beaconchain/develop/node/join-mainnet) or [testnnet](https://docs.bnbchain.world/docs/beaconchain/develop/node/join-testnet).

## Assets

### Issuing assets

Assets may be issued through `bnbcli` while the blockchain is running; see here for an example:

```bash
$ bnbcli tokens issue bnb -n bnb -s 100000
```

This will post a transaction with an `IssueMsg` to the blockchain, which contains the data needed for token issuance.

### Checking a balance

Start your node, then list your keys as below:

```bash
$ bnbcli keys list
All keys:
pepe    B71E119324558ABA3AE3F5BC854F1225132465A0
you     DEBF30B59A5CD0111FDF4F86664BC063BF450A1A
```

Check a balance with this command, e.g.:

```bash
$ bnbcli account DEBF30B59A5CD0111FDF4F86664BC063BF450A1A
```

Alternatively through http when `bnbcli api-server` is running. Amounts are returned as decimal numbers in strings.

```bash
$ curl -s http://localhost:8080/balances/cosmosaccaddr173hyu6dtfkrj9vujjhvz2ayehrng64rxq3h4yp | json_pp
{
   "address" : "cosmosaccaddr173hyu6dtfkrj9vujjhvz2ayehrng64rxq3h4yp",
   "balances" : [
      {
         "symbol" : "BNB",
         "free" : "2.00000000",
         "locked" : "0.00000000",
         "frozen" : "0.00000000"
      },
      {
         "symbol" : "XYZ",
         "free" : "0.98999900",
         "locked" : "0.00000100",
         "frozen" : "0.00000000"
      }
   ]
}
```

### Sending assets

You have to send a transaction to send assets to another address, which is possible with the cli tool:

Make sure `chain-id` is set correctly; you can find it in your `genesis.json`.

```bash
$ bnbcli send --chain-id=$CHAIN_ID --name=you --amount=1000mycoin --to=B71E119324558ABA3AE3F5BC854F1225132465A0 --sequence=0
Password to sign with 'you': xxx
Committed at block 88. Hash: 492B08FFE364D389BB508FD3507BBACD3DB58A98
```

You can look at the contents of the tx, use the tx hash above:

```bash
$ bnbcli tx 492B08FFE364D389BB508FD3507BBACD3DB58A98
```

Then you can check the balance of pepe's key to see that he now has 1000 satoshi units of `mycoin`:

```bash
$ bnbcli account B71E119324558ABA3AE3F5BC854F1225132465A0
{
  "type": "16542275FBFAB8",
  "value": {
    "BaseAccount": {
      "address": "B71E119324558ABA3AE3F5BC854F1225132465A0",
      "coins": [
        {
          "denom": "mycoin",
          "amount": 1000
        }
      ],
      ...
    },
    ...
  }
}
```

Amounts are represented as ints, and all coins have a fixed scale of 8. This means that if a balance of `100000000` were to be shown here, that would represent a balance of `1` coin.

## DEX

### Placing an order

```bash
$ bnbcli dex order -i uniqueid1 -l XYZ_BNB -s 1 -p 100000000 -q 100000000 --from me --chain-id=$CHAIN_ID -t 1
```

### Viewing the order book

```bash
$ bnbcli dex show -l XYZ_BNB
```

Alternatively through http when `bnbcli api-server` is running. Prices and quantities are returned as decimal numbers in strings.

```bash
$ curl -s http://localhost:8080/api/v1/depth?symbol=XYZ_BNB&limit=5 | json_pp
{"asks":[["0.00000000","0.00000000"],["0.00000000","0.00000000"],["0.00000000","0.00000000"],["0.00000000","0.00000000"],["0.00000000","0.00000000"]],"bids":[["0.10000000","1.00000000"],["0.00000000","0.00000000"],["0.00000000","0.00000000"],["0.00000000","0.00000000"],["0.00000000","0.00000000"]]}
```

## Contribution
It is welcomed to contribute to this repo from everyone. If you'd like to contribute, please fork, fix, commit and submit a pull request to review and merge into the main code base. Please make sure your contributions adhere to our coding guidelines:

- Code must adhere to the official Go formatting guidelines (i.e. please use gofmt tool).
- Code must be documented adhering to the official Go commentary guidelines.
- Pull requests need to be based on and opened against the master branch.
