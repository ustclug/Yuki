package server

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/stretchr/testify/require"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/docker"
	"github.com/ustclug/Yuki/pkg/model"
	testutils "github.com/ustclug/Yuki/test/utils"
)

func TestInitRepoMetas(t *testing.T) {
	te := NewTestEnv(t)
	require.NoError(t, te.server.db.Create([]model.Repo{
		{
			Name: "repo0",
		},
		{
			Name: "repo1",
		},
	}).Error)
	require.NoError(t, te.server.db.Create([]model.RepoMeta{
		{
			Name:     "repo0",
			Size:     100,
			ExitCode: 0,
		},
	}).Error)

	require.NoError(t, te.server.initRepoMetas())

	var metas []model.RepoMeta
	require.NoError(t, te.server.db.Order("name").Find(&metas).Error)
	require.Len(t, metas, 2)
	require.EqualValues(t, -1, metas[0].Size)
	require.EqualValues(t, 0, metas[0].ExitCode)

	require.EqualValues(t, -1, metas[1].Size)
	require.EqualValues(t, -1, metas[1].ExitCode)
}

type fakeImageClient struct {
	docker.Client
	pullImage func(ctx context.Context, image string) error
}

func (f *fakeImageClient) PullImage(ctx context.Context, image string) error {
	return f.pullImage(ctx, image)
}

func TestUpgradeImages(t *testing.T) {
	te := NewTestEnv(t)
	var (
		mu           sync.Mutex
		pulledImages []string
	)
	dockerCli := &fakeImageClient{
		Client: te.server.dockerCli,
		pullImage: func(ctx context.Context, image string) error {
			mu.Lock()
			defer mu.Unlock()
			pulledImages = append(pulledImages, image)
			return nil
		},
	}
	te.server.dockerCli = dockerCli

	require.NoError(t, te.server.db.Create([]model.Repo{
		{
			Name:  "repo0",
			Image: "image0",
		},
		{
			Name:  "repo1",
			Image: "image1",
		},
		{
			Name:  "repo2",
			Image: "image0",
		},
	}).Error)
	te.server.upgradeImages()

	require.Len(t, pulledImages, 2)
	require.Contains(t, pulledImages, "image0")
	require.Contains(t, pulledImages, "image1")
}

func TestWaitRunningContainers(t *testing.T) {
	te := NewTestEnv(t)
	_, err := te.server.dockerCli.RunContainer(
		context.TODO(),
		&container.Config{
			Labels: map[string]string{
				api.LabelRepoName:   "repo0",
				api.LabelStorageDir: "/data",
			},
		},
		&container.HostConfig{},
		&network.NetworkingConfig{},
		"sync-repo0",
	)
	require.NoError(t, err)
	require.NoError(t, te.server.waitRunningContainers())

	_, ok := te.server.syncingContainers.Load("repo0")
	require.True(t, ok)

	testutils.PollUntilTimeout(t, time.Minute, func() bool {
		_, exist := te.server.syncingContainers.Load("repo0")
		return !exist
	})
}
