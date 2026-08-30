package server

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/require"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/model"
	testutils "github.com/ustclug/Yuki/test/utils"
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
	require.Equal(t, "/data/2", repos[1].StorageDir)
}

func TestHandlerReloadAllRepos(t *testing.T) {
	te := NewTestEnv(t)
	rootDir, err := os.MkdirTemp("", t.Name())
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(rootDir)
	})
	cfgDir1 := filepath.Join(rootDir, "cfg1")
	cfgDir2 := filepath.Join(rootDir, "cfg2")
	require.NoError(t, os.Mkdir(cfgDir1, 0o755))
	require.NoError(t, os.Mkdir(cfgDir2, 0o755))
	te.server.config = Config{
		RepoLogsDir:   filepath.Join(rootDir, "logs"),
		RepoConfigDir: []string{"/no/such/dir", cfgDir1, cfgDir2},
	}
	te.server.repoSchedules.Set("should-be-deleted", cron.Schedule(nil))

	require.NoError(t, te.server.db.Create([]model.Repo{
		{
			Name: "should-be-deleted",
		},
		{
			Name: "repo0",
			Cron: "1 * * * *",
		},
	}).Error)

	require.NoError(t, te.server.db.Create([]model.RepoMeta{
		{
			Name: "should-be-deleted",
		},
		{
			Name:     "repo0",
			Upstream: "http://foo.com",
		},
	}).Error)

	for i := 0; i < 2; i++ {
		mirrorz := `
mirrorz:
  - desc: Default mapping
`
		if i == 1 {
			mirrorz = "\nmirrorz: []\n"
		}
		testutils.WriteFile(
			t,
			filepath.Join(cfgDir1, fmt.Sprintf("repo%d.yaml", i)),
			fmt.Sprintf(`
name: repo%d
cron: "* * * * *"
image: "alpine:latest"
storageDir: /tmp
%s`, i, mirrorz),
		)
	}
	testutils.WriteFile(t, filepath.Join(cfgDir2, "repo0.yaml"), `
image: ubuntu
envs:
  $UPSTREAM: http://bar.com
`)

	cli := te.RESTClient()
	resp, err := cli.R().Post("/repos")
	require.NoError(t, err)
	require.True(t, resp.IsSuccess(), "Unexpected response: %s", resp.Body())

	require.Equal(t, 2, te.server.repoSchedules.Count())

	var repos []model.Repo
	require.NoError(t, te.server.db.Order("name").Find(&repos).Error)
	require.Len(t, repos, 2)

	require.Equal(t, "repo0", repos[0].Name)
	require.Equal(t, "ubuntu", repos[0].Image)
	require.Equal(t, "* * * * *", repos[0].Cron)
	require.NotEmpty(t, repos[0].Envs)
	require.Equal(t, []model.MirrorzRepo{{
		Name: "repo0",
		Desc: "Default mapping",
	}}, repos[0].Mirrorz)

	require.Equal(t, "repo1", repos[1].Name)
	require.Equal(t, "alpine:latest", repos[1].Image)
	require.Empty(t, repos[1].Mirrorz)
	require.NotNil(t, repos[1].Mirrorz)

	var metas []model.RepoMeta
	require.NoError(t, te.server.db.Order("name").Find(&metas).Error)
	require.Len(t, repos, 2)
	require.Equal(t, "repo0", metas[0].Name)
	require.Equal(t, "http://bar.com", metas[0].Upstream)

	require.Equal(t, "repo1", metas[1].Name)
}

func TestHandlerReloadRepoRejectsDuplicateMirrorzRepo(t *testing.T) {
	te := NewTestEnv(t)
	cfgDir := t.TempDir()
	te.server.config.RepoConfigDir = []string{cfgDir}
	te.server.config.RepoLogsDir = t.TempDir()
	testutils.WriteFile(t, filepath.Join(cfgDir, "repo.yaml"), `
name: repo
cron: "* * * * *"
image: alpine:latest
storageDir: /tmp
mirrorz:
  - name: logical-repo
  - name: logical-repo
`)

	resp, err := te.RESTClient().R().Post("/repos/repo")
	require.NoError(t, err)
	require.Equal(t, 400, resp.StatusCode())
	require.Contains(t, string(resp.Body()), "duplicate MirrorZ repository")
}

func TestHandlerSyncRepo(t *testing.T) {
	te := NewTestEnv(t)
	name := te.RandomString()
	require.NoError(t, te.server.db.Create(&model.Repo{
		Name:       name,
		Cron:       "@every 1h",
		Image:      "alpine:latest",
		StorageDir: "/data",
	}).Error)
	schedule, _ := cron.ParseStandard("@every 1h")
	te.server.repoSchedules.Set(name, schedule)

	require.NoError(t, te.server.db.Create(&model.RepoMeta{Name: name}).Error)

	cli := te.RESTClient()
	resp, err := cli.R().Post(fmt.Sprintf("/repos/%s/sync", name))
	require.NoError(t, err)
	require.True(t, resp.IsSuccess(), "Unexpected response: %s", resp.Body())

	meta := model.RepoMeta{
		Name: name,
	}
	testutils.PollUntilTimeout(t, time.Minute, func() bool {
		require.NoError(t, te.server.db.Take(&meta).Error)
		return !meta.Syncing
	})

	require.NoError(t, te.server.db.Take(&meta).Error)
	require.NotEmpty(t, meta.PrevRun, "PrevRun")
	require.Empty(t, meta.ExitCode, "ExitCode")
	require.NotEmpty(t, meta.LastSuccess, "LastSuccess")
	require.NotEmpty(t, meta.NextRun, "NextRun")
}

func TestHandlerRemoveRepo(t *testing.T) {
	te := NewTestEnv(t)
	name := te.RandomString()
	require.NoError(t, te.server.db.Create(&model.Repo{
		Name:       name,
		Cron:       "@every 1h",
		Image:      "alpine:latest",
		StorageDir: "/data",
	}).Error)
	require.NoError(t, te.server.db.Create(&model.RepoMeta{Name: name}).Error)
	schedule, _ := cron.ParseStandard("@every 1h")
	te.server.repoSchedules.Set(name, schedule)

	cli := te.RESTClient()
	resp, err := cli.R().Delete(fmt.Sprintf("/repos/%s", name))
	require.NoError(t, err)
	require.True(t, resp.IsSuccess(), "Unexpected response: %s", resp.Body())

	require.False(t, te.server.repoSchedules.Has(name))
	require.ErrorContains(t, te.server.db.First(&model.Repo{Name: name}).Error, "record not found")
	require.ErrorContains(t, te.server.db.First(&model.RepoMeta{Name: name}).Error, "record not found")

	resp, err = cli.R().Delete("/repos/nonexist")
	require.NoError(t, err)
	require.Equal(t, 404, resp.StatusCode(), "Removing non-exist repo does not return 404")
}
