#!/usr/bin/env bash

# start a validator and witness on local machine
# later on db indexer and publisher can be setup and started by newIndexer.sh and newPublisher.sh
# later on order generation can be kicked off by ordergen.sh
# TODO: support two validator without docker in same machine

########################### SETUP #########################
src='/Users/luerheng/go/src/github.com/binance-chain/node'
home='/Users/luerheng'
deamonhome='/Users/luerheng/.bnbchaind'
witnesshome='/Users/luerheng/.bnbchaind_witness'
clihome='/Users/luerheng/.bnbcli'
chain_id='test-chain-n4b735'

key_seed_path="${home}"
executable="${src}/build/bnbchaind"
clipath="${src}/build/bnbcli"
cli="${clipath} --home ${clihome}"
scripthome="${src}/networks/publisher"
############################ END ##########################

#x1mini_symbol="x1mini-ED3"
result=$(${cli} miniToken issue --from=zc --token-name="X1MINI Coin" --symbol=X1M --total-supply=80000000000000 --token-type=2 --chain-id ${chain_id})
x1mini_symbol=$(echo "${result}" | tail -n 1 | grep -o "x1mini-[0-9A-Z]*")
echo ${x1mini_symbol}
sleep 2
echo 1234qwerasdf|${cli} dex list-mini -s=${x1mini_symbol} --quote-asset-symbol=BNB --init-price=1000000000 --from=zc --chain-id ${chain_id}
sleep 1
zz_addr=$(${cli} keys list | grep "zz.*local" | grep -o "bnb[0-9a-zA-Z]*" | grep -v "bnbp")
echo 1234qwerasdf|${cli} send --from=zc --to=${zz_addr} --amount=20000000000000:${x1mini_symbol} --chain-id ${chain_id}
sleep 5
