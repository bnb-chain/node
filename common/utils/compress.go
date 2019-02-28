package utils

import (
	"bytes"
	"compress/zlib"
)

func Compress(bz []byte) ([]byte, error) {
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	defer func() {
		if err := w.Close(); err != nil {
			panic(err)
		}
	}()
	_, err := w.Write(bz)
	if err != nil {
		return nil, err
	}
	err = w.Flush()
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
