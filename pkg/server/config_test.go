package server

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadSyncTimeoutConfig(t *testing.T) {
	f, err := os.CreateTemp("", "sync_timeout*.toml")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = f.Close()
		_ = os.Remove(f.Name())
	})
	require.NoError(t, os.WriteFile(f.Name(), []byte(`
db_url = "test"

repo_logs_dir = "/tmp/log_yuki/"

repo_config_dir = "/tmp/config_yuki"

sync_timeout = "15s"
`), 0o644))

	config, err := loadConfig(f.Name())
	require.NoError(t, err)
	require.EqualValues(t, time.Second*15, config.SyncTimeout)
}
