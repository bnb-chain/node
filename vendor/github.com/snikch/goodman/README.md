# Goodman [![Godoc Reference](http://img.shields.io/badge/godoc-reference-5272B4.svg?style=flat-square)](https://godoc.org/github.com/snikch/goodman) ![Travis Status](https://travis-ci.org/snikch/goodman.svg?branch=master)

Goodman is a [Dredd](https://github.com/apiaryio/dredd) hook handler implementation in Go. The API may change, please Vendor this library.

## About
This package contains a Go Dredd hook handler which provides a bridge between the [Dredd API Testing Framework](http://dredd.readthedocs.org/en/latest/)
 and Go environment to ease implementation of testing hooks provided by [Dredd](http://dredd.readthedocs.org/en/latest/). Write Dredd hooks in Go to glue together [API Blueprint](https://apiblueprint.org/) with your Go project

Not sure what these Dredd Hooks are?  Read the Dredd documentation on [them](http://dredd.readthedocs.org/en/latest/hooks/)

The following are a few examples of what hooks can be used for:

- loading db fixtures
- cleanup after test step or steps
- handling authentication and sessions
- passing data between transactions (saving state from responses to stash)
- modifying request generated from blueprint
- changing generated expectations
- setting custom expectations
- debugging via logging stuff


## Installing

**Must use Dredd v1.1.0 or greater**

```bash
go get github.com/snikch/goodman/cmd/goodman
```

## Usage

1). Create a hook file in `hooks.go`

```go
package main

import (
  "fmt"

  "github.com/snikch/goodman/hooks"
  trans "github.com/snikch/goodman/transaction"
)

func main() {
      h := hooks.NewHooks()
      server := hooks.NewServer(hooks.NewHooksRunner(h))
      h.Before("/message > GET", func(t *trans.Transaction) {
          fmt.Println("before modification")
      })
      server.Serve()
      defer server.Listener.Close()
})

```

2). Compile your hooks program

```bash
go build -o hooks path/to/hooks.go
```

3). Run it with dredd

`dredd apiary.apib localhost:3000 --language go --hookfiles ./hooks`

## API

The `hooks.Server` struct provides the following methods to hook into the following dredd transactions: `before`, `after`, `before_all`, `after_all`, `before_each`, `after_each`, `before_validation`, and `before_each_validation`.
The `before`, `before_validation` and `after` hooks are identified by [transaction name](http://dredd.readthedocs.org/en/latest/hooks/#getting-transaction-names).

## How to Contribute

1. Fork it
2. Create your feature branch (git checkout -b my-newfeature)
3. Commit your changes (git commit -am 'Add some feature')
4. Push (git push origin my-new-feature)
5. Create a new Pull Request

## Tests

The test suite consists of go test suite and aruba/cucumber tests

Running the tests

- go tests `go test github.com/snikch/{,/hooks,/transaction}`

- aruba tests
  - Install local dredd copy `npm install`
  - Install aruba ruby gem `bundle install`
  - Run test suite `bundle exec cucumber`
