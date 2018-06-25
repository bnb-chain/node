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

There would be only 1 price selected in one match round as the best prices among all the fillable orders, to show the fairness.

All the orders would be matched first by the price aggressiveness and then block height. 

### Conclude Execution Price
The execution price would be selected as the below logic, in order to:
1. maximize the execution qty
2. execute all orders or at least all orders on one side that are fillable against the selected price.
3. indicate the market pressure from either buy or sell

- Step 0: no match for one side market, or market without crossed order book

- Step 1: Maximum matched volume. The Equilibrium Price (EP) should be the price at which the maximum
volume can be traded. In the case of more than one price level with the same executable volume,
the algorithm should go to step 2

- Step 2: Minimum surplus. In the case of more than one price level with the same maximum executable
volume, the EP should be the price with the lowest surplus (imbalance) volume. The surplus is
absolute leftover volume at the EP. If multiple surplus amounts have the same lowest value, precede to step 3.

- Step 3: Market Pressure. If multiple prices satisfy 1 and 2, establish where market pressure of the potential
price exists. Surplus with a positive sign indicates buy side pressure while surplus with a negative
sign indicates sell side pressure. 
   - For scenarios that all the the equivalent surplus amounts are positive, if all the prices are below the reference price plus an upper limit percentage (e.g. 5%), then
algorithm uses the highest of the potential equilibrium prices. If all the prices are above the reference price plus an upper limit, use the lowest price; for other cases, use the reference price plus the upper limit directly. 
   - Conversely, if market pressure is on
the sell side, if all prices are above the reference price minus a lower percentage limit, then the algorithm uses the lowest of the potential prices. If all the price are below the reference price minus the lower percentage limit, use the highest price, otherwise use the reference price minus the lower percentage limit.

    If both positive and
negative surplus amounts exist, precede to Step 4.

- Step 4: When both positive and negative surplus amounts exists at the lowest, if the reference price falls at / into these prices, the reference price should be chose, otherwise the price closest to the reference price would be chosen.


### Examples
The chosen price level row would have ``*`` on the deciding colume.
```
1. Choose the largest execution (Step 1)
-------------------------------------------------------------
SUM    SELL    PRICE    BUY    SUM    EXECUTION    IMBALANCE
300            100      150    150    150          -150
300            99              150    150          -150
300    250     98       150    300    300*         0
50     50      97              300    50           250

2. Choose the largest execution (Step 1)
--------------------------------------------------------------
SUM    SELL    PRICE    BUY    SUM    EXECUTION    IMBALANCE
300            100      150    150    150          -150
300            99       50     200    200          -100
300            98              200    200          -100
300    200     97       300    500    300*         200
100    100     96              500    100          400


3. the least abs surplus imbalance (Step 2)
--------------------------------------------------------------
SUM    SELL    PRICE    BUY    SUM    EXECUTION    IMBALANCE
1500           102      300    300    300          -1200
1500           101             300    300          -1200
1500           100      100    400    400          -1100
1500           99       200    600    600          -900
1500   250     98       300    900    900          -600
1250   250     97              900    900          -350
1000   1000    96              900    900          -100*

4. the least abs surplus imbalance (Step 2)
--------------------------------------------------------------
SUM    SELL    PRICE    BUY    SUM    EXECUTION    IMBALANCE
110            102      30     30     30           -80
110            101      10     40     40           -70
110            100             40     40           -70
110            99       50     90     90           -20
110    10      98              90     90           -20
100    50      97              90     90           -10*
50             96       15     105    50           55
50     50      95              105    50           55

5.1 choose the lowest for all the same value of sell surplus imbalance, reference price is 80 and 5% upper limit (Step 3)
--------------------------------------------------------------
SUM    SELL    PRICE    BUY    SUM    EXECUTION    IMBALANCE
50             102      10     10     10           -40
50             101             10     10           -40
50             100             10     10           -40
50             99              10     10           -40
50             98              10     10           -40
50             97       10     20     20           -30
50             96              20     20           -30
50     50      95              20     20           -30*

5.2 choose the lowest for all the same value of sell surplus imbalance, reference price is 100 and 5% upper limit (Step 3)
--------------------------------------------------------------
SUM    SELL    PRICE    BUY    SUM    EXECUTION    IMBALANCE
50             99       10     10     10           -40
50             98              10     10           -40
50             97              10     10           -40
50             96              10     10           -40
50             95              10     10           -40
50             94       10     20     20           -30*
50             93              20     20           -30
50     50      92              20     20           -30

5.3 choose the lowest for all the same value of sell surplus imbalance, reference price is 90 and 5% upper limit (Step 3)
--------------------------------------------------------------
SUM    SELL    PRICE    BUY    SUM    EXECUTION    IMBALANCE
50             99       100    100    50           50
50             98              100    50           50
50             97              100    50           50
50             96              100    50           50
50             95              100    50           50
50             94              100    50           50*
50             93              100    50           50
50     50      92              100    50           50

5.4 choose the lowest for all the same value of sell surplus imbalance, reference price is 100 and 5% upper limit (Step 3)
--------------------------------------------------------------
SUM    SELL    PRICE    BUY    SUM    EXECUTION    IMBALANCE
50             101      10     10     10           -40
50             100             10     10           -40
50             99              10     10           -40
50             98              10     10           -40
50             97              10     10           -40
50             96       10     20     20           -30
50             95              20     20           -30*
50     50      94              20     20           -30

6.1 choose the closest to the last trade price 99 (Step 4)
--------------------------------------------------------------
SUM    SELL    PRICE    BUY    SUM    EXECUTION    IMBALANCE
50             100      25     25     25           -25
50             99              25     25           -25*
50     25      98              25     25           -25
25             97       25     50     25           25
25             96              50     25           25
25     25      95              50     25           25

6.2 choose the closest to the last trade price 97 (Step 4)
--------------------------------------------------------------
SUM    SELL    PRICE    BUY    SUM    EXECUTION    IMBALANCE
50             100      25     25     25           -25
50             99              25     25           -25
50     25      98              25     25           -25
25             97       25     50     25           25*
25             96              50     25           25
25     25      95              50     25           25

```

### Order Matches
After the execution price is concluded. Order match would happen in sequence of the price and time, i.e.
1. orders with best bid price would match with order with best ask price;
2. if the orders on one price cannot be fully filled by the opposite orders:
   1. for the orders with the same price, the orders from the earlier blocks would be selected and filled first
   2. If the orders have the same price and block height, and cannot be fully filled, the execution would be allocated to each order in proportion to their quantity (floored if the number has a partial lot). If the allocation cannot be accurately divided, a deterministic algo would gurantee that no consistent bias to any orders: according to a sorted sequence of a de facto random order ID.

Each match between 2 orders would generate one trade structure, which is composed of trade quantity, trade price, and 2 order IDs. Validator code would perform **two** transfers between the accounts of the two orders.
