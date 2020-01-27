package core

import (
	"testing"

	"github.com/globalsign/mgo/bson"
	"github.com/stretchr/testify/assert"
)

func TestGetMeta(t *testing.T) {
	t.Parallel()
	name := "test-get-meta"
	as := assert.New(t)
	c.AddRepository(&Repository{
		Name:        name,
		Interval:    "1 * * * *",
		Image:       "ustcmirror/hackage:latest",
		StorageDir:  "/tmp/hackage",
		LogRotCycle: 10,
	})
	err := c.AddMeta(&Meta{
		Name:     name,
		Size:     -1,
		ExitCode: -1,
	})
	as.Nil(err)

	m, err := c.GetMeta(name)
	as.Nil(err)
	as.Equal("https://hackage.haskell.org/", m.Upstream)

	err = c.UpdateRepository(name, bson.M{"image": "ustcmirror/homebrew-bottles:latest"})
	as.Nil(err)

	m, err = c.GetMeta(name)
	as.Nil(err)
	as.Equal("http://homebrew.bintray.com/", m.Upstream)
}

func TestRemoveMeta(t *testing.T) {
	t.Parallel()
	name := "test-remove-meta"
	as := assert.New(t)
	err := c.AddMeta(&Meta{
		Name:     name,
		Size:     -1,
		ExitCode: -1,
	})
	as.Nil(err)

	err = c.RemoveMeta(name)
	as.Nil(err)

	_, err = c.GetMeta(name)
	as.NotNil(err)
}
