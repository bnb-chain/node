#!/bin/bash

DEFAULT_NETWORK=("mainnet" "testnet")
NETWORK=${NETWORK:-mainnet}
HOME=${HOME:-/data}
DEFAULT_CONFIG=${DEFAULT_CONFIG:-/configs}

if echo ${DEFAULT_NETWORK[@]} | grep -q -w "${NETWORK}"
then
    mkdir -p ${HOME}/config
    cp ${DEFAULT_CONFIG}/${NETWORK}/* ${HOME}/config/
fi

exec "bnbchaind" "start" "--home" ${HOME} "$@"