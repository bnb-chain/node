#!/bin/bash

curl -s http://localhost:8080/orderbook/$1 | json_pp

