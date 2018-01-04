package events

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmitter(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	e := NewEmitter()
	e.
		On(SyncEnd, func(data Payload) {
			t.Log("SyncEnd")
			as.Equal("debian", data.Attrs["name"])
			as.Equal("19700101", data.Attrs["createdAt"])
		}).
		On(SyncStart, func(data Payload) {
			t.Log("SyncStart")
		}).
		On(ExportConfig, func(data Payload) {
			t.Log("Export")
		}).
		On(ImportConfig, func(data Payload) {
			t.Log("Import")
		}).
		Emit(Payload{SyncStart, nil}).
		Emit(Payload{
			Evt: SyncEnd,
			Attrs: M{
				"name":      "debian",
				"createdAt": "19700101",
			},
		}).
		Emit(Payload{ImportConfig, nil}).
		Emit(Payload{ExportConfig, nil})
}
