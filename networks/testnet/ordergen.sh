#!/usr/bin/env bash

# This script is still working in progress. Basically it should be used under private-2 box in our jumpservers because kafka is deployed their

clipath='/home/bijieprd/gowork/src/github.com/BiJie/BinanceChain/build/bnbcli'
clihome='/server/bnc/node2/gaiacli'
chainId='chain-rKZsqV'

cli="${clipath} --home ${clihome}"

${cli} keys add zc --recover
${cli} keys add zz
${cli} token issue --from=zc --token-name="New BNB Coin" --symbol=NNB --total-supply=2000000000000000 --chain-id ${chainId}
sleep 10
${cli} dex list -s=NNB --quote-asset-symbol=BNB --init-price=110000000 --from=zc --chain-id ${chainId}
sleep 10
${cli} token issue --from=zc --token-name="ZC Coin" --symbol=ZCB --total-supply=2000000000000000 --chain-id ${chainId}
sleep 10
${cli} dex list -s=ZCB --quote-asset-symbol=BNB --init-price=110000000 --from=zc --chain-id ${chainId}
sleep 10
${cli} send --from=zc --to=cosmosaccaddr1872gjuvfakc6nrrf8qdqsar7anp9ezjly8rrwh --amount=1000000000000000BNB --chain-id ${chainId}
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

    symbol="NNB_BNB"
    if [ $symbolNum -lt 4 ]
    then
        symbol="ZCB_BNB"
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