#!/bin/bash

# ./cancel.sh --id f4705e9d-279a-4fbc-8718-a1f53448dc63 --from bob

chain_id=$CHAIN_ID

while true ; do
    case "$1" in
        --id )
            id=$2
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

expect ./cancel.exp $id $from $chain_id > /dev/null

echo "Order ${id} cancelled success."