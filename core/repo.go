package core

import (
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

// M is an alias for a map[string]string map.
type M map[string]string

// Repository contains a list of syncing options.
type Repository struct {
	Name        string `bson:"_id" json:"name" validate:"-"`
	Interval    string `bson:"interval" json:"interval" validate:"required,cron"`
	Image       string `bson:"image" json:"image" validate:"required,containsrune=:"`
	StorageDir  string `bson:"storageDir" json:"storageDir" validate:"required"`
	LogRotCycle int    `bson:"logRotCycle,omitempty" json:"logRotCycle" validate:"min=0"`
	Envs        M      `bson:"envs,omitempty" json:"envs" validate:"omitempty,dive,keys,required,endkeys,required"`
	Volumes     M      `bson:"volumes,omitempty" json:"volumes" validate:"omitempty,dive,keys,required,endkeys,required"`
	User        string `bson:"user,omitempty" json:"user" validate:"-"`
	BindIP      string `bson:"bindIP,omitempty" json:"bindIP" validate:"omitempty,ip"`
	Retry       int    `bson:"retry,omitempty" json:"retry" validate:"min=0"`
	CreatedAt   int64  `bson:"createdAt,omitempty" json:"createdAt" validate:"-"`
	UpdatedAt   int64  `bson:"updatedAt,omitempty" json:"updatedAt" validate:"-"`
}

// GetRepository returns the Repository with the given name.
func (c *Core) GetRepository(name string) (*Repository, error) {
	sess := c.MgoSess.Copy()
	defer sess.Close()
	r := new(Repository)
	if err := c.repoColl.With(sess).FindId(name).One(r); err != nil {
		return nil, err
	}
	return r, nil
}

// AddRepository creates one or more Repositories.
func (c *Core) AddRepository(repos ...*Repository) error {
	sess := c.MgoSess.Copy()
	defer sess.Close()
	now := time.Now().Unix()
	rs := make([]interface{}, len(repos))
	for i, r := range repos {
		r.CreatedAt = now
		r.UpdatedAt = now
		rs[i] = r
	}
	return c.repoColl.With(sess).Insert(rs...)
}

// UpdateRepository updates the syncing options of the given Repository.
func (c *Core) UpdateRepository(name string, update bson.M) error {
	var set bson.M
	sess := c.MgoSess.Copy()
	defer sess.Close()
	switch v := update["$set"].(type) {
	case map[string]interface{}:
		set = bson.M(v)
	case bson.M:
		set = v
	default:
		set = bson.M{}
	}
	set["updatedAt"] = time.Now().Unix()
	return c.repoColl.With(sess).UpdateId(name, update)
}

// RemoveRepository removes the Repository with given name.
func (c *Core) RemoveRepository(name string) error {
	sess := c.MgoSess.Copy()
	defer sess.Close()
	return c.repoColl.With(sess).RemoveId(name)
}

// ListAllRepositories returns all Repositories.
func (c *Core) ListAllRepositories() []Repository {
	sess := c.MgoSess.Copy()
	defer sess.Close()
	result := []Repository{}
	c.repoColl.With(sess).Find(nil).Sort("_id").All(&result)
	return result
}

// FindRepository simply re-export the mgo API.
func (c *Core) FindRepository(query interface{}) *mgo.Query {
	return c.repoColl.Find(query)
}
