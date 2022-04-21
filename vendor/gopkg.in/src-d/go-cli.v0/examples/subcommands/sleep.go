package main

import (
	"context"
	"fmt"
	"time"

	"gopkg.in/src-d/go-cli.v0"
)

func init() {
	sub.AddCommand(&sleepCommand{})
}

type sleepCommand struct {
	cli.Command `name:"sleep" short-description:"sleeps forever" long-description:"sleeps indefinitely until it receives SIGTERM or SIGINT"`

	Sleep time.Duration `long:"duration" default:"1s" description:"sleep intervals"`
}

func (c *sleepCommand) ExecuteContext(ctx context.Context, args []string) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		fmt.Println("Sleeping...")
		time.Sleep(c.Sleep)
	}
}
