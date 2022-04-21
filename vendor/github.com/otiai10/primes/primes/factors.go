package main

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/otiai10/jsonindent"
	"github.com/otiai10/primes"
	"github.com/urfave/cli"
)

var numericExp = regexp.MustCompile("([0-9]+)")

var factorize = func(ctx *cli.Context) error {
	if len(ctx.Args()) == 0 {
		return fmt.Errorf("`factorize` needs second arg like `primes f 12`")
	}
	num, err := strconv.ParseInt(ctx.Args().First(), 10, 64)
	if err != nil {
		return err
	}
	factors := primes.Factorize(num)
	if !ctx.Bool("json") {
		fmt.Println(factors.All())
		return nil
	}
	dict := factors.Powers()
	return jsonindent.NewEncoder(ctx.App.Writer).Encode(dict)
}
