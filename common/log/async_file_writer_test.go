package log

import (
	"testing"
	"time"
)

func TestWriter(t *testing.T) {
	w := NewAsyncFileWriter("./hello.log", 100)
	w.Start()
	w.Write([]byte("hello\n"))
	w.Write([]byte("world\n"))
	w.Stop()
	time.Sleep(10 * time.Second)
}
