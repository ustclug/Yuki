package server

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/model"
)

func TestHandlerListRepos(t *testing.T) {
	te := NewTestEnv(t)
	require.NoError(t, te.server.db.Create([]model.Repo{
		{
			Name:       te.RandomString(),
			StorageDir: "/data/1",
		},
		{
			Name:       te.RandomString(),
			StorageDir: "/data/2",
		},
	}).Error)

	var repos api.ListReposResponse
	cli := te.RESTClient()
	resp, err := cli.R().SetResult(&repos).Get("/repos")
	require.NoError(t, err)
	require.True(t, resp.IsSuccess(), "Unexpected response: %s", resp.Body())

	require.Len(t, repos, 2)
	require.EqualValues(t, "/data/2", repos[1].StorageDir)
}

func TestHandlerGetRepoLogs(t *testing.T) {
	const repoName = "test-repo"
	te := NewTestEnv(t)
	logDir, err := os.MkdirTemp("", t.Name())
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(logDir)
	})
	repoLogDir := filepath.Join(logDir, repoName)
	_ = os.MkdirAll(repoLogDir, 0o755)

	te.server.config = &Config{
		LogDir: logDir,
	}
	log0Name := filepath.Join(repoLogDir, "result.log.0")
	require.NoError(t, os.WriteFile(log0Name, []byte("log0"), 0o644))

	log1Name := filepath.Join(repoLogDir, "result.log.1.gz")
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, _ = gw.Write([]byte("log1"))
	_ = gw.Flush()
	_ = gw.Close()
	require.NoError(t, os.WriteFile(log1Name, buf.Bytes(), 0o644))

	cli := te.RESTClient()
	testCases := map[string]struct {
		n string

		expectedContent string
	}{
		"plain text logs": {
			n:               "0",
			expectedContent: "log0",
		},
		"gzipped logs": {
			n:               "1",
			expectedContent: "log1",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			resp, err := cli.R().
				SetDoNotParseResponse(true).
				SetPathParam("name", repoName).
				SetQueryParam("n", tc.n).
				Get("/repos/{name}/logs")
			require.NoError(t, err)
			body := resp.RawBody()
			defer body.Close()
			rawBytes, err := io.ReadAll(body)
			require.NoError(t, err)
			data := string(rawBytes)
			require.True(t, resp.IsSuccess(), "Unexpected response: %s", data)
			require.EqualValues(t, tc.expectedContent, data)
		})
	}
}

func TestHandlerReloadAllRepos(t *testing.T) {
	te := NewTestEnv(t)
	configDir, err := os.MkdirTemp("", t.Name())
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(configDir)
	})
	te.server.config = &Config{
		RepoConfigDir: []string{"/no/such/dir", configDir},
	}

	for i := 0; i < 2; i++ {
		require.NoError(t, os.WriteFile(
			filepath.Join(configDir, fmt.Sprintf("repo%d.yaml", i)),
			[]byte(fmt.Sprintf(`
name: repo%d
interval: "* * * * *"
image: "alpine:latest"
storageDir: /data
`, i)),
			0o644,
		))
	}

	cli := te.RESTClient()
	resp, err := cli.R().Post("/repos")
	require.NoError(t, err)
	require.True(t, resp.IsSuccess(), "Unexpected response: %s", resp.Body())

	var repos []model.Repo
	require.NoError(t, te.server.db.Find(&repos).Error)
	require.Len(t, repos, 2)
}

func TestHandlerSyncRepo(t *testing.T) {
	te := NewTestEnv(t)
	name := te.RandomString()
	require.NoError(t, te.server.db.Create(&model.Repo{
		Name:       name,
		Interval:   "@every 1h",
		Image:      "alpine:latest",
		StorageDir: "/data",
		Envs: model.StringMap{
			"UPSTREAM": "http://foo.com",
		},
	}).Error)

	require.NoError(t, te.server.db.Create(&model.RepoMeta{Name: name}).Error)

	cli := te.RESTClient()
	resp, err := cli.R().Post(fmt.Sprintf("/repos/%s/sync", name))
	require.NoError(t, err)
	require.True(t, resp.IsSuccess(), "Unexpected response: %s", resp.Body())

	pollUntilTimeout(t, time.Minute, func() bool {
		_, exist := te.server.syncingContainers.Load(name)
		return !exist
	})

	meta := model.RepoMeta{
		Name: name,
	}
	require.NoError(t, te.server.db.First(&meta).Error)
	require.EqualValues(t, "http://foo.com", meta.Upstream)
	require.NotEmpty(t, meta.PrevRun)
	require.NotEmpty(t, meta.LastSuccess)
}
