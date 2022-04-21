# Why we create this repo

This repo is forked from [https://github.com/cosmos/cosmos-sdk](https://github.com/cosmos/cosmos-sdk).

Our BinanceChain app leverages cosmos-sdk to fast build a dApp running with tendermint. As our app becomes more and more complex, the original cosmos-sdk can hardly fit all our requirements. 
We changed a lot to our copied sdk, but it makes the future integration harder and harder. So we decided to fork cosmos-sdk.

# How to use this repo

We need to remove the original cosmos-sdk repo and clone our repo into that directory.
The reason is that we need to keep the import path.

```bash
> cd $GOPATH/src/github.com
> rm -rf cosmos/cosmos-sdk
> git clone https://github.com/binance-chain/bnc-cosmos-sdk.git cosmos/cosmos-sdk
> cd cosmos-sdk
> git checkout develop
> make get_vendor_deps
```

# Cosmos SDK
![banner](docs/graphics/cosmos-sdk-image.png)

[![version](https://img.shields.io/github/tag/cosmos/cosmos-sdk.svg)](https://github.com/cosmos/cosmos-sdk/releases/latest)
[![CircleCI](https://circleci.com/gh/cosmos/cosmos-sdk/tree/master.svg?style=shield)](https://circleci.com/gh/cosmos/cosmos-sdk/tree/master)
[![codecov](https://codecov.io/gh/cosmos/cosmos-sdk/branch/master/graph/badge.svg)](https://codecov.io/gh/cosmos/cosmos-sdk)
[![Go Report Card](https://goreportcard.com/badge/github.com/cosmos/cosmos-sdk)](https://goreportcard.com/report/github.com/cosmos/cosmos-sdk)
[![license](https://img.shields.io/github/license/cosmos/cosmos-sdk.svg)](https://github.com/cosmos/cosmos-sdk/blob/master/LICENSE)
[![LoC](https://tokei.rs/b1/github/cosmos/cosmos-sdk)](https://github.com/cosmos/cosmos-sdk)
[![API Reference](https://godoc.org/github.com/cosmos/cosmos-sdk?status.svg
)](https://godoc.org/github.com/cosmos/cosmos-sdk)
[![riot.im](https://img.shields.io/badge/riot.im-JOIN%20CHAT-green.svg)](https://riot.im/app/#/room/#cosmos-sdk:matrix.org)

The Cosmos-SDK is a framework for building blockchain applications in Golang.
It is being used to build `Gaia`, the first implementation of the [Cosmos Hub](https://cosmos.network/docs/),

**WARNING**: The SDK has mostly stabilized, but we are still making some
breaking changes.

**Note**: Requires [Go 1.10+](https://golang.org/dl/)

## Gaia Testnet

To join the latest testnet, follow
[the guide](https://cosmos.network/docs/getting-started/full-node.html#setting-up-a-new-node).

For status updates and genesis files, see the
[testnets repo](https://github.com/cosmos/testnets).

## Install

See the
[install instructions](https://cosmos.network/docs/getting-started/installation.html).

## Quick Start

See the [Cosmos Docs](https://cosmos.network/docs/)

- [Getting started with the SDK](https://cosmos.network/docs/sdk/core/intro.html)
- [SDK Examples](/examples)
- [Join the testnet](https://cosmos.network/docs/getting-started/full-node.html#run-a-full-node)

## Disambiguation

This Cosmos-SDK project is not related to the [React-Cosmos](https://github.com/react-cosmos/react-cosmos) project (yet). Many thanks to Evan Coury and Ovidiu (@skidding) for this Github organization name. As per our agreement, this disambiguation notice will stay here.
