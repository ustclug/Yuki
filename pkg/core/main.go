package core

import (
	"fmt"

	"github.com/docker/docker/client"
	"github.com/globalsign/mgo"

	"github.com/ustclug/Yuki/pkg/fs"
)

// Config contains a list of options used when creating `Core`.
type Config struct {
	Debug bool
	// DbURL contains username and password
	DbURL          string
	DbName         string
	GetSizer       fs.GetSizer
	DockerEndpoint string
}

// Core is the basic type of this package. It provides methods for interaction with MongoDB and Docker API.
type Core struct {
	mgoSess  *mgo.Session
	repoColl *mgo.Collection
	metaColl *mgo.Collection
	docker   *client.Client
	getSizer fs.GetSizer
}

// NewWithConfig returns a `Core` instance with specified config.
func NewWithConfig(cfg Config) (*Core, error) {
	d, err := client.NewClientWithOpts(
		client.WithHost(cfg.DockerEndpoint),
		client.WithAPIVersionNegotiation(),
	)
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
		docker:   d,
		mgoSess:  sess,
		getSizer: cfg.GetSizer,
	}

	c.repoColl = db.C("repositories")
	c.metaColl = db.C("metas")

	return &c, nil
}

func (c *Core) GetSize(dir string) int64 {
	return c.getSizer.GetSize(dir)
}

func (c *Core) PingMongoSession() error {
	return c.mgoSess.Ping()
}

func (c *Core) CloseMongoSession() {
	c.mgoSess.Close()
}
