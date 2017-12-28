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
	"strconv"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/knight42/Yuki/core"
	"github.com/knight42/Yuki/queue"
	"github.com/labstack/echo"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func (s *Server) listRepos(c echo.Context) error {
	return c.JSON(http.StatusOK, s.c.ListRepositories(nil, bson.M{
		"interval": 1,
		"image":    1,
	}))
}

func (s *Server) addRepo(c echo.Context) error {
	repo := core.Repository{}
	if err := c.Bind(&repo); err != nil {
		s.logger.Warn(err)
		return BadRequest(err.Error())
	}
	err := s.c.AddRepository(&repo)
	if err != nil {
		s.logger.Error(err)
		if mgo.IsDup(err) {
			return Conflict(err.Error())
		} else {
			return err
		}
	}
	return c.NoContent(http.StatusCreated)
}

func (s *Server) getRepo(c echo.Context) error {
	name := c.Param("name")
	repo, err := s.c.GetRepository(name)
	if err != nil {
		s.logger.Error(err)
		return NotFound(err.Error())
	}
	return c.JSON(http.StatusOK, repo)
}

func (s *Server) getRepoLogs(c echo.Context) error {
	type repoLogsOptions struct {
		N    int `query:"n"`
		Tail int `query:"tail"`
	}

	opts := repoLogsOptions{}
	if err := c.Bind(&opts); err != nil {
		return err
	}

	if opts.Tail < 0 {
		opts.Tail = 0
	}
	if opts.Tail > 64 {
		opts.Tail = 64
	}

	logdir := path.Join(s.config.LogDir, c.Param("name"))
	if err := os.MkdirAll(logdir, os.ModePerm); err != nil {
		return fmt.Errorf("not a directory: %s", logdir)
	}

	files, err := ioutil.ReadDir(logdir)
	if err != nil {
		return err
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
		return NotFound(fmt.Errorf("no such file: %s", wantedName))
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
	q.ReadFrom(reader)
	q.WriteTo(c.Response())
	return nil
}

func (s *Server) updateRepo(c echo.Context) error {
	t := bson.M{}
	decoder := json.NewDecoder(c.Request().Body)
	if err := decoder.Decode(&t); err != nil {
		s.logger.Warn(err)
		return BadRequest(err.Error())
	}
	name := c.Param("name")
	if err := s.c.UpdateRepository(name, t); err != nil {
		s.logger.Error(err)
		return err
	}
	r, _ := s.c.GetRepository(name)
	s.cron.AddJob(r.Name, r.Interval, s.newJob(r.Name))
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) removeRepo(c echo.Context) error {
	name := c.Param("name")
	if err := s.c.RemoveRepository(name); err != nil {
		s.logger.Error(err)
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
			"label": {"ustcmirror.images"},
		},
	}
	apiCts, err := s.c.Docker.ListContainers(opts)
	if err != nil {
		s.logger.Warn(err)
		return err
	}
	cts := []container{}
	for _, ct := range apiCts {
		cts = append(cts, container{
			ID:      "id:" + ct.ID[:10],
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

	err = s.c.Sync(core.SyncOptions{
		Name:       name,
		NamePrefix: s.config.NamePrefix,
		LogDir:     s.config.LogDir,
		Owner:      s.config.Owner,
		MountDir:   !IsTest,
		Debug:      debug,
		BindIP:     s.config.BindIP,
	})
	if err != nil {
		s.logger.Error(err)
		if err == docker.ErrContainerAlreadyExists {
			return Conflict(err.Error())
		}
		return err
	}
	if debug {
		fw := NewFlushWriter(c.Response())
		if err = s.c.GetContainerLogs(core.LogsOptions{
			ID:     s.config.NamePrefix + name,
			Stream: fw,
			Tail:   "all",
			Follow: true,
		}); err != nil {
			s.logger.Warn(err)
		}
		return err
	}
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
		return BadRequest(err.Error())
	}
	fw := NewFlushWriter(c.Response())
	if err := s.c.GetContainerLogs(core.LogsOptions{
		ID:     name,
		Stream: fw,
		Tail:   opts.Tail,
		Follow: opts.Follow,
	}); err != nil {
		return err
	}
	return nil
}

func (s *Server) stopCt(c echo.Context) error {
	name := c.Param("name")
	if err := s.c.StopContainer(s.config.NamePrefix + name); err != nil {
		s.logger.Error(err)
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) removeCt(c echo.Context) error {
	name := c.Param("name")
	err := s.c.RemoveContainer(s.config.NamePrefix + name)
	if err != nil {
		s.logger.Error(err)
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
		s.logger.Error(err)
		return NotFound(err.Error())
	}
	return c.JSON(http.StatusOK, m)
}

func (s *Server) exportConfig(c echo.Context) error {
	repos := s.c.ListRepositories(nil, bson.M{
		"updatedAt": 0,
	})
	return c.JSON(http.StatusOK, repos)
}

func (s *Server) importConfig(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) RegisterAPIs(g *echo.Group) {
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
	// FIXME
	g.POST("config", s.importConfig)
}
