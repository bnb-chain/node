BinanceChain
------------

BinanceChain is a blockchain with a flexible set of native assets and pluggable modules. It uses [tendermint](https://tendermint.com) for consensus and app logic is written in golang. It targets fast block times, a native dApp layer and multi-token support with no smart contract VM.

This is a fork of [basecoin](https://github.com/cosmos/cosmos-sdk/tree/master/examples/basecoin) and is already functional as a multi-asset cryptocurrency blockchain and DEX; see below for instructions on how to use it.

## Overview

* This uses BFT consensus so up to 1/3 of all validator nodes can be rogue or bad.
* Validator nodes are part of the "validator set" so they are known, trusted and controlled by the network.
* Full nodes are not validator nodes, but anyone can get a copy of the whole blockchain and validate it.
* No PoW means block times are very fast.
* UXTO/account does not matter as we just use the [cosmos](https://github.com/cosmos/cosmos-sdk/tree/master/x/bank) bank.
* Features like the DEX will run directly on the node as apps written in golang.

<img src="https://d.pr/i/5kNDH1+" alt="tendermint architecture" width="500" />

[Read](https://tendermint.readthedocs.io/en/master/introduction.html) [more](https://blog.cosmos.network/tendermint-explained-bringing-bft-based-pos-to-the-public-blockchain-domain-f22e274a0fdb) about Tendermint and ABCI.

## Getting Started

### Environment setup

If you do not have golang yet, please [install it](https://golang.org/dl) or use brew on macOS: `brew install go` and `brew install dep`.


**Mac & Linux**

```bash
$ export GOPATH=~/go
$ export PATH=~/go/bin:$PATH
$ export BNBCHAINPATH=~/go/src/github.com/binance-chain/node
$ mkdir -p $BNBCHAINPATH
$ git clone git@github.com:binance-chain/node.git $BNBCHAINPATH
$ cd $BNBCHAINPATH
$ make get_vendor_deps
$ make install
```

**Windows**

If you are working on windows, `GOPATH` and `PATH` should already be set when you install golang.
You may need add BNBCHAINPATH to the environment variables.

```bat
> md %BNBCHAINPATH%
> git clone git@github.com:binance-chain/node.git %BNBCHAINPATH%
> cd %BNBCHAINPATH%
> make get_vendor_deps
> make install
```

> If you encounter some network issues when downloading the dependencies, make sure you have configured shadowsocks correctly and switch to global mode. Run `set(win)/export(linux/mac) https_proxy=127.0.0.1:1080` if you still have https issues. 

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
$ bnbchaind start
```

If everything worked you will see blocks being generated around every 1s in your console.

### Reset

When you make a change you probably want to reset your chain, remember to kill the node first.

```bash
$ bnbchaind unsafe_reset_all
```

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

### Future

#### Pegging

If we use a native asset (BNB) as an ICO quote currency this will be straightforward as a plugin, but other examples of how to peg ethereum tokens to assets on tendermint chains do exist e.g.

* [Peggy](https://github.com/cosmos/peggy) [read more](https://blog.cosmos.network/understanding-the-value-proposition-of-cosmos-ecaef63350d#f158)

#### Others

To add: [ICO], [Staking], [Freezing], [Burning]
