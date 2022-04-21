#!/usr/bin/env bash

t=$1
n=$2
tt=$(date '+%s')

./keygen.sh $1 $2 $tt
sleep 60

address=$(./tbnbcli keys list | grep "tss_test0_default" | awk '{print $3}')
echo "$address"
touch ./tests/"${tt}"/"$address"
expect ./send_normal.exp "${address}" > /dev/null 2>&1

for i in {0..20}
do
    ./kill_all.sh
    ./sign.sh "$t" "${tt}" $i
    sleep 60
done

sequence=$(curl https://testnet-dex.binance.org/api/v1/account/"${address}" | sed -n 's/.*sequence\":\(.*\)\}$/\1/p')
if [ "$sequence" == "21" ]; then
    echo "good"
else
    echo $'\n'"${tt}" >> bad.txt
fi;