package server

import (
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestSyncTimeoutConfig(t *testing.T) {
	viper.Reset()
	viper.SetConfigFile("../../test/fixtures/sync_timeout.toml")
	config, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.SyncTimeout != time.Hour*48 {
		t.Fatalf("Expected SyncTimeout to be 48h, got %d", config.SyncTimeout)
	}
}
