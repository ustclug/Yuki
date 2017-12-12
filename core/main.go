package core

import (
	docker "github.com/fsouza/go-dockerclient"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Manager interface {
	GetRepository(name string) (*Repository, error)
	AddRepository(repo *Repository) error
	UpdateRepository(name string, update bson.M) error
	RemoveRepository(name string) error
	ListRepositories(query, proj bson.M) []Repository

	//Sync(name string) error

	//CleanContainers() error
	//StartContainer(name string) error
	//StopContainer(name string) error
	//RemoveContainer() error
	//ListContainers() error

	//ImportConfig() error
	//ExportConfig() error
}

type ManagerConfig struct {
	Debug          bool
	DbURL          string
	DbName         string
	DockerEndpoint string
	NamePrefix     string
}

type Yukid struct {
	NamePrefix string
	Docker     *docker.Client
	DB         *mgo.Database
	repoColl   *mgo.Collection
	metaColl   *mgo.Collection
	userColl   *mgo.Collection
}

func NewWithConfig(c ManagerConfig) (Manager, error) {
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

	m := Yukid{
		NamePrefix: c.NamePrefix,
		DB:         sess.DB(c.DbName),
		Docker:     d,
	}
	m.repoColl = m.DB.C("repositories")
	m.metaColl = m.DB.C("metas")
	m.userColl = m.DB.C("users")
	return &m, nil
}
