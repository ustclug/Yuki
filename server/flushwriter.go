package server

type flushWriter struct {
	w FlushWriter
}

type FlushWriter interface {
	Write(p []byte) (int, error)
	Flush()
}

func (f *flushWriter) Flush() {
	f.w.Flush()
}

func (f *flushWriter) Write(p []byte) (n int, err error) {
	n, err = f.w.Write(p)
	f.w.Flush()
	return
}

func NewFlushWriter(w FlushWriter) FlushWriter {
	return &flushWriter{w}
}
