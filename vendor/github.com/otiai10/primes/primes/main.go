package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Version = "0.1.0"
	app.Name = "primes"
	app.Usage = "Find primes, factors, and reduce fractions"
	app.Commands = []cli.Command{
		cli.Command{
			Name:      "prime",
			Aliases:   []string{"p"},
			Usage:     "Find primes until specified number",
			Action:    findPrimes,
			ArgsUsage: "number to find primes until",
		},
		cli.Command{
			Name:      "factorize",
			Aliases:   []string{"f"},
			Usage:     "Factorize given number",
			Action:    factorize,
			ArgsUsage: "number to factorize",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "json,j",
					Usage: "Show factors by json format",
				},
			},
		},
		cli.Command{
			Name:      "reduce",
			Aliases:   []string{"r"},
			Usage:     "Reduce fraction",
			Action:    reduce,
			ArgsUsage: "fraction expression",
		},
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
