package log

import (
	"testing"
)

func TestWriter(t *testing.T) {
	w := NewAsyncFileWriter("./hello.log", 100)
	w.Start()
	w.Write([]byte("hello\n"))
	w.Write([]byte("world\n"))
	w.Stop()
}
