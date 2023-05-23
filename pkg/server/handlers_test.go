package server

import (
	"context"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestRepoLoad(t *testing.T) {
	req := require.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	viper.Reset()
	viper.SetConfigFile("../../test/fixtures/sync_timeout.toml")

	prefix := "syncing-"
	name := "yuki-test-sync-repo"
	d := "/tmp/" + name
	server, err := getTestingServer(ctx, prefix, name, d)
	req.Nil(err)

	repo, err := server.loadRepo([]string{"../../test/fixtures/repo/"}, "test.yaml")
	req.Nil(err)

	req.Equal("test", repo.Name)
	req.Equal("cernet", repo.Network)
}
