package curr

import (
	"os"
	"testing"

	. "github.com/otiai10/mint"
)

var pkgpath string

func init() {
	gopath := os.Getenv("GOPATH")
	pkgpath = gopath + "/src/github.com/otiai10/curr"
}

func TestLine(t *testing.T) {
	Because(t, "Line() should provide current line", func(t *testing.T) {
		Expect(t, Line()).ToBe(19)
		// <- line 20
		Expect(t, Line()).ToBe(21)
	})
}

func TestFunc(t *testing.T) {
	Because(t, "Func() should provide current function name", func(t *testing.T) {
		Expect(t, Foobaa()).ToBe("Foobaa")
	})
	Expect(t, Func()).ToBe("TestFunc")
}

func TestFile(t *testing.T) {
	Because(t, "File() should provide current file name", func(t *testing.T) {
		Expect(t, File()).ToBe(pkgpath + "/all_test.go")
	})
}

func TestBasename(t *testing.T) {
	Because(t, "Basename() should provide current file basename", func(t *testing.T) {
		Expect(t, Basename()).ToBe("all_test.go")
	})
}

func TestDir(t *testing.T) {
	Because(t, "Dir() should provide current directory", func(t *testing.T) {
		Expect(t, Dir()).ToBe(pkgpath)
	})
}

func Foobaa() string {
	return Func()
}
