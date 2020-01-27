package core

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSync(t *testing.T) {
	t.Parallel()
	name := "yuki-test-sync-repo"
	as := assert.New(t)
	d := "/tmp/" + name
	os.MkdirAll(d, os.ModePerm)
	defer os.RemoveAll(d)
	c.AddRepository(&Repository{
		Name:        name,
		Interval:    "1 * * * *",
		Image:       "ustcmirror/test:latest",
		StorageDir:  d,
		LogRotCycle: 10,
	})
	prefix := "syncing-"
	ct, err := c.Sync(SyncOptions{
		LogDir:     logDir,
		MountDir:   false,
		Name:       name,
		NamePrefix: prefix,
	})
	as.Nil(err)
	code, err := c.Docker.WaitContainer(ct.ID)
	defer c.RemoveContainer(ct.ID)
	as.Nil(err)
	as.Equal(0, code)
}
