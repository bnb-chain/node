#!/usr/bin/env bash

src='/Users/zhaocong/go/src/github.com/BiJie/BinanceChain'
executable='/Users/zhaocong/go/src/github.com/BiJie/BinanceChain/build/bnbchaind'
cli='/Users/zhaocong/go/src/github.com/BiJie/BinanceChain/build/bnbcli'
home='/Users/zhaocong'

## clean history data
#rm -r ${home}/.bnbchaind_val2
#
## init a witness node
#${executable} init --name val2 --home ${home}/.bnbchaind_val2 > ~/init2.json

# config witness node
cp ${home}/.bnbchaind/config/genesis.json ${home}/.bnbchaind_val2/config/

sed -i -e "s/26/30/g" ${home}/.bnbchaind_val2/config/config.toml
sed -i -e "s/6060/10060/g" ${home}/.bnbchaind_val2/config/config.toml

# get validator id
validator_pid=$(ps aux | grep "bnbchaind start$" | awk '{print $2}')
validatorStatus=$(${cli} status)
validatorId=$(echo ${validatorStatus} | grep -o "\"id\":\"[a-zA-Z0-9]*\"" | sed "s/\"//g" | sed "s/id://g")
#echo ${validatorId}

# set witness peer to validator and start witness
sed -i -e "s/persistent_peers = \"\"/persistent_peers = \"${validatorId}@127.0.0.1:26656\"/g" ${home}/.bnbchaind_val2/config/config.toml
sed -i -e "s/index_all_tags = false/index_all_tags = true/g" ${home}/.bnbchaind_val2/config/config.toml
${executable} start --home ${home}/.bnbchaind_val2 > ${home}/.bnbchaind_val2/log.txt 2>&1 &
validator_pid=$!
echo ${validator_pid}
