package core

import (
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"

	"github.com/ustclug/Yuki/pkg/api"
)

// GetRepository returns the Repository with the given name.
func (c *Core) GetRepository(name string) (*api.Repository, error) {
	sess := c.MgoSess.Copy()
	defer sess.Close()
	r := new(api.Repository)
	if err := c.repoColl.With(sess).FindId(name).One(r); err != nil {
		return nil, err
	}
	return r, nil
}

// AddRepository creates one or more Repositories.
func (c *Core) AddRepository(repos ...api.Repository) error {
	sess := c.MgoSess.Copy()
	defer sess.Close()
	now := time.Now().Unix()
	rs := make([]interface{}, 0, len(repos))
	for _, r := range repos {
		if r.LogRotCycle == nil {
			cycle := 10
			r.LogRotCycle = &cycle
		}
		r.CreatedAt = now
		r.UpdatedAt = now
		rs = append(rs, r)
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
func (c *Core) ListAllRepositories() []api.Repository {
	sess := c.MgoSess.Copy()
	defer sess.Close()
	var result []api.Repository
	_ = c.repoColl.With(sess).Find(nil).Sort("_id").All(&result)
	return result
}

// FindRepository simply re-export the mgo API.
func (c *Core) FindRepository(query interface{}) *mgo.Query {
	return c.repoColl.Find(query)
}
