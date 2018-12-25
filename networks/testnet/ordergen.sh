#!/usr/bin/env bash

# This script is still working in progress. Basically it should be used under private-2 box in our jumpservers because kafka is deployed their

# For dev
#clipath='/home/bijieprd/gowork/src/github.com/BiJie/BinanceChain/build/bnbcli'
#clihome='/server/bnc/node2/gaiacli'
#chainId='chain-rKZsqV'

clipath='/root/zjubfd/src/github.com/BiJie/BinanceChain/build/bnbcli'
clihome='/server/bnc/node0/gaiacli'
chainId='chain-bnb'


cli="${clipath} --home ${clihome}"
#cli="${clipath} --home ${clihome} --node tcp://172.27.41.151:26657"

${cli} keys add zc --recover
${cli} keys add zz
result=$(${cli} token issue --from=zc --token-name="New BNB Coin" --symbol=NNB --total-supply=2000000000000000 --chain-id ${chainId})
nnb_symbol=$(echo "${result}" | tail -n 1 | grep -o "NNB-[0-9A-Z]*")
echo ${nnb_symbol}
sleep 10
${cli} gov submit-list-proposal --chain-id ${chainId} --from zc --deposit 200000000000:BNB --base-asset-symbol ${nnb_symbol} --quote-asset-symbol BNB --init-price 1000000000 --title "list NNB/BNB" --description "list NNB/BNB" --expire-time 1644486400
sleep 2
${cli} gov vote --from zc --chain-id ${chainId} --proposal-id 7 --option Yes
sleep 61
${cli} dex list -s=${nnb_symbol} --quote-asset-symbol=BNB --init-price=1000000000 --from=zc --chain-id ${chainId} --proposal-id 7
sleep 1
result=$(${cli} token issue --from=zc --token-name="ZC Coin" --symbol=ZCB --total-supply=2000000000000000 --chain-id ${chainId})
zcb_symbol=$(echo "${result}" | tail -n 1 | grep -o "NNB-[0-9A-Z]*")
echo ${zcb_symbol}
sleep 10
${cli} gov submit-list-proposal --chain-id ${chainId} --from zc --deposit 200000000000:BNB --base-asset-symbol ${zcb_symbol} --quote-asset-symbol BNB --init-price 1000000000 --title "list NNB/BNB" --description "list NNB/BNB" --expire-time 1644486400
sleep 2
${cli} gov vote --from zc --chain-id ${chainId} --proposal-id 8 --option Yes
sleep 61
${cli} dex list -s=${zcb_symbol} --quote-asset-symbol=BNB --init-price=1000000000 --from=zc --chain-id ${chainId} --proposal-id 8
sleep 1
${cli} send --from=zc --to=cosmosaccaddr1872gjuvfakc6nrrf8qdqsar7anp9ezjly8rrwh --amount=1000000000000000:BNB --chain-id ${chainId}
sleep 10

function random()
{
    min=$1;
    max=$(($2-$1+1));
    num=$(date +%s%N);
    ((retnum=num%max+min));
    echo $retnum;
}

while :
do
    side=$(random 1 2)
    price=$(random 1 20)
    qty=$(random 10 20)
    pause=$(random 5 7)
    symbolNum=$(random 1 10)

    symbol="NNB-94A_BNB"
    if [ $symbolNum -lt 4 ]
    then
        symbol="ZCB-E21_BNB"
    fi

    from="zc"
    if [ $side == 1 ]
    then
        from="zz"
    fi

    printf "\ncli dex order --symbol=${symbol} --side=${side} --price=${price}00000000 --qty=${qty}00000000 --tif="GTC" --from=${from} --chain-id=${chainId}\n"

    ${cli} dex order --symbol=${symbol} --side=${side} --price=${price}00000000 --qty=${qty}00000000 --tif="GTC" --from=${from} --chain-id=${chainId}

    #printf "\nsleep ${pause} seconds...\n"
    #sleep ${pause}
done