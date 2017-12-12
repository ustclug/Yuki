package events

import (
	"testing"
)

func TestEmitter(t *testing.T) {
	type EvtPayload struct {
		name      string
		createdAt string
	}
	e := NewEmitter()
	e.
		On(SyncEnd, func(data *EvtPayload) {
			if data.name != "debian" {
				t.Error("Unexpected name")
			}
			if data.createdAt != "19700101" {
				t.Error("Unexpected createdAt")
			}
		}).
		On(SyncStart, func() {
			t.Log("SyncStart")
		}).
		On([]Event{ImportConfig, ExportConfig}, func() {
			t.Log("Import or Export")
		}).
		Emit(SyncStart).
		Emit(SyncEnd, &EvtPayload{
			name:      "debian",
			createdAt: "19700101",
		}).
		Emit(ImportConfig).
		Emit(ExportConfig)
}
