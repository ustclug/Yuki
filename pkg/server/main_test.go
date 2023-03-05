package server

import (
	"context"
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/core"
)

func getTestingServer(prefix string, name string, storageDir string) (*Server, error) {
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
	_ = s.c.RemoveContainer(context.TODO(), prefix+name)

	s.ctx = context.TODO()
	go s.onPreSync()
	go s.onPostSync()

	return s, nil
}

func TestWaitForSync(t *testing.T) {
	as := assert.New(t)

	viper.Reset()
	viper.SetConfigFile("../../test/fixtures/sync_timeout.toml")

	prefix := "syncing-"
	name := "yuki-test-sync-repo"
	d := "/tmp/" + name
	cycle := 10

	s, err := getTestingServer(prefix, name, d)
	as.Nil(err)

	err = s.c.AddRepository(api.Repository{
		Name:        name,
		Interval:    "1 * * * *",
		Image:       "ustcmirror/test:latest",
		StorageDir:  d,
		LogRotCycle: &cycle,
		Envs:        map[string]string{"SLEEP_INFINITY": "1"},
	})
	as.Nil(err)

	ct, err := s.c.Sync(context.TODO(), core.SyncOptions{
		MountDir:   false,
		Name:       name,
		NamePrefix: prefix,
	})
	as.Nil(err)
	defer s.c.RemoveContainer(context.TODO(), ct.ID)
	err = s.waitForSync(ct)
	if err != context.DeadlineExceeded {
		t.Fatalf("Expected error to be context.DeadlineExceeded, got %v", err)
	}
}
