#!/bin/sh
git fetch
git checkout origin/p2p_poc
go build
../tss -listen_addr="/ip4/172.31.31.230/tcp/27148" -dht_sever_mode=true