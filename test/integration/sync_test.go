package integration

import (
	"context"
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
	logDir := filepath.Join(tmpdir, "log")
	cfgDir := filepath.Join(tmpdir, "config")
	os.MkdirAll(logDir, 0755)
	os.MkdirAll(cfgDir, 0755)

	cfg := server.DefaultConfig
	cfg.DbURL = filepath.Join(tmpdir, "yukid.db")
	cfg.RepoConfigDir = []string{cfgDir}
	cfg.RepoLogsDir = logDir
	cfg.ListenAddr = "127.0.0.1:0"
	srv, err := server.NewWithConfig(cfg)
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

	testutils.WriteFile(t, filepath.Join(cfgDir, "foo.yaml"), `
name: "foo"
cron: "@every 1h"
image: "ustcmirror/test:latest"
storageDir: "/tmp"
`)

	time.Sleep(3 * time.Second)
	if t.Failed() {
		return
	}
	restCli := resty.New().SetBaseURL("http://" + srv.ListenAddr())
	resp, err := restCli.R().Post("/api/v1/repos/foo")
	require.NoError(t, err)
	require.True(t, resp.IsSuccess(), "Unexpected response: %s", resp.Body())

	resp, err = restCli.R().Post("/api/v1/repos/foo/sync")
	require.NoError(t, err)
	require.True(t, resp.IsSuccess(), "Unexpected response: %s", resp.Body())

	var meta api.GetRepoMetaResponse
	testutils.PollUntilTimeout(t, 5*time.Minute, func() bool {
		resp, err = restCli.R().SetResult(&meta).Get("/api/v1/metas/foo")
		require.NoError(t, err)
		require.True(t, resp.IsSuccess(), "Unexpected response: %s", resp.Body())
		if !meta.Syncing {
			return true
		}
		t.Log("Waiting for syncing to finish")
		return false
	})
}
