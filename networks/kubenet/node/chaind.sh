#!/bin/bash
#

function startChaind() {
    /server/bnbchaind/validator/bnbchaind start --home /server/bnbchaind/validator/node/gaiad --p2p.laddr  "tcp://0.0.0.0:26656" --address "tcp://0.0.0.0:26658" --rpc.laddr "tcp://0.0.0.0:26657" >> /server/bnbchaind/validator/node.log 2>&1
}

function stopChaind() {
    pid=`ps -ef | grep /server/bnbchaind/validator/bnbchaind | grep -v grep | awk '{print $2}'`
    if [ -n "$pid" ]; then
        for((i=1;i<=4;i++));
        do
            kill $pid
            sleep 5
            pid=`ps -ef | grep /server/bnbchaind/validator/bnbchaind | grep -v grep | awk '{print $2}'`
            if [ -z "$pid" ]; then
                #echo "bnbchaind stoped"
                break
            elif [ $i -eq 4 ]; then
                kill -9 $kid
            fi
        done
    fi
}

CMD=$1

case $CMD in
-start)
    echo "start"
    startChaind
    ;;
-stop)
    echo "stop"
    stopChaind
    ;;
-restart)
    stopChaind
    sleep 3
    startChaind
    ;;
*)
    echo "Usage: chaind.sh -start | -stop | -restart .Or use systemctl start | stop | restart bnbchaind.service "
    ;;
esac