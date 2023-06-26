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

  secret=$(./bnbchaind init --moniker testnode --home ${home} --home-client ${cli_home} --chain-id ${chain_id} | grep secret | grep -o ":.*" | grep -o "\".*" | sed "s/\"//g")
  echo ${secret} >${home}/secret

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
  $(cd "./${home}/config" && sed -i -e "s/LimitConsAddrUpdateIntervalHeight = 9223372036854775807/LimitConsAddrUpdateIntervalHeight = 11/g" app.toml)
  $(cd "./${home}/config" && sed -i -e "s/breatheBlockInterval = 0/breatheBlockInterval = 5/g" app.toml)
  $(cd "./${home}/config" && sed -i -e "s/EnableReconciliationHeight = 9223372036854775807/EnableReconciliationHeight = 3/g" app.toml)

  # stop and start node
  ps -ef | grep bnbchaind | grep testnoded | awk '{print $2}' | xargs kill -9
  ./bnbchaind start --home ${home} >./testnoded/node.log 2>&1 &

  echo ${secret}
}

function exit_test() {
  # stop node
  ps -ef | grep bnbchaind | grep testnoded | awk '{print $2}' | xargs kill -9
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
sleep 10

alice_addr=$(./bnbcli keys list --home ${cli_home} | grep alice | grep -o "bnb1[0-9a-zA-Z]*")
bob_addr=$(./bnbcli keys list --home ${cli_home} | grep bob | grep -o "bnb1[0-9a-zA-Z]*")
carl_addr=$(./bnbcli keys list --home ${cli_home} | grep carl | grep -o "bnb1[0-9a-zA-Z]*")

sleep 5
# send
result=$(expect ./send.exp ${cli_home} alice ${chain_id} "100000000000000:BNB" ${bob_addr})
check_operation "Send Token" "${result}" "${chain_operation_words}"

sleep 1
# send
result=$(expect ./send.exp ${cli_home} alice ${chain_id} "100000000000000:BNB" ${carl_addr})
check_operation "Send Token" "${result}" "${chain_operation_words}"

# staking related
function staking() {
  # get parameters
  result=$(./bnbcli staking parameters --home ${cli_home} --trust-node)
  check_operation "Query Staking Parameters" "${result}" "proposer"

  # get params side-params
  result=$(./bnbcli params side-params --home ${cli_home} --trust-node --side-chain-id bsc)
  check_operation "Query Staking Parameters" "${result}" "StakeParamSet"

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
  sleep 5
  result=$(./bnbcli staking validators --home ${cli_home} --trust-node)
  check_operation "Get Validators" "${result}" "Operator"
  result=$(./bnbcli staking validator ${bob_val_addr} --home ${cli_home} --trust-node)
  check_operation "Get Validator" "${result}" "bob"
  check_operation "Get Validator" "${result}" "${bob_pubkey}"

  # edit validator
  result=$(expect ./edit-validator.exp ${cli_home} bob ${chain_id} ${bob_pubkey_new})
  check_operation "edit validator" "${result}" "${chain_operation_words}"
  sleep 5
  result=$(./bnbcli staking validator ${bob_val_addr} --home ${cli_home} --trust-node)
  check_operation "Get Validator" "${result}" "bob-new"
  check_operation "Get Validator" "${result}" "${bob_pubkey_new}"
  bob_val_addr=$(echo "${result}" | grep Operator | grep -o "bva[0-9a-zA-Z]*")

  # run test with go-sdk
  cd ../e2e && go run .
}

# token related
function token() {
  sleep 1
  # issue token
  result=$(expect ./issue.exp BTC Bitcoin 1000000000000000 true bob ${chain_id} ${cli_home})
  btc_symbol=$(echo "${result}" | tail -n 1 | grep -o "BTC-[0-9A-Z]*")
  check_operation "Issue Token" "${result}" "${chain_operation_words}"

  sleep 1
  # bind
  result=$(expect ./bind.exp ${btc_symbol} 0x6aade9709155a8386c63c1d2e5939525b960b4e7 10000000000000 4083424190 bob ${chain_id} ${cli_home})
  check_operation "Bind Token" "${result}" "${chain_operation_words}"

  sleep 1
  # issue token
  result=$(expect ./issue.exp ETH Ethereum 1000000000000000 true bob ${chain_id} ${cli_home})
  eth_symbol=$(echo "${result}" | tail -n 1 | grep -o "ETH-[0-9A-Z]*")
  check_operation "Issue Token" "${result}" "${chain_operation_words}"

  sleep 1
  # freeze token
  result=$(expect ./freeze.exp ${btc_symbol} 100000000 bob ${chain_id} ${cli_home})
  check_operation "Freeze Token" "${result}" "${chain_operation_words}"

  sleep 1
  # send
  result=$(expect ./send.exp ${cli_home} alice ${chain_id} "100000000000000:BNB" ${bob_addr})
  check_operation "Send Token" "${result}" "${chain_operation_words}"

  sleep 1
  # multi send
  echo ${bob_addr}
  result=$(expect ./multi_send.exp ${cli_home} alice ${chain_id} "[{\"to\":\"${bob_addr}\",\"amount\":\"100000000000000:BNB\"},{\"to\":\"${alice_addr}\",\"amount\":\"100000000000000:BNB\"}]")
  check_operation "Multi Send Token" "${result}" "${chain_operation_words}"

  sleep 1
  # mint token
  result=$(expect ./mint.exp ${btc_symbol} 1000000000000000 bob ${chain_id} ${cli_home})
  check_operation "Mint Token" "${result}" "${chain_operation_words}"

  sleep 1
  # burn token
  result=$(expect ./burn.exp ${btc_symbol} 50000000 bob ${chain_id} ${cli_home})
  check_operation "Burn Token" "${result}" "${chain_operation_words}"

  sleep 1
  # unfreeze token
  result=$(expect ./unfreeze.exp ${btc_symbol} 100000000 bob ${chain_id} ${cli_home})
  check_operation "Freeze Token" "${result}" "${chain_operation_words}"
}

# gov related
function gov() {
  sleep 1
  # propose list
  ((expire_time = $(date '+%s') + 1000))
  lower_case_btc_symbol=$(echo ${btc_symbol} | tr 'A-Z' 'a-z')
  result=$(expect ./propose_list.exp ${chain_id} alice 200000000000:BNB ${lower_case_btc_symbol} bnb 100000000 "list BTC/BNB" "list BTC/BNB" ${cli_home} ${expire_time} 5)
  check_operation "Propose list" "${result}" "${chain_operation_words}"

  sleep 2
  # vote for propose
  result=$(expect ./vote.exp alice ${chain_id} 1 Yes ${cli_home})
  check_operation "Vote" "${result}" "${chain_operation_words}"

}

# account related
function account() {
  sleep 1
  ## query account balance
  result=$(./bnbcli account $bob_addr --trust-node)
  balance1=$(echo "${result}" | jq -r '.value.base.coins[0].amount')

  sleep 1
  # set an account flag which isn't bounded to transfer memo checker script
  result=$(expect ./set_account_flags.exp 0x02 bob ${chain_id} ${cli_home})
  check_operation "Set account flags" "${result}" "${chain_operation_words}"

  sleep 1
  ## query account balance
  result=$(./bnbcli account $bob_addr --trust-node)
  balance2=$(echo "${result}" | jq -r '.value.base.coins[0].amount')
  check_operation "Check fee deduction for set account flags transaction" "$(expr $balance2 - $balance1)" "100000000"

  sleep 1
  result=$(expect ./send.exp ${cli_home} alice ${chain_id} "100000000000000:BNB" ${bob_addr})
  check_operation "Send Token" "${result}" "${chain_operation_words}"

  sleep 1
  result=$(expect ./send.exp ${cli_home} alice ${chain_id} "100000000000000:BNB" ${bob_addr} "123456abcd")
  check_operation "Send Token" "${result}" "${chain_operation_words}"

  sleep 1
  # set an account flag which is bounded to transfer memo checker script
  result=$(expect ./set_account_flags.exp 0x01 bob ${chain_id} ${cli_home})
  check_operation "Set account flags" "${result}" "${chain_operation_words}"

  sleep 1
  result=$(expect ./send.exp ${cli_home} alice ${chain_id} "100000000000000:BNB" ${bob_addr})
  check_operation "Send Token" "${result}" "ERROR"

  sleep 1
  result=$(expect ./send.exp ${cli_home} alice ${chain_id} "100000000000000:BNB" ${bob_addr} "123456abcd")
  check_operation "Send Token" "${result}" "ERROR:"

  sleep 1
  result=$(expect ./send.exp ${cli_home} alice ${chain_id} "100000000000000:BNB" ${bob_addr} "1234567890")
  check_operation "Send Token" "${result}" "${chain_operation_words}"

}

# swap related
function swap() {
  sleep 1
  # Create an atomic swap
  result=$(expect ./HTLT-cross-chain.exp 2000 "100000000:BNB" "100000000:BNB" $bob_addr 0xf2fbB6C41271064613D6f44C7EE9A6c471Ec9B25 alice ${chain_id} ${cli_home})
  check_operation "Create an atomic swap" "${result}" "${chain_operation_words}"
  randomNumber=$(sed 's/Random number: //g' <<<$(echo "${result}" | grep -o "Random number: [0-9a-z]*"))
  timestamp=$(sed 's/Timestamp: //g' <<<$(echo "${result}" | grep -o "Timestamp: [0-9]*"))
  randomNumberHash=$(sed 's/Random number hash: //g' <<<$(echo "${result}" | grep -o "Random number hash: [0-9a-z]*"))
  swapID=$(sed 's/swapID: //g' <<<$(echo "${result}" | tail -n 1 | grep -o "swapID: [0-9a-z]*"))
  sleep 1

  atomicSwap=$(./bnbcli token query-swap --swap-id ${swapID} --trust-node)
  swapFrom=$(echo "${atomicSwap}" | jq -r '.from')
  check_operation "Check swap creator address" $swapFrom $alice_addr
  swapTo=$(echo "${atomicSwap}" | jq -r '.to')
  check_operation "swap recipient address" $swapTo $bob_addr

  result=$(./bnbcli account bnb1wxeplyw7x8aahy93w96yhwm7xcq3ke4f8ge93u --trust-node)
  swapDeadAddrBalance=$(echo "${result}" | jq -r '.value.base.coins[0].amount')
  check_operation "the balance of swap dead address" $swapDeadAddrBalance "100000000"

  result=$(./bnbcli account $bob_addr --trust-node)
  balanceBobBeforeClaim=$(echo "${result}" | jq -r '.value.base.coins[0].amount')

  # Claim an atomic swap
  result=$(expect ./claim.exp ${swapID} $randomNumber alice ${chain_id} ${cli_home})
  check_operation "claim an atomic swap" "${result}" "${chain_operation_words}"

  sleep 1

  result=$(./bnbcli account $bob_addr --trust-node)
  balanceBobAfterClaim=$(echo "${result}" | jq -r '.value.base.coins[0].amount')
  check_operation "Bob balance after claim swap" "$(expr $balanceBobAfterClaim - $balanceBobBeforeClaim)" "100000000"

  # Create an atomic swap
  result=$(expect ./HTLT-cross-chain.exp 2000 "100000000:BNB" "100000000:BNB" $alice_addr 0xf2fbB6C41271064613D6f44C7EE9A6c471Ec9B25 bob ${chain_id} ${cli_home})
  check_operation "Create an atomic swap" "${result}" "${chain_operation_words}"
  swapID=$(sed 's/swapID: //g' <<<$(echo "${result}" | tail -n 1 | grep -o "swapID: [0-9a-z]*"))

  sleep 1

  # Refund an atomic swap
  result=$(expect ./refund.exp ${swapID} alice ${chain_id} ${cli_home})
  check_operation "refund an atomic swap which is still not expired" "${result}" "ERROR"

  sleep 1

  result=$(./bnbcli account bnb1wxeplyw7x8aahy93w96yhwm7xcq3ke4f8ge93u --trust-node)
  swapDeadAddrBalance=$(echo "${result}" | jq -r '.value.base.coins[0].amount')
  check_operation "the balance of swap dead address" $swapDeadAddrBalance "100000000"

  sleep 1
  # Create a single chain atomic swap
  result=$(expect ./HTLT-single-chain.exp 2000 "100000000:BNB" "10000:${eth_symbol}" $bob_addr alice ${chain_id} ${cli_home})
  check_operation "Create a single chain atomic swap" "${result}" "${chain_operation_words}"
  randomNumber=$(sed 's/Random number: //g' <<<$(echo "${result}" | grep -o "Random number: [0-9a-z]*"))
  timestamp=$(sed 's/Timestamp: //g' <<<$(echo "${result}" | grep -o "Timestamp: [0-9]*"))
  randomNumberHash=$(sed 's/Random number hash: //g' <<<$(echo "${result}" | grep -o "Random number hash: [0-9a-z]*"))
  swapID=$(sed 's/swapID: //g' <<<$(echo "${result}" | tail -n 1 | grep -o "swapID: [0-9a-z]*"))

  sleep 1
  # Deposit to a single chain atomic swap
  result=$(expect ./deposit.exp ${swapID} "10000:${eth_symbol}" bob ${chain_id} ${cli_home})
  check_operation "Deposit to a single chain atomic swap" "${result}" "${chain_operation_words}"

  sleep 1
  # Claim a single chain atomic swap
  result=$(expect ./claim.exp ${swapID} ${randomNumber} alice ${chain_id} ${cli_home})
  check_operation "claim a single chain atomic swap" "${result}" "${chain_operation_words}"

  sleep 1
  # Deposit to a single chain atomic swap
  result=$(expect ./deposit.exp ${swapID} "10000:${eth_symbol}" bob ${chain_id} ${cli_home})
  check_operation "Deposit to a closed single chain atomic swap" "${result}" "ERROR"

  sleep 1
  # Create a single chain atomic swap
  result=$(expect ./HTLT-single-chain.exp 360 "100000000:BNB" "10000:${eth_symbol}" $bob_addr alice ${chain_id} ${cli_home})
  check_operation "Create a single chain atomic swap" "${result}" "${chain_operation_words}"
  randomNumber=$(sed 's/Random number: //g' <<<$(echo "${result}" | grep -o "Random number: [0-9a-z]*"))
  timestamp=$(sed 's/Timestamp: //g' <<<$(echo "${result}" | grep -o "Timestamp: [0-9]*"))
  randomNumberHash=$(sed 's/Random number hash: //g' <<<$(echo "${result}" | grep -o "Random number hash: [0-9a-z]*"))
  swapID=$(sed 's/swapID: //g' <<<$(echo "${result}" | tail -n 1 | grep -o "swapID: [0-9a-z]*"))

  sleep 1
  # Deposit to a single chain atomic swap
  result=$(expect ./deposit.exp ${swapID} "10000:${eth_symbol}" bob ${chain_id} ${cli_home})
  check_operation "Deposit to a single chain atomic swap" "${result}" "${chain_operation_words}"

  sleep 1
  # Deposit to a single chain atomic swap
  result=$(expect ./deposit.exp ${swapID} "10000:${eth_symbol}" bob ${chain_id} ${cli_home})
  check_operation "Deposit to a deposited single chain atomic swap" "${result}" "ERROR"

}

# bridge related
function bridge() {
  echo "skip, due to  crosschain needed"
  #  sleep 1
  #  # Transfer out
  #  result=$(expect ./transfer-out.exp ${cli_home} alice ${chain_id} "10000:${btc_symbol}" 0x4307fa0f0b4a9fe83e4ed88ae93a33b03892be03 4083424190)
  #  check_operation "Transfer Out" "${result}" "${chain_operation_words}"

  #  sleep 1
  #  # Unbind
  #  result=$(expect ./unbind.exp ${btc_symbol} alice ${chain_id} ${cli_home})
  #  check_operation "Unbind" "${result}" "${chain_operation_words}"
}

sleep 10

token
bridge
gov
account
swap
staking

exit_test 0
