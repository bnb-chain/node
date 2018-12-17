#!/bin/bash
#

function start_bnbcli() {
   /server/bnbchaind/validator/bnbcli --chain-id chain-bnb --home  /server/bnbchaind/validator/node/gaiacli --laddr tcp://0.0.0.0:8080 --node tcp://localhost:26657 api-server>>  /server/bnbchaind/validator/api-server.log 2>&1
}

function stop_bnbcli() {
    pid=`ps -ef | grep /server/bnbchaind/validator/bnbcli | grep -v grep | awk '{print $2}'`
    if [ -n "$pid" ]; then
        for((i=1;i<=4;i++));
        do
            kill $pid
            sleep 5
            pid=`ps -ef | grep /server/bnbchaind/validator/bnbcli | grep -v grep | awk '{print $2}'`
            if [ -z "$pid" ]; then
                #echo "bnbcli stoped"
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
    start_bnbcli
    ;;
-stop)
    echo "stop"
    stop_bnbcli
    ;;
-restart)
    stop_bnbcli
    sleep 3
    start_bnbcli
    ;;
*)
    echo "Usage: bnbcli.sh -start | -stop | -restart .Or use systemctl start | stop | restart bnbcli.service "
    ;;
esac