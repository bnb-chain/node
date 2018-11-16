#!/usr/bin/env bash

for i in {0..1000}
do
 ./add_key.exp node2_user$i /home/test/go/src/github.com/BiJie/BinanceChain/build/bnbcli /home/test/.bnbcli/
done
