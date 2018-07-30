#!/bin/bash

# ./show.sh -l ADA_BNB --from alice

chain_id=$CHAIN_ID

while true ; do
    case "$1" in
        -l|--list-pair )
            pair=$2
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

expect ./show.exp $pair $from $chain_id

echo "Order sent success."