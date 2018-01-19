package server

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/knight42/Yuki/core"
	"github.com/knight42/Yuki/queue"
	"github.com/labstack/echo"
	log "github.com/sirupsen/logrus"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
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

func notAcceptable(msg interface{}) error {
	return &echo.HTTPError{
		Code:    http.StatusNotAcceptable,
		Message: msg,
	}
}

func conflict(msg interface{}) error {
	return &echo.HTTPError{
		Code:    http.StatusConflict,
		Message: msg,
	}
}

func forbidden(msg interface{}) error {
	return &echo.HTTPError{
		Code:    http.StatusForbidden,
		Message: msg,
	}
}

func (s *Server) listRepos(c echo.Context) error {
	return c.JSON(http.StatusOK, s.c.ListRepositories(nil, bson.M{
		"interval": 1,
		"image":    1,
	}))
}

func (s *Server) addRepo(c echo.Context) error {
	repo := new(core.Repository)
	if err := c.Bind(repo); err != nil {
		return badRequest(err)
	}
	if err := c.Validate(repo); err != nil {
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
		infos := []fileInfo{}
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
	if opts.Tail > 64 {
		opts.Tail = 64
	}

	wantedName := fmt.Sprintf("result.log.%d", opts.N)
	fileName := ""
	for _, f := range files {
		if strings.HasPrefix(f.Name(), wantedName) {
			// result.log.0
			// result.log.1.gz
			// result.log.2.gz
			fileName = f.Name()
			break
		}
	}

	if fileName == "" {
		return notFound(fmt.Sprintf("no such file: %s", wantedName))
	}

	content, err := os.Open(path.Join(logdir, fileName))
	if err != nil {
		return err
	}
	defer content.Close()

	var reader io.Reader

	switch path.Ext(fileName) {
	case ".gz":
		gr, err := gzip.NewReader(content)
		if err != nil {
			return err
		}
		defer gr.Close()
		reader = gr
	default:
		reader = content
	}

	if opts.Tail == 0 {
		io.Copy(c.Response(), reader)
		return nil
	}

	q := queue.New(opts.Tail)
	if err = q.ReadAll(reader); err != nil {
		return err
	}
	q.WriteAll(c.Response())
	return nil
}

func convertUpdate(update bson.M) bson.M {
	if _, ok := update["$set"]; !ok {
		update["$set"] = make(map[string]interface{})
	}
	set := update["$set"].(map[string]interface{})
	for k, v := range update {
		if !strings.HasPrefix(k, "$") {
			set[k] = v
			delete(update, k)
		}
	}
	return update
}

func (s *Server) updateRepo(c echo.Context) error {
	t := bson.M{}
	decoder := json.NewDecoder(c.Request().Body)
	if err := decoder.Decode(&t); err != nil {
		return badRequest(err)
	}
	t = convertUpdate(t)
	set := t["$set"].(map[string]interface{})
	myva := s.e.Validator.(*myValidator)
	if err := myva.CheckMap(set, core.Repository{}); err != nil {
		return badRequest(err)
	}
	name := c.Param("name")
	if err := s.c.UpdateRepository(name, t); err != nil {
		return err
	}
	r, _ := s.c.GetRepository(name)
	s.logger.Infof("Rescheduled %s", name)
	if err := s.cron.AddJob(r.Name, r.Interval, s.newJob(*r)); err != nil {
		s.logger.Errorln(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) removeRepo(c echo.Context) error {
	name := c.Param("name")
	if err := s.c.RemoveRepository(name); err != nil {
		return err
	}
	if err := s.c.RemoveMeta(name); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) listCts(c echo.Context) error {
	type container struct {
		ID      string `json:"Id"`
		Image   string `json:"Image"`
		Created int64  `json:"Created"`
		State   string `json:"State"`
		Status  string `json:"Status"`
		Name    string `json:"Name"`
	}
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
	cts := []container{}
	for _, ct := range apiCts {
		cts = append(cts, container{
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

	s.logger.Infof("Syncing %s", name)
	ct, err := s.c.Sync(core.SyncOptions{
		Name:       name,
		NamePrefix: s.config.NamePrefix,
		LogDir:     s.config.LogDir,
		Owner:      s.config.Owner,
		MountDir:   !IsTest,
		Debug:      debug,
		BindIP:     s.config.BindIP,
	})
	if err != nil {
		if err == docker.ErrContainerAlreadyExists {
			return conflict(err)
		}
		return err
	}

	go func() {
		s.c.WaitForSync(*ct, 0)
	}()

	return c.NoContent(http.StatusCreated)
}

func (s *Server) getCtLogs(c echo.Context) error {
	type logsOptions struct {
		Follow bool   `query:"follow"`
		Tail   string `query:"tail"`
	}
	name := s.config.NamePrefix + c.Param("name")
	opts := logsOptions{}
	if err := c.Bind(&opts); err != nil {
		return badRequest(err)
	}
	fw := NewFlushWriter(c.Response())
	return s.c.GetContainerLogs(core.LogsOptions{
		ID:     name,
		Stream: fw,
		Tail:   opts.Tail,
		Follow: opts.Follow,
	})
}

func (s *Server) stopCt(c echo.Context) error {
	name := c.Param("name")
	if err := s.c.StopContainer(s.config.NamePrefix + name); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) removeCt(c echo.Context) error {
	name := c.Param("name")
	err := s.c.RemoveContainer(s.config.NamePrefix + name)
	if err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) listMetas(c echo.Context) error {
	return c.JSON(http.StatusOK, s.c.ListMetas(nil, nil))
}

func (s *Server) getMeta(c echo.Context) error {
	name := c.Param("name")
	m, err := s.c.GetMeta(name)
	if err != nil {
		return notFound(err)
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
	repos := s.c.ListRepositories(query, bson.M{
		"updatedAt": 0,
	})
	return c.JSON(http.StatusOK, repos)
}

func (s *Server) importConfig(c echo.Context) error {
	var repos []*core.Repository
	if err := c.Bind(&repos); err != nil {
		return err
	}
	if err := s.c.AddRepository(repos...); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) registerAPIs(g *echo.Group) {
	g.GET("repositories", s.listRepos)
	g.POST("repositories/:name", s.addRepo)
	g.GET("repositories/:name", s.getRepo)
	g.PUT("repositories/:name", s.updateRepo)
	g.DELETE("repositories/:name", s.removeRepo)
	g.GET("repositories/:name/logs", s.getRepoLogs)

	g.GET("containers", s.listCts)
	g.POST("containers/:name", s.sync)
	g.POST("containers/:name/stop", s.stopCt)
	g.DELETE("containers/:name", s.removeCt)
	g.GET("containers/:name/logs", s.getCtLogs)

	g.GET("metas", s.listMetas)
	g.GET("metas/:name", s.getMeta)

	g.GET("config", s.exportConfig)
	g.POST("config", s.importConfig)
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
		if he.Inner != nil {
			msg = fmt.Sprintf("%v, %v", err, he.Inner)
		}
	} else if de, ok := err.(*docker.Error); ok {
		code = de.Status
		msg = de.Message
	} else {
		msg = err.Error()
	}
	respMsg = echo.Map{"message": msg}

	s.logger.WithFields(log.Fields{
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
			s.logger.WithFields(log.Fields{
				"method": c.Request().Method,
				"uri":    c.Request().RequestURI,
			}).Errorln(err)
		}
	}
}
