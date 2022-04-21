package capturer

import (
	"fmt"
	"os"
)

func ExampleCaptureStdout() {
	out := CaptureStdout(func() {
		fmt.Fprint(os.Stdout, "foo")
	})

	fmt.Println(out)
	// Output: foo
}

func ExampleCaptureStderr() {
	out := CaptureStderr(func() {
		fmt.Fprint(os.Stderr, "bar")
	})

	fmt.Println(out)
	// Output: bar
}

func ExampleCaptureOutput() {
	out := CaptureOutput(func() {
		fmt.Fprint(os.Stdout, "foo")
		fmt.Fprint(os.Stderr, "bar")
	})

	fmt.Println(out)
	// Output: foobar
}
