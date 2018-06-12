# Binance DEX Specification

## Goal
Binance DEX (B-DEX) is a decentralized exchange and blockchain. Please check business documentation for details. It allows:
- Token issurance, with customized burn and/or freeze
- Token asset transfer
- Trade across tokens via buy/sell orders upon token pairs, directly from blockchain wallet
- Do not need to deposit before trading
- public ledger for all information, gurantees for no front-running from anyone, including the all the validator nodes

## Architecture
The system diagram is as below.

![System Diagram](https://github.com/BiJie/BinanceChain/blob/danjundev/doc/B-DEX%20Diagram.png)

### Components
#### Validator
Validator nodess are responsible for generating the blockchain. Validator is a Tendermint based ABCI application. There would be multiple validator nodes, e.g. 7, forming the backbone of the mainnet of Binance DEX chain.

Each validator would communicate to **all** the rest validator via direct TCP connections.

#### Frontier
Frontiers are non-validator nodes. They are not generating any blocks but responsible for accepting requests and publishing data. Several frontier nodes work with one validator node.

#### Bridge
Bridge is the communication channels between Validator and Frontier. It would be a one-to-many broadcast for relaying the stream of blocks. 

#### Client
Clients are GUI applications. Users use client to enter orders, check account status and explore other information.

## Workflow

### Critical Concepts

#### Address and Account

#### Orders

#### Transfer

#### Issurance

#### Freeze

#### Burn

#### List/De-List

### Genesis

### Transaction Workflow After Genesis
The below diagram shows the sequence of Time.

#### Transaction Entry

#### Blocking

#### Execution

## Data Structure and Storage

### Block

### Transaction Data

### Storage 

## Base Components

### Frontier - Transaction Entry

### Bridge - Transaction Transportation

### Validator - Mempool

### Validator - Transaction Check

### Validator - [Match Engine](./match_engine.md)

### Validator - Execution 

### Validator - Fees

### Bridge - Broadcast

### Frontier - Block Saving

### Frontier - Exeuction 

### Frontier - Market Data Propagation

### Client - P2P Bootstrap
#### Account Authetication

#### Connection Authetication

### Client - API

## Periphery

### Explorer 

### Market Data Federal Net

### Pegged Token

### Data Prune
