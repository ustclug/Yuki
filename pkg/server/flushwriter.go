package server

import (
	"io"
)

type flushWriter struct {
	w FlushWriter
}

type FlushWriter interface {
	io.Writer
	Flush()
}

func (f *flushWriter) Flush() {
	f.w.Flush()
}

func (f *flushWriter) Write(p []byte) (n int, err error) {
	n, err = f.w.Write(p)
	f.w.Flush()
	return n, err
}

func NewFlushWriter(w FlushWriter) FlushWriter {
	return &flushWriter{w}
}
