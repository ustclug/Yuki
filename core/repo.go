package core

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

// M is an alias for a map[string]string map.
type M map[string]string

// Repository contains a list of syncing options.
type Repository struct {
	Name        string `bson:"_id,omitempty" json:"name,omitempty" validate:"-"`
	Interval    string `bson:"interval,omitempty" json:"interval,omitempty" validate:"required,cron"`
	Image       string `bson:"image,omitempty" json:"image,omitempty" validate:"required,containsrune=:"`
	StorageDir  string `bson:"storageDir,omitempty" json:"storageDir,omitempty" validate:"required"`
	LogRotCycle int    `bson:"logRotCycle,omitempty" json:"logRotCycle,omitempty" validate:"omitempty,min=0,max=30"`
	Envs        M      `bson:"envs,omitempty" json:"envs,omitempty" validate:"omitempty,dive,keys,required,endkeys,required"`
	Volumes     M      `bson:"volumes,omitempty" json:"volumes,omitempty" validate:"omitempty,dive,keys,required,endkeys,required"`
	User        string `bson:"user,omitempty" json:"user,omitempty" validate:"-"`
	BindIP      string `bson:"bindIP,omitempty" json:"bindIP,omitempty" validate:"omitempty,ip"`
	Retry       int    `bson:"retry,omitempty" json:"retry,omitempty" validate:"omitempty,min=1,max=3"`
	CreatedAt   int64  `bson:"createdAt,omitempty" json:"createdAt,omitempty" validate:"-"`
	UpdatedAt   int64  `bson:"updatedAt,omitempty" json:"updatedAt,omitempty" validate:"-"`
}

// GetRepository returns the Repository with the given name.
func (c *Core) GetRepository(name string) (*Repository, error) {
	r := new(Repository)
	if err := c.repoColl.FindId(name).One(r); err != nil {
		return nil, err
	}
	return r, nil
}

// AddRepository creates one or more Repositories.
func (c *Core) AddRepository(repos ...*Repository) error {
	now := time.Now().Unix()
	rs := make([]interface{}, len(repos))
	for i, r := range repos {
		r.CreatedAt = now
		r.UpdatedAt = now
		rs[i] = r
	}
	return c.repoColl.Insert(rs...)
}

// UpdateRepository updates the syncing options of the given Repository.
func (c *Core) UpdateRepository(name string, update bson.M) error {
	var set bson.M
	switch v := update["$set"].(type) {
	case map[string]interface{}:
		set = bson.M(v)
	case bson.M:
		set = v
	default:
		set = bson.M{}
	}
	set["updatedAt"] = time.Now().Unix()
	return c.repoColl.UpdateId(name, update)
}

// RemoveRepository removes the Repository with given name.
func (c *Core) RemoveRepository(name string) error {
	return c.repoColl.RemoveId(name)
}

// ListRepositories returns all Repositories.
func (c *Core) ListRepositories(query, proj bson.M) []Repository {
	result := []Repository{}
	c.repoColl.Find(query).Select(proj).Sort("_id").All(&result)
	return result
}
