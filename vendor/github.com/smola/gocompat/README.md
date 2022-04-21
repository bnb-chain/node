
# gocompat [![godoc](https://godoc.org/github.com/smola/gocompat?status.svg)](https://godoc.org/github.com/smola/gocompat) [![Build Status](https://travis-ci.org/smola/gocompat.svg)](https://travis-ci.org/smola/gocompat) [![codecov.io](https://codecov.io/github/smola/gocompat/coverage.svg)](https://codecov.io/github/smola/gocompat)

**gocompat** is a tool to check compatibility between Go API versions.

## Usage

### Listing all symbols

**gocompat** considers an API as all exported symbols in a given set of packages as well as all exported symbols reachable from them. You can check this for the current package as follows:

```bash
gocompat reach .
```

### Compare current version against reference data

**gocompat** can save your API for later comparison. Usage example:

```bash
git checkout v1.0.0
gocompat save ./...
git checkout master
gocompat compare ./...
```

### Comparing two git reference

**gocompat** can compare the API of two git references in a repository. For example:

```
gocompat compare --git-refs=v0.1.0..master ./...
```

## Declaring your compatibility guarantees

There is almost no API change in Go that is fully backwards compatibility ([see this post for more](https://blog.merovius.de/2015/07/29/backwards-compatibility-in-go.html)). By default, gocompat uses a strict approach in which most changes to exported symbols are considered incompatible. The `--exclude=` flag can be used to exclude a change type from results.

Most users will probably want to use compatibility guarantees analogous to the [Go 1 compatibility promise](https://golang.org/doc/go1compat). You can use the `--go1compat` for that, which is a shorthand for `--exclude=SymbolAdded --exclude=FieldAdded --exclude=MethodAdded`. For example:

```
gocompat compare --go1compat --from-git=v1.0.0..v1.1.0 ./...
```

If you are using [Semantic Versioning](https://semver.org/), you might want to use the strict defaults for patch versions.

## License

Released under the terms of the Apache License Version 2.0, see [LICENSE](LICENSE).