# Binance DEX Specification

## Goal
Binance DEX (B-DEX) is a decentralized exchange and blockchain. Please check business documentation for details. It allows:
- Token issurance, with customized burn and/or freeze
- Token asset transfer
- Trade across tokens via buy/sell orders upon the listed token pairs, directly from blockchain wallet
- Do not need to deposit before trading
- public ledger for all information, gurantees for no front-running from anyone, including the all the validator nodes

## Architecture
The system diagram is as below.

![System Diagram](https://github.com/BiJie/BinanceChain/blob/danjundev/doc/B-DEX%20Diagram.png)

### Components
#### Validator
Validator nodes comprise the core network of the block chain. They  are responsible for :
- generating the blockchain, after consensus
- executing the trasfering/freezing/burning/issuing/listing/de-listing transactions
- matching orders on the list currency pairs and transfering the asserts between accounts that have execution trades
- according to the blocks, saving proper world of state, as the authority of the state

Validator is a Tendermint based ABCI application. There would be multiple validator nodes, e.g. 7, forming the backbone of the mainnet of Binance DEX chain.

Each validator would communicate to **all** the rest validator via direct TCP connections.

#### Frontier
Frontiers are non-validator nodes. Multiple frontiers would talk to one validator and together comprise a 'site'. The frontiers in a site are not generating any blocks but responsible for:
- accepting transactions requests from clients that connect to it, such as transfer, orders, etc. , and pass along to Validator
- sending order acknowledgement and execution results and pass along to Validator
- responding to clients queries, such as order status/executions, account balances, and block informations
- publishing market data snapshot after every matchand publishing data. 

Besides Frontiers keep order book information inside its memory, they also save the state of world including:
1. all blocks
2. all state the validator saves
3. redundant state for the query purpose, such as execution trades, even they can be reproduced via the above

#### Bridge
Bridge is the communication channels between Validator and Frontier. It would be a one-to-many broadcast for relaying the stream of blocks. The implementation can be either Kafka clusters or Multicast subnet. Different solution would not impact the core block chain architecture but it is better they are the same across different 'sites'.

1 Bridge, 1 Validator, and a few of Frontiers should stay within one LAN. 

#### Client
Clients are native GUI applications on different platforms or Web interfaces. They should have 4 major functions:
1. enter orders and transfering requests
2. check account status
3. show realtime and historical market data, and even some technical metrics
4. and explore other block chain information.

The client GUI should resemble the current Binance GUIs. Programming API similar to current Binance API would be also provided by Frontiers.

The block chain information can also be accessed via Explorer web site similar to 'etherscan.io'. 

## Security
- Client should verify the Frontier addresses
- Only Frontiers and Validators can access the specified addresses and ports of Validators via secure transportation. No others can.
- Both Frontiers and Validators would check the requests parameters for their types and values. 
- Frontiers/Explorer should implenment the similar controls on API request frequency and other rules to prevent DDoS 
- Once 1/3 validator cannot work, the chain would stop blocking. Exchange would be marked as down and new orders should be rejected.

## Workflow

### Critical Concepts

#### User Setup, Keys, Address and Account
Users do not have to register with B-DEX in order to trade. User is identified only by its private key and address. Their setup happens in the client GUIs, and only take effect in the B-DEX block chain after the address gets the 1st transfer of tokens on B-DEX.

Account is an internal concept associated with the user address, which is a 1:1 mapping. Tokens owned by users is recorded as balance on the user account. One account can hold multiple tokens and as many orders as the balance can afford.

##### Address
//TODO - B-DEX address can be represented by 20 bytes, which can be generated via Client GUI or command line tools.

#### Orders
Orders are the requests for client to buy or sell tokens into other tokens on B-DEX. Orders are composed of the below parameters.
0. Symbol Pairs: the list pair the order wants to trade. 
1. Order Type: B-DEX only accept LIMIT orders, which is adhering to SEC definitions of LIMIT orders
2. Price: price users would like to pay for the specified token quantity, presented as a float number of base currency. Internally it can be multiplied by Price Factor and store as an intergater in the range of int64.
3. Quantity: number of tokens users want to buy or sell. Internally it can be multiplied by Price Factor and store as an intergater in the range of int64.
4. Side: buy or sell
5. Time: entry time of the order, which is the block number(height) the order gets booked in.
6. TimeInForce:
   1. GTE: Good Till Expire. Order would stay effective until expire time. Order may expire, 259, 200 blocks, which is 72 hours in term of blocking time, 1s. Since the expiry is not checked in every block round, so orders may not expire right after the expiry time, but it would do after 72~96 hours. 
   2. IOC: Immediate or Cancel. Orders would be executed as much as it can in the booking block round and then got canceled back if there is still quantity left.
   
Orders would be rejected when:
0. user address cannot be located with asset
1. account doesn't possess enough token to buy or sell
2. Exchange is down or has problem to match it
3. the token is not listed against any base currencies
4. other order parameters are not valid
5. duplicated order ID

Orders may be canceled / expired back when:
1. IOC order not fully filled
2. Order expired
3. Exchange has problem to handle further with the orders

#### Transfer
Transfering can happen between any 2 addresses on any tokens issued by B-DEX Chain, even the token is not listed against any base currencies.

A transfer transaction contains:
1. Source Address, with owner signature
2. Target Address
3. Quantity

Transfer would happen right after the request is blocked. Transfer transaction would be rejected when:
0. user address cannot be located with asset
1. account doesn't possess enough token to transfer
2. Exchange is down or has problem to execute
 
#### Issurance
//TODO: Issurance can be done via ICO or direct setup.

#### Freeze / Unfreeze
Freeze is to move certain amount of token to be temporarily unspendable/un-usable for a defined period of time. The Freeze transaction can be either defined in the ICO spec or issued via user request. The Freeze transaction would define the time (or time series) and percentage amount to be unfreezed in the future time. The Unfreeze actions would be triggered by Validator itself after the specified time points, instead of the users.


#### Burn
Burn is to destroy certain amount of token, after which that amount is not spendable/usable anymore. This is 
implemented by sending the amount to a non-readable account, right after the containing block is booked.

The Burn interface is not open on the public User client GUI or command line tools.

#### List/De-List
The List/De-List requests are used to define the tradable currency pair universe. They take effect right after the containing block is booked. The List/De-List interface is not open on the public User client GUI or command line tools.

### Genesis
Genesis is starting point of the chain. It defined all the original parameters for the chain, in a JSON format file. The parameters are in the below:

//TODO:

-  ``genesis_time``: Official time of blockchain start.
-  ``chain_id``: ID of the blockchain. This must be unique for every
   blockchain. If your testnet blockchains do not have unique chain IDs,
   you will have a bad time.
-  ``validators``:
-  ``pub_key``: The first element specifies the pub\_key type. 1 ==
   Ed25519. The second element are the pubkey bytes.
-  ``power``: The validator's voting power.
-  ``name``: Name of the validator (optional).
-  ``app_hash``: The expected application hash (as returned by the
   ``ResponseInfo`` ABCI message) upon genesis. If the app's hash does not
   match, Tendermint will panic.
-  ``app_state``: The application state (e.g. initial distribution of tokens).

During Genesis, all the validators would be connected and recognized with each other. The chain will start generating blocks and saving new states. The readiness of Frontiers is not a dependency here.


### Transaction Workflow After Genesis
The below diagram shows the sequence of Time.
#### Account Creation

#### Transaction Entry
1. Client GUI or command line tools would validate the request parameters and generate a transaction requests
2. Client GUI or command line tools would choose an access point (please see below) via consistent hash, connect and autheticate the connection, and send requests to it
3. Frontier would validate the request and parameters, if all right, relay it to the connected Valdiator
4. Validator would validate the request again by calling ABCI interface ``CheckTX`` , if all right, put it into mempool to broadcast. Please note no acknowledgement would be sent back even at this moment.

#### P2P communication
This would stay the same as tendermint for now. 
#### Consensus
This would stay the same as tendermint for now.
##### How to Select Transactions into Blocking
It would not force a hard fork if the selection logic is changed. So far it stays the same as Tendermint, but the more optimized is to:
1. select orders with better prices with more chance to get executed in the upcoming blocks.

#### Blocking
The blocking is done via ABCI interfaces, ``BeginBlock``, ``DeliverTx``, ``EndBlock`` and ``Commit``.

Validator would perform the last and most crucial check on the transactions in ``DeliverTx``, majorly to ensure there is enough asset to sponsor the transaction, e.g. enough BNB to buy Token ABC and pay the fees.

After ``Commit`` is called, the block is concluded to book. This is the moment to trigger all the executions.

#### Execution
Executions are different according to different type of transactions, but the main purpose is to generate the correct state, and persist. All the validators are expected to generate the same result of execution and the states.

All the below can happen concurrently. The last step of saving state has contention so that locks may be used.

##### Order Match
The most number of requests are expected to be orders. After the block is committed:
1. iterate all the new order requests, new orders in the new block would be inserted into the order book; while cancel order would be located from the outstanding order map and net off from the order book;
2. perform match among all the left orders and generate all the trades
3. clear the fully filled orders from order book
4. iterate the execution list, move the asset accordingly to change the balance and charge the fees
5. refresh/store the state of account. Please note trades would not be saved.

##### Order Expire

##### Transfer 

##### Burn

##### Freeze / Unfreeze

##### Token Issue and ICO

## Data Structure and Storage

### Encoding

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
