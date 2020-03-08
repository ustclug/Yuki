package server

import (
	"compress/gzip"
	"context"
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

	"github.com/docker/docker/errdefs"
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
		var infos []api.LogFileStat
		for _, f := range files {
			name := f.Name()
			if !strings.HasPrefix(name, "result.log.") {
				continue
			}
			infos = append(infos, api.LogFileStat{
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	apiCts, err := s.c.ListContainers(ctx, map[string][]string{
		"label": {"org.ustcmirror.syncing=true"},
	})
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
	ct, err := s.c.Sync(s.context(), core.SyncOptions{
		Name:          name,
		NamePrefix:    s.config.NamePrefix,
		LogDir:        s.config.LogDir,
		DefaultOwner:  s.config.Owner,
		MountDir:      true,
		Debug:         debug,
		DefaultBindIP: s.config.BindIP,
	})
	if err != nil {
		if errdefs.IsConflict(err) {
			return conflict(err)
		}
		return err
	}
	go func() {
		err := s.waitForSync(ct)
		if err != nil {
			logrus.Warningf("waitForSync: %s", err)
		}
	}()
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
	return s.c.GetContainerLogs(c.Request().Context(), core.LogsOptions{
		ID:     id,
		Stream: fw,
		Tail:   opts.Tail,
		Follow: opts.Follow,
	})
}

func (s *Server) removeCt(c echo.Context) error {
	id := s.getCtID(c.Param("name"))
	ctx, cancel := s.contextWithTimeout(time.Second * 10)
	_ = s.c.StopContainer(ctx, id)
	cancel()
	ctx, cancel = s.contextWithTimeout(time.Second * 10)
	_ = s.c.RemoveContainer(ctx, id)
	cancel()
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) updateSyncStatus(m *api.Meta) {
	_, ok := s.syncStatus.Load(m.Name)
	m.Syncing = ok
}

func (s *Server) listMetas(c echo.Context) error {
	ms := s.c.ListAllMetas()
	jobs := s.cron.Jobs()
	for i := 0; i < len(ms); i++ {
		job, ok := jobs[ms[i].Name]
		if ok {
			ms[i].NextRun = job.Next.Unix()
		}
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
	jobs := s.cron.Jobs()
	job, ok := jobs[m.Name]
	if ok {
		m.NextRun = job.Next.Unix()
	}
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
	var repos []api.Repository
	_ = s.c.FindRepository(query).Select(bson.M{"updatedAt": 0, "createdAt": 0}).Sort("_id").All(&repos)
	return c.JSON(http.StatusOK, repos)
}

func (s *Server) loadRepo(fp string) (*api.Repository, error) {
	data, err := ioutil.ReadFile(fp)
	if err != nil {
		return nil, err
	}
	var repo api.Repository
	err = yaml.Unmarshal(data, &repo)
	if err != nil {
		return nil, badRequest(err)
	}
	if err := s.e.Validator.Validate(&repo); err != nil {
		return nil, badRequest(err)
	}
	err = s.c.RemoveRepository(repo.Name)
	if err != nil && err != mgo.ErrNotFound {
		return nil, err
	}
	err = s.c.AddRepository(repo)
	if err != nil {
		return nil, err
	}
	err = s.cron.AddJob(repo.Name, repo.Interval, s.newJob(repo.Name))
	if err != nil {
		return nil, err
	}
	_ = s.c.UpsertRepoMeta(repo.Name, repo.StorageDir, -1)
	return &repo, nil
}

func (s *Server) reloadRepo(c echo.Context) error {
	name := c.Param("name")
	fp := filepath.Join(s.config.RepoConfigDir, name+".yaml")
	_, err := s.loadRepo(fp)
	if err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) reloadAllRepos(c echo.Context) error {
	repos := s.c.ListAllRepositories()
	toDelete := map[string]struct{}{}
	for _, r := range repos {
		toDelete[r.Name] = struct{}{}
	}
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
		repo, err := s.loadRepo(fp)
		if err != nil {
			return err
		}
		delete(toDelete, repo.Name)
	}
	for name := range toDelete {
		err := s.c.RemoveRepository(name)
		if err != nil {
			logrus.WithField("repo", name).Errorf("remove repository: %s", err)
		}
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) httpErrorHandler(err error, c echo.Context) {
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
