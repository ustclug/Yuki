package server

import (
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestDefaultTimeoutConfig(t *testing.T) {
	viper.Reset()
	viper.SetConfigFile("../../dist/daemon.toml")
	_, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
}

func TestSyncTimeoutConfig(t *testing.T) {
	viper.Reset()
	viper.SetConfigFile("../../test/fixtures/sync_timeout.toml")
	config, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.SyncTimeout != time.Second*15 {
		t.Fatalf("Expected SyncTimeout to be 15s, got %d", config.SyncTimeout)
	}
}
