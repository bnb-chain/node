#!/bin/bash

cd ./build
if [ $? -ne 0 ]; then
	echo "path build does not exists"
	exit 1
fi

cli_home="./testnodecli"
home="./testnoded"
chain_id="bnbchain-1000"

keys_operation_words="bnc"
chain_operation_words="Committed"
order_book_words="10.00000000"

round="1"
rounds="2"

function prepare_node() {
	cp -f ../networks/demo/*.exp .

	rm -rf ${cli_home}
	rm -rf ${home}
	mkdir ${cli_home}
	mkdir ${home}

	secret=$(./bnbchaind init --name testnode --home ${home} --home-client ${cli_home} --chain-id ${chain_id} | grep secret | grep -o ":.*" | grep -o "\".*"  | sed "s/\"//g")

	$(cd "./${home}/config" && sed -i -e "s/skip_timeout_commit = false/skip_timeout_commit = true/g" config.toml)

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
	printf "\n=================== Checking $1 (${round}/${rounds}) ===================\n"
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
result=$(expect ./add_key.exp "${bob_secret}" "bob")
check_operation "Add Key" "${result}" "${keys_operation_words}"

alice_addr=$(./bnbcli keys list --home ${cli_home} | grep alice | grep -o "bnc[0-9a-zA-Z]*")
bob_addr=$(./bnbcli keys list --home ${cli_home} | grep bob | grep -o "bnc[0-9a-zA-Z]*")

# wait for the chain
sleep 10s


## ROUND 1 ##

# send
result=$(expect ./send.exp ${cli_home} alice ${chain_id} "100000000000000:BNB" ${bob_addr})
check_operation "Send Token" "${result}" "${chain_operation_words}"

sleep 1s
# issue token
result=$(expect ./issue.exp BTC Bitcoin 1000000000000000 bob ${chain_id} ${cli_home})
check_operation "Issue Token" "${result}" "${chain_operation_words}"

sleep 1s
# list trading pair
result=$(expect ./list.exp BTC BNB 100000000 bob ${chain_id} ${cli_home})
check_operation "List Trading Pair" "${result}" "${chain_operation_words}"

sleep 1s
# place buy order
result=$(expect ./order.exp BTC_BNB 1 100000000 1000000000 alice ${chain_id} gtc ${cli_home})
check_operation "Place Order" "${result}" "${chain_operation_words}"

order_id=$(echo "${result}" | tail -n 1 | grep -o "[0-9A-Z]*-[0-9]*")
printf "Order ID: $order_id\n"

sleep 2s
# cancel order
result=$(expect ./cancel.exp BTC_BNB ${order_id} alice ${chain_id} ${cli_home})
check_operation "Cancel Order" "${result}" "${chain_operation_words}"

sleep 1s
# place buy order
result=$(expect ./order.exp BTC_BNB 1 100000000 1000000000 alice ${chain_id} gtc ${cli_home})
check_operation "Place Order" "${result}" "${chain_operation_words}"

echo ""
./bnbcli dex show -l BTC_BNB

sleep 1s
# place Sell order
result=$(expect ./order.exp BTC_BNB 2 100000000 2000000000 bob ${chain_id} gtc ${cli_home})
check_operation "Place Order" "${result}" "${chain_operation_words}"

result=$(./bnbcli dex show -l BTC_BNB)
check_operation "Order Book" "${result}" "${order_book_words}"


## ROUND 2 ##

round="2"

sleep 1s
# place buy order
result=$(expect ./order.exp BTC_BNB 1 100000000 2000000000 alice ${chain_id} gtc ${cli_home})
check_operation "Place Order" "${result}" "${chain_operation_words}"

order_id=$(echo "${result}" | tail -n 1 | grep -o "[0-9A-Z]*-[0-9]*")
printf "Order ID: $order_id\n"

sleep 2s
# cancel order
result=$(expect ./cancel.exp BTC_BNB ${order_id} alice ${chain_id} ${cli_home})
check_operation "Cancel Order" "${result}" "${chain_operation_words}"

sleep 1s
# place buy order
result=$(expect ./order.exp BTC_BNB 1 100000000 1000000000 alice ${chain_id} gtc ${cli_home})
check_operation "Place Order" "${result}" "${chain_operation_words}"

echo ""
./bnbcli dex show -l BTC_BNB

sleep 1s
# place Sell order
result=$(expect ./order.exp BTC_BNB 2 100000000 2000000000 bob ${chain_id} gtc ${cli_home})
check_operation "Place Order" "${result}" "${chain_operation_words}"

result=$(./bnbcli dex show -l BTC_BNB)
check_operation "Order Book" "${result}" "${order_book_words}"


exit_test 0
