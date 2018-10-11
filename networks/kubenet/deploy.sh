#!/bin/bash
### Before execute, make sure that the kubernetes cluster labeled with tendermint-identity=node${i}

basedir=$(cd `dirname $0`; pwd)
workspace=$basedir/../../

home=("$workspace/build/kubenode0" "$workspace/build/kubenode1" "$workspace/build/kubenode2"
       "$workspace/build/kubenode3" "$workspace/build/kubenode4" "$workspace/build/kubenode5"
       "$workspace/build/kubenode6" "$workspace/build/kubenode7" "$workspace/build/kubenode8"
    )
src_ips=("172.18.10.204" "172.18.10.205" "172.18.10.206"
          "172.18.10.207" "172.18.10.208" "172.18.10.209"
          "172.18.10.210" "172.18.10.211" "172.18.10.212"
    )

chain_operation_words="Committed"
bridge_addr=""

command=$1
des_ips=($2)
cluster_num=$3
bridge_ips=($4)
kafka_ip=$5
deploy_mode=qa

namespace=k8s-ecoysystem-apps

function build-image(){
    if [ ! -f "${workspace}/build/bnbchaind" ]; then
        make build-linux
        make build-docker-node
    fi
    cp ${workspace}/build/bnbchaind $basedir/node
    cp ${workspace}/build/bnbcli $basedir/node
    docker build --tag 172.22.41.103:5000/kube-bnbchain $basedir/node
    docker push 172.22.41.103:5000/kube-bnbchain
    rm $basedir/node/bnbchaind
}

function prepare(){
    cd ${workspace}
    if ! [ -f ${home[0]}/gaiad/config/genesis.json ];
    then
        docker run --rm -v $(pwd)/build:/bnbchaind:Z binance/bnbdnode testnet --v 9 --o . --starting-ip-address 172.18.10.204 --node-dir-prefix=kubenode
    fi
    for ihome in ${home[@]}; do
        $(cd "${ihome}/gaiad/config" && sed -i -e "s/skip_timeout_commit = false/skip_timeout_commit = true/g" config.toml)
        $(cd "${ihome}/gaiad/config" && sed -i -e "s/flush_throttle_timeout = 100/flush_throttle_timeout = 0/g" config.toml)
        $(cd "${ihome}/gaiad/config" && sed -i -e "s/peer_gossip_sleep_duration = 100/peer_gossip_sleep_duration = 0/g" config.toml)
        $(cd "${ihome}/gaiad/config" && sed -i -e "s/timeout_commit = 5000/timeout_commit = 0/g" config.toml)
        $(cd "${ihome}/gaiad/config" && sed -i -e "s/pex = true/pex = false/g" config.toml)
        $(cd "${ihome}/gaiad/config" && sed -i -e "s/skip_timeout_commit = false/skip_timeout_commit = true/g" config.toml)
    done
    for j in {0..8}
    do
        for i in {0..9}
        do
            sed -i -e "s/${src_ips[$i]}:[0-9]\{5\}/${des_ips[$i]}:26656/g" "${home[$j]}/gaiad/config/config.toml"
        done
    done
}

function build-configmap(){
    # We can change `p2p.persistent_peers` parameter of bnbchaind while deploying in different environment.
    for i in {0..2}; do
        j=$(echo "$cluster_num*3" |bc)
        j=$((j + i))
        ${kubectl} create configmap  validator-${i}-config --from-file ${home[$j]}/gaiad/config/app.toml  --from-file ${home[$j]}/gaiad/config/config.toml --from-file ${home[$j]}/gaiad/config/genesis.json -n ${namespace}
    done
}

function build-secret(){
   for i in {0..2}; do
        j=$(echo "$cluster_num*3" |bc)
        j=$((j + i))
        ${kubectl} create secret generic validator-${i}-secret --from-file ${home[$j]}/gaiad/config/node_key.json --from-file ${home[$j]}/gaiad/config/priv_validator.json -n ${namespace}
   done
}

