package gpool

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPoolMaxWorkers(t *testing.T) {
	as := assert.New(t)
	stopCh := make(chan struct{})
	const n int32 = 5
	p := New(stopCh, WithMaxWorkers(n))
	for i := int32(0); i < n; i++ {
		p.Submit(func() {
			time.Sleep(time.Second * 1)
		})
	}
	p.Submit(func() {
		time.Sleep(time.Second * 1)
	})
	as.Equal(n, p.Workers())
	close(stopCh)
}

func TestPoolIdleTimeout(t *testing.T) {
	as := assert.New(t)
	stopCh := make(chan struct{})
	const n int32 = 5
	p := New(stopCh, WithIdleTimeout(time.Second), WithMaxWorkers(n))

	for i := int32(0); i < n; i++ {
		p.Submit(func() {
			time.Sleep(time.Millisecond * 50)
		})
	}
	time.Sleep(time.Millisecond * 1100)
	as.Equal(int32(2), p.Workers())
	close(stopCh)
}
