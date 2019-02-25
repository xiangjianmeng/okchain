package util

import (
	"fmt"
	"hash"
)

type ConWriter interface {
	Write([]byte) ConWriter
	Error() error
}

type failConWriter struct {
	Err error
}

func (w *failConWriter) Write([]byte) ConWriter {
	return w
}

func (w *failConWriter) Error() error {
	return w.Err
}

type hashConWriter struct {
	H hash.Hash
}

func (w hashConWriter) Write(p []byte) ConWriter {
	i, err := w.H.Write(p)

	if err != nil {
		return &failConWriter{err}
	} else if i < len(p) {
		return &failConWriter{fmt.Errorf("Write %d for %d bytes", i, len(p))}
	} else {
		return w
	}
}

func (w hashConWriter) Error() error {
	return nil
}

func NewHashWriter(h hash.Hash) ConWriter {
	return &hashConWriter{h}
}