function build-bridge-config(){
    ## prepare bridge node
    bridge_home=${workspace}/build/bridge

    rm -rf ${bridge_home}
    cp -r ${workspace}/build/kubenode0 ${bridge_home}
    rm -rf ${bridge_home}/gaiad/config/gentx ${bridge_home}/gaiad/config/node_key.json ${bridge_home}/gaiad/config/priv_validator.json
    
    sed -i -e "s/pex = false/pex = true/g" ${bridge_home}/gaiad/config/config.toml
    start=$(echo "$cluster_num*3" |bc)
    end=$((start + 3))
    while [ ${start} -lt ${end} ]; do
        nid=$(cat ${workspace}/build/gentxs/kubenode${start}.json|jq .node_id | sed 's/\"//g')
        if [ "$bridge_seeds"x == ""x ];then
            bridge_seeds="${nid}@${des_ips[$start]}:26656"
        else
            bridge_seeds="${bridge_seeds},${nid}@${des_ips[$start]}:26656"
        fi
        start=$((start + 1))
    done
    sed -i -e "s/seeds = \"\"/seeds = \"${bridge_seeds}\"/g" ${bridge_home}/gaiad/config/config.toml
    sed -i -e 's/persistent_peers = ".*"/persistend_peers = ""/g' ${bridge_home}/gaiad/config/config.toml
    
    private_ids=$(echo ${bridge_seeds} | sed 's/@[0-9]*.[0-9]*.[0-9]*.[0-9]*:[0-9]*//g')
    sed -i -e "s/private_peer_ids = \"\"/private_peer_ids = \"${private_ids}\"/g" ${bridge_home}/gaiad/config/config.toml
    
    # create configmap
    ${kubectl} create configmap  bridge-config --from-file ${bridge_home}/gaiad/config/app.toml --from-file ${bridge_home}/gaiad/config/config.toml --from-file ${bridge_home}/gaiad/config/genesis.json -n ${namespace}
}

function build-seed-config(){
    # prepare config directory
    seed_home=${workspace}/build/seed
    rm -rf ${seed_home}
    cp -r ${workspace}/build/kubenode0 ${seed_home}
    rm -rf ${seed_home}/gaiad/config/gentx ${seed_home}/gaiad/config/node_key.json ${seed_home}/gaiad/config/priv_validator.json

    # turn on pex
    sed -i -e "s/pex = false/pex = true/g" ${seed_home}/gaiad/config/config.toml
    sed -i -e "s/seeds = \"\"/seeds = \"${bridge_addr}\"/g" ${seed_home}/gaiad/config/config.toml
    sed -i -e 's/persistent_peers = ".*"/persistend_peers = ""/g' ${seed_home}/gaiad/config/config.toml

    ${kubectl} create configmap  seed-config --from-file ${seed_home}/gaiad/config/app.toml --from-file ${seed_home}/gaiad/config/config.toml --from-file ${seed_home}/gaiad/config/genesis.json -n ${namespace}
}

