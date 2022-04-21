#!/usr/bin/env bash

t=$1
tt=$2
iter=$3

go build
for (( i=0; i<=$t; i++ ))
do
    home=./tests/${tt}/${i}
    echo "start $i client"
#    ./tbnbcli send --amount 1000:BNB --to tbnb1mh3w2kxmdmnvctt7t5nu7hhz9jnp422edqdw2d --from tss_test${i}_default --chain-id Binance-Chain-Nile --node https://data-seed-pre-0-s1.binance.org:443 --trust-node > ${home}/sign.log 2>&1 &
    expect ./send.exp $home tss_test${i}_default > ${home}/sign_${iter}.log 2>&1 &
done