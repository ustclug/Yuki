package server

import (
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestLoadDefaultConfig(t *testing.T) {
	_, err := loadConfig(viper.New(), "../../deploy/daemon.toml")
	require.NoError(t, err)
}

func TestLoadSyncTimeoutConfig(t *testing.T) {
	config, err := loadConfig(viper.New(), "../../test/fixtures/sync_timeout.toml")
	require.NoError(t, err)
	require.EqualValues(t, time.Second*15, config.SyncTimeout)
}
