package core

import (
	"testing"
	"gopkg.in/mgo.v2/bson"
	"github.com/stretchr/testify/assert"
)

func TestGetMeta(t *testing.T) {
	t.Parallel()
	name := "test-get-meta"
	as := assert.New(t)
	C.AddRepository(&Repository{
		Name: name,
		Interval: "1 * * * *",
		Image: "ustcmirror/hackage:latest",
		StorageDir: "/tmp/hackage",
		LogRotCycle: 10,
	})
	err := C.AddMeta(&Meta{
		Name: name,
		Size: -1,
		LastExitCode: -1,
	})
	as.Nil(err)

	m, err := C.GetMeta(name)
	as.Nil(err)
	as.Equal("https://hackage.haskell.org/", m.Upstream)

	err = C.UpdateRepository(name, bson.M{"image": "ustcmirror/homebrew-bottles:latest"})
	as.Nil(err)

	m, err = C.GetMeta(name)
	as.Nil(err)
	as.Equal("http://homebrew.bintray.com/" , m.Upstream)
}

func TestUpdateMeta(t *testing.T) {
	t.Parallel()
	name := "test-update-meta"
	as := assert.New(t)
	err := C.AddMeta(&Meta{
		Name: name,
		Size: -1,
		LastExitCode: -1,
	})
	as.Nil(err)

	err = C.UpdateMeta(name, bson.M{"size": 1024})
	as.Nil(err)

	m, err := C.GetMeta(name)
	as.Nil(err)

	as.Equal(1024, m.Size)
}

func TestRemoveMeta(t *testing.T) {
	t.Parallel()
	name := "test-remove-meta"
	as := assert.New(t)
	err := C.AddMeta(&Meta{
		Name: name,
		Size: -1,
		LastExitCode: -1,
	})
	as.Nil(err)

	err = C.RemoveMeta(name)
	as.Nil(err)

	_, err = C.GetMeta(name)
	as.NotNil(err)
}
