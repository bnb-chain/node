#!/bin/bash

# ./issue.sh --from alice -s TRX -n 10000000 --name tron

chain_id=$CHAIN_ID

while true ; do
    case "$1" in
        -s|--symbol )
            symbol=$2
            shift 2
        ;;
        --name )
            token_name=$2
            shift 2
        ;;
		-n|--total-supply )
			total_supply=$2
			shift 2
		;;
		--from )
			from=$2
			shift 2
		;;
        *)
            break
        ;;
    esac
done;

expect ./issue.exp $symbol $token_name $total_supply $from $chain_id > /dev/null

echo "Token $symbol issued success. Total supply is $total_supply."
