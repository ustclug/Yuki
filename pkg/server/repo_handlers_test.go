package server

import (
	"fmt"
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
		writeFile(
			t,
			filepath.Join(configDir, fmt.Sprintf("repo%d.yaml", i)),
			fmt.Sprintf(`
name: repo%d
interval: "* * * * *"
image: "alpine:latest"
storageDir: /tmp
`, i),
		)
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
