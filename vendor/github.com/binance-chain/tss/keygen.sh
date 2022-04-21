#!/usr/bin/env bash

# usage ./keygen_36.sh t n home
t=$1
n=$2
tt=$3

go build

for (( i=0; i<$n; i++ ))
do
    home=./tests/${tt}/${i}
    ./tss init --home "${home}" --vault_name "default" --moniker "test${i}" --password "123456789" --log_level debug
    ./tss keygen --home "${home}" --vault_name "default" --parties $n --threshold $t --password "123456789" --channel_password "123456789" --channel_id "18368134266" > ${home}/keygen.log 2>&1 &
done