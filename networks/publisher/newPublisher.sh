#!/usr/bin/env bash

########################### SETUP #########################
src='/Users/zhaocong/go/src/github.com/BiJie/BinanceChain'
home='/Users/zhaocong'
deamonhome='/Users/zhaocong/.bnbchaind'
witnesshome='/Users/zhaocong/.bnbchaind_publisher'
clihome='/Users/zhaocong/.bnbcli'
chain_id='test-chain-n4b735'

key_seed_path="${home}"
executable="${src}/build/bnbchaind"
clipath="${src}/build/bnbcli"
cli="${clipath} --home ${clihome}"
scripthome="${src}/networks/publisher"
############################ END ##########################

# clean history data
rm -r ${witnesshome}
mkdir -p ${witnesshome}/config

# config witness node
cp ${deamonhome}/config/genesis.json ${witnesshome}/config/
cp ${deamonhome}/config/config.toml ${witnesshome}/config/
cp ${deamonhome}/config/app.toml ${witnesshome}/config/

sed -i -e "s/26/29/g" ${witnesshome}/config/config.toml
sed -i -e "s/6060/9060/g" ${witnesshome}/config/config.toml

# get validator id
validator_pid=$(ps aux | grep "bnbchaind start$" | awk '{print $2}')
validatorStatus=$(${cli} status)
validator_id=$(echo ${validatorStatus} | grep -o "\"id\":\"[a-zA-Z0-9]*\"" | sed "s/\"//g" | sed "s/id://g")

# set witness peer to validator and start witness
sed -i -e "s/persistent_peers = \"\"/persistent_peers = \"${validator_id}@127.0.0.1:26656\"/g" ${witnesshome}/config/config.toml
sed -i -e "s/prometheus = false/prometheus = true/g" ${witnesshome}/config/config.toml
sed -i -e "s/publishOrderUpdates = false/publishOrderUpdates = true/g" ${witnesshome}/config/app.toml
sed -i -e "s/publishAccountBalance = false/publishAccountBalance = true/g" ${witnesshome}/config/app.toml
sed -i -e "s/publishOrderBook = false/publishOrderBook = true/g" ${witnesshome}/config/app.toml
sed -i -e "s/accountBalanceTopic = \"accounts\"/accountBalanceTopic = \"test\"/g" ${witnesshome}/config/app.toml
sed -i -e "s/orderBookTopic = \"books\"/orderBookTopic = \"test\"/g" ${witnesshome}/config/app.toml

# turn on debug level log
sed -i -e "s/log_level = \"main:info,state:info,\*:error\"/log_level = \"debug\"/g" ${witnesshome}/config/config.toml

${executable} start --home ${witnesshome} > ${witnesshome}/log.txt 2>&1 &
witness_pid=$!
echo ${witness_pid}
