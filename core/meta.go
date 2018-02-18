package core

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

var (
	re = regexp.MustCompile("([^:])//+")
)

// Meta represents the metadata of a Repository.
type Meta struct {
	Name        string `bson:"_id" json:"name"`
	Upstream    string `bson:"-" json:"upstream"`
	Size        int    `bson:"size" json:"size"`
	ExitCode    int    `bson:"exitCode" json:"exitCode"`
	LastSuccess int64  `bson:"lastSuccess,omitempty" json:"lastSuccess"`
	CreatedAt   int64  `bson:"createdAt,omitempty" json:"createdAt"`
	UpdatedAt   int64  `bson:"updatedAt,omitempty" json:"updatedAt"`
}

func getUpstream(t string, envs M) (upstream string) {
	var ok bool
	if upstream, ok := envs["$UPSTREAM"]; ok {
		return upstream
	}
	switch t {
	case "archvsync":
		fallthrough
	case "rsync":
		return fmt.Sprintf("rsync://%s/%s/", envs["RSYNC_HOST"], envs["RSYNC_PATH"])
	case "aptsync":
		return envs["APTSYNC_URL"]
	case "debian-cd":
		return fmt.Sprintf("rsync://%s/%s/", envs["RSYNC_HOST"], envs["RSYNC_MODULE"])
	case "freebsd-pkg":
		if upstream, ok = envs["FBSD_PKG_UPSTREAM"]; !ok {
			return "http://pkg.freebsd.org/"
		}
	case "freebsd-ports":
		if upstream, ok = envs["FBSD_PORTS_DISTFILES_UPSTREAM"]; !ok {
			return "http://distcache.freebsd.org/ports-distfiles/"
		}
	case "gitsync":
		return envs["GITSYNC_URL"]
	case "gsutil-rsync":
		return envs["GS_URL"]
	case "hackage":
		if upstream, ok = envs["HACKAGE_BASE_URL"]; !ok {
			return "https://hackage.haskell.org/"
		}
	case "homebrew-bottles":
		if upstream, ok = envs["HOMEBREW_BOTTLE_DOMAIN"]; !ok {
			return "http://homebrew.bintray.com/"
		}
	case "lftpsync":
		return fmt.Sprintf("%s/%s", envs["LFTPSYNC_HOST"], envs["LFTPSYNC_PATH"])
	case "nodesource":
		return "https://nodesource.com/"
	case "pypi":
		return "https://pypi.python.org/"
	case "rubygems":
		if upstream, ok = envs["UPSTREAM"]; !ok {
			return "http://rubygems.org/"
		}
	case "stackage":
		upstream = "https://github.com/commercialhaskell/"
	}
	return
}

func (c *Core) transformMeta(m *Meta) {
	if r, err := c.GetRepository(m.Name); err == nil {
		image := strings.Split(r.Image, ":")[0]
		t := strings.Split(image, "/")
		// remove extra slashes
		m.Upstream = re.ReplaceAllString(getUpstream(t[len(t)-1], r.Envs), "${1}/")
	}
	if m.Upstream == "" {
		m.Upstream = "unknown"
	}
}

// GetMeta returns the metadata of the given Repository.
func (c *Core) GetMeta(name string) (*Meta, error) {
	m := new(Meta)
	sess := c.MgoSess.Copy()
	defer sess.Close()
	if err := c.metaColl.With(sess).FindId(name).One(m); err != nil {
		return nil, err
	}
	c.transformMeta(m)
	return m, nil
}

// AddMeta inserts one or more Metas.
func (c *Core) AddMeta(ms ...*Meta) error {
	now := time.Now().Unix()
	docs := make([]interface{}, len(ms))
	for i, m := range ms {
		m.CreatedAt = now
		docs[i] = m
	}
	sess := c.MgoSess.Copy()
	defer sess.Close()
	return c.metaColl.With(sess).Insert(docs...)
}

// InitMetas creates metadata for each Repository.
func (c *Core) InitMetas() {
	repos := c.ListAllRepositories()
	now := time.Now().Unix()
	for _, r := range repos {
		go func(id, dir string) {
			size := c.getSizer.GetSize(dir)
			c.metaColl.UpsertId(id, bson.M{
				"$set": bson.M{
					"size": size,
				},
				"$setOnInsert": bson.M{
					"createdAt": now,
					"exitCode":  -1,
				},
			})
		}(r.Name, r.StorageDir)
	}
}

// UpsertRepoMeta updates the metadata of the given Repository.
func (c *Core) UpsertRepoMeta(name, dir string, code int) error {
	now := time.Now().Unix()
	set := bson.M{
		"exitCode":  code,
		"updatedAt": now,
		"size":      c.getSizer.GetSize(dir),
	}
	if code == 0 {
		set["lastSuccess"] = now
	}
	sess := c.MgoSess.Copy()
	defer sess.Close()
	_, err := c.metaColl.With(sess).UpsertId(name, bson.M{
		"$set": set,
		"$setOnInsert": bson.M{
			"createdAt": now,
		},
	})
	return err
}

// RemoveMeta removes the metadata of the given Repository.
func (c *Core) RemoveMeta(name string) error {
	sess := c.MgoSess.Copy()
	defer sess.Close()
	return c.metaColl.With(sess).RemoveId(name)
}

// ListAllMetas returns the list of metadata of all Repositories.
func (c *Core) ListAllMetas() []Meta {
	sess := c.MgoSess.Copy()
	defer sess.Close()
	result := []Meta{}
	m := Meta{}
	iter := c.metaColl.With(sess).Find(nil).Sort("_id").Iter()
	defer iter.Close()
	for iter.Next(&m) {
		c.transformMeta(&m)
		result = append(result, m)
	}
	return result
}

// FindMeta simply re-export the mgo API.
func (c *Core) FindMeta(query interface{}) *mgo.Query {
	return c.metaColl.Find(query)
}
