package core

import (
	docker "github.com/fsouza/go-dockerclient"
	"gopkg.in/mgo.v2"
)

type Config struct {
	Debug bool `json:"debug,omitempty"`
	// DbURL contains username and password
	DbURL          string `json:"db_url,omitempty"`
	DbName         string `json:"db_name,omitempty"`
	DockerEndpoint string `json:"docker_endpoint,omitempty"`
}

type Core struct {
	DB     *mgo.Database
	Docker *docker.Client

	repoColl *mgo.Collection
	metaColl *mgo.Collection
	userColl *mgo.Collection
}

func NewWithConfig(cfg Config) (*Core, error) {
	d, err := docker.NewClient(cfg.DockerEndpoint)
	if err != nil {
		return nil, err
	}

	mgo.SetDebug(cfg.Debug)
	sess, err := mgo.Dial(cfg.DbURL)
	if err != nil {
		return nil, err
	}
	if err = sess.Ping(); err != nil {
		return nil, err
	}
	sess.SetMode(mgo.Monotonic, true)

	c := Core{
		DB:     sess.DB(cfg.DbName),
		Docker: d,
	}
	c.repoColl = c.DB.C("repositories")
	c.metaColl = c.DB.C("metas")
	c.userColl = c.DB.C("users")
	return &c, nil
}
