#!/usr/bin/env bash

# start a validator and witness on local machine
# later on db indexer and publisher can be setup and started by newIndexer.sh and newPublisher.sh
# later on order generation can be kicked off by ordergen.sh
# TODO: support two validator without docker in same machine

########################### SETUP #########################
src='/Users/zhaocong/go/src/github.com/binance-chain/node'
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
sed -i -e "s/publishLocal = false/publishLocal = true/g" ${deamonhome}/config/app.toml
sed -i -e 's/"voting_period": "1209600000000000"/"voting_period": "5000000000"/g' ${deamonhome}/config/genesis.json

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
sed -i -e "s/state_sync = false/state_sync = true/g" ${witnesshome}/config/config.toml
${executable} start --pruning breathe --home ${witnesshome} > ${witnesshome}/log.txt 2>&1 &
witness_pid=$!
echo ${witness_pid}

# init accounts
result=$(expect ${scripthome}/recover.exp "${secret}" "zc" "${clipath}" "${clihome}")
result=$(expect ${scripthome}/add_key.exp "zz" "${clipath}" "${clihome}")
zz_addr=$(${cli} keys list | grep "zz.*local" | grep -o "bnb[0-9a-zA-Z]*" | grep -v "bnbp")

# issue&list NNB and ZCB for ordergen
result=$(${cli} token issue --from=zc --token-name="New BNB Coin" --symbol=NNB --total-supply=2000000000000000 --chain-id ${chain_id})
nnb_symbol=$(echo "${result}" | tail -n 1 | grep -o "NNB-[0-9A-Z]*")
echo ${nnb_symbol}
sleep 5
${cli} gov submit-list-proposal --chain-id ${chain_id} --from zc --deposit 200000000000:BNB --base-asset-symbol ${nnb_symbol} --quote-asset-symbol BNB --init-price 1000000000 --title "list NNB/BNB" --description "list NNB/BNB" --expire-time 1644486400 --json
sleep 2
${cli} gov vote --from zc --chain-id ${chain_id} --proposal-id 1 --option Yes --json
sleep 6
${cli} dex list -s=${nnb_symbol} --quote-asset-symbol=BNB --init-price=1000000000 --from=zc --chain-id ${chain_id} --proposal-id 1
sleep 1
result=$(${cli} token issue --from=zc --token-name="ZC Coin" --symbol=ZCB --total-supply=2000000000000000 --chain-id ${chain_id})
zcb_symbol=$(echo "${result}" | tail -n 1 | grep -o "ZCB-[0-9A-Z]*")
echo ${zcb_symbol}
sleep 5
${cli} gov submit-list-proposal --chain-id ${chain_id} --from zc --deposit 200000000000:BNB --base-asset-symbol ${zcb_symbol} --quote-asset-symbol BNB --init-price 1000000000 --title "list NNB/BNB" --description "list NNB/BNB" --expire-time 1644486400 --json
sleep 2
${cli} gov vote --from zc --chain-id ${chain_id} --proposal-id 2 --option Yes --json
sleep 6
${cli} dex list -s=${zcb_symbol} --quote-asset-symbol=BNB --init-price=1000000000 --from=zc --chain-id ${chain_id} --proposal-id 2
sleep 1
${cli} send --from=zc --to=${zz_addr} --amount=1000000000000000:BNB --chain-id ${chain_id}
sleep 5