package gpool

import (
	"time"
)

type Option func(pool *poolImpl)

func WithMaxWorkers(n int32) Option {
	return func(pool *poolImpl) {
		pool.maxWorkers = n
	}
}

func WithIdleTimeout(timeout time.Duration) Option {
	return func(pool *poolImpl) {
		pool.idleTimeout = timeout
	}
}
