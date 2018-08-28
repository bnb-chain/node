#!/usr/bin/env bash

# This script is still working in progress. Basically it should be used under private-2 box in our jumpservers because kafka is deployed their

# ./bnbcli token issue --from=zc --token-name="ZC Coin" --symbol=ZCB --total-supply=2000000000000000 --chain-id test-chain-JVx2RL
# ./bnbcli dex list --symbol=ZCB --quote-symbol=BNB --init-price=110000000 --from=zc --chain-id=test-chain-JVx2RL

cli='/home/bijieprd/gowork/src/github.com/BiJie/BinanceChain/build/bnbcli --node tcp://localhost:26757'
chainId='test-chain-vJ4ggW'

function random()
{
    min=$1;
    max=$(($2-$1+1));
    num=$(date +%s%N);  # macos: %N doesn't work
    ((retnum=num%max+min));
    echo $retnum;
}

while :
do
    side=$(random 1 2)
    price=$(random 5 7)
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

    printf "\nsleep ${pause} seconds...\n"
    sleep ${pause}
done