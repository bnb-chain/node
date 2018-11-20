#!/usr/bin/env bash

# start a validator and witness on local machine
# later on db indexer and publisher can be setup and started by newIndexer.sh and newPublisher.sh
# later on order generation can be kicked off by ordergen.sh
# TODO: support two validator without docker in same machine

########################### SETUP #########################
src='/Users/zhaocong/go/src/github.com/BiJie/BinanceChain'
home='/Users/zhaocong'
deamonhome='/Users/zhaocong/.bnbchaind'
witnesshome='/Users/zhaocong/.bnbchaind_witness'
clihome='/Users/zhaocong/.bnbcli'
chain_id='test-chain-n4b735'

key_seed_path="${home}"
executable="${src}/build/bnbchaind"
clipath="${src}/build/bnbcli"
cli="${clipath} --home ${clihome}"
scripthome="${src}/networks/publisher"
############################ END ##########################

# clean history data
rm -r ${deamonhome}
rm -r ${clihome}
rm -r ${home}/.bnbchaind_witness

# build
cd ${src}
make build

# init a validator and witness node
${executable} init --name xxx --chain-id ${chain_id} > ${key_seed_path}/key_seed.json # cannot save into ${deamonhome}/init.json
secret=$(cat ${key_seed_path}/key_seed.json | grep secret | grep -o ":.*" | grep -o "\".*"  | sed "s/\"//g")
#echo ${secret}

mkdir -p ${home}/.bnbchaind_witness/config

#sed -i -e "s/skip_timeout_commit = false/skip_timeout_commit = true/g" ${deamonhome}/config/config.toml
sed -i -e "s/log_level = \"main:info,state:info,\*:error\"/log_level = \"debug\"/g" ${deamonhome}/config/config.toml

# config witness node
cp ${deamonhome}/config/genesis.json ${witnesshome}/config/
cp ${deamonhome}/config/config.toml ${witnesshome}/config/

sed -i -e "s/26/27/g" ${witnesshome}/config/config.toml
sed -i -e "s/6060/7060/g" ${witnesshome}/config/config.toml

# start validator
${executable} start > ${deamonhome}/log.txt 2>&1 &
validator_pid=$!
echo ${validator_pid}
sleep 10 # sleep in case cli status call failed to get node id
validatorStatus=$(${cli} status)
validator_id=$(echo ${validatorStatus} | grep -o "\"id\":\"[a-zA-Z0-9]*\"" | sed "s/\"//g" | sed "s/id://g")
#echo ${validator_id}

# set witness peer to validator and start witness
sed -i -e "s/persistent_peers = \"\"/persistent_peers = \"${validator_id}@127.0.0.1:26656\"/g" ${witnesshome}/config/config.toml
${executable} start --home ${witnesshome} > ${witnesshome}/log.txt 2>&1 &
witness_pid=$!
echo ${witness_pid}

# init accounts
result=$(expect ${scripthome}/recover.exp "${secret}" "zc" "${clipath}" "${clihome}")
result=$(expect ${scripthome}/add_key.exp "zz" "${clipath}" "${clihome}")
zz_addr=$(${cli} keys list | grep "zz.*local" | grep -o "bnc[0-9a-zA-Z]*" | grep -v "bncp")

# issue&list NNB and ZCB for ordergen
${cli} token issue --from=zc --token-name="New BNB Coin" --symbol=NNB --total-supply=2000000000000000 --chain-id ${chain_id}
sleep 5
${cli} dex list -s=NNB --quote-asset-symbol=BNB --init-price=100000000 --from=zc --chain-id ${chain_id}
sleep 5
${cli} token issue --from=zc --token-name="ZC Coin" --symbol=ZCB --total-supply=2000000000000000 --chain-id ${chain_id}
sleep 5
${cli} dex list -s=ZCB --quote-asset-symbol=BNB --init-price=100000000 --from=zc --chain-id ${chain_id}
sleep 5
${cli} send --from=zc --to=${zz_addr} --amount=1000000000000000:BNB --chain-id ${chain_id}
sleep 5