function build-witness-order-config(){
    # prepare config directory
    witness_order_home=${workspace}/build/witness_order
    rm -rf ${witness_order_home}
    cp -r ${workspace}/build/kubenode0 ${witness_order_home}
    rm -rf ${witness_order_home}/gaiad/config/gentx ${witness_order_home}/gaiad/config/node_key.json ${witness_order_home}/gaiad/config/priv_validator.json

    sed -i -e "s/pex = false/pex = true/g" ${witness_order_home}/gaiad/config/config.toml
    sed -i -e "s/seeds = \"\"/seeds = \"${bridge_addr}\"/g" ${witness_order_home}/gaiad/config/config.toml
    sed -i -e 's/persistent_peers = ".*"/persistend_peers = ""/g' ${witness_order_home}/gaiad/config/config.toml
    sed -i -e "s/publishAccountBalance = false/publishAccountBalance = true/g" ${witness_order_home}/gaiad/config/app.toml
    sed -i -e "s/orderUpdatesKafka = \"127.0.0.1:9092\"/orderUpdatesKafka = \"${kafka_ip}:9092\"/g" ${witness_order_home}/gaiad/config/app.toml
    sed -i -e "s/accountBalanceKafka = \"127.0.0.1:9092\"/accountBalanceKafka = \"${kafka_ip}:9092\"/g" ${witness_order_home}/gaiad/config/app.toml
    sed -i -e "s/orderBookKafka = \"127.0.0.1:9092\"/orderBookKafka = \"${kafka_ip}:9092\"/g" ${witness_order_home}/gaiad/config/app.toml
    sed -i -e "s/publishOrderUpdates = false/publishOrderUpdates = true/g" ${witness_order_home}/gaiad/config/app.toml
    sed -i -e "s/publishOrderBook = false/publishOrderBook = true/g" ${witness_order_home}/gaiad/config/app.toml
    sed -i -e "s/orderUpdatesTopic = \"test\"/orderUpdatesTopic = \"orders\"/g" ${witness_order_home}/gaiad/config/app.toml
    sed -i -e "s/log_level = \"main:info,state:info,\*:error\"/log_level = \"debug\"/g" ${witness_order_home}/gaiad/config/config.toml
    sed -i -e "s/prometheus = false/prometheus = true/g" ${witness_order_home}/gaiad/config/config.toml
    if [ ${cluster_num} -eq 1 ];then
        sed -i -e "s/orderUpdatesTopic = \"orders\"/orderUpdatesTopic = \"orders_backup\"/g" ${witness_order_home}/gaiad/config/config.toml
        sed -i -e "s/accountBalanceTopic = \"accounts\"/accountBalanceTopic = \"accounts\"/g" ${witness_order_home}/gaiad/config/config.toml
        sed -i -e "s/orderBookTopic = \"orders\"/orderBookTopic = \"orders\"/g" ${witness_order_home}/gaiad/config/config.toml
    fi
    ${kubectl} create configmap  witness-order-config --from-file ${witness_order_home}/gaiad/config/config.toml --from-file ${witness_order_home}/gaiad/config/genesis.json --from-file ${witness_order_home}/gaiad/config/app.toml -n ${namespace}
}

function build-witness-explorer-config(){
    # prepare config directory
    witness_explorer_home=${workspace}/build/witness_explorer
    rm -rf ${witness_explorer_home}
    cp -r ${workspace}/build/kubenode0 ${witness_explorer_home}
    rm -rf ${witness_explorer_home}/gaiad/config/gentx ${witness_explorer_home}/gaiad/config/node_key.json ${witness_explorer_home}/gaiad/config/priv_validator.json

    # turn on pex
    sed -i -e "s/pex = false/pex = true/g" ${witness_explorer_home}/gaiad/config/config.toml
    sed -i -e "s/seeds = \"\"/seeds = \"${bridge_addr}\"/g" ${witness_explorer_home}/gaiad/config/config.toml
    sed -i -e 's/persistent_peers = ".*"/persistend_peers = ""/g' ${witness_explorer_home}/gaiad/config/config.toml
    sed -i -e "s/index_tags = \"\"/index_tags = \"tx.height\"/g" ${witness_explorer_home}/gaiad/config/config.toml
    sed -i -e "s/accountBalanceKafka = \"127.0.0.1:9092\"/accountBalanceKafka = \"${kafka_ip}:9092\"/g" ${witness_explorer_home}/gaiad/config/app.toml
    sed -i -e "s/publishAccountBalance = false/publishAccountBalance = true/g" ${witness_explorer_home}/gaiad/config/app.toml
    sed -i -e "s/log_level = \"main:info,state:info,\*:error\"/log_level = \"debug\"/g" ${witness_explorer_home}/gaiad/config/config.toml
    sed -i -e "s/prometheus = false/prometheus = true/g" ${witness_explorer_home}/gaiad/config/config.toml
    if [ ${cluster_num} -eq 1 ];then
        sed -i -e "s/orderUpdatesTopic = \"orders\"/orderUpdatesTopic = \"orders_backup\"/g" ${witness_explorer_home}/gaiad/config/config.toml
        sed -i -e "s/accountBalanceTopic = \"accounts\"/accountBalanceTopic = \"accounts\"/g" ${witness_explorer_home}/gaiad/config/config.toml
        sed -i -e "s/orderBookTopic = \"orders\"/orderBookTopic = \"orders\"/g" ${witness_explorer_home}/gaiad/config/config.toml
    fi
    ${kubectl} create configmap  witness-explorer-config --from-file ${witness_explorer_home}/gaiad/config/app.toml --from-file ${witness_explorer_home}/gaiad/config/config.toml --from-file ${witness_explorer_home}/gaiad/config/genesis.json -n ${namespace}
}

