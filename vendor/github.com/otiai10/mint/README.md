# mint

[![Build Status](https://travis-ci.org/otiai10/mint.svg?branch=master)](https://travis-ci.org/otiai10/mint)
[![codecov](https://codecov.io/gh/otiai10/mint/branch/master/graph/badge.svg)](https://codecov.io/gh/otiai10/mint)
[![GoDoc](https://godoc.org/github.com/otiai10/mint?status.png)](https://godoc.org/github.com/otiai10/mint)

The very minimum assertion for Go.

```go
package your_test

import (
    "testing"
    "pkg/your"
    . "github.com/otiai10/mint"
)

func TestFoo(t *testing.T) {

    foo := your.Foo()
    Expect(t, foo).ToBe(1234)
    Expect(t, foo).TypeOf("int")
    Expect(t, foo).Not().ToBe(nil)
    Expect(t, func() { yourFunc() }).Exit(1)

    // If assertion failed, exit 1 with message.
    Expect(t, foo).ToBe("foobarbuz")

    // You can run assertions without os.Exit
    res := Expect(t, foo).Dry().ToBe("bar")
    // res.OK() == false

    // You can omit repeated `t`.
    m := mint.Blend(t)
    m.Expect(foo).ToBe(1234)
}
```

# features

- Simple syntax
- Loosely coupled
- Plain implementation

# tests
```
go test ./...
```

# use cases

Projects bellow use `mint`

- [github.com/otiai10/gosseract](https://github.com/otiai10/gosseract/blob/master/all_test.go)
- [github.com/otiai10/marmoset](https://github.com/otiai10/marmoset/blob/master/all_test.go#L168-L190)
