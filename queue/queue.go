package queue

import (
	"bufio"
	"io"
)

type LinesQueue struct {
	next int
	len  int
	buf  []string
}

func New(cap int) LinesQueue {
	return LinesQueue{
		buf: make([]string, cap),
	}
}

func (q *LinesQueue) Push(s string) {
	q.buf[q.next] = s
	cap := len(q.buf)
	q.next = (q.next + 1) % cap
	if q.len < cap {
		q.len++
	}
}

func (q *LinesQueue) ReadFrom(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		q.Push(scanner.Text())
	}
}

func (q *LinesQueue) WriteTo(w io.Writer) (n int, err error) {
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
