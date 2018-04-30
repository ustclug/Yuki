// Package events implements functions for listening or emitting events.
package events

import (
	"fmt"
	"sync"
	"time"
)

// Event is an alias of int.
type Event = int

// M is an alias of map[string]interface{}.
type M = map[string]interface{}

const (
	// SyncStart event triggered before syncing a repository.
	SyncStart Event = iota
	// SyncEnd event triggered after syncing a repository.
	SyncEnd
	// ImportConfig event triggered after config imported.
	ImportConfig
	// ExportConfig event triggered after config exported.
	ExportConfig
)

// Payload is the event Payload.
type Payload struct {
	Evt   Event
	Attrs M
}

// Listener is the event listener.
type Listener func(data Payload)

var (
	globalEmitter *Emitter
	// ErrTimeout is returned for an expired deadline.
	ErrTimeout = fmt.Errorf("timeout")
)

// Emitter is the event emitter.
type Emitter struct {
	listeners map[Event][]Listener
}

func init() {
	globalEmitter = NewEmitter()
}

// On register listeners for global events.
func On(evt Event, listener Listener) *Emitter {
	return globalEmitter.On(evt, listener)
}

// Emit triggers global events.
func Emit(payload Payload) error {
	return globalEmitter.Emit(payload)
}

// On registers a listener for the given event.
func (e *Emitter) On(evt Event, listener Listener) *Emitter {
	e.listeners[evt] = append(e.listeners[evt], listener)
	return e
}

func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) error {
	c := make(chan struct{})
	go func() {
		wg.Wait()
		close(c)
	}()
	select {
	case <-c:
		return nil
	case <-time.After(timeout):
		return ErrTimeout
	}
}

// Emit emits the given event with the padload.
func (e *Emitter) Emit(payload Payload) error {
	var (
		wg sync.WaitGroup
	)

	evt := payload.Evt
	listeners, ok := e.listeners[evt]
	if !ok {
		return nil
	}

	wg.Add(len(listeners))

	for _, fn := range listeners {
		go func(fn Listener) {
			defer wg.Done()
			fn(payload)
		}(fn)
	}

	return waitTimeout(&wg, 5*time.Second)
}

// NewEmitter returns an instance of Emitter.
func NewEmitter() *Emitter {
	e := new(Emitter)
	e.listeners = make(map[Event][]Listener)
	return e
}
