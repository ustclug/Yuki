package core

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ustclug/Yuki/pkg/api"
)

func TestAddRepository(t *testing.T) {
	t.Parallel()
	name := "test-add-repo"
	as := assert.New(t)
	cycle := 10
	err := c.AddRepository(api.Repository{
		Name:        name,
		Interval:    "1 * * * *",
		Image:       "ustcmirror/hackage:latest",
		StorageDir:  "/tmp/hackage",
		LogRotCycle: &cycle,
	})
	as.Nil(err)
	repo, err := c.GetRepository(name)
	as.Nil(err)
	as.Equal("1 * * * *", repo.Interval)
}

func TestUpdateRepository(t *testing.T) {
}

func TestRemoveRepository(t *testing.T) {
	t.Parallel()
	name := "test-remove-repo"
	as := assert.New(t)
	cycle := 10
	err := c.AddRepository(api.Repository{
		Name:        name,
		Interval:    "1 * * * *",
		Image:       "ustcmirror/hackage:latest",
		StorageDir:  "/tmp/hackage",
		LogRotCycle: &cycle,
	})
	as.Nil(err)

	_, err = c.GetRepository(name)
	as.Nil(err)

	err = c.RemoveRepository(name)
	as.Nil(err)

	_, err = c.GetRepository(name)
	as.NotNil(err)
}
