#!/bin/bash

curl -s http://localhost:8080/api/v1/depth/$1 | json_pp

