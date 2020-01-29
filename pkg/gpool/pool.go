package gpool

import (
	"sync"
	"time"
)

type Job = func()

type Pool interface {
	Submit(job Job)
	Workers() int32
}

type poolImpl struct {
	maxWorkers  int32
	idleTimeout time.Duration
	stopCh      <-chan struct{}

	mu         sync.RWMutex
	curWorkers int32
	jobQueue   chan Job
}

func (p *poolImpl) worker() {
	for {
		select {
		case <-p.stopCh:
			return
		case job := <-p.jobQueue:
			job()
		case <-time.After(p.idleTimeout):
			p.mu.RLock()
			if p.curWorkers < 3 {
				p.mu.RUnlock()
				continue
			}
			p.mu.RUnlock()

			p.mu.Lock()
			p.curWorkers--
			p.mu.Unlock()
			return
		}
	}
}

func (p *poolImpl) Workers() int32 {
	p.mu.RLock()
	n := p.curWorkers
	p.mu.RUnlock()
	return n
}

func (p *poolImpl) Submit(job Job) {
	for {
		select {
		case p.jobQueue <- job:
			return
		case <-time.After(time.Millisecond * 10):
			p.mu.RLock()
			if p.curWorkers < p.maxWorkers {
				p.mu.RUnlock()

				p.mu.Lock()
				p.curWorkers++
				p.mu.Unlock()

				go p.worker()
				continue
			}
			p.mu.RUnlock()
		}
	}
}

func New(stopCh <-chan struct{}, options ...Option) Pool {
	p := &poolImpl{
		stopCh:      stopCh,
		maxWorkers:  5,
		idleTimeout: time.Minute * 5,
		jobQueue:    make(chan Job),
	}
	for _, op := range options {
		op(p)
	}
	return p
}
