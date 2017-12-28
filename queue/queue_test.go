package queue

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestQueue(t *testing.T) {
	q := New(22 / 10)
	q.Push("1")
	q.Push("2")
	q.Push("3")
	q.Push("4")
	buf := bytes.NewBufferString("")
	q.WriteTo(buf)
	assert.Equal(t, "3\n4\n", buf.String())
}

func BenchmarkQueue(b *testing.B) {
	cap := b.N / 10
	if cap == 0 {
		cap = 1
	}
	q := New(cap)
	for i := 0; i < b.N; i++ {
		q.Push("Hello World!")
	}
}
