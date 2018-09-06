#!/bin/bash

cd ./build
if [ $? -ne 0 ]; then
	exit 1
fi

cli_home="./testnodecli"
home="./testnoded"
chain_id="test-chain-5dvMS7"

function prepare_node() {
	cp -f ../networks/demo/*.exp .

	rm -rf ${cli_home}
	rm -rf ${home}
	mkdir ${cli_home}
	mkdir ${home}

	secret=$(./bnbchaind init --name testnode --home ./testnoded --home-client ./testnodecli --chain-id ${chain_id} | grep secret | grep -o ":.*" | grep -o "\".*"  | sed "s/\"//g")

	$(cd "./${home}/config" && sed -i -e "s/skip_timeout_commit = false/skip_timeout_commit = true/g" config.toml)

	# stop and start node
	ps -ef  | grep bnbchaind | grep testnoded | awk '{print $2}' | xargs kill -9
	./bnbchaind start --home ./testnoded  > ./testnoded/node.log 2>&1 &

	echo ${secret}
}

function check_keys_operation() {
	echo "=================== Checking $1 ==================="
	echo "$2"

	echo "$2" | grep -q cosmosaccaddr
	if [ $? -ne 0 ]; then
		echo "Checking $1 Failed"
		exit 1
	fi
}

function check_chain_operation() {
	echo "=================== Checking $1 ==================="
	echo "$2"

	echo "$2" | grep -q Committed
	if [ $? -ne 0 ]; then
		echo "Checking $1 Failed"
		exit 1
	fi
}

function check_order_book_operation() {
	echo "=================== Checking $1 ==================="
	echo "$2"

	echo "$2" | grep -q 10.00000000
	if [ $? -ne 0 ]; then
		echo "Checking $1 Failed"
		exit 1
	fi
}

secret=$(prepare_node)

result=$(expect ./recover.exp "${secret}" "alice" true)
check_keys_operation "Recover Key" "${result}"

bob_secret="bottom quick strong ranch section decide pepper broken oven demand coin run jacket curious business achieve mule bamboo remain vote kid rigid bench rubber"
result=$(expect ./add_key.exp "${bob_secret}" "bob")
check_keys_operation "Add Key" "${result}"

alice_addr=$(./bnbcli keys list --home ./testnodecli | grep alice | grep -o "cosmosaccaddr[0-9a-zA-Z]*")
bob_addr=$(./bnbcli keys list --home ./testnodecli | grep bob | grep -o "cosmosaccaddr[0-9a-zA-Z]*")

# wait for the chain
sleep 10s

# send
result=$(expect ./send.exp ./testnodecli alice ${chain_id} 100000000000000BNB ${bob_addr})
check_chain_operation "Send Token" "${result}"

sleep 1s
# issue token
result=$(expect ./issue.exp BTC Bitcoin 1000000000000000 bob ${chain_id} "./testnodecli")
check_chain_operation "Issue Token" "${result}"

sleep 1s
# list trading pair
result=$(expect ./list.exp BTC BNB 100000000 bob ${chain_id} ./testnodecli)
check_chain_operation "List Trading Pair" "${result}"

sleep 1s
# place buy order
buy_order=$(expect ./order.exp BTC_BNB 1 100000000 1000000000 alice ${chain_id} gtc ./testnodecli/)
check_chain_operation "Place Order" "${result}"

order_id=$(echo "${buy_order}" | grep -o "cosmosaccaddr[0-9a-zA-Z]*-[0-9]*")

sleep 1s
# cancel order
result=$(expect ./cancel.exp ${order_id} alice ${chain_id} ./testnodecli/)
check_chain_operation "Cancel Order" "${result}"

sleep 1s
# place buy order
result=$(expect ./order.exp BTC_BNB 1 100000000 1000000000 alice ${chain_id} gtc ./testnodecli/)
check_chain_operation "Place Order" "${result}"

./bnbcli dex show -l BTC_BNB

sleep 1s
# place Sell order
result=$(expect ./order.exp BTC_BNB 2 100000000 2000000000 bob ${chain_id} gtc ./testnodecli/)
check_chain_operation "Place Order" "${result}"

result=$(./bnbcli dex show -l BTC_BNB)
check_order_book_operation "Order Book" "${result}"