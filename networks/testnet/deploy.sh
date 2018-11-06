#!/usr/bin/env bash

# options
skip_timeout=true
is_build=false

while true ; do
    case "$1" in
		--build )
			is_build=$2
			shift 2
		;;
		--skip_timeout )
			skip_timeout=$2
			shift 2
		;;
        *)
            break
        ;;
    esac
done;

cd ../..
work_path=$(pwd)

if ! [ -f build/node0/gaiad/config/genesis.json ];
then
	# these two lines are just used for generate a build directory own by bijieprd otherwise docker would create a build directory own by root
	make build-linux
	make build-docker-node
	docker run --rm -v $(pwd)/build:/bnbchaind:Z binance/bnbdnode testnet --v 4 --o . --starting-ip-address 172.20.0.2
fi

# variables
paths=("node0" "node1" "node2" "node3")
des_ips=("172.31.47.173" "172.31.47.173" "172.31.47.252" "172.31.47.252")
src_ips=("172.20.0.2" "172.20.0.3" "172.20.0.4" "172.20.0.5")
machines=("172.31.47.173" "172.31.47.252" "172.31.35.68")
bridge_ip="172.31.35.68"
witness_ip="172.31.35.68"
kafka_ip="172.31.47.173"
home_path="/server/bnc"

if [ "${is_build}" = true ]
then
	# build
	cd ${work_path}
	make get_vendor_deps
	make build

	cd ${work_path}/..

	cp -rf utils.go /home/bijieprd/gowork/src/github.com/BiJie/BinanceChain/vendor/github.com/cosmos/cosmos-sdk/client/keys/ > /dev/null
	tar -zcvf BinanceChain.tar.gz --exclude=BinanceChain/build BinanceChain > /dev/null
	for i in {0..2}
	do
		echo "Copying repo to host ${machines[$i]}..."
		ssh bijieprd@${machines[$i]} "sudo rm -rf ~/gowork/src/github.com/BiJie/BinanceChain"
		scp BinanceChain.tar.gz bijieprd@${machines[$i]}:/home/bijieprd/gowork/src/github.com/BiJie > /dev/null
		ssh bijieprd@${machines[$i]} "source ~/.zshrc && cd ~/gowork/src/github.com/BiJie && tar -zxvf BinanceChain.tar.gz > /dev/null && cd BinanceChain && make build"
	done
fi

cd ${work_path}/build

## prepare validators
# close pex
for p in "${paths[@]}"
do
  	$(cd "${p}/gaiad/config" && sed -i -e "s/pex = true/pex = false/g" config.toml)
    $(cd "${p}/gaiad/config" && sed -i -e "s/logToConsole = true/logToConsole = false/g" app.toml)

	if [ "${skip_timeout}" = true ]
	then
	  	$(cd "${p}/gaiad/config" && sed -i -e "s/skip_timeout_commit = false/skip_timeout_commit = true/g" config.toml)
	fi
done

# change gentx
for i in {0..3}
do
	idx=`expr $i % 2`
	$(cd "${paths[$i]}/gaiad/config/gentx/" && sed -i -e "s/172.20.0.[0-9]/${des_ips[$i]}/g" *.json)
done

# change port
for i in {0..3}
do
	if [ `expr $i % 2` == 1 ]
	then
		sed -i -e "s/26658/26668/g" "${paths[$i]}/gaiad/config/config.toml"
		sed -i -e "s/26656/26666/g" "${paths[$i]}/gaiad/config/config.toml"
		sed -i -e "s/6060/6070/g" "${paths[$i]}/gaiad/config/config.toml"
		sed -i -e "s/26657/26667/g" "${paths[$i]}/gaiad/config/config.toml"
		sed -i -e "s/26660/26670/g" "${paths[$i]}/gaiad/config/config.toml"
	fi
done

