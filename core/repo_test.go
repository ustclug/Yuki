package core

import (
	"testing"
	"gopkg.in/mgo.v2/bson"
	"github.com/stretchr/testify/assert"
)

func TestAddRepository(t *testing.T) {
	t.Parallel()
	name := "test-add-repo"
	as := assert.New(t)
	C.AddRepository(&Repository{
		Name: name,
		Interval: "1 * * * *",
		Image: "ustcmirror/hackage:latest",
		StorageDir: "/tmp/hackage",
		LogRotCycle: 10,
	})
	repo, err := C.GetRepository(name)
	as.Nil(err)
	as.Equal("1 * * * *", repo.Interval)
}

func TestUpdateRepository(t *testing.T) {
	t.Parallel()
	name := "test-update-repo"
	as := assert.New(t)
	err := C.AddRepository(&Repository{
		Name: name,
		Interval: "1 * * * *",
		Image: "ustcmirror/hackage:latest",
		StorageDir: "/tmp/hackage",
		LogRotCycle: 10,
	})
	as.Nil(err)

	err = C.UpdateRepository(name, bson.M{"logRotCycle": 20})
	as.Nil(err)

	repo, err := C.GetRepository(name)
	as.Nil(err)
	as.Equal(uint(20), repo.LogRotCycle)
}

func TestRemoveRepository(t *testing.T) {
	t.Parallel()
	name := "test-remove-repo"
	as := assert.New(t)
	err := C.AddRepository(&Repository{
		Name: name,
		Interval: "1 * * * *",
		Image: "ustcmirror/hackage:latest",
		StorageDir: "/tmp/hackage",
		LogRotCycle: 10,
	})
	as.Nil(err)

	_, err = C.GetRepository(name)
	as.Nil(err)

	err = C.RemoveRepository(name)
	as.Nil(err)

	_, err = C.GetRepository(name)
	as.NotNil(err)
}
