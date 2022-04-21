package main

import (
	"gopkg.in/src-d/go-cli.v0"
)

var (
	version string
	build   string
)

var app = cli.New("basic", version, build, "my basic command")

type subcommand struct {
	cli.PlainCommand `name:"sub" short-description:"my subcommand" long-description:"my subcommand"`
}

var sub = app.AddCommand(&subcommand{})

func main() {
	app.RunMain()
}
