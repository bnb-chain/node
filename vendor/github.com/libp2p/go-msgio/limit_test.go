package msgio

import (
	"bytes"
	"testing"
)

func TestLimitReader(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	reader, _ := LimitedReader(buf) // limit is set to 0
	n, err := reader.Read([]byte{})
	if n != 0 || err.Error() != "EOF" {
		t.Fatal("Expected not to read anything")
	}
}

func TestLimitWriter(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	writer := NewLimitedWriter(buf)
	n, err := writer.Write([]byte{1, 2, 3})
	if n != 3 || err != nil {
		t.Fatal("Expected to write 3 bytes with no errors")
	}
	err = writer.Flush()
}
