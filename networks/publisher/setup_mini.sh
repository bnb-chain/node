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

# clean history data
rm -r ${deamonhome}
rm -r ${clihome}
rm -r ${home}/.bnbchaind_witness

# build
cd ${src}
make build

# init a validator and witness node
${executable} init --moniker xxx --chain-id ${chain_id} > ${key_seed_path}/key_seed.json # cannot save into ${deamonhome}/init.json
secret=$(cat ${key_seed_path}/key_seed.json | grep secret | grep -o ":.*" | grep -o "\".*"  | sed "s/\"//g")
#echo ${secret}

mkdir -p ${home}/.bnbchaind_witness/config

#sed -i -e "s/skip_timeout_commit = false/skip_timeout_commit = true/g" ${deamonhome}/config/config.toml
sed -i -e "s/log_level = \"main:info,state:info,\*:error\"/log_level = \"debug\"/g" ${deamonhome}/config/config.toml
sed -i -e "s/allow_duplicate_ip = false/allow_duplicate_ip = true/g" ${deamonhome}/config/config.toml
sed -i -e "s/addr_book_strict = true/addr_book_strict = false/g" ${deamonhome}/config/config.toml

sed -i -e 's/logToConsole = true/logToConsole = false/g' ${deamonhome}/config/app.toml
sed -i -e 's/breatheBlockInterval = 0/breatheBlockInterval = 100/g' ${deamonhome}/config/app.toml
sed -i -e "s/publishOrderUpdates = false/publishOrderUpdates = true/g" ${deamonhome}/config/app.toml
sed -i -e "s/publishAccountBalance = false/publishAccountBalance = true/g" ${deamonhome}/config/app.toml
sed -i -e "s/publishOrderBook = false/publishOrderBook = true/g" ${deamonhome}/config/app.toml
sed -i -e "s/publishBlockFee = false/publishBlockFee = true/g" ${deamonhome}/config/app.toml
sed -i -e "s/publishTransfer = false/publishTransfer = true/g" ${deamonhome}/config/app.toml
sed -i -e "s/publishLocal = false/publishLocal = true/g" ${deamonhome}/config/app.toml

# config witness node
cp ${deamonhome}/config/genesis.json ${witnesshome}/config/
cp ${deamonhome}/config/app.toml ${witnesshome}/config/
cp ${deamonhome}/config/config.toml ${witnesshome}/config/

sed -i -e "s/26/27/g" ${witnesshome}/config/config.toml
sed -i -e "s/6060/7060/g" ${witnesshome}/config/config.toml
#sed -i -e "s/fastest_sync_height = -1/fastest_sync_height = 10/g" ${witnesshome}/config/config.toml

# start validator
${executable} start --pruning breathe > ${deamonhome}/log.txt 2>&1 &
validator_pid=$!
echo ${validator_pid}
sleep 60 # sleep in case cli status call failed to get node id
validatorStatus=$(${cli} status)
validator_id=$(echo ${validatorStatus} | grep -o "\"id\":\"[a-zA-Z0-9]*\"" | sed "s/\"//g" | sed "s/id://g")
#echo ${validator_id}

# set witness peer to validator and start witness
sed -i -e "s/persistent_peers = \"\"/persistent_peers = \"${validator_id}@127.0.0.1:26656\"/g" ${witnesshome}/config/config.toml
sed -i -e "s/state_sync_height = -1/state_sync_height = 0/g" ${witnesshome}/config/config.toml
${executable} start --pruning breathe --home ${witnesshome} > ${witnesshome}/log.txt 2>&1 &
witness_pid=$!
echo ${witness_pid}

# init accounts
result=$(expect ${scripthome}/recover.exp "${secret}" "zc" "${clipath}" "${clihome}")
result=$(expect ${scripthome}/add_key.exp "zz" "${clipath}" "${clihome}")
zz_addr=$(${cli} keys list | grep "zz.*local" | grep -o "bnb[0-9a-zA-Z]*" | grep -v "bnbp")

