package server

import (
	"context"
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/core"
)

func getTestingServer(ctx context.Context, prefix string, name string, storageDir string) (*Server, error) {
	s, err := New()
	if err != nil {
		return nil, err
	}

	defer os.RemoveAll("/tmp/log_yuki")
	defer os.RemoveAll("/tmp/config_yuki")

	err = os.MkdirAll(storageDir, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(storageDir)
	// remove existing repository and container
	_ = s.c.RemoveRepository(name)
	_ = s.c.RemoveContainer(ctx, prefix+name)

	s.ctx = ctx
	go s.onPreSync()
	go s.onPostSync(context.TODO())

	return s, nil
}

func TestWaitForSync(t *testing.T) {
	as := assert.New(t)
	req := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	viper.Reset()
	viper.SetConfigFile("../../test/fixtures/sync_timeout.toml")

	prefix := "syncing-"
	name := "yuki-wait-test-sync-repo"
	d := "/tmp/" + name
	cycle := 10

	s, err := getTestingServer(ctx, prefix, name, d)
	req.Nil(err)

	err = s.c.AddRepository(api.Repository{
		Name:        name,
		Interval:    "1 * * * *",
		Image:       "ustcmirror/test:latest",
		StorageDir:  d,
		LogRotCycle: &cycle,
		Envs:        map[string]string{"SLEEP_INFINITY": "1"},
	})
	as.Nil(err)

	ct, err := s.c.Sync(ctx, core.SyncOptions{
		MountDir:   false,
		Name:       name,
		NamePrefix: prefix,
	})
	as.Nil(err)
	defer s.c.RemoveContainer(ctx, ct.ID)
	err = s.waitForSync(ct)
	if err != context.DeadlineExceeded && err.Error() != "" {
		t.Fatalf("Expected error to be context.DeadlineExceeded, got %v", err)
	}
}
