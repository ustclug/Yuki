package server

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	testutils "github.com/ustclug/Yuki/test/utils"
)

func TestLoadConfig(t *testing.T) {
	tmp, err := os.CreateTemp("", "TestLoadConfig*.toml")
	require.NoError(t, err)
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	testutils.WriteFile(t, tmp.Name(), `
db_url = ":memory:"
repo_logs_dir = "/tmp"
repo_config_dir = "/tmp"
sync_timeout = "15s"
`)
	srv, err := New(tmp.Name())
	require.NoError(t, err)
	require.Equal(t, time.Second*15, srv.config.SyncTimeout)
	require.Equal(t, "/tmp", srv.config.RepoConfigDir[0])
}
