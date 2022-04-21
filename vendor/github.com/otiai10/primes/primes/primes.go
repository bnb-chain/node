package main

import (
	"fmt"
	"strconv"

	"github.com/otiai10/primes"
	"github.com/urfave/cli"
)

var findPrimes = func(ctx *cli.Context) error {
	if len(ctx.Args()) == 0 {
		return fmt.Errorf("`primes` needs second arg like `primes p 12`")
	}
	num, err := strconv.ParseInt(ctx.Args().First(), 10, 64)
	if err != nil {
		return err
	}
	fmt.Println(primes.Until(num).List())
	return nil
}
