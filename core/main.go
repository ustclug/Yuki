package core

import (
	"fmt"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/knight42/Yuki/fs"
	"gopkg.in/mgo.v2"
)

type Config struct {
	Debug bool
	// DbURL contains username and password
	DbURL          string
	DbName         string
	FileSystem     string
	DockerEndpoint string
}

type Core struct {
	DB     *mgo.Database
	Docker *docker.Client

	fs       fs.GetSizer
	repoColl *mgo.Collection
	metaColl *mgo.Collection
	userColl *mgo.Collection
}

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
	if err = sess.Ping(); err != nil {
		return nil, err
	}
	sess.SetMode(mgo.Monotonic, true)

	c := Core{
		DB:     sess.DB(cfg.DbName),
		Docker: d,
	}

	switch cfg.FileSystem {
	case "xfs":
		c.fs = fs.New(fs.XFS)
	case "zfs":
		c.fs = fs.New(fs.ZFS)
	default:
		c.fs = fs.New(fs.DEFAULT)
	}

	c.repoColl = c.DB.C("repositories")
	c.metaColl = c.DB.C("metas")
	c.userColl = c.DB.C("users")
	return &c, nil
}
