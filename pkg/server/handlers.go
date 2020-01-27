package server

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/core"
	"github.com/ustclug/Yuki/pkg/tail"
)

func badRequest(msg interface{}) error {
	return &echo.HTTPError{
		Code:    http.StatusBadRequest,
		Message: msg,
	}
}

func notFound(msg interface{}) error {
	return &echo.HTTPError{
		Code:    http.StatusNotFound,
		Message: msg,
	}
}

func conflict(msg interface{}) error {
	return &echo.HTTPError{
		Code:    http.StatusConflict,
		Message: msg,
	}
}

func (s *Server) registerAPIs(g *echo.Group) {
	// public APIs
	g.GET("metas", s.listMetas)
	g.GET("metas/:name", s.getMeta)

	// private APIs
	g.GET("repositories", s.listRepos)
	g.POST("repositories", s.reloadAllRepos)
	g.GET("repositories/:name", s.getRepo)
	g.POST("repositories/:name", s.reloadRepo)
	g.DELETE("repositories/:name", s.removeRepo)
	g.GET("repositories/:name/logs", s.getRepoLogs)

	g.GET("containers", s.listCts)
	g.POST("containers/:name", s.sync)
	// g.POST("containers/:name/stop", s.stopCt)
	g.DELETE("containers/:name", s.removeCt)
	g.GET("containers/:name/logs", s.getCtLogs)

	g.GET("config", s.exportConfig)
}

func (s *Server) listRepos(c echo.Context) error {
	var repos []api.RepoSummary
	_ = s.c.FindRepository(nil).Select(bson.M{
		"interval":   1,
		"image":      1,
		"storageDir": 1,
	}).Sort("_id").All(&repos)
	return c.JSON(http.StatusOK, repos)
}

func (s *Server) addRepo(c echo.Context) error {
	var repo api.Repository
	if err := c.Bind(&repo); err != nil {
		return badRequest(err)
	}
	if err := c.Validate(&repo); err != nil {
		return badRequest(err)
	}
	name := c.Param("name")
	if repo.Name == "" {
		repo.Name = name
	}
	err := s.c.AddRepository(repo)
	if err != nil {
		if mgo.IsDup(err) {
			return conflict(err)
		}
		return err
	}
	return c.NoContent(http.StatusCreated)
}

func (s *Server) getRepo(c echo.Context) error {
	name := c.Param("name")
	repo, err := s.c.GetRepository(name)
	if err != nil {
		return notFound(err)
	}
	return c.JSON(http.StatusOK, repo)
}

func (s *Server) getRepoLogs(c echo.Context) error {
	type repoLogsOptions struct {
		N     int  `query:"n"`
		Tail  int  `query:"tail"`
		Stats bool `query:"stats"`
	}
	type fileInfo struct {
		Name  string    `json:"name"`
		Size  int64     `json:"size"`
		Mtime time.Time `json:"mtime"`
	}

	opts := repoLogsOptions{}
	if err := c.Bind(&opts); err != nil {
		return badRequest(err)
	}

	logdir := path.Join(s.config.LogDir, c.Param("name"))
	if err := os.MkdirAll(logdir, os.ModePerm); err != nil {
		return fmt.Errorf("not a directory: %s", logdir)
	}

	files, err := ioutil.ReadDir(logdir)
	if err != nil {
		return err
	}

	if opts.Stats {
		var infos []fileInfo
		for _, f := range files {
			name := f.Name()
			if !strings.HasPrefix(name, "result.log.") {
				continue
			}
			infos = append(infos, fileInfo{
				Name:  name,
				Size:  f.Size(),
				Mtime: f.ModTime(),
			})
		}
		sort.Slice(infos, func(i, j int) bool {
			return infos[j].Mtime.After(infos[i].Mtime)
		})
		return c.JSON(http.StatusOK, infos)
	}

	if opts.Tail < 0 {
		opts.Tail = 0
	}

	wantedName := fmt.Sprintf("result.log.%d", opts.N)
	fileName := ""
	for _, f := range files {
		realName := f.Name()
		if realName == wantedName || (realName == wantedName+".gz") {
			// result.log.0
			// result.log.1.gz
			// result.log.2.gz
			// result.log.10.gz
			fileName = realName
			break
		}
	}

	if len(fileName) == 0 {
		return notFound(fmt.Sprintf("no such file: %s", wantedName))
	}

	content, err := os.Open(path.Join(logdir, fileName))
	if err != nil {
		return err
	}
	defer content.Close()

	var t *tail.Tail

	switch path.Ext(fileName) {
	case ".gz":
		gr, err := gzip.NewReader(content)
		if err != nil {
			return err
		}
		defer gr.Close()
		tmpfile, err := ioutil.TempFile(logdir, "extracted")
		if err != nil {
			return err
		}
		defer os.Remove(tmpfile.Name())
		defer tmpfile.Close()
		_, err = io.Copy(tmpfile, gr)
		if err != nil {
			return err
		}
		_, err = tmpfile.Seek(0, io.SeekStart)
		if err != nil {
			return err
		}
		t = tail.New(tmpfile, opts.Tail)
	default:
		t = tail.New(content, opts.Tail)
	}

	_, err = t.WriteTo(c.Response())
	return err
}

