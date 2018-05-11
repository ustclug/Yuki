// Package events implements functions for listening or emitting events.
package events

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
func Emit(payload Payload) {
	globalEmitter.Emit(payload)
}

// On registers a listener for the given event.
func (e *Emitter) On(evt Event, listener Listener) *Emitter {
	e.listeners[evt] = append(e.listeners[evt], listener)
	return e
}

// Emit emits the given event with the padload.
func (e *Emitter) Emit(payload Payload) {
	evt := payload.Evt
	listeners, ok := e.listeners[evt]
	if !ok {
		return
	}

	for _, fn := range listeners {
		go fn(payload)
	}
}

// NewEmitter returns an instance of Emitter.
func NewEmitter() *Emitter {
	e := new(Emitter)
	e.listeners = make(map[Event][]Listener)
	return e
}
