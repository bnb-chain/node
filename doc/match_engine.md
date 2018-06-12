# B-DEX Match Engine Specification

B-DEX doesn't carry continuous matching as the most traditional centralized exchange. Instead, B-DEX match by **rounds*, which is consistent with the blocking pace. 

## Match Candidates
Orders meet any of the below conditions would be considered as the candidates of next match round:

- new orders that come in just now and get confirmed by being accepted into the **last** block
- existing orders that come in the past blocks before the last, and have not been filled or expired 

## Match Time
Candidates would be matched right after one block is committed.

## Match Logic
The below match logic would be applied on every listed token pairs.

The match only happens when the best bid and ask prices are _'crossed'_, i.e. best bid > best ask. 

All the orders would be matched first by the price aggressiveness and then block height. 

### Conclude Execution Price
The execution price would be selected as the below logic, in order to:
1. maximize the execution qty
2. execute all orders or at least all orders on one side
3. indicate the market pressure from either buy or sell

- Step 0: no match for one side market, or market without crossed order book

- Step 1: Maximum matched volume. The Equilibrium Price (EP) should be the price at which the maximum
volume can be traded. In the case of more than one price level with the same executable volume,
the algorithm should go to step 2

- Step 2: Minimum surplus. In the case of more than one price level with the same maximum executable
volume, the EP should be the price with the lowest surplus (imbalance) volume. The surplus is
absolute leftover volume at the EP.

- Step 3: Market Pressure. If multiple prices satisfy 1 and 2, establish where market pressure of the potential
price exists. Surplus with a positive sign indicates buy side pressure while surplus with a negative
sign indicates sell side pressure. If multiple positive equivalent surplus amounts exist, then
algorithm uses the highest of the potential equilibrium prices. Conversely, if market pressure is on
the sell side then the algorithm uses the lowest of the potential prices. If both positive and
negative surplus amounts exist, precede to Step 4.

- Step 4: Reference to previous trade price. Select the price closest to the last trade price

please check [match cases]() for example.

### Order Matches
After the execution price is concluded. Order match would happen in sequence of the price and time, i.e.
1. orders with best bid price would match with order with best ask price;
2. if the orders on one price cannot be fully filled by the opposite orders:
   1. for the orders with the same price, the orders from the earlier blocks would be selected and filled first
   2. If the orders have the same price and block height, and cannot be fully filled, the execution would be allocated to each order in proportion to their quantity. If the allocation cannot be accurately divided, a deterministic algo would gurantee that no consistent bias to any orders.

Each match between 2 orders would generate one trade structure, which is composed of trade quantity, trade price, and 2 order IDs. Validator code would perform **two** transfers between the accounts of the two orders.
