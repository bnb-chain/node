#!/usr/bin/env bash

for i in {0..1000}
do
 ./add_key.exp node0_user$i tbnbcli /home/test/.bnbcli/
done
