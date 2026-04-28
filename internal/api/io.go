package api

import (
	"bytes"
	"io"
)

func readAll(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}

func newBytesReader(b []byte) io.Reader {
	return bytes.NewReader(b)
}
