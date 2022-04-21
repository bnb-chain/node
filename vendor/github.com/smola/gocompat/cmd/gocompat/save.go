package main

import (
	"os"

	"github.com/mailru/easyjson"
	compat "github.com/smola/gocompat"
	"gopkg.in/src-d/go-cli.v0"
)

func init() {
	app.AddCommand(&saveCommand{})
}

type saveCommand struct {
	cli.Command `name:"save" short-desc:"List all types reachable from a package."`
	Path        string `long:"path" default:".gocompat.json" description:"path to save the API data to"`
	Positional  struct {
		Packages []string `positional-arg-name:"package" description:"Package to start from."`
	} `positional-args:"yes" required:"yes"`
}

func (c saveCommand) Execute(args []string) (err error) {
	api, err := compat.ReachableFromPackages(c.Positional.Packages...)
	if err != nil {
		return err
	}

	f, err := os.Create(c.Path)
	if err != nil {
		return err
	}

	defer checkClose(f, &err)

	_, err = easyjson.MarshalToWriter(api, f)
	return err
}
