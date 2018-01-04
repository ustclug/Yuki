package events

import (
	"sync"
)

type Event int
type M map[string]interface{}

const (
	SyncStart Event = iota
	SyncEnd
	ImportConfig
	ExportConfig
)

type Payload struct {
	Evt   Event
	Attrs M
}

type Listener func(data Payload)

var (
	globalEmitter *Emitter
)

type Emitter struct {
	listeners map[Event][]Listener
}

func init() {
	globalEmitter = NewEmitter()
}

func On(evt Event, listener Listener) *Emitter {
	return globalEmitter.On(evt, listener)
}

func Emit(payload Payload) *Emitter {
	return globalEmitter.Emit(payload)
}

func (e *Emitter) On(evt Event, listener Listener) *Emitter {
	e.listeners[evt] = append(e.listeners[evt], listener)
	return e
}

func (e *Emitter) Emit(payload Payload) *Emitter {
	var (
		wg sync.WaitGroup
	)

	evt := payload.Evt
	listeners, ok := e.listeners[evt]
	if !ok {
		return e
	}

	wg.Add(len(listeners))

	for _, fn := range listeners {
		go func(fn Listener) {
			defer wg.Done()
			fn(payload)
		}(fn)
	}

	wg.Wait()

	return e
}

func NewEmitter() *Emitter {
	e := new(Emitter)
	e.listeners = make(map[Event][]Listener)
	return e
}
