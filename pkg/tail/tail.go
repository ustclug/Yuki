// Package tail provides support for outputing the last N lines of a ReadSeeker.
package tail

import (
	"bytes"
	"io"
)

const blockSize = 1024

var eol = []byte("\n")

// Tail prints the last N lines.
type Tail struct {
	r io.ReadSeeker
	n int
}

// New returns an instance of Tail.
func New(r io.ReadSeeker, n int) *Tail {
	return &Tail{r, n}
}

// WriteTo writes last N lines to the Writer.
func (t *Tail) WriteTo(w io.Writer) (n int64, err error) {
	if t.n == 0 {
		n, err = io.Copy(w, t.r)
		return
	}

	size, err := t.r.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}

	var b, data []byte
	block := -1
	cnt := 0

	for t.n >= cnt {
		step := int64(block * blockSize)
		left := size + step // how many bytes to beginning
		if left < 0 {
			if _, err = t.r.Seek(0, io.SeekStart); err != nil {
				return 0, err
			}
			b = make([]byte, blockSize+left)
			if _, err = t.r.Read(b); err != nil {
				return 0, err
			}
			data = append(b, data...)
			break
		} else {
			if _, err = t.r.Seek(left, io.SeekStart); err != nil {
				return 0, err
			}
			b = make([]byte, blockSize)
			if _, err = t.r.Read(b); err != nil {
				return 0, err
			}
			data = append(b, data...)
		}
		cnt += bytes.Count(b, eol)
		block--
	}

	if bytes.HasSuffix(data, eol) {
		data = data[:len(data)-1]
	}
	lines := bytes.Split(data, eol)
	nLines := len(lines)
	if t.n < nLines {
		lines = lines[nLines-t.n:]
	}

	for _, l := range lines {
		l = append(l, eol...)
		written, err := w.Write(l)
		n += int64(written)
		if err != nil {
			return n, err
		}
	}
	return n, nil
}
