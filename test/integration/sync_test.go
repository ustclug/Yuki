package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"

	testutils "github.com/ustclug/Yuki/test/utils"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/server"
)

func TestSyncRepo(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", t.Name())
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(tmpdir)
	})

	configPath := filepath.Join(tmpdir, "config.toml")
	testutils.WriteFile(t, configPath, fmt.Sprintf(`
db_url = "%s/yukid.db"
repo_logs_dir = "%s/log/"
repo_config_dir = "%s/config/"
`, tmpdir, tmpdir, tmpdir))

	srv, err := server.New(configPath)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		err := srv.Start(ctx)
		if err != nil {
			t.Errorf("Fail to start server: %v", err)
		}
		cancel()
	}()

	testutils.WriteFile(t, filepath.Join(tmpdir, "config/foo.yaml"), `
name: "foo"
interval: "@every 1h"
image: "ustcmirror/test:latest"
storageDir: "/tmp"
`)

	time.Sleep(5 * time.Second)
	restCli := resty.New()
	resp, err := restCli.R().Post("http://127.0.0.1:9999/api/v1/repos/foo")
	require.NoError(t, err)
	require.True(t, resp.IsSuccess(), "Unexpected response: %s", resp.Body())

	resp, err = restCli.R().Post("http://127.0.0.1:9999/api/v1/repos/foo/sync")
	require.NoError(t, err)
	require.True(t, resp.IsSuccess(), "Unexpected response: %s", resp.Body())

	var meta api.GetRepoMetaResponse
	testutils.PollUntilTimeout(t, 5*time.Minute, func() bool {
		resp, err = restCli.R().SetResult(&meta).Get("http://127.0.0.1:9999/api/v1/metas/foo")
		require.NoError(t, err)
		require.True(t, resp.IsSuccess(), "Unexpected response: %s", resp.Body())
		if !meta.Syncing {
			return true
		}
		t.Log("Waiting for syncing to finish")
		return false
	})
}