function build-config(){
   for i in {0..2}; do
        j=$(echo "$cluster_num*3" |bc)
        j=$((j + i))
        cp -r ${basedir}/node ${home[$j]}/gaiad
   done
   build-configmap
   build-secret
}

function build-deployment(){
    for i in {0..2}; do
        j=$(echo "$cluster_num*3" |bc)
        j=$((j + i))
        sed -i "s/{{INSTANCE}}/$i/g"  ${home[$j]}/gaiad/node/deployment.yaml
        sed -i "s/{{INSTANCE}}/$i/g"  ${home[$j]}/gaiad/node/validator-svc.yaml
        sed -i "s/{{DEPLOY_MODE}}/$deploy_mode/g"  ${home[$j]}/gaiad/node/validator-svc.yaml
        ${kubectl} create -f  ${home[$j]}/gaiad/node/deployment.yaml -n ${namespace}
        ${kubectl} create -f  ${home[$j]}/gaiad/node/validator-svc.yaml -n ${namespace}
    done
}

function clean(){
    for i in {0..2}; do
        ${kubectl} delete deploy validator-${i} --ignore-not-found=true -n ${namespace}
    done
    ${kubectl} delete deploy seed bridge witness-explorer witness-order -n ${namespace} --ignore-not-found=true
}

function clean-config(){
    ## Notice: notice is not able to clean data that in remote vm, should delete manually.
    ${kubectl} delete cm bridge-config seed-config witness-explorer-config witness-order-config -n ${namespace} --ignore-not-found=true
    for i in {0..2}; do
        ${kubectl} delete cm validator-${i}-config --ignore-not-found=true -n ${namespace}
        ${kubectl} delete secret validator-${i}-secret --ignore-not-found=true -n ${namespace}
        ${kubectl} delete svc validator-${i} --ignore-not-found=true -n ${namespace}
    done
    ${kubectl} delete svc seed witness-explorer -n ${namespace} --ignore-not-found=true

}
function check_operation() {
    echo "$2" | grep -q $3
    if [ $? -ne 0 ]; then
        echo "Checking $1 Failed"
        exit 1
    fi
}

