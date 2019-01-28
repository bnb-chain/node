package main

import (
	cmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
)

func main() {
	if err := cmd.LiteCmd.Execute(); err != nil {
		panic(err)
	}
}
