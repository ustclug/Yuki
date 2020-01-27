package core

import (
	"testing"

	"github.com/globalsign/mgo/bson"
	"github.com/stretchr/testify/assert"
)

func TestAddRepository(t *testing.T) {
	t.Parallel()
	name := "test-add-repo"
	as := assert.New(t)
	c.AddRepository(&Repository{
		Name:        name,
		Interval:    "1 * * * *",
		Image:       "ustcmirror/hackage:latest",
		StorageDir:  "/tmp/hackage",
		LogRotCycle: 10,
	})
	repo, err := c.GetRepository(name)
	as.Nil(err)
	as.Equal("1 * * * *", repo.Interval)
}

func TestUpdateRepository(t *testing.T) {
	t.Parallel()
	name := "test-update-repo"
	as := assert.New(t)
	err := c.AddRepository(&Repository{
		Name:        name,
		Interval:    "1 * * * *",
		Image:       "ustcmirror/hackage:latest",
		StorageDir:  "/tmp/hackage",
		LogRotCycle: 10,
	})
	as.Nil(err)

	err = c.UpdateRepository(name, bson.M{
		"$set": bson.M{
			"logRotCycle": 20,
		},
	})
	as.Nil(err)

	repo, err := c.GetRepository(name)
	as.Nil(err)
	as.Equal(20, repo.LogRotCycle)
}

func TestRemoveRepository(t *testing.T) {
	t.Parallel()
	name := "test-remove-repo"
	as := assert.New(t)
	err := c.AddRepository(&Repository{
		Name:        name,
		Interval:    "1 * * * *",
		Image:       "ustcmirror/hackage:latest",
		StorageDir:  "/tmp/hackage",
		LogRotCycle: 10,
	})
	as.Nil(err)

	_, err = c.GetRepository(name)
	as.Nil(err)

	err = c.RemoveRepository(name)
	as.Nil(err)

	_, err = c.GetRepository(name)
	as.NotNil(err)
}
