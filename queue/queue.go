// Package queue is designed to save the last N lines from a Reader.
package queue

import (
	"bufio"
	"io"
)

// LinesQueue save the last N lines.
type LinesQueue struct {
	next int
	len  int
	buf  []string
}

// New returns an instance of LinesQueue with a fixed capacity.
func New(cap int) LinesQueue {
	return LinesQueue{
		buf: make([]string, cap),
	}
}

// Push appends a new line to the queue and pops the oldest one if the queue is already full.
func (q *LinesQueue) Push(s string) {
	q.buf[q.next] = s
	cap := len(q.buf)
	q.next = (q.next + 1) % cap
	if q.len < cap {
		q.len++
	}
}

// ReadAll reads from a Reader line by line and saves the last N lines.
func (q *LinesQueue) ReadAll(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		q.Push(scanner.Text())
	}
	return scanner.Err()
}

// WriteAll writes all the lines to the Writer.
func (q *LinesQueue) WriteAll(w io.Writer) (n int, err error) {
	if q.len == 0 {
		return 0, nil
	}
	n = 0
	cap := len(q.buf)
	head := q.next - q.len
	if head < 0 {
		head += cap
	}

	for {
		written, err := w.Write([]byte(q.buf[head] + "\n"))
		n += written
		if err != nil {
			return n, err
		}
		head = (head + 1) % cap
		if head == q.next {
			break
		}
	}
	return n, nil
}
