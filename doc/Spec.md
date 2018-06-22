# Binance DEX Specification

- [Binance DEX Specification](#binance-dex-specification)
    - [Goal](#goal)
    - [Architecture](#architecture)
        - [Components](#components)
            - [Validator](#validator)
            - [Frontier](#frontier)
            - [Bridge](#bridge)
            - [Client](#client)
    - [Security](#security)
    - [Workflow](#workflow)
        - [Critical Concepts](#critical-concepts)
            - [User Setup, Keys, Address and Account](#user-setup-keys-address-and-account)
                - [Address](#address)
            - [Orders](#orders)
            - [Transfer](#transfer)
            - [Issuance](#issuance)
            - [Freeze / Unfreeze](#freeze--unfreeze)
            - [Burn](#burn)
            - [List/De-List](#listde-list)
        - [Genesis](#genesis)
        - [Transaction Workflow After Genesis](#transaction-workflow-after-genesis)
            - [Account Creation](#account-creation)
            - [Transaction Entry](#transaction-entry)
            - [P2P communication](#p2p-communication)
            - [Consensus](#consensus)
                - [How to Select Transactions into Blocking](#how-to-select-transactions-into-blocking)
            - [Blocking](#blocking)
            - [Execution](#execution)
                - [Order Match](#order-match)
                - [Order Expire](#order-expire)
                - [Transfer](#transfer)
                - [Burn](#burn)
                - [Freeze / Unfreeze](#freeze--unfreeze)
                - [Token Issue and ICO](#token-issue-and-ico)
    - [Data Structure and Storage](#data-structure-and-storage)
        - [Encoding](#encoding)
        - [Block](#block)
        - [Transaction Data](#transaction-data)
        - [Storage](#storage)
    - [Base Components](#base-components)
        - [Frontier - Transaction Entry](#frontier---transaction-entry)
        - [Bridge - Transaction Transportation](#bridge---transaction-transportation)
        - [Validator - Mempool](#validator---mempool)
        - [Validator - Transaction Check](#validator---transaction-check)
        - [Validator - Match Engine](#validator---match-engine)
        - [Validator - Execution](#validator---execution)
        - [Validator - Fees](#validator---fees)
            - [Fee Collection and Rebate](#fee-collection-and-rebate)
        - [Bridge - Broadcast](#bridge---broadcast)
        - [Frontier - Block Saving](#frontier---block-saving)
        - [Frontier - Exeuction](#frontier---exeuction)
        - [Frontier - Market Data Propagation](#frontier---market-data-propagation)
        - [Client - P2P Bootstrap](#client---p2p-bootstrap)
            - [Account Authetication](#account-authetication)
            - [Connection Authetication](#connection-authetication)
        - [Client - API](#client---api)
    - [Periphery](#periphery)
        - [Explorer](#explorer)
        - [Market Data Federal Net](#market-data-federal-net)
        - [Pegged Token](#pegged-token)
        - [Data Prune](#data-prune)

## Goal
Binance DEX (B-DEX) is a decentralized exchange and blockchain. Please check business documentation for details. It allows:
- Token issuance, with customized burn and/or freeze
- Token asset transfer
- Trade across tokens via buy/sell orders upon the listed token pairs, directly from blockchain wallet
- Do not need to deposit before trading
- public ledger for all information, guarantees for no front-running from anyone, including the all the validator nodes

## Architecture
The system diagram is as below.

![System Diagram](https://github.com/BiJie/BinanceChain/blob/danjundev/doc/B-DEX%20Diagram.png)

### Components
#### Validator
Validator nodes comprise the core network of the block chain. They  are responsible for :
- generating the blockchain, after consensus
- executing the transferring/freezing/burning/issuing/listing/de-listing transactions
- matching orders on the list currency pairs and transferring the asserts between accounts that have execution trades
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
1. enter orders and transferring requests
2. check account status
3. show realtime and historical market data, and even some technical metrics
4. and explore other block chain information.

The client GUI should resemble the current Binance GUIs. Programming API similar to current Binance API would be also provided by Frontiers.

The block chain information can also be accessed via Explorer web site similar to 'etherscan.io'. 

## Security
- Client should verify the Frontier addresses
- Only Frontiers and Validators can access the specified addresses and ports of Validators via secure transportation. No others can.
- Both Frontiers and Validators would check the requests parameters for their types and values. 
- Frontiers/Explorer should implement the similar controls on API request frequency and other rules to prevent DDoS 
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
1. Account does not possess enough token to buy or sell
2. Exchange is down or has problem to match it
3. The token is not listed against any base currencies
4. Other order parameters are not valid
5. Duplicated order ID

Orders may be canceled / expired back when: 

1. IOC order not fully filled
2. Order expired
3. Exchange has problem to handle further with the orders

#### Transfer
Transferring can happen between any 2 addresses on any tokens issued by B-DEX Chain, even the token is not listed against any base currencies.

A transfer transaction contains:

1. Source Address, with owner signature
2. Target Address
3. Quantity

Transfer would happen right after the request is blocked. Transfer transaction would be rejected when:

1. User address cannot be located with asset
2. Account does not possess enough token to transfer
3. Account does not possess enough BNB to pay the fees for transferring
4. Exchange is down or has problem to execute
 
#### Issuance

Issuance is to create a new token that does not exist on Binance Chain. To make it simple and fast, we seperate the issuance and ICO process. All the specific tokens are minted in the issuing process, and never be mined within the later blocks.

Since this operation costs many fees, operators should ensure their account has enough BNB before exeucting.

An issuance transaction contains:

1. Source Address: the sender address of the transaction and it will become the **owner** of the token
2. Token Name: the length of the token name is limited to 30 characters. e.g. "Bitcoin"
3. Symbol: identifier of the token, limited to 10 alphanumeric characters and is case insensitive. To avoid conflicts with existing symbols on other blockchains, a suffix '.B' is added internally,  e.g. "BTC.B" 
4. Total Supply:  // TODO
5. Decimals:  // TODO

Issuance transaction would be rejected when:

1. The name and the symbol do not meet the naming limits
2. The symbol already exists. This circumstance happens when you double execute the issuance operation, or the symbol is occupied by other token
3. The total supply and the decimals do not meet the range limits
4. The account of the sender does not possess enough BNB


#### Freeze / Unfreeze
Freeze is to lock certain amount of token to be temporarily unspendable/un-usable while Unfreeze is to unlock them.

Anyone can freeze his/her own tokens as long as his/her account possesses enough tokens.
Anyone can unfreeze his/her own frozen tokens as long as his/her account has enough frozen tokens. 

An freeze/unfreeze transaction contains:

1. Source Address
2. Symbol
3. Amount

Freeze/unfreeze transaction would be rejected when:

1. The symbol does not exist
2. The account does not possess enough token to be frozen/unfrozen
3. The account does not possess enough BNB to pay for the transaction fees


#### Burn
Burn is to destroy certain amount of token, after which that amount of tokens will be subtracted from the operator's balance.
The total supply should be updated at the same time. Notice that only the owner of the token has the permission to burn token. 

A burn transaction contains:

1. Source Address
2. Symbol
3. Amount

Burn transaction would be rejected when:

1. The symbol does not exist
2. The sender is not the owner of the token
3. Owner account does not possess enough tokens to burn
4. Account does not possess enough BNB to pay for the transaction fees

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
//TODO

#### Transaction Entry
1. Client GUI or command line tools would validate the request parameters and generate a transaction requests
2. Client GUI or command line tools would choose an access point (please see below) via consistent hash, connect and autheticate the connection, and send requests to it
3. Frontier would validate the request and parameters, if all right, relay it to the connected Valdiator
4. Validator would validate the request again by calling ABCI interface ``CheckTX`` . One of the check is ensure there is enough asset to sponsor the transaction, e.g. enough BNB to buy Token ABC and pay the fees, freezed and previous locked quantity would be excluded as avaiable.  If all right, put it into mempool to broadcast. Please note no acknowledgement would be sent back even at this moment.

#### P2P communication
This would stay the same as tendermint for now. 
#### Consensus
This would stay the same as tendermint for now.
##### How to Select Transactions into Blocking
It would not force a hard fork if the selection logic is changed. So far it stays the same as Tendermint, but the more optimized is to:
1. select orders with better prices with more chance to get executed in the upcoming blocks.

#### Blocking
The blocking is done via ABCI interfaces, ``BeginBlock``, ``DeliverTx``, ``EndBlock`` and ``Commit``. It should be triggered when there is enough transaction to fill the whole block or the time is up to generate a block:

**//?? Each block can contain at most 4K transactions, but it should be generated within half a second (or less to catch time). Thus another half a second would be used for executions, until the ABCI application can handle further transactions.**

Validator would perform the last and most crucial check on the transactions in ``DeliverTx``.  **After the DeliverTx, order quantity would be locked in the account, i.e. the amount of quantity cannot be spended / used to transfer or generate new orders.**

After ``Commit`` is called, the block is concluded to book. This is the moment to trigger all the executions.

#### Execution
Executions are different according to different type of transactions, but the main purpose is to generate the correct state, and persist. All the validators are expected to generate the same result of execution and the states.

All the below can happen concurrently. The last step of saving state has contention so that locks may be used.

##### Order Match
The most number of requests are expected to be orders. After the block is committed:
1. iterate all the new order requests, new orders in the new block would be inserted into the order book; while cancel order would be located from the outstanding order map and net off from the order book;
2. perform match among all the left orders and generate all the trades
3. clear the fully filled orders from order book
4. iterate all unfully filled order, for IOC orders, remove from orders, and release the locked quantity back to 
5. iterate the execution list, move the asset accordingly to change the balance and charge the fees
6. refresh/store the state of account. Please note trades would not be saved.

##### Order Expire
A whole order book scan would happen every 86400 blocks (around 24 hours) to filter out all the expired orders. After the scan, all the expired orders would be removed from the order book, the locked quantity in the account would be unlocked. Before this action all the existing orders in the order book is subject to matching. 

##### Transfer 
This is dedicated for Transfer transaction, to move the asset accordingly to change the balance and charge the fees, and then refresh/store the state of account.

##### Burn
This is to reduce the total supply of the token. Following steps would be executed in sequence:

1. Validate the transaction parameters
2. Check whether the token exists
3. Check whether the sender is the owner
4. Check whether the sender account has sufficient tokens
5. Check whether the provided amount is bigger than total supply
6. Subtract tokens from sender's balance
7. Update total supply

##### Freeze / Unfreeze
Freezing is to lock some amount of tokens in user's balance.

1. Validate the transaction parameters
2. Check whether the token exists
3. Check whether the sender account has sufficient tokens
4. Move tokens from `balance` to `frozen`

Unfreezing is to unlock some amount of tokens in user's frozen balance

1. Validate the transaction parameters
2. Check whether the token exists
3. Check whether the sender account has sufficient frozen tokens
4. Move tokens from `frozen` to `balance`.

##### Token Issue
The issuing process is simple as followed:

1. Check the range restriction of total supply and decimals
2. Check the uniqueness of the symbol
3. Save the token info to tokenStore
4. Add the total tokens to the owner's account 

##### ICO

## Data Structure and Storage
//TODO: to determine which part should be changed upon Tendermint data structure & storage

### Encoding
//TODO: Amino, any change to do upon Tendermint?

### Block

### Transaction Data

### Storage 

## Base Components

### Frontier - Transaction Entry
Frontier is accessed by user client ends via an assess point, which handles load balancing and session management. Frontier performs [Checks](./transaction_entry_checks.md) to make sure it's valid. For every request, **Frontier would generate an UUID as the unique identifier for the transaction for the whole life cycle**, , and then relay the requests to Validators.


Every Frontier only has one validator to connect with, via direct TCP??. 

For the requests below, the response would be generated after the block is booked, though the transaction would take effect after the execution is done.
- New order and cancel order requests
- Transfer 
- Burn
- Freeze
- List/De-List

### Bridge - Transaction Transportation
In order to relay the request from Frontier to Validator, the transaction would be wired into the same encoding format. The transportation is via a RPC mechnism opened by Validator ABCI application, gRPC??

### Validator - Mempool
//TODO

### Validator - Transaction Check
//TODO

### Validator - [Match Engine](./match_engine.md)

### Validator - Execution 
The Execution would happen inside the function callback in ABCI. 
//TODO

### Validator - Fees
Fees would be calculated based on the trade notional value of quote currency and paid in BNB. The rate of quote currency to BNB from the last blocking around would be used to calculate.

The fee rate is set and saved in the state of world. It can be reset via a special transaction type??. 

#### Fee Collection and Rebate

Fees would be collected and transferred to the blocking proposer Validator account?? after all the execution of the blocking round.

Every one pays the same transaction fees, and there would be fee rebate framework, which is implemented outside the chain: 
1. In perodic time, a routine would calculate the fee rebate for different accounts;
2. the total rebates would collected evenly from all validator accounts.
3. Transfer transactions would be generated


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
