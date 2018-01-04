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
	C.AddRepository(&Repository{
		Name: name,
		Interval: "1 * * * *",
		Image: "ustcmirror/test:latest",
		StorageDir: d,
		LogRotCycle: 10,
	})
	prefix := "syncing-"
	ct, err := C.Sync(SyncOptions{
		LogDir: LogDir,
		MountDir: false,
		Name: name,
		NamePrefix: prefix,
	})
	as.Nil(err)
	code, err := C.Docker.WaitContainer(ct.ID)
	defer C.RemoveContainer(ct.ID)
	as.Nil(err)
	as.Equal(0, code)
}
