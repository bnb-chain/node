package log

import (
	"os"
	"strings"
	"testing"
)

func TestWriter(t *testing.T) {
	w := NewAsyncFileWriter("./hello.log", 100)
	w.Start()
	w.Write([]byte("hello\n"))
	w.Write([]byte("world\n"))
	w.Stop()
	files, _ := os.ReadDir("./")
	for _, f := range files {
		fn := f.Name()
		if strings.HasPrefix(fn, "hello") {
			t.Log(fn)
			content, _ := os.ReadFile(fn)
			t.Log(content)
			os.Remove(fn)
		}
	}
}
