package core

import (
	"context"
	"fmt"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/globalsign/mgo"
	"github.com/ustclug/Yuki/fs"
)

// Config contains a list of options used when creating `Core`.
type Config struct {
	Debug bool
	// DbURL contains username and password
	DbURL          string
	DbName         string
	GetSizer       fs.GetSizer
	SessionAge     time.Duration
	DockerEndpoint string
}

// Core is the basic type of this package. It provides methods for interaction with MongoDB and Docker API.
type Core struct {
	Docker  *docker.Client
	MgoSess *mgo.Session

	ctx      context.Context
	getSizer fs.GetSizer
	repoColl *mgo.Collection
	metaColl *mgo.Collection
	sessColl *mgo.Collection
}

const ttlIdxName = "sessions-TTL-index-on-lastAccess"

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

		ctx:      context.Background(),
		getSizer: cfg.GetSizer,
	}

	c.repoColl = db.C("repositories")
	c.metaColl = db.C("metas")
	c.sessColl = db.C("sessions")

	if err = ensureTTLIndex(c.sessColl, cfg.SessionAge); err != nil {
		if err = c.sessColl.DropIndexName(ttlIdxName); err != nil {
			return nil, err
		}
		if err = ensureTTLIndex(c.sessColl, cfg.SessionAge); err != nil {
			return nil, err
		}
	}

	return &c, nil
}

func ensureTTLIndex(coll *mgo.Collection, age time.Duration) error {
	return coll.EnsureIndex(mgo.Index{
		Key:         []string{"lastAccess"},
		Name:        ttlIdxName,
		ExpireAfter: age,
	})
}
