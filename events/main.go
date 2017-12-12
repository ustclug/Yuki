package events

import (
	"errors"
	"reflect"
	"sync"
)

type Event int

const (
	SyncStart Event = iota
	SyncEnd
	ImportConfig
	ExportConfig
)

var (
	ErrNotAFunc = errors.New("The type of listener is not a function")
)

type Emitter struct {
	listeners map[interface{}][]reflect.Value
}

func (e *Emitter) On(events, listener interface{}) *Emitter {
	fn := reflect.ValueOf(listener)

	if reflect.Func != fn.Kind() {
		panic(ErrNotAFunc)
	}

	evs := reflect.ValueOf(events)
	if reflect.Array == evs.Kind() || reflect.Slice == evs.Kind() {
		len := evs.Len()
		for i := 0; i < len; i++ {
			ev := evs.Index(i).Interface()
			e.listeners[ev] = append(e.listeners[ev], fn)
		}
	} else {
		e.listeners[events] = append(e.listeners[events], fn)
	}

	return e
}

func (e *Emitter) Emit(evt interface{}, payload ...interface{}) *Emitter {
	var (
		wg sync.WaitGroup
	)

	listeners, ok := e.listeners[evt]
	if !ok {
		return e
	}

	wg.Add(len(listeners))

	for _, fn := range listeners {
		go func(fn reflect.Value) {
			defer wg.Done()
			var vals []reflect.Value
			for _, arg := range payload {
				vals = append(vals, reflect.ValueOf(arg))
			}
			fn.Call(vals)
		}(fn)
	}

	wg.Wait()

	return e
}

func NewEmitter() *Emitter {
	e := new(Emitter)
	e.listeners = make(map[interface{}][]reflect.Value)
	return e
}
