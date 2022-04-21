package main

import (
	"fmt"
	"regexp"

	"github.com/otiai10/primes"
	"github.com/urfave/cli"
)

var fractionExp = regexp.MustCompile("([0-9]+)/([0-9]+)")

var reduce = func(ctx *cli.Context) error {
	if len(ctx.Args()) == 0 {
		return fmt.Errorf("`reduce` needs second arg like `primes r 144/360`")
	}
	fraction, err := primes.ParseFractionString(ctx.Args().First())
	if err != nil {
		return err
	}
	fmt.Println(fraction.Reduce(-1).String())
	return nil
}
