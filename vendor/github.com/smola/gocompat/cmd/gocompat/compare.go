package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/mailru/easyjson"

	compat "github.com/smola/gocompat"
	"gopkg.in/src-d/go-cli.v0"
)

func init() {
	app.AddCommand(&compareCommand{})
}

type compareCommand struct {
	cli.Command    `name:"compare" short-desc:"List all symbols reachable from a package."`
	Path           string   `long:"path" default:".gocompat.json" description:"path to load reference API data from"`
	GitRefs        string   `long:"git-refs" description:"compare two git reference instead of loading form file, format is from-ref..to-ref"`
	Exclude        []string `long:"exclude" description:"excluded change type"`
	ExcludePackage []string `long:"exclude-package" description:"excluded package"`
	ExcludeSymbol  []string `long:"exclude-symbol" description:"excluded symbol" unquote:"false"`
	Go1Compat      bool     `long:"go1compat" description:"Based on Go 1 promise of compatibility. Equivalent to --exclude=SymbolAdded --exclude=FieldAdded --exclude=MethodAdded"`
	Positional     struct {
		Packages []string `positional-arg-name:"package" description:"Package to start from."`
	} `positional-args:"yes" required:"yes"`

	excluded map[compat.ChangeType]bool
}

func (c compareCommand) Execute(args []string) error {
	c.excluded = make(map[compat.ChangeType]bool)
	for _, e := range c.Exclude {
		ct, err := compat.ChangeTypeFromString(e)
		if err != nil {
			return err
		}

		c.excluded[ct] = true
	}

	if c.Go1Compat {
		c.excluded[compat.SymbolAdded] = true
		c.excluded[compat.FieldAdded] = true
		c.excluded[compat.MethodAdded] = true
	}

	var (
		from, to *compat.API
		err      error
	)

	if c.GitRefs != "" {
		from, to, err = c.getFromGit()
		if err != nil {
			return err
		}
	} else {
		from, err = c.getFromFile()
		if err != nil {
			return err
		}

		to, err = c.getCurrent()
		if err != nil {
			return err
		}
	}

	return c.compareResults(from, to)
}

func (c compareCommand) compareResults(from, to *compat.API) error {
	changed := false
	changes := compat.Compare(from, to)
	for _, change := range changes {
		if c.excluded[change.Type] {
			continue
		}

		exclude := false
		for _, pkg := range c.ExcludePackage {
			prefix := fmt.Sprintf(`"%s"`, pkg)
			if strings.HasPrefix(change.Symbol, prefix) {
				exclude = true
				break
			}
		}
		if exclude {
			continue
		}

		for _, sym := range c.ExcludeSymbol {
			if change.Symbol == sym {
				exclude = true
				break
			}
		}
		if exclude {
			continue
		}

		changed = true
		fmt.Println(change)
	}

	if changed {
		return fmt.Errorf("found backwards incompatible changes")
	}

	return nil
}

func (c compareCommand) getCurrent() (*compat.API, error) {
	return compat.ReachableFromPackages(c.Positional.Packages...)
}

func (c compareCommand) getFromFile() (from *compat.API, err error) {
	f, err := os.Open(c.Path)
	if err != nil {
		return nil, err
	}

	from = &compat.API{}
	err = easyjson.UnmarshalFromReader(f, from)
	return
}

func (c compareCommand) getFromGit() (from, to *compat.API, err error) {
	fields := strings.SplitN(c.GitRefs, "..", 2)

	fromRef := fields[0]
	toRef := fields[1]
	head, err := getHEAD()
	if err != nil {
		return nil, nil, err
	}

	defer func() {
		gitCheckout(head)
	}()

	if err := gitCheckout(fromRef); err != nil {
		return nil, nil, err
	}

	from, err = c.getCurrent()
	if err != nil {
		return nil, nil, err
	}

	if err := gitCheckout(toRef); err != nil {
		return nil, nil, err
	}

	to, err = c.getCurrent()
	if err != nil {
		return nil, nil, err
	}

	return from, to, nil
}

func getHEAD() (string, error) {
	headBytes, err := ioutil.ReadFile(".git/HEAD")
	if err != nil {
		return "", err
	}

	head := string(headBytes)
	head = strings.TrimSpace(head)
	if strings.HasPrefix(head, "ref: refs/heads/") {
		return head[len("ref: refs/heads/"):], nil
	}

	if strings.HasPrefix(head, "ref: ") {
		return head[len("ref: "):], nil
	}

	return head, nil
}

func gitCheckout(ref string) error {
	cmd := exec.Command("git", "checkout", ref)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
