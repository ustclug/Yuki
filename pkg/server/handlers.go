package server

import (
	"github.com/labstack/echo/v4"
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
}
