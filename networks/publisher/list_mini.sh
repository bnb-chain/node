#!/usr/bin/env bash

# start a validator and witness on local machine
# later on db indexer and publisher can be setup and started by newIndexer.sh and newPublisher.sh
# later on order generation can be kicked off by ordergen.sh
# TODO: support two validator without docker in same machine

########################### SETUP #########################
home=$HOME
src="${home}/go/src/github.com/binance-chain/node"
deamonhome="${home}/.bnbchaind"
witnesshome="${home}/.bnbchaind_witness"
clihome="${home}/.bnbcli"
chain_id='test-chain-n4b735'
echo $src
echo $deamonhome
echo $witnesshome
echo $clihome

key_seed_path="${home}"
executable="${src}/build/bnbchaind"
clipath="${src}/build/bnbcli"
cli="${clipath} --home ${clihome}"
scripthome="${src}/networks/publisher"
############################ END ##########################

#X1Mini_symbol="X1Mini-ED3"
result=$(${cli} token issue-tiny --from=zc --token-name="X1M Coin" --symbol=X1M --total-supply=800000000000 --chain-id ${chain_id})
X1Mini_symbol=$(echo "${result}" | tail -n 1 | grep -o "X1M-[0-9A-Z]*")
echo ${X1Mini_symbol}
sleep 2
echo 1234qwerasdf|${cli} dex list-mini -s=${X1Mini_symbol} --quote-asset-symbol=BNB --init-price=100000000 --from=zc --chain-id ${chain_id}
sleep 1
zz_addr=$(${cli} keys list | grep "zz.*local" | grep -o "bnb[0-9a-zA-Z]*" | grep -v "bnbp")
echo 1234qwerasdf|${cli} send --from=zc --to=${zz_addr} --amount=200000000000:${X1Mini_symbol} --chain-id ${chain_id}
sleep 5
