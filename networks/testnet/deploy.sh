#!/usr/bin/env bash

cd ../../build/

paths=("node0" "node1" "node2" "node3")
des_ips=("172.31.47.173" "172.31.47.173" "172.31.47.252" "172.31.47.252")
src_ips=("172.20.0.2" "172.20.0.3" "172.20.0.4" "172.20.0.5")
bridge_ip="172.31.35.68"
witness_ip="172.31.35.68"

## prepare validators
# close pex
for p in "${paths[@]}"
do
  $(cd "${p}/gaiad/config" && sed -i -e "s/pex = true/pex = false/g" config.toml)
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
	ssh bijieprd@${des_ips[$i]} "rm -rf ~/${paths[$i]}"
	echo "Copying ${paths[$i]} config to host ${des_ips[$i]}..."
	scp -r ${paths[$i]} bijieprd@${des_ips[$i]}:/home/bijieprd > /dev/null
done

# deploy
for i in {0..3}
do
	echo "Starting validator ${paths[$i]} in host ${des_ips[$i]}..."
	ssh bijieprd@${des_ips[$i]} "cd ~/gowork/src/github.com/BiJie/BinanceChain/build && nohup ./bnbchaind --home ~/${paths[$i]}/gaiad start > ~/${paths[$i]}.log 2>&1 &"
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
ssh bijieprd@${bridge_ip} "rm -rf ~/node_bridge"

echo "Copying config to bridge node ${bridge_ip}..."
scp -r node_bridge bijieprd@${bridge_ip}:/home/bijieprd > /dev/null

echo "Starting bridge node..."
ssh bijieprd@${bridge_ip} "cd ~/gowork/src/github.com/BiJie/BinanceChain/build && nohup ./bnbchaind --home ~/node_bridge/gaiad start > ~/node_bridge.log 2>&1 &"

## prepare seed node
bridge_id=$(ssh bijieprd@${bridge_ip} "cd ~/gowork/src/github.com/BiJie/BinanceChain/build && ./bnbcli --home ~/node_bridge/gaiad status")
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
rm -rf ~/node_seed
cp -r node_seed ~/node_seed

echo "Starting seed node..."
nohup ./bnbchaind --home ~/node_seed/gaiad start > ~/node_seed.log 2>&1 &

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
ssh bijieprd@${witness_ip} "rm -rf ~/node_witness"

echo "Copying config to witness node ${witness_ip}..."
scp -r node_witness bijieprd@${witness_ip}:/home/bijieprd > /dev/null

echo "Starting witness node..."
ssh bijieprd@${witness_ip} "cd ~/gowork/src/github.com/BiJie/BinanceChain/build && nohup ./bnbchaind --home ~/node_witness/gaiad start > ~/node_witness.log 2>&1 &"
