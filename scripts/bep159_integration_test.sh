#!/bin/bash

set -ex

cd ./build
if [ $? -ne 0 ]; then
	echo "path build does not exists"
	exit 1
fi

cli_home="./testnodecli"
home="./testnoded"
chain_id="bnbchain-1000"

keys_operation_words="bnb"
chain_operation_words="Committed"

function prepare_node() {
	cp -f ../networks/demo/*.exp .

	rm -rf ${cli_home}
	rm -rf ${home}
	mkdir ${cli_home}
	mkdir ${home}

	secret=$(./bnbchaind init --moniker testnode --home ${home} --home-client ${cli_home} --chain-id ${chain_id} | grep secret | grep -o ":.*" | grep -o "\".*"  | sed "s/\"//g")
	echo ${secret} > ${home}/secret

    $(cd "./${home}/config" && sed -i -e "s/BEP12Height = 9223372036854775807/BEP12Height = 1/g" app.toml)
    $(cd "./${home}/config" && sed -i -e "s/BEP3Height = 9223372036854775807/BEP3Height = 1/g" app.toml)
    $(cd "./${home}/config" && sed -i -e "s/timeout_commit = \"1s\"/timeout_commit = \"500ms\"/g" config.toml)
	$(cd "./${home}/config" && sed -i -e "s/log_level = \"main\:info,state\:info,\*\:error\"/log_level = \"*\:debug\"/g" config.toml)
	$(cd "./${home}/config" && sed -i -e "s/\"min_self_delegation\": \"1000000000000\"/\"min_self_delegation\": \"10000000000\"/g" genesis.json)
	$(cd "./${home}/config" && sed -i -e "s/BEP3Height = 9223372036854775807/BEP3Height = 1/g" app.toml)
	$(cd "./${home}/config" && sed -i -e "s/BEP8Height = 9223372036854775807/BEP8Height = 1/g" app.toml)
	$(cd "./${home}/config" && sed -i -e "s/BEP67Height = 9223372036854775807/BEP67Height = 1/g" app.toml)
	$(cd "./${home}/config" && sed -i -e "s/BEP82Height = 9223372036854775807/BEP82Height = 1/g" app.toml)
	$(cd "./${home}/config" && sed -i -e "s/BEP84Height = 9223372036854775807/BEP84Height = 1/g" app.toml)
	$(cd "./${home}/config" && sed -i -e "s/BEP87Height = 9223372036854775807/BEP87Height = 1/g" app.toml)
	$(cd "./${home}/config" && sed -i -e "s/FixFailAckPackageHeight = 9223372036854775807/FixFailAckPackageHeight = 1/g" app.toml)
	$(cd "./${home}/config" && sed -i -e "s/EnableAccountScriptsForCrossChainTransferHeight = 9223372036854775807/EnableAccountScriptsForCrossChainTransferHeight = 1/g" app.toml)
	$(cd "./${home}/config" && sed -i -e "s/BEP70Height = 9223372036854775807/BEP70Height = 1/g" app.toml)
	$(cd "./${home}/config" && sed -i -e "s/BEP128Height = 9223372036854775807/BEP128Height = 1/g" app.toml)
	$(cd "./${home}/config" && sed -i -e "s/BEP151Height = 9223372036854775807/BEP151Height = 1/g" app.toml)
	$(cd "./${home}/config" && sed -i -e "s/BEP153Height = 9223372036854775807/BEP153Height = 2/g" app.toml)
	$(cd "./${home}/config" && sed -i -e "s/BEP159Height = 9223372036854775807/BEP159Height = 3/g" app.toml)
	$(cd "./${home}/config" && sed -i -e "s/BEP159Phase2Height = 9223372036854775807/BEP159Phase2Height = 11/g" app.toml)
	$(cd "./${home}/config" && sed -i -e "s/breatheBlockInterval = 0/breatheBlockInterval = 5/g" app.toml)
#	$(cd "./${home}/config" && sed -i -e "s/publishStaking = false/publishStaking = true/g" app.toml)
#	$(cd "./${home}/config" && sed -i -e "s/publishKafka = false/publishKafka = true/g" app.toml)
#	$(cd "./${home}/config" && sed -i -e "s/publishLocal = false/publishLocal = true/g" app.toml)

	# stop and start node
	ps -ef  | grep bnbchaind | grep testnoded | awk '{print $2}' | xargs kill -9
	./bnbchaind start --home ${home}  > ./testnoded/node.log 2>&1 &

	echo ${secret}
}

function exit_test() {
	# stop node
	ps -ef  | grep bnbchaind | grep testnoded | awk '{print $2}' | xargs kill -9
	exit $1
}

function check_operation() {
	printf "\n=================== Checking $1 ===================\n"
	echo "$2"

	echo "$2" | grep -q $3
	if [ $? -ne 0 ]; then
		echo "Checking $1 Failed"
		exit_test 1
	fi
}

secret=$(prepare_node)
result=$(expect ./recover.exp "${secret}" "alice" true)
check_operation "Recover Key" "${result}" "${keys_operation_words}"

bob_secret="bottom quick strong ranch section decide pepper broken oven demand coin run jacket curious business achieve mule bamboo remain vote kid rigid bench rubber"
bob_val_addr=bva1ddt3ls9fjcd8mh69ujdg3fxc89qle2a7k8spre
bob_pubkey=bcap1zcjduepqes09r5x3kqnv7nlrcrveh5sxsrqxw222wu8999fa2wpnjher4yxst89v4a
bob_pubkey_new=bcap1zcjduepqcde4hk9kac248hqr3vqxle049f9l5zc58rcacy6nphuay5wt6c5q3ydes7
result=$(expect ./recover.exp "${bob_secret}" "bob" true)
check_operation "Add Key" "${result}" "${keys_operation_words}"
carl_secret="mad calm portion vendor fine weather thunder ensure simple fish enrich genre plate kind minor random where crop hero soda isolate pelican provide chimney"
result=$(expect ./recover.exp "${carl_secret}" "carl" true)
check_operation "Add Key" "${result}" "${keys_operation_words}"
# wait for the chain
sleep 10s

alice_addr=$(./bnbcli keys list --home ${cli_home} | grep alice | grep -o "bnb1[0-9a-zA-Z]*")
bob_addr=$(./bnbcli keys list --home ${cli_home} | grep bob | grep -o "bnb1[0-9a-zA-Z]*")
carl_addr=$(./bnbcli keys list --home ${cli_home} | grep carl | grep -o "bnb1[0-9a-zA-Z]*")

# send
result=$(expect ./send.exp ${cli_home} alice ${chain_id} "100000000000000:BNB" ${bob_addr})
check_operation "Send Token" "${result}" "${chain_operation_words}"
sleep 5s
result=$(expect ./send.exp ${cli_home} alice ${chain_id} "100000000000000:BNB" ${carl_addr})
check_operation "Send Token" "${result}" "${chain_operation_words}"
sleep 5s

# get parameters
result=$(./bnbcli staking parameters --home ${cli_home} --trust-node)
check_operation "Query Staking Parameters" "${result}" "proposer"

# get validators
result=$(./bnbcli staking validators --home ${cli_home} --trust-node)
check_operation "Get Validators" "${result}" "Operator"

# get validator
operator_address=$(echo "${result}" | grep Operator | grep -o "bva[0-9a-zA-Z]*" | head -n1)
result=$(./bnbcli staking validator ${operator_address} --home ${cli_home} --trust-node)
check_operation "Get Validator" "${result}" "Operator"

# get delegations
result=$(./bnbcli staking delegations ${alice_addr} --home ${cli_home} --trust-node)
check_operation "Get Delegations" "${result}" "Validator"

# get delegation
validator_address=$(echo "${result}" | grep Validator | grep -o "bva[0-9a-zA-Z]*")
delegator_address=$(echo "${result}" | grep Delegator | grep -o "bnb1[0-9a-zA-Z]*")
result=$(./bnbcli staking delegation --address-delegator ${delegator_address} --validator ${validator_address} --home ${cli_home} --trust-node)
check_operation "Get Delegation" "${result}" "Validator"

# get pool
result=$(./bnbcli staking pool --home ${cli_home} --trust-node)

# create validator
result=$(expect ./create-validator-open.exp ${cli_home} bob ${chain_id} ${bob_pubkey})
check_operation "create validator open" "${result}" "${chain_operation_words}"
sleep 5s
result=$(./bnbcli staking validators --home ${cli_home} --trust-node)
check_operation "Get Validators" "${result}" "Operator"
result=$(./bnbcli staking validator ${bob_val_addr} --home ${cli_home} --trust-node)
check_operation "Get Validator" "${result}" "bob"
check_operation "Get Validator" "${result}" "${bob_pubkey}"

# edit validator
result=$(expect ./edit-validator.exp ${cli_home} bob ${chain_id} ${bob_pubkey_new})
check_operation "edit validator" "${result}" "${chain_operation_words}"
sleep 5s
result=$(./bnbcli staking validator ${bob_val_addr} --home ${cli_home} --trust-node)
check_operation "Get Validator" "${result}" "bob-new"
check_operation "Get Validator" "${result}" "${bob_pubkey_new}"
bob_val_addr=$(echo "${result}" | grep Operator | grep -o "bva[0-9a-zA-Z]*")

## delegate
#result=$(expect ./delegate.exp ${cli_home} carl ${chain_id} "1000000000:BNB" ${validator_address})
#check_operation "delegate" "${result}" "${chain_operation_words}"
#sleep 5s
#result=$(./bnbcli staking delegation --address-delegator ${carl_addr} --validator ${validator_address} --home ${cli_home} --trust-node)
#check_operation "Get Delegation" "${result}" "Validator"
#
## redelegate
#result=$(expect ./redelegate.exp ${cli_home} carl ${chain_id} "600000000:BNB" ${validator_address} ${bob_val_addr})
#check_operation "redelegate" "${result}" "${chain_operation_words}"
#sleep 5s
#
## undelegate
#result=$(expect ./undelegate.exp ${cli_home} carl ${chain_id} "400000000:BNB" ${validator_address})
#check_operation "undelegate" "${result}" "${chain_operation_words}"
#sleep 5s
#
## get redelegations
#result=$(./bnbcli staking redelegations ${carl_addr} --home ${cli_home} --trust-node)
#check_operation "Get Redelegations" "${result}" "delegator_addr"
#
## get redelegation
#result=$(./bnbcli staking redelegation --address-delegator ${carl_addr} --addr-validator-source ${validator_address} --addr-validator-dest ${bob_val_addr} --home ${cli_home} --trust-node)
#check_operation "Get Redelegation" "${result}" "Delegator"
#
## get unbonding-delegations
#result=$(./bnbcli staking unbonding-delegations ${carl_addr} --home ${cli_home} --trust-node)
#check_operation "Get Unbonding-Delegations" "${result}" "delegator_addr"
#
## get unbonding-delegation
#result=$(./bnbcli staking unbonding-delegation --address-delegator ${carl_addr} --validator ${validator_address} --home ${cli_home} --trust-node)
#check_operation "Get Unbonding-Delegation" "${result}" "Delegator"

# run test with go-sdk
cd ../e2e && go run .

exit_test 0
