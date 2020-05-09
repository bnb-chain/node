#!/bin/bash

########################### SETUP #########################
home=$HOME
src="${home}/go/src/github.com/binance-chain/node"
clipath="${home}/go/src/github.com/binance-chain/node/build/bnbcli"
clihome="${home}/.bnbcli"
chainId='test-chain-n4b735' # should be same with publisher/setup.sh or testnet/deploy.sh


cli="${clipath} --home ${clihome}"
scripthome="${src}/networks/publisher"
############################ END ##########################

function random()
{
    unameOut="$(uname -s)"
    case "${unameOut}" in
        Linux*)     machine=Linux;;
        Darwin*)    machine=Mac;;
        CYGWIN*)    machine=Cygwin;;
        MINGW*)     machine=MinGw;;
        *)          machine="UNKNOWN:${unameOut}"
    esac

    min=$1;
    max=$(($2-$1+1));
    if [ "${machine}" = "Mac" ]
        then
            num=$(date +%s);  # macos: %N doesn't work
        else
            num=$(date +%s%N);
    fi
    ((retnum=num%max+min));

    echo $retnum;
}

while :
do
    side=$(random 1 2)
    price=$(random 1 4)
    qty=$(random 1 2)
    pause=$(random 5 7)
    symbolNum=$(random 1 10)

    symbol="YCB-E9D_BNB"
    if [ $symbolNum -lt 4 ]
    then
        symbol="X1M-71AM_BNB"
    elif [ $symbolNum -lt 6 ]
    then
        symbol="Y2N-02AM_BNB"
    else [ $symbolNum -lt 8 ]
        symbol="ZCK-064_BNB"
    fi
    from="zc"
    if [ $side == 1 ]
    then
        from="zz"
    fi

    printf "\n${cli} dex order --symbol=${symbol} --side=${side} --price=${price}00000000 --qty=${qty}00000000 --tif="GTE" --from=${from} --chain-id=${chainId}\n"

    echo 1234qwerasdf|${cli} dex order --symbol=${symbol} --side=${side} --price=${price}00000000 --qty=${qty}00000000 --tif="GTE" --from=${from} --chain-id=${chainId}

    # -d is used for get response of expect script. TODO: better log redirection
    #result=$(expect -d ${scripthome}/ordergen.exp "${clipath}" "${clihome}" "${symbol}" "${side}" "${price}00000000" "${qty}00000000" "${from}" "${chainId}")

    #printf "\nsleep ${pause} seconds...\n"
    sleep ${pause}
done