function test-chain(){
    address=$(cat ${home[0]}/gaiad/config/genesis.json|jq .app_state.tokens[0].owner)
    ## find who own the finance
    owner=kubenode0
    for i in {0..2}; do
        caddress=$(cat ${home[0]}/gaiad/config/genesis.json|jq .app_state.accounts[$i].address)
        name=$(cat ${home[0]}/gaiad/config/genesis.json|jq .app_state.accounts[$i].name)
        if [ ${caddress} == ${address} ] ;then
            owner=$(echo ${name}|sed 's/\"//g')
            break
        fi
    done
    secret=$(cat ${workspace}/build/${owner}/gaiacli/key_seed.json|jq .secret|sed 's/\"//g')
    chain_id=$(cat ${workspace}/build/${owner}/gaiad/config/genesis.json|jq .chain_id|sed 's/\"//g')
    cd ${workspace}/build/
    cp -f ${workspace}/networks/demo/*.exp ./

    result=$(expect ./recover.exp "${secret}" "alice" true ${owner}/gaiacli)

    wihte_secret="bottom quick strong ranch section decide pepper broken oven demand coin run jacket curious business achieve mule bamboo remain vote kid rigid bench rubber"
    result=$(expect ./add_key.exp "${wihte_secret}" "white" ${owner}/gaiacli)

    white_addr=$(./bnbcli keys list --home ./${owner}/gaiacli | grep white | grep -o "cosmosaccaddr[0-9a-zA-Z]*")
    # wait for the chain
    timeout=0
    while [ $(${kubectl}  get deploy -n ${namespace}|grep validator-0|awk '{print $5}') -ne 1 ]; do
        sleep 1
        timeout=$((timeout + 1))
        if [ ${timeout} -gt 60 ]; then
            echo "Error: Wait timeout for node0 to be ready."
            exit 1
        fi
    done
    sleep 20
    # send
    result=$(expect ./send.exp ./${owner}/gaiacli alice ${chain_id} 1000BNB ${white_addr})
    check_operation "Send Token" "${result}" "${chain_operation_words}"
}

function deploy-bridge(){
    bridge_home=${workspace}/build/bridge
    sed -i "s/{{DEPLOY_MODE}}/$deploy_mode/g"  ${basedir}/node/bridge-deployment.yaml
    ${kubectl} create -f  ${basedir}/node/bridge-deployment.yaml -n ${namespace}
    while [ $(${kubectl}  get deploy -n ${namespace}|grep bridge|awk '{print $5}') -ne 2 ]; do
        sleep 1
        timeout=$((timeout + 1))
        if [ ${timeout} -gt 120 ]; then
            echo "Error: Wait timeout for bridge to be ready."
            exit 1
        fi
    done
    sleep 5
    ## prepare seed node
    for i in {0..1}; do
        bridge_id=$(${workspace}/build/bnbcli --home ${bridge_home}/gaiad  --node "tcp://${bridge_ips[$i]}:26657" status)
        bridge_id=$(echo ${bridge_id} | grep -o "\"id\":\"[a-zA-Z0-9]*\"" | sed "s/\"//g" | sed "s/id://g")
        if [ "$bridge_addr"x == ""x ];then
            bridge_addr=${bridge_id}@${bridge_ips[${i}]}:26656
        else
            bridge_addr=${bridge_addr},${bridge_id}@${bridge_ips[$i]}:26656
        fi
    done
}

function deploy-seed(){
    if [ ${cluster_num} -eq 0 ];then
        ${kubectl} create -f  ${basedir}/node/seed-svc.yaml -n ${namespace}
    fi
    sed -i "s/{{DEPLOY_MODE}}/$deploy_mode/g"  ${basedir}/node/seed-deployment.yaml
    ${kubectl} create -f  ${basedir}/node/seed-deployment.yaml -n ${namespace}
}

function deploy-witness-explorer(){
    sed -i "s/{{DEPLOY_MODE}}/$deploy_mode/g"  ${basedir}/node/witness-explorer-deployment.yaml
    ${kubectl} create -f  ${basedir}/node/witness-explorer-deployment.yaml -n ${namespace}
    ${kubectl} create -f  ${basedir}/node/witness-explorer-svc.yaml -n ${namespace}
}

function deploy-witness-order(){
    sed -i "s/{{DEPLOY_MODE}}/$deploy_mode/g"  ${basedir}/node/witness-order-deployment.yaml
    ${kubectl} create -f  ${basedir}/node/witness-order-deployment.yaml -n ${namespace}
}

set -e

if [ "$command"x == "prepare"x ];then
    echo "--> Start build-image..."
    build-image
    echo "--> Start Prepare..."
    prepare
elif [ "$command"x == "install"x ];then
    export kubectl="kubectl --kubeconfig=/home/cluster${cluster_num}-config"
    echo "--> Start build-image..."
    build-image
    echo "--> Start build-config..."
    build-config
    echo "--> Start build-deployment..."
    build-deployment
    echo "--> Start build-bridge-config"
    build-bridge-config
    echo "--> Start deploy bridge"
    deploy-bridge
    echo "--> Start build seed config"
    build-seed-config
    echo "--> Start deploy seed"
    deploy-seed
    if [ ${cluster_num} -lt 2 ];then
        echo "--> Start build explorer witness config"
        build-witness-explorer-config
        echo "--> Start deploy explorer witness"
        deploy-witness-explorer
        echo "--> Start build order witness config"
        build-witness-order-config
        echo "--> Start deploy order witness"
        deploy-witness-order
    fi
elif [ "$command"x == "clean"x ];then
    cluster_num=$2
    export kubectl="kubectl --kubeconfig=/home/cluster${cluster_num}-config"
    echo "--> Start clean..."
    clean
    echo "--> Start clean config..."
    clean-config
fi
echo "--> Finish."