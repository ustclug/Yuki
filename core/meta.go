package core

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"gopkg.in/mgo.v2/bson"
)

var (
	re = regexp.MustCompile("([^:])//+")
)

// Meta represents the metadata of a Repository.
type Meta struct {
	Name        string `bson:"_id,omitempty" json:"name,omitempty"`
	Upstream    string `bson:"-" json:"upstream,omitempty"`
	Size        int    `bson:"size,omitempty" json:"size,omitempty"`
	ExitCode    int    `bson:"exitCode,omitempty" json:"exitCode,omitempty"`
	LastSuccess int64  `bson:"lastSuccess,omitempty" json:"lastSuccess,omitempty"`
	CreatedAt   int64  `bson:"createdAt,omitempty" json:"createdAt,omitempty"`
	UpdatedAt   int64  `bson:"updatedAt,omitempty" json:"updatedAt,omitempty"`
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
	if err := c.metaColl.FindId(name).One(m); err != nil {
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
	return c.metaColl.Insert(docs...)
}

// InitMetas creates metadata for each Repository.
func (c *Core) InitMetas() {
	repos := c.ListRepositories(nil, nil)
	now := time.Now().Unix()
	for _, r := range repos {
		go func(id, dir string) {
			size := c.fs.GetSize(dir)
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
		"size":      c.fs.GetSize(dir),
	}
	if code == 0 {
		set["lastSuccess"] = now
	}
	_, err := c.metaColl.UpsertId(name, bson.M{
		"$set": set,
		"$setOnInsert": bson.M{
			"createdAt": now,
		},
	})
	return err
}

// RemoveMeta removes the metadata of the given Repository.
func (c *Core) RemoveMeta(name string) error {
	return c.metaColl.RemoveId(name)
}

// ListMetas returns the list of metadata of all Repositories.
func (c *Core) ListMetas(query, proj bson.M) []Meta {
	result := []Meta{}
	m := Meta{}
	iter := c.metaColl.Find(query).Select(proj).Sort("_id").Iter()
	defer iter.Close()
	for iter.Next(&m) {
		c.transformMeta(&m)
		result = append(result, m)
	}
	return result
}
