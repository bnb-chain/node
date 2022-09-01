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
result=$(expect ./recover.exp "${bob_secret}" "bob" true)
check_operation "Add Key" "${result}" "${keys_operation_words}"
# wait for the chain
sleep 10s

#secret=$(cat ${home}/secret)

alice_addr=$(./bnbcli keys list --home ${cli_home} | grep alice | grep -o "bnb1[0-9a-zA-Z]*")
bob_addr=$(./bnbcli keys list --home ${cli_home} | grep bob | grep -o "bnb1[0-9a-zA-Z]*")

# send
result=$(expect ./send.exp ${cli_home} alice ${chain_id} "100000000000000:BNB" ${bob_addr})
check_operation "Send Token" "${result}" "${chain_operation_words}"

go run ../cmd/test_client

exit_test 0