# change persistent peers
for j in {0..3}
do
	for i in {0..3}
	do
		if [ `expr $i % 2` == 0 ]
		then
			sed -i -e "s/${src_ips[$i]}:[0-9]\{5\}/${des_ips[$i]}:26656/g" "${paths[$j]}/gaiad/config/config.toml"
		else
			sed -i -e "s/${src_ips[$i]}:[0-9]\{5\}/${des_ips[$i]}:26666/g" "${paths[$j]}/gaiad/config/config.toml"
		fi
	done
done


# distribute config
for i in {0..3}
do
	echo "Stop validator ${paths[$i]} in host ${des_ips[$i]}..."
	ssh bijieprd@${des_ips[$i]} "ps -ef | grep bnbchain | grep ${paths[$i]} | awk '{print \$2}' | xargs kill -9"
	ssh bijieprd@${des_ips[$i]} "rm -rf ${home_path}/${paths[$i]}"
	echo "Copying ${paths[$i]} config to host ${des_ips[$i]}..."
	scp -r ${paths[$i]} bijieprd@${des_ips[$i]}:${home_path} > /dev/null
done

# deploy
for i in {0..3}
do
	echo "Starting validator ${paths[$i]} in host ${des_ips[$i]}..."
	ssh bijieprd@${des_ips[$i]} "cd ~/gowork/src/github.com/BiJie/BinanceChain/build && nohup ./bnbchaind --home ${home_path}/${paths[$i]}/gaiad start > ${home_path}/${paths[$i]}/${paths[$i]}.log 2>&1 &"
done


# post deploy
for i in {0..3}
do
	echo "Adding account at validator ${paths[$i]} in host ${des_ips[$i]}..."
	ssh bijieprd@${des_ips[$i]} "expect ~/gowork/src/github.com/BiJie/BinanceChain/networks/testnet/add_key.exp \"zc\" \"~/gowork/src/github.com/BiJie/BinanceChain/build/bnbcli\" \"${home_path}/${paths[$i]}/gaiacli 2>&1 &"
	ssh bijieprd@${des_ips[$i]} "expect ~/gowork/src/github.com/BiJie/BinanceChain/networks/testnet/add_key.exp \"zz\" \"~/gowork/src/github.com/BiJie/BinanceChain/build/bnbcli\" \"${home_path}/${paths[$i]}/gaiacli 2>&1 &"
done

## prepare bridge node
rm -rf node_bridge
cp -r node0 node_bridge
rm -rf node_bridge/gaiad/config/gentx node_bridge/gaiad/config/node_key.json node_bridge/gaiad/config/priv_validator.json

# turn on pex
sed -i -e "s/pex = false/pex = true/g" node_bridge/gaiad/config/config.toml

# set seeds
seeds=$(grep persistent_peers node_bridge/gaiad/config/config.toml | grep -o '".*"')
sed -i -e "s/seeds = \"\"/seeds = ${seeds}/g" node_bridge/gaiad/config/config.toml

# clear persistent peers
sed -i -e 's/persistent_peers = ".*"/persistend_peers = ""/g' node_bridge/gaiad/config/config.toml

# set private_ids
private_ids=$(echo ${seeds} | sed 's/@[0-9]*.[0-9]*.[0-9]*.[0-9]*:[0-9]*//g')
sed -i -e "s/private_peer_ids = \"\"/private_peer_ids = ${private_ids}/g" node_bridge/gaiad/config/config.toml

# distribute config
echo "Stopping bridge node in host  ${bridge_ip}..."
ssh bijieprd@${bridge_ip} "ps -ef | grep bnbchain | grep node_bridge | awk '{print \$2}' | xargs kill -9"
ssh bijieprd@${bridge_ip} "rm -rf ${home_path}/node_bridge"

echo "Copying config to bridge node ${bridge_ip}..."
scp -r node_bridge bijieprd@${bridge_ip}:${home_path} > /dev/null

