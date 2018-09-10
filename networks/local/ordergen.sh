#!/usr/bin/env sh

# This script can be used by node3 after "make localnet-start"
# within docker, we only have sh rather than sh, so some implementation is different between ordergen.sh in other modules

chainId='chain-DzRg94'

clipath='/bnbchaind/bnbcli'
clihome='/bnbchaind/node3/gaiacli'
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
${cli} send --from=zc --to=cosmosaccaddr1suav6l5pzc0yta5760jwj6j072s4dx5ty4nt4h --amount=1000000000000000BNB --chain-id ${chainId}
sleep 10

random() {
    min=$1;
    max=$(($2-$1+1));
    num=$(date +%s%N);
    retnum=`expr $num % $max + $min`
    echo $retnum;
}

while :
do
    side=`random 1 2`
    price=$(random 5 7)
    qty=$(random 10 20)
    pause=$(random 5 7)
    symbolNum=$(random 1 10)

    symbol="NNB_BNB"
    if [ $symbolNum -lt 4 ]
    then
        symbol="ZCB_BNB"
    fi

    echo $side
    from="zc"
    if [ $side == 1 ]
    then
        from="zz"
    fi

    printf "\ncli dex order --symbol=${symbol} --side=${side} --price=${price}00000000 --qty=${qty}00000000 --tif="GTC" --from=${from} --chain-id=${chainId}\n"

    ${cli} dex order --symbol=${symbol} --side=${side} --price=${price}00000000 --qty=${qty}00000000 --tif="GTC" --from=${from} --chain-id=${chainId}
done