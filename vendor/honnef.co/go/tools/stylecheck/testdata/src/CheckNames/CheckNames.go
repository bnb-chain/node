// Package pkg_foo ...
package pkg_foo // MATCH "should not use underscores in package names"

var range_ int
var _abcdef int
var abcdef_ int
var abc_def int  // MATCH "should not use underscores in Go names; var abc_def should be abcDef"
var abc_def_ int // MATCH "should not use underscores in Go names; var abc_def_ should be abcDef_"

func fn_1()  {} // MATCH "func fn_1 should be fn1"
func fn2()   {}
func fn_Id() {} // MATCH "func fn_Id should be fnID"
func fnId()  {} // MATCH "func fnId should be fnID"

var FOO_BAR int // MATCH "should not use ALL_CAPS in Go names; use CamelCase instead"
var Foo_BAR int // MATCH "var Foo_BAR should be FooBAR"
var foo_bar int // MATCH "foo_bar should be fooBar"
var kFoobar int // not a check we inherited from golint. more false positives than true ones.

func fn(x []int) {
	var (
		a_b = 1 // MATCH "var a_b should be aB"
		c_d int // MATCH "var c_d should be cD"
	)
	a_b += 2
	for e_f := range x { // MATCH "range var e_f should be eF"
		_ = e_f
	}

	_ = a_b
	_ = c_d
}

//export fn_3
func fn_3() {}

//export not actually the export keyword
func fn_4() {} // MATCH "func fn_4 should be fn4"

//export
func fn_5() {} // MATCH "func fn_5 should be fn5"

// export fn_6
func fn_6() {} // MATCH "func fn_6 should be fn6"

//export fn_8
func fn_7() {} // MATCH "func fn_7 should be fn7"

//go:linkname fn_8 time.Now
func fn_8() {}
