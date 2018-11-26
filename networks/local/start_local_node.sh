#!/bin/bash
set -e

ORG="BiJie"
REPO="$ORG/BinanceChain"

# initial checks

if [ ! -d "$GOPATH" ]; then
	echo "GOPATH must be set and exist for this script to work"
	exit 1
fi

command -v dep
if [ $? -ne 0 ]; then
	echo "Installing dep"
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
fi

mkdir -p $GOPATH/src/github.com/$ORG

if [ ! -d "$GOPATH/src/github.com/$REPO" ]; then
	cd $GOPATH/src/github.com/$ORG && git clone git@github.com:$REPO.git
fi

# build the chain

cd $GOPATH/src/github.com/$REPO && make get_vendor_deps && make build

cd $GOPATH/src/github.com/$REPO/build
if [ $? -ne 0 ]; then
	echo "The build path does not exist"
	exit 1
fi

# start the chain

cli_home="./testnodecli"
home="./testnoded"
chain_id="bnbchain-1000"

keys_operation_words="bnc"

# prepare_node generates a secret for alice and starts the node
function prepare_node() {
	cp -f ../networks/demo/*.exp .

	rm -rf ${cli_home}
	rm -rf ${home}
	mkdir ${cli_home}
	mkdir ${home}

	alice_secret=$(./bnbchaind init --name testnode --home ${home} --home-client ${cli_home} --chain-id ${chain_id} | grep secret | grep -o ":.*" | grep -o "\".*"  | sed "s/\"//g")

	$(cd "./${home}/config" && sed -i -e "s/skip_timeout_commit = false/skip_timeout_commit = true/g" config.toml)

	# stop and start node
	ps -ef  | grep bnbchaind | grep testnoded | awk '{print $2}' | xargs kill -9
	./bnbchaind start --home ${home}  > ./testnoded/node.log 2>&1 &

	echo ${alice_secret}
}

# stop_node stops the chain node
function stop_node() {
	ps -ef  | grep bnbchaind | grep testnoded | awk '{print $2}' | xargs kill -9
}

alice_secret=$(prepare_node)

result=$(expect ./recover.exp "${alice_secret}" "alice" true)

bob_secret="bottom quick strong ranch section decide pepper broken oven demand coin run jacket curious business achieve mule bamboo remain vote kid rigid bench rubber"
result=$(expect ./add_key.exp "${bob_secret}" "bob")

alice_addr=$(./bnbcli keys list --home ${cli_home} | grep alice | grep -o "bnc[0-9a-zA-Z]*")
bob_addr=$(./bnbcli keys list --home ${cli_home} | grep bob | grep -o "bnc[0-9a-zA-Z]*")

# export a function to kill the node, as well as alice and bob's keys and secrets

export -f stop_node

