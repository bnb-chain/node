#!/usr/bin/env bash

########################### SETUP #########################
home=$HOME
src="${home}/go/src/github.com/bnb-chain/node"
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

# clean history data
rm -r ${witnesshome}
mkdir -p ${witnesshome}/config

# config witness node
cp ${deamonhome}/config/genesis.json ${witnesshome}/config/
cp ${deamonhome}/config/config.toml ${witnesshome}/config/

sed -i -e "s/26/28/g" ${witnesshome}/config/config.toml
sed -i -e "s/6060/8060/g" ${witnesshome}/config/config.toml

# get validator id
validator_pid=$(ps aux | grep "bnbchaind start$" | awk '{print $2}')
validatorStatus=$(${cli} status)
validatorId=$(echo ${validatorStatus} | grep -o "\"id\":\"[a-zA-Z0-9]*\"" | sed "s/\"//g" | sed "s/id://g")
#echo ${validatorId}

# set witness peer to validator and start witness
sed -i -e "s/persistent_peers = \"\"/persistent_peers = \"${validatorId}@127.0.0.1:26656\"/g" ${witnesshome}/config/config.toml
#sed -i -e "s/index_tags = \"\"/index_tags = \"tx.height\"/g" ${witnesshome}/config/config.toml
sed -i -e "s/index_all_tags = false/index_all_tags = true/g" ${witnesshome}/config/config.toml
${executable} start --home ${witnesshome} > ${witnesshome}/log.txt 2>&1 &
witness_pid=$!
echo ${witness_pid}
