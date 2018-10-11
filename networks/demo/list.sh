#!/bin/bash

# ./list.sh -s ADA --quote-symbol BNB --from alice --init-price 1

chain_id=$CHAIN_ID

while true ; do
    case "$1" in
        -s|--base-asset-symbol )
            base_asset=$2
            shift 2
        ;;
        --quote-asset-symbol )
            quote_asset=$2
            shift 2
        ;;
		--init-price )
			init_price=$2
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

expect ./list.exp $base_asset $quote_asset $init_price $from $chain_id "tcp://localhost:26657"> /dev/null

echo "Pair $(symbol)_$(quote_symbol) listed success."
