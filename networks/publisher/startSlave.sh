#!/usr/bin/env bash

src='/Users/zhaocong/go/src/github.com/BiJie/BinanceChain'
executable='/Users/zhaocong/go/src/github.com/BiJie/BinanceChain/build/bnbchaind'
cli='/Users/zhaocong/go/src/github.com/BiJie/BinanceChain/build/bnbcli'
home='/Users/zhaocong'

witnessId=$1

${executable} start --home ${home}/.bnbchaind_nonVal > ${home}/.bnbchaind_nonVal${witnessId}/log.txt 2>&1 &
witness_pid=$!
echo ${witness_pid}