package core

import (
	docker "github.com/fsouza/go-dockerclient"
	"gopkg.in/mgo.v2"
)

type Config struct {
	Debug bool
	// DbURL contains username and password
	DbURL          string
	DbName         string
	DockerEndpoint string
}

type Core struct {
	DB      *mgo.Database
	Docker  *docker.Client

	repoColl *mgo.Collection
	metaColl *mgo.Collection
	userColl *mgo.Collection
}

func NewWithConfig(c Config) (*Core, error) {
	mgo.SetDebug(c.Debug)
	sess, err := mgo.Dial(c.DbURL)
	if err != nil {
		return nil, err
	}
	if err = sess.Ping(); err != nil {
		return nil, err
	}
	sess.SetMode(mgo.Monotonic, true)

	d, err := docker.NewClient(c.DockerEndpoint)
	if err != nil {
		return nil, err
	}
	m := Core{
		DB:      sess.DB(c.DbName),
		Docker:  d,
	}
	m.repoColl = m.DB.C("repositories")
	m.metaColl = m.DB.C("metas")
	m.userColl = m.DB.C("users")
	return &m, nil
}
