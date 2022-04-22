#!/usr/bin/env bash
set -ex

go install golang.org/x/tools/cmd/goimports@latest
go get github.com/petermattis/goid@b0b1615b78e5ee59739545bb38426383b2cda4c9
go get github.com/sasha-s/go-deadlock@d68e2bc52ae3291765881b9056f2c1527f245f1e