func (s *Server) removeRepo(c echo.Context) error {
	name := c.Param("name")
	if err := s.c.RemoveRepository(name); err != nil {
		return err
	}
	_ = s.c.RemoveMeta(name)
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) listCts(c echo.Context) error {
	opts := docker.ListContainersOptions{
		All: true,
		Filters: map[string][]string{
			"label": {"org.ustcmirror.syncing=true"},
		},
	}
	apiCts, err := s.c.Docker.ListContainers(opts)
	if err != nil {
		return err
	}
	cts := make([]api.ContainerDetail, 0, len(apiCts))
	for _, ct := range apiCts {
		cts = append(cts, api.ContainerDetail{
			ID:      ct.ID[:10],
			Image:   ct.Image,
			Created: ct.Created,
			State:   ct.State,
			Status:  ct.Status,
			Name:    strings.TrimLeft(ct.Names[0], "/"),
		})
	}
	return c.JSON(http.StatusOK, cts)
}

func (s *Server) sync(c echo.Context) error {
	name := c.Param("name")
	debug, err := strconv.ParseBool(c.QueryParam("debug"))
	if err != nil {
		debug = false
	}

	logrus.Infof("Syncing %s", name)
	ct, err := s.c.Sync(core.SyncOptions{
		Name:          name,
		NamePrefix:    s.config.NamePrefix,
		LogDir:        s.config.LogDir,
		DefaultOwner:  s.config.Owner,
		MountDir:      true,
		Debug:         debug,
		DefaultBindIP: s.config.BindIP,
	})
	if err != nil {
		if err == docker.ErrContainerAlreadyExists {
			return conflict(err)
		}
		return err
	}

	go s.waitForSync(ct)
	return c.NoContent(http.StatusCreated)
}

func (s *Server) getCtID(name string) string {
	if s.cron.HasJob(name) {
		// repo name
		return s.config.NamePrefix + name
	}
	// container ID
	return name
}

func (s *Server) getCtLogs(c echo.Context) error {
	type logsOptions struct {
		Follow bool   `query:"follow"`
		Tail   string `query:"tail"`
	}
	id := s.getCtID(c.Param("name"))
	var opts logsOptions
	if err := c.Bind(&opts); err != nil {
		return badRequest(err)
	}
	fw := NewFlushWriter(c.Response())
	return s.c.GetContainerLogs(core.LogsOptions{
		ID:          id,
		Stream:      fw,
		Tail:        opts.Tail,
		Follow:      opts.Follow,
		CloseNotify: c.Request().Context().Done(),
	})
}