echo "Starting bridge node..."
ssh bijieprd@${bridge_ip} "cd ~/gowork/src/github.com/BiJie/BinanceChain/build && nohup ./bnbchaind --home ${home_path}/node_bridge/gaiad start > ${home_path}/node_bridge/node_bridge.log 2>&1 &"

## prepare seed node
bridge_id=$(ssh bijieprd@${bridge_ip} "cd ~/gowork/src/github.com/BiJie/BinanceChain/build && ./bnbcli --home ${home_path}/node_bridge/gaiad status")
bridge_id=$(echo ${bridge_id} | grep -o "\"id\":\"[a-zA-Z0-9]*\"" | sed "s/\"//g" | sed "s/id://g")

# prepare config directory
rm -rf node_seed
cp -r node0 node_seed
rm -rf node_seed/gaiad/config/gentx node_seed/gaiad/config/node_key.json node_seed/gaiad/config/priv_validator.json

# turn on pex
sed -i -e "s/pex = false/pex = true/g" node_seed/gaiad/config/config.toml

# set seeds
sed -i -e "s/seeds = \"\"/seeds = \"${bridge_id}@${bridge_ip}:26656\"/g" node_bridge/gaiad/config/config.toml

# clear persistent peers
sed -i -e 's/persistent_peers = ".*"/persistend_peers = ""/g' node_bridge/gaiad/config/config.toml

# stop previous node
echo "Stopping seed node..."
ps -ef | grep bnbchain | grep seed | awk '{print $2}' | xargs kill -9

# start seed node
rm -rf ${home_path}/node_seed
cp -r node_seed ${home_path}/node_seed

echo "Starting seed node..."
nohup ./bnbchaind --home ${home_path}/node_seed/gaiad start > ${home_path}/node_seed/node_seed.log 2>&1 &

## prepare witness
rm -rf node_witness
cp -r node0 node_witness
rm -rf node_witness/gaiad/config/gentx node_witness/gaiad/config/node_key.json node_witness/gaiad/config/priv_validator.json

# turn on pex
sed -i -e "s/pex = false/pex = true/g" node_witness/gaiad/config/config.toml

# change port
sed -i -e "s/26658/26668/g" "node_witness/gaiad/config/config.toml"
sed -i -e "s/26656/26666/g" "node_witness/gaiad/config/config.toml"
sed -i -e "s/6060/6070/g" "node_witness/gaiad/config/config.toml"
sed -i -e "s/26657/26667/g" "node_witness/gaiad/config/config.toml"
sed -i -e "s/26660/26670/g" "node_witness/gaiad/config/config.toml"

# set seeds
sed -i -e "s/seeds = \"\"/seeds = \"${bridge_id}@${bridge_ip}:26656\"/g" node_witness/gaiad/config/config.toml

# clear persistent peers
sed -i -e 's/persistent_peers = ".*"/persistend_peers = ""/g' node_witness/gaiad/config/config.toml

# distribute config
echo "Stopping witness node in host  ${witness_ip}..."
ssh bijieprd@${witness_ip} "ps -ef | grep bnbchain | grep node_witness | awk '{print \$2}' | xargs kill -9"
ssh bijieprd@${witness_ip} "rm -rf ${home_path}/node_witness"

echo "Copying config to witness node ${witness_ip}..."
scp -r node_witness bijieprd@${witness_ip}:${home_path} > /dev/null

echo "Starting witness node..."
ssh bijieprd@${witness_ip} "cd ~/gowork/src/github.com/BiJie/BinanceChain/build && nohup ./bnbchaind --home ${home_path}/node_witness/gaiad start >> ${home_path}/node_witness/gaiad/bnc.log 2>&1 &"

## prepare publisher
rm -rf node_publisher
cp -r node0 node_publisher
rm -rf node_publisher/gaiad/config/gentx node_publisher/gaiad/config/node_key.json node_publisher/gaiad/config/priv_validator.json

# turn on pex
sed -i -e "s/pex = false/pex = true/g" node_publisher/gaiad/config/config.toml

