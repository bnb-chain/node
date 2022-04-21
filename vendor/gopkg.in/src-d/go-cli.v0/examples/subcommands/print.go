package main

import (
	"fmt"

	"gopkg.in/src-d/go-cli.v0"
)

func init() {
	sub.AddCommand(&printCommand{})
}

type printCommand struct {
	cli.Command `name:"print" short-description:"prints a message" long-description:"prints a very nice message"`

	Message string `long:"message" env:"BASIC_MY_OPTION" default:"my-message" description:"my option does something"`
}

func (c *printCommand) Execute(args []string) error {
	fmt.Printf("Message: %s\n", c.Message)
	return nil
}
