#!/bin/bash

curl -s http://localhost:1317/orderbook/$1 | json_pp

