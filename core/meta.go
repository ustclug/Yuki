package core

import (
	"fmt"
	"strings"
	"time"

	"gopkg.in/mgo.v2/bson"
)

type Meta struct {
	Name         string    `bson:"_id,omitempty" json:"name,omitempty"`
	Upstream     string    `bson:",omitempty" json:"upstream,omitempty"`
	Size         int       `bson:"size,omitempty" json:"size,omitempty"`
	LastExitCode int       `bson:"lastExitCode,omitempty" json:"lastExitCode,omitempty"`
	LastSuccess  time.Time `bson:"lastSuccess,omitempty" json:"lastSuccess,omitempty"`
	CreatedAt    time.Time `bson:"createdAt,omitempty" json:"createdAt,omitempty"`
	UpdatedAt    time.Time `bson:"updatedAt,omitempty" json:"updatedAt,omitempty"`
}

func getUpstream(t string, envs M) (upstream string) {
	if v, ok := envs["$upstream"]; ok {
		return v
	}
	var ok bool
	switch t {
	case "archvsync":
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
		upstream = ""
	}
	return
}

func (c *Core) transformMeta(m *Meta) {
	if r, err := c.GetRepository(m.Name); err == nil {
		image := strings.Split(r.Image, ":")[0]
		t := strings.Split(image, "/")
		m.Upstream = getUpstream(t[len(t) - 1], r.Envs)
	}
	if m.Upstream == "" {
		m.Upstream = "unknown"
	}
}

func (c *Core) GetMeta(name string) (*Meta, error) {
	m := new(Meta)
	if err := c.metaColl.FindId(name).One(m); err != nil {
		return nil, err
	}
	c.transformMeta(m)
	return m, nil
}

func (c *Core) AddMeta(m *Meta) error {
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()
	return c.metaColl.Insert(*m)
}

func (c *Core) UpdateMeta(name string, update bson.M) error {
	return c.metaColl.UpdateId(name, bson.M{
		"$set": update,
		"$currentDate": bson.M{"updatedAt": true},
	})
}

func (c *Core) RemoveMeta(name string) error {
	return c.metaColl.RemoveId(name)
}

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
