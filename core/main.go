package core

import (
	"context"
	"fmt"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/globalsign/mgo"
	"github.com/knight42/Yuki/fs"
)

// Config contains a list of options used when creating `Core`.
type Config struct {
	Debug bool
	// DbURL contains username and password
	DbURL          string
	DbName         string
	FileSystem     string
	DockerEndpoint string
}

// Core is the basic type of this package. It provides methods for interaction with MongoDB and Docker API.
type Core struct {
	Docker  *docker.Client
	MgoSess *mgo.Session

	fs       fs.GetSizer
	ctx      context.Context
	repoColl *mgo.Collection
	metaColl *mgo.Collection
	userColl *mgo.Collection
}

// NewWithConfig returns a `Core` instance with specified config.
func NewWithConfig(cfg Config) (*Core, error) {
	d, err := docker.NewClient(cfg.DockerEndpoint)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to the Docker daemon at %s", cfg.DockerEndpoint)
	}

	mgo.SetDebug(cfg.Debug)
	sess, err := mgo.Dial(cfg.DbURL)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to the MongoDB at %s", cfg.DbURL)
	}
	sess.SetMode(mgo.Monotonic, true)

	db := sess.DB(cfg.DbName)
	c := Core{
		Docker:  d,
		MgoSess: sess,

		ctx: context.Background(),
	}

	switch cfg.FileSystem {
	case "xfs":
		c.fs = fs.New(fs.XFS)
	case "zfs":
		c.fs = fs.New(fs.ZFS)
	default:
		c.fs = fs.New(fs.DEFAULT)
	}

	c.repoColl = db.C("repositories")
	c.metaColl = db.C("metas")
	c.userColl = db.C("users")
	return &c, nil
}
