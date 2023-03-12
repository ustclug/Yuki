package core

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"

	"github.com/ustclug/Yuki/pkg/api"
)

var (
	extraSlashes = regexp.MustCompile("([^:])//+")
)

func getUpstream(t string, envs api.M) (upstream string) {
	var ok bool
	if upstream, ok := envs["$UPSTREAM"]; ok {
		return upstream
	}
	switch t {
	case "archvsync", "rsync":
		return fmt.Sprintf("rsync://%s/%s/", envs["RSYNC_HOST"], envs["RSYNC_PATH"])
	case "aptsync", "apt-sync":
		return envs["APTSYNC_URL"]
	case "crates-io-index":
		return "https://github.com/rust-lang/crates.io-index"
	case "debian-cd":
		return fmt.Sprintf("rsync://%s/%s/", envs["RSYNC_HOST"], envs["RSYNC_MODULE"])
	case "docker-ce":
		return "https://download.docker.com/"
	case "fedora":
		remote, ok := envs["REMOTE"]
		if !ok {
			remote = "rsync://dl.fedoraproject.org"
		}
		return fmt.Sprintf("%s/%s", remote, envs["MODULE"])
	case "freebsd-pkg":
		if upstream, ok = envs["FBSD_PKG_UPSTREAM"]; !ok {
			return "http://pkg.freebsd.org/"
		}
	case "freebsd-ports":
		if upstream, ok = envs["FBSD_PORTS_DISTFILES_UPSTREAM"]; !ok {
			return "http://distcache.freebsd.org/ports-distfiles/"
		}
	case "ghcup":
		return "https://www.haskell.org/ghcup/"
	case "github-release":
		return "https://github.com"
	case "gitsync":
		return envs["GITSYNC_URL"]
	case "google-repo":
		if upstream, ok = envs["UPSTREAM"]; !ok {
			return "https://android.googlesource.com/mirror/manifest"
		}
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
	case "julia-storage":
		return "https://us-east.storage.juliahub.com, https://kr.storage.juliahub.com"
	case "nix-channels":
		if upstream, ok = envs["NIX_MIRROR_UPSTREAM"]; !ok {
			return "https://nixos.org/channels"
		}
	case "lftpsync":
		return fmt.Sprintf("%s/%s", envs["LFTPSYNC_HOST"], envs["LFTPSYNC_PATH"])
	case "nodesource":
		return "https://nodesource.com/"
	case "pypi":
		return "https://pypi.python.org/"
	case "rclone":
		remoteType := envs["RCLONE_CONFIG_REMOTE_TYPE"]
		path := envs["RCLONE_PATH"]
		domain := ""
		if remoteType == "swift" {
			domain = envs["RCLONE_SWIFT_STORAGE_URL"] + "/"
		} else if remoteType == "http" {
			domain = envs["RCLONE_CONFIG_REMOTE_URL"]
		} else if remoteType == "s3" {
			domain = envs["RCLONE_CONFIG_REMOTE_ENDPOINT"]
		}
		return fmt.Sprintf("%s%s", domain, path)
	case "rubygems":
		if upstream, ok = envs["UPSTREAM"]; !ok {
			return "http://rubygems.org/"
		}
	case "stackage":
		upstream = "https://github.com/commercialhaskell/"
	case "winget-source":
		if upstream, ok = envs["WINGET_REPO_URL"]; !ok {
			return "https://cdn.winget.microsoft.com/cache"
		}
	case "yum-sync":
		return envs["YUMSYNC_URL"]
	}
	return
}

func (c *Core) transformMeta(m *api.Meta) {
	if r, err := c.GetRepository(m.Name); err == nil {
		image := strings.Split(r.Image, ":")[0]
		t := strings.Split(image, "/")
		// remove extra slashes
		m.Upstream = extraSlashes.ReplaceAllString(getUpstream(t[len(t)-1], r.Envs), "${1}/")
	}
	if m.Upstream == "" {
		m.Upstream = "unknown"
	}
}

// GetMeta returns the metadata of the given Repository.
func (c *Core) GetMeta(name string) (*api.Meta, error) {
	m := new(api.Meta)
	sess := c.mgoSess.Copy()
	defer sess.Close()
	if err := c.metaColl.With(sess).FindId(name).One(m); err != nil {
		return nil, err
	}
	c.transformMeta(m)
	return m, nil
}

// AddMeta inserts one or more Metas.
func (c *Core) AddMeta(ms ...*api.Meta) error {
	now := time.Now().Unix()
	docs := make([]interface{}, 0, len(ms))
	for _, m := range ms {
		m.CreatedAt = now
		docs = append(docs, *m)
	}
	sess := c.mgoSess.Copy()
	defer sess.Close()
	return c.metaColl.With(sess).Insert(docs...)
}

// InitMetas creates metadata for each Repository.
func (c *Core) InitMetas() {
	repos := c.ListAllRepositories()
	now := time.Now().Unix()
	for _, r := range repos {
		size := c.GetSize(r.StorageDir)
		_, _ = c.metaColl.UpsertId(r.Name, bson.M{
			"$set": bson.M{
				"size": size,
			},
			"$setOnInsert": bson.M{
				"createdAt": now,
				"exitCode":  -1,
			},
		})
	}
}

func (c *Core) UpdatePrevRun(name string) error {
	sess := c.mgoSess.Copy()
	defer sess.Close()
	err := c.metaColl.With(sess).UpdateId(name, bson.M{
		"$set": bson.M{
			"prevRun": time.Now().Unix(),
		},
	})
	return err
}

// UpsertRepoMeta updates the metadata of the given Repository.
func (c *Core) UpsertRepoMeta(name, dir string, code int) error {
	now := time.Now().Unix()
	set := bson.M{
		"updatedAt": now,
		"size":      c.GetSize(dir),
	}
	if code != -1 {
		set["exitCode"] = code
	}
	if code == 0 {
		set["lastSuccess"] = now
	}
	sess := c.mgoSess.Copy()
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
	sess := c.mgoSess.Copy()
	defer sess.Close()
	return c.metaColl.With(sess).RemoveId(name)
}

// ListAllMetas returns the list of metadata of all Repositories.
func (c *Core) ListAllMetas() []api.Meta {
	sess := c.mgoSess.Copy()
	defer sess.Close()
	var (
		result []api.Meta
		m      api.Meta
	)
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
