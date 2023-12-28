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

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/server"
)

func TestSyncRepo(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", t.Name())
	require.NoError(t, err)
	t.Log(tmpdir)
	t.Cleanup(func() {
		// _ = os.RemoveAll(tmpdir)
	})

	configPath := filepath.Join(tmpdir, "config.toml")
	require.NoError(t, os.WriteFile(configPath, []byte(fmt.Sprintf(`
db_url = "%s/yukid.db"
log_dir = "%s/log/"
repo_config_dir = "%s/config/"
`, tmpdir, tmpdir, tmpdir)), 0o644))

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

	require.NoError(t, os.WriteFile(filepath.Join(tmpdir, "config/foo.yaml"), []byte(`
name: "foo"
interval: "@every 1h"
image: "ustcmirror/test:latest"
storageDir: "/tmp"
`), 0o644))

	time.Sleep(5 * time.Second)
	restCli := resty.New()
	resp, err := restCli.R().Post("http://127.0.0.1:9999/api/v1/repos/foo")
	require.NoError(t, err)
	require.True(t, resp.IsSuccess(), "Unexpected response: %s", resp.Body())

	resp, err = restCli.R().Post("http://127.0.0.1:9999/api/v1/repos/foo/sync")
	require.NoError(t, err)
	require.True(t, resp.IsSuccess(), "Unexpected response: %s", resp.Body())

	var meta api.GetMetaResponse
	for {
		resp, err = restCli.R().SetResult(&meta).Get("http://127.0.0.1:9999/api/v1/metas/foo")
		require.NoError(t, err)
		require.True(t, resp.IsSuccess(), "Unexpected response: %s", resp.Body())
		if !meta.Syncing {
			break
		}
		t.Log("Waiting for syncing to finish")
		time.Sleep(3 * time.Second)
	}
}
