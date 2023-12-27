package server

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/ustclug/Yuki/pkg/model"
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

	v1API.GET("config", s.handlerExportConfig)
}

func (s *Server) handlerExportConfig(c echo.Context) error {
	l := getLogger(c)
	l.Debug("Invoked")

	names := c.QueryParam("names")
	tx := s.getDB(c)
	if names != "" {
		tx = tx.Where("name IN ?", strings.Split(names, ","))
	}
	var repos []model.Repo
	err := tx.Order("name").Find(&repos).Error
	if err != nil {
		const msg = "Failed to list repositories"
		l.Error(msg, slogErrAttr(err), slog.String("names", names))
		return newHTTPError(http.StatusInternalServerError, msg)
	}
	return c.JSON(http.StatusOK, repos)
}
