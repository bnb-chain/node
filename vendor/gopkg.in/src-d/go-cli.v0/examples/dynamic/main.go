package main

import (
	"gopkg.in/src-d/go-cli.v0"
)

var (
	version string
	build   string
)

var app = cli.New("basic", version, build, "my basic command")

func main() {
	app.RunMain()
}