func (s *Server) stopCt(c echo.Context) error {
	id := s.getCtID(c.Param("name"))
	if err := s.c.StopContainer(id); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) removeCt(c echo.Context) error {
	id := s.getCtID(c.Param("name"))
	_ = s.c.StopContainer(id)
	err := s.c.RemoveContainer(id)
	if err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) updateSyncStatus(m *api.Meta) {
	_, ok := s.syncStatus.Load(m.Name)
	m.Syncing = ok
}

func (s *Server) listMetas(c echo.Context) error {
	ms := s.c.ListAllMetas()
	for i := 0; i < len(ms); i++ {
		s.updateSyncStatus(&ms[i])
	}
	return c.JSON(http.StatusOK, ms)
}

func (s *Server) getMeta(c echo.Context) error {
	name := c.Param("name")
	m, err := s.c.GetMeta(name)
	if err != nil {
		return notFound(err)
	}
	s.updateSyncStatus(m)
	return c.JSON(http.StatusOK, m)
}

func (s *Server) exportConfig(c echo.Context) error {
	var query bson.M
	names := c.QueryParam("names")
	if names != "" {
		nameLst := strings.Split(names, ",")
		query = bson.M{
			"_id": bson.M{"$in": nameLst},
		}
	}
	var repos []bson.M
	_ = s.c.FindRepository(query).Select(bson.M{"updatedAt": 0, "createdAt": 0}).Sort("_id").All(&repos)
	for i := 0; i < len(repos); i++ {
		r := repos[i]
		r["name"] = r["_id"]
		delete(r, "_id")
		delete(r, "__v")
	}
	return c.JSON(http.StatusOK, repos)
}

func (s *Server) loadRepo(fp string) error {
	data, err := ioutil.ReadFile(fp)
	if err != nil {
		return err
	}
	var repo api.Repository
	err = yaml.Unmarshal(data, &repo)
	if err != nil {
		return badRequest(err)
	}
	if err := s.e.Validator.Validate(&repo); err != nil {
		return badRequest(err)
	}
	err = s.c.RemoveRepository(repo.Name)
	if err != nil && err != mgo.ErrNotFound {
		return err
	}
	err = s.c.AddRepository(repo)
	if err != nil {
		return err
	}
	err = s.cron.AddJob(repo.Name, repo.Interval, s.newJob(repo.Name))
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) reloadRepo(c echo.Context) error {
	name := c.Param("name")
	fp := filepath.Join(s.config.RepoConfigDir, name+".yaml")
	err := s.loadRepo(fp)
	if err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) reloadAllRepos(c echo.Context) error {
	infos, err := ioutil.ReadDir(s.config.RepoConfigDir)
	if err != nil {
		return err
	}
	for _, info := range infos {
		fileName := info.Name()
		if info.IsDir() || fileName[0] == '.' || !strings.HasSuffix(fileName, ".yaml") {
			continue
		}
		fp := filepath.Join(s.config.RepoConfigDir, fileName)
		err := s.loadRepo(fp)
		if err != nil {
			return err
		}
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) HTTPErrorHandler(err error, c echo.Context) {
	var (
		code    = http.StatusInternalServerError
		msg     string
		respMsg echo.Map
	)

	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		msg = fmt.Sprintf("%v", he.Message)
		if he.Internal != nil {
			msg = fmt.Sprintf("%v, %v", err, he.Internal)
		}
	} else if de, ok := err.(*docker.Error); ok {
		code = de.Status
		msg = de.Message
	} else {
		msg = err.Error()
	}
	respMsg = echo.Map{"message": msg}

	logrus.WithFields(logrus.Fields{
		"remote_ip": c.RealIP(),
		"status":    code,
		"method":    c.Request().Method,
		"uri":       c.Request().RequestURI,
	}).Error(msg)

	// Send response
	if !c.Response().Committed {
		if c.Request().Method == echo.HEAD { // Issue echo#608
			err = c.NoContent(code)
		} else {
			err = c.JSON(code, respMsg)
		}
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"method": c.Request().Method,
				"uri":    c.Request().RequestURI,
			}).Errorln(err)
		}
	}
}
