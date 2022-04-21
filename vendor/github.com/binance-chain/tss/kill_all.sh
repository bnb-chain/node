#!/usr/bin/env bash

kill -9 $(ps aux | grep "tss" | grep -v "grep" | grep -v "___tss" | grep -v "bnbcli_tss" | grep -v "_atsserver" | awk '{print $2}')
kill -9 $(ps aux | grep "tbnbcli" | grep -v "grep" | awk '{print $2}')
