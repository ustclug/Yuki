package core

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

type M map[string]string

type Repository struct {
	Name        string    `bson:"_id,omitempty" json:"name,omitempty"`
	Interval    string    `bson:"interval,omitempty" json:"interval,omitempty"`
	Image       string    `bson:"image,omitempty" json:"image,omitempty"`
	StorageDir  string    `bson:"storageDir,omitempty" json:"storageDir,omitempty"`
	LogRotCycle uint      `bson:"logRotCycle,omitempty" json:"logRotCycle,omitempty"`
	Envs        M         `bson:"envs,omitempty" json:"envs,omitempty"`
	Volumes     M         `bson:"volumes,omitempty" json:"volumes,omitempty"`
	User        string    `bson:"user,omitempty" json:"user,omitempty"`
	BindIp      string    `bson:"bindIp,omitempty" json:"bindIp,omitempty"`
	Retry       int       `bson:"retry,omitempty" json:"retry,omitempty"`
	CreatedAt   time.Time `bson:"createdAt,omitempty" json:"createdAt,omitempty"`
	UpdatedAt   time.Time `bson:"updatedAt,omitempty" json:"updatedAt,omitempty"`
}

func (y *Core) GetRepository(name string) (*Repository, error) {
	r := new(Repository)
	if err := y.repoColl.FindId(name).One(r); err != nil {
		return nil, err
	}
	return r, nil
}

func (y *Core) AddRepository(repo *Repository) error {
	repo.CreatedAt = time.Now()
	repo.UpdatedAt = time.Now()
	return y.repoColl.Insert(*repo)
}

func (y *Core) UpdateRepository(name string, update bson.M) error {
	return y.repoColl.UpdateId(name, bson.M{
		"$set":         update,
		"$currentDate": bson.M{"updatedAt": true},
	})
}

func (y *Core) RemoveRepository(name string) error {
	return y.repoColl.RemoveId(name)
}

func (y *Core) ListRepositories(query, proj bson.M) []Repository {
	result := []Repository{}
	y.repoColl.Find(query).Select(proj).Sort("_id").All(&result)
	return result
}