# turn on debug level log
sed -i -e "s/log_level = \"main:info,state:info,\*:error\"/log_level = \"debug\"/g" node_publisher/gaiad/config/config.toml

# turn on prometheus
sed -i -e "s/prometheus = false/prometheus = true/g" node_publisher/gaiad/config/config.toml

# change port
sed -i -e "s/26658/27658/g" "node_publisher/gaiad/config/config.toml"
sed -i -e "s/26656/27656/g" "node_publisher/gaiad/config/config.toml"
sed -i -e "s/6060/7060/g" "node_publisher/gaiad/config/config.toml"
sed -i -e "s/26657/27657/g" "node_publisher/gaiad/config/config.toml"
sed -i -e "s/26660/27660/g" "node_publisher/gaiad/config/config.toml"

# set seeds
sed -i -e "s/seeds = \"\"/seeds = \"${bridge_id}@${bridge_ip}:26656\"/g" node_publisher/gaiad/config/config.toml

# clear persistent peers
sed -i -e 's/persistent_peers = ".*"/persistend_peers = ""/g' node_publisher/gaiad/config/config.toml

# set tx index config
sed -i -e "s/index_tags = \"\"/index_tags = \"tx.height\"/g" node_publisher/gaiad/config/config.toml

# set publish config - ip
sed -i -e "s/orderUpdatesKafka = \"127.0.0.1:9092\"/orderUpdatesKafka = \"${kafka_ip}:9092\"/g" node_publisher/gaiad/config/app.toml
sed -i -e "s/accountBalanceKafka = \"127.0.0.1:9092\"/accountBalanceKafka = \"${kafka_ip}:9092\"/g" node_publisher/gaiad/config/app.toml
sed -i -e "s/orderBookKafka = \"127.0.0.1:9092\"/orderBookKafka = \"${kafka_ip}:9092\"/g" node_publisher/gaiad/config/app.toml
sed -i -e "s/publishOrderUpdates = false/publishOrderUpdates = true/g" node_publisher/gaiad/config/app.toml
sed -i -e "s/publishAccountBalance = false/publishAccountBalance = true/g" node_publisher/gaiad/config/app.toml
sed -i -e "s/publishOrderBook = false/publishOrderBook = true/g" node_publisher/gaiad/config/app.toml
sed -i -e "s/accountBalanceTopic = \"accounts\"/accountBalanceTopic = \"test\"/g" node_publisher/gaiad/config/app.toml
sed -i -e "s/orderBookTopic = \"orders\"/orderBookTopic = \"test\"/g" node_publisher/gaiad/config/app.toml
sed -i -e "s/orderUpdatesTopic = \"orders\"/orderUpdatesTopic = \"test\"/g" node_publisher/gaiad/config/app.toml

# distribute config
echo "Stopping publisher node in host  ${witness_ip}..."
ssh bijieprd@${witness_ip} "ps -ef | grep bnbchain | grep node_publisher | awk '{print \$2}' | xargs kill -9"
ssh bijieprd@${witness_ip} "rm -rf ${home_path}/node_publisher"

echo "Copying config to publisher node ${witness_ip}..."
scp -r node_publisher bijieprd@${witness_ip}:${home_path} > /dev/null

# start an api-server to query tx
ps -ef | grep "bnbcli" | grep "api-server" | awk '{print $2}' | xargs kill -9
nohup /home/bijieprd/gowork/src/github.com/BiJie/BinanceChain/build/bnbcli --laddr tcp://0.0.0.0:8080 --node tcp://${witness_ip}:27657 api-server > ${home_path}/cong/api-server.log 2>&1 &

echo "Starting publisher node..."
ssh bijieprd@${witness_ip} "cd ~/gowork/src/github.com/BiJie/BinanceChain/build && nohup ./bnbchaind --home ${home_path}/node_publisher/gaiad start > ${home_path}/node_publisher/node_publisher.log 2>&1 &"