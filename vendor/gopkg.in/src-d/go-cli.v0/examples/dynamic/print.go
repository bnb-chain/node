package main

import (
	"fmt"

	flags "github.com/jessevdk/go-flags"
	"gopkg.in/src-d/go-cli.v0"
)

func init() {
	app.AddCommand(&printCommand{}, addDefaultMessage)
}

const defaultMessage = "my-message"

func addDefaultMessage(cmd *flags.Command) {
	for _, opt := range cmd.Options() {
		if opt.LongName == "message" {
			opt.Default = []string{defaultMessage}
		}
	}
}

type printCommand struct {
	cli.Command `name:"print" short-description:"prints a message" long-description:"prints a very nice message"`

	Message string `long:"message" env:"BASIC_MY_OPTION" description:"my option does something"`
}

func (c *printCommand) Execute(args []string) error {
	fmt.Printf("Message: %s\n", c.Message)
	return nil
}
