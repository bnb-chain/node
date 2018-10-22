#!/usr/bin/env bash

basedir=$(cd `dirname $0`; pwd)
workspace=$basedir/../../

command=$1

chain_operation_words="Committed"
rain_secrets=("bottom quick strong ranch section decide pepper broken oven demand coin run jacket curious business achieve mule bamboo remain vote kid rigid bench rubber"
          "bench bottom quick strong ranch section decide pepper broken oven demand coin run jacket curious business achieve mule bamboo remain vote kid rigid  rubber"
          "section bottom quick strong ranch decide pepper broken oven demand coin run jacket curious business achieve mule bamboo remain vote kid rigid bench rubber"
)
users=(rainmaker0 rainmaker1 rainmaker2)

cli_home=${workspace}/build/kubenode0
chain_id=""
owner=kubenode0
namespace=k8s-ecoysystem-apps
rootuser=rootuser

function check_operation() {
	echo "$2" | grep -q $3
	if [ $? -ne 0 ]; then
		echo "Checking $1 Failed"
		exit_test 1
	fi
}

function find_owner(){
    address=$(cat ${cli_home}/gaiad/config/genesis.json|jq .app_state.tokens[0].owner)
    ## find who own the finance
    for i in {0..8}; do
        caddress=$(cat ${cli_home}/gaiad/config/genesis.json|jq .app_state.accounts[$i].address)
        name=$(cat ${cli_home}/gaiad/config/genesis.json|jq .app_state.accounts[$i].name)
        if [ ${caddress} == ${address} ] ;then
            owner=$(echo ${name}|sed 's/\"//g')
            break
        fi
    done
    cli_home=${workspace}/build/${owner}/gaiacli
}

function rain_prepare(){
    ## find who own the finance
    find_owner
    secret=$(cat ${workspace}/build/${owner}/gaiacli/key_seed.json|jq .secret|sed 's/\"//g')
    chain_id=$(cat ${workspace}/build/${owner}/gaiad/config/genesis.json|jq .chain_id|sed 's/\"//g')
    cd ${workspace}/build/
    cp -f ${workspace}/networks/demo/*.exp ./

    expect ./recover.exp "${secret}" "${rootuser}" true ${owner}/gaiacli

    rain_addrs=("addr0" "addr1" "addr2")
    # add user
    for i in {0..2}; do
        rain_secret=${rain_secrets[$i]}
        expect ./add_key.exp "${rain_secret}"  ${users[$i]} ${owner}/gaiacli
        rain_addrs[$i]=$(./bnbcli keys list --home ./${owner}/gaiacli | grep ${users[$i]} | grep -o "cosmosaccaddr[0-9a-zA-Z]*")
    done

    # send bnb
    for rain_addr in ${rain_addrs[@]}; do
        result=$(expect ./send.exp ./${owner}/gaiacli ${rootuser} ${chain_id} 50000000BNB ${rain_addr} ${node_addr})
        check_operation "Send Token" "${result}" "${chain_operation_words}"
        sleep 1
    done

    # issue token and send token
    for ((i=0;i<${#coins[@]}; i++));do
        result=$(expect ./issue.exp ${conins_short[$i]} ${coins[$i]} 90000000000 ${rootuser} ${chain_id} ${cli_home} ${node_addr})
        check_operation "Issue Token" "${result}" "${chain_operation_words}"
        sleep 1

        result=$(expect ./list.exp ${conins_short[$i]} BNB 10 ${rootuser} ${chain_id} ${node_addr} ${cli_home})
        check_operation "List Trading Pair" "${result}" "${chain_operation_words}"
        sleep 1

        for rain_addr in ${rain_addrs[@]}; do
            result=$(expect ./send.exp ./${owner}/gaiacli ${rootuser} ${chain_id} 3000000000${conins_short[$i]} ${rain_addr} ${node_addr})
            check_operation "Send Token" "${result}" "${chain_operation_words}"
            sleep 1
        done
    done

    for ((i=0;i<${#coins[@]}; i++));do
        for ((j=$i+1;j<${#coins[@]};j++));do
            result=$(expect ./list.exp ${conins_short[$i]} ${conins_short[$j]} 1000000 ${rootuser} ${chain_id} ${node_addr} ${cli_home})
            check_operation "List Trading Pair" "${result}" "${chain_operation_words}"
            sleep 1
        done
    done

}

function rain_install(){
    find_owner
    export kubectl="kubectl --kubeconfig=/home/cluster${cluster_num}-config"

    ldbs=($(ls  ${cli_home}/keys/keys.db/*.ldb))
    current=$(cat ${cli_home}/keys/keys.db/CURRENT)

    args="--from-file ${cli_home}/keys/keys.db/CURRENT --from-file ${cli_home}/keys/keys.db/CURRENT.bak"
    args="$args  --from-file ${cli_home}/keys/keys.db/$current"
    args="$args --from-file ${cli_home}/keys/keys.db/LOCK"

    for ldb in ${ldbs[@]};do
        args="$args --from-file $ldb"
    done

    sed -i "s/{{NODE_DOMAIN}}/${node_addr}/g" ${basedir}/node/rainmaker.toml
    ${kubectl} create configmap  rainmaker --from-file  ${basedir}/node/rainmaker.toml -n ${namespace}
    ${kubectl} create secret generic rainmaker ${args}    -n ${namespace}
    sed -i "s/{{DEPLOY_MODE}}/$deploy_mode/g"  ${basedir}/node/rainmaker-statefulset.yaml
    sed -i "s/{{DOCKER_REGISTRY}}/${docker_registry}/g" ${basedir}/node/rainmaker-statefulset.yaml
    ${kubectl} create -f ${basedir}/node/rainmaker-statefulset.yaml -n ${namespace}
    ${kubectl} create -f ${basedir}/node/rainmaker-svc.yaml -n ${namespace}
    ${kubectl} create -f ${basedir}/node/rainmaker-nodeport.yaml -n ${namespace}
}

function rain_clean(){
    export kubectl="kubectl --kubeconfig=/home/cluster${cluster_num}-config"
    ${kubectl} delete configmap rainmaker --ignore-not-found=true -n ${namespace}
    ${kubectl} delete secret rainmaker --ignore-not-found=true -n ${namespace}
    ${kubectl} delete statefulset rainmaker --ignore-not-found=true -n ${namespace}
    ${kubectl} delete svc rainmaker --ignore-not-found=true -n ${namespace}
    ${kubectl} delete svc bnbrainmaker --ignore-not-found=true -n ${namespace}
}
set -e
if [ "$command"x == "prepare"x ];then
    export coins=($2)
    export conins_short=($3)
    export node_addr=$4
    rain_prepare
elif [ "$command"x == "install"x ];then
    export cluster_num=$2
    export docker_registry=$3
    export deploy_mode=$4
    export node_addr=$5
    rain_install
elif [ "$command"x == "clean"x ];then
    export cluster_num=$2
    rain_clean
fi