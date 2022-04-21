#!/bin/sh
git fetch
git checkout origin/p2p_poc
go build
../tss -listen_addr="/ip4/0.0.0.0/tcp/27151" -dht_server_addr="/dns4/ec2-3-211-209-139.compute-1.amazonaws.com/tcp/27148/p2p/12D3KooWLXx68ortikYWRiyjgSnWercccrqrNmLpu7Yng37xKvTo"