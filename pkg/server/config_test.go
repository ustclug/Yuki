package server

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, path, content string) {
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestLoadSyncTimeoutConfig(t *testing.T) {
	f, err := os.CreateTemp("", "sync_timeout*.toml")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = f.Close()
		_ = os.Remove(f.Name())
	})

	writeFile(t, f.Name(), `
db_url = "test"

repo_logs_dir = "/tmp/log_yuki/"

repo_config_dir = "/tmp/config_yuki"

sync_timeout = "15s"
`)

	config, err := loadConfig(f.Name())
	require.NoError(t, err)
	require.EqualValues(t, time.Second*15, config.SyncTimeout)
}
