package curr

import (
	"path"
	"runtime"
	"strings"
)

const (
	depthOfFunctionCaller = 1
)

// File is current file name provider,
// like `__FILE__` of PHP.
func File() string {
	_, fi, _, _ := runtime.Caller(depthOfFunctionCaller)
	return fi
}

// Basename is current file basename provider,
// like `basename(__FILE__)` of PHP.
func Basename() string {
	_, fi, _, _ := runtime.Caller(depthOfFunctionCaller)
	return path.Base(fi)
}

// Dir is current directory provider,
// like `__DIR__` of PHP.
func Dir() string {
	_, fi, _, _ := runtime.Caller(depthOfFunctionCaller)
	return path.Dir(fi)
}

// Func is current function name provider,
// like `__FUNCTION__` of PHP.
func Func() string {
	pc, _, _, _ := runtime.Caller(depthOfFunctionCaller)
	fn := runtime.FuncForPC(pc)
	elems := strings.Split(fn.Name(), ".")
	return elems[len(elems)-1]
}

// Line is current line provider,
// like `__LINE__` of PHP.
func Line() int {
	_, _, li, _ := runtime.Caller(depthOfFunctionCaller)
	return li
}
