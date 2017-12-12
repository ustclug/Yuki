package core

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

type Repository struct {
	Name        string    `bson:"_id" json:"name"`
	Interval    string    `bson:"interval" json:"interval"`
	Image       string    `bson:"image" json:"image"`
	StorageDir  string    `bson:"storageDir" json:"storageDir"`
	LogRotCycle string    `bson:"logRotCycle" json:"logRotCycle"`
	Envs        bson.M    `bson:"envs" json:"envs"`
	Volumes     bson.M    `bson:"volumes" json:"volumes"`
	User        string    `bson:"user" json:"user"`
	BindIp      string    `bson:"bindIp" json:"bindIp"`
	Retry       int       `bson:"retry" json:"retry"`
	CreatedAt   time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time `bson:"updatedAt" json:"updatedAt"`
}

func (y *Yukid) GetRepository(name string) (*Repository, error) {
	result := Repository{}
	if err := y.repoColl.FindId(name).One(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (y *Yukid) AddRepository(repo *Repository) error {
	return y.repoColl.Insert(*repo)
}

func (y *Yukid) UpdateRepository(name string, update bson.M) error {
	return y.repoColl.UpdateId(name, bson.M{"$set": update})
}

func (y *Yukid) RemoveRepository(name string) error {
	if err := y.repoColl.RemoveId(name); err != nil {
		return err
	}
	if err := y.metaColl.RemoveId(name); err != nil {
		return err
	}
	return nil
}

func (y *Yukid) ListRepositories(query, proj bson.M) []Repository {
	result := []Repository{}
	y.repoColl.Find(query).Select(proj).Sort("_id").All(&result)
	return result
}
