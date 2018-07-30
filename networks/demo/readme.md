# Wrap Scripts

At first, move the scripts to `build` path. 

Then `export CHAIN_ID=your_chain_id`.

### issue.sh
issue token
``` shell
./issue.sh --from alice -s TRX -n 10000000 --name tron
```

### info.sh
show token info
``` shell
./info.sh -s ADA
```

### list.sh
list trade pair

```shell
./list.sh -s ADA --quote-symbol BNB --from alice --init-price 1
```

### order.sh
make orders

```shell
./order.sh --list-pair BTC_BNB --side 1 --price 1 --quantity 100 --from alice --tif 1
```

### show.sh
show order book

```shell 
./show.sh -l ADA_BNB --from alice
```

### cancel.sh
cancel order
```shell 
./cancel.sh --id f4705e9d-279a-4fbc-8718-a1f53448dc63 --from bob
```