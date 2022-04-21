curr
=====

[![CircleCI](https://circleci.com/gh/otiai10/curr.svg?style=svg)](https://circleci.com/gh/otiai10/curr)
[![codecov](https://codecov.io/gh/otiai10/curr/branch/master/graph/badge.svg)](https://codecov.io/gh/otiai10/curr)
[![WTFPL](https://img.shields.io/badge/License-WTFPL-orange.svg)](http://www.wtfpl.net/)

Current file and dir privider for Golang.

Just a sugar for [runtime](https://golang.org/pkg/runtime/).

```go
import "curr"

// __FILE__
f := curr.File()

// __DIR__
d := curr.Dir()

// __LINE__
l := curr.Line()

// __FUNCTION__
fn := curr.Func()

// basename(__FILE__)
b := curr.Basename()
```
