package server

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/errdefs"
	"github.com/globalsign/mgo/bson"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/core"
)

func (s *Server) registerAPIs(e *echo.Echo) {
	v1API := e.Group("/api/v1/")

	// public APIs
	v1API.GET("metas", s.handlerListRepoMetas)
	v1API.GET("metas/:name", s.handlerGetRepoMeta)

	// private APIs
	v1API.GET("repos", s.handlerListRepos)
	v1API.GET("repos/:name", s.handlerGetRepo)
	v1API.DELETE("repos/:name", s.handlerRemoveRepo)
	v1API.GET("repos/:name/logs", s.handlerGetRepoLogs)
	v1API.POST("repos/:name", s.handlerReloadRepo)
	v1API.POST("repos", s.handlerReloadAllRepos)
	v1API.POST("repos/:name/sync", s.handlerSyncRepo)

	v1API.GET("config", s.exportConfig)
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
	ct, err := s.c.Sync(c.Request().Context(), core.SyncOptions{
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
			return conflict(err.Error())
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
		return badRequest(err.Error())
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
	reqCtx := c.Request().Context()
	ctx, cancel := context.WithTimeout(reqCtx, time.Second*10)
	_ = s.c.StopContainer(ctx, id)
	cancel()
	ctx, cancel = context.WithTimeout(reqCtx, time.Second*10)
	_ = s.c.RemoveContainer(ctx, id)
	cancel()
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) updateSyncStatus(m *api.Meta) {
	_, ok := s.syncStatus.Load(m.Name)
	m.Syncing = ok
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
