package compat

import (
	"go/types"

	"golang.org/x/tools/go/packages"
)

func init() {
	conf := &packages.Config{
		Mode:  packages.LoadTypes,
		Tests: true,
	}

	loadedPackages, err := packages.Load(conf, "file=fixtures_test.go")
	if err != nil {
		panic(err)
	}

	FixtureObjects = make(map[string]types.Object)
	scope := loadedPackages[0].Types.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		FixtureObjects[obj.Name()] = obj
	}
}

var FixturePackage *packages.Package

var FixtureObjects map[string]types.Object

type StructA1 struct {
}

type StructA2 struct {
	Field1 string
}

type StructA3 struct {
	Field1 string
	Field2 string
}

type StructA4 struct {
	Field1 string
	Field2 int
}

type StructA5 struct {
	Field1 string
	Field2 int
	Field3 string
}

type TypeAlias1 = int

type TypeAlias2 = string

type TypeAlias3 = StructA3

type TypeAlias4 = StructA5

type Named1 int

type Named2 string

type Named3 StructA3

type Named4 StructA5

func Func1() {}

func Func2(arg1 string) {}

func Func3(arg1 string, arg2 string) {}

func Func4(arg1 string, arg2 int) {}

func Func5(arg1 string, arg2 int) string { return "" }

func Func6(arg1 string, arg2 int) int { return 0 }

func Func7(arg1 string, arg2 int) (string, int) { return "", 0 }

func Func8(arg1 ...string) {}

func Func9(arg1 StructA5) {}

type StructB1 struct{}

func (StructB1) Func1() {}

func (StructB1) func1() {}

type StructB2 struct{}

func (StructB2) Func1(arg1 int) {}

func (StructB2) func1() {}

var Array1 [20]int

var ChanRead1 <-chan int

var Map1 map[int]string

var Pointer1 *int

const Const1 = "foo"

var VarInit1 = "foo"

var VarInit2 *string = nil

var VarInit3 = ""

type Interface1 interface{}

type Interface2 interface {
	F()
}

type Interface3 interface {
	F(s string)
}

type Interface4 interface {
	f()
}
