package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/model"
)

func (s *Server) getDB(c echo.Context) *gorm.DB {
	return s.db.WithContext(c.Request().Context())
}

func (s *Server) handlerListRepoMetas(c echo.Context) error {
	var metas []model.RepoMeta
	err := s.getDB(c).Find(&metas).Error
	if err != nil {
		return err
	}
	resp := make(api.ListRepoMetasResponse, len(metas))
	jobs := s.cron.Jobs()
	for i, meta := range metas {
		resp[i] = s.convertModelRepoMetaToGetMetaResponse(meta, jobs)
	}
	return c.JSON(http.StatusOK, resp)
}

func (s *Server) handlerGetRepoMeta(c echo.Context) error {
	name, err := getRequiredParamFromEchoContext(c, "name")
	if err != nil {
		return err
	}

	var meta model.RepoMeta
	err = s.getDB(c).
		Where(model.RepoMeta{
			Name: name,
		}).
		Limit(1).
		Find(&meta).Error
	if err != nil {
		return err
	}

	resp := s.convertModelRepoMetaToGetMetaResponse(meta, s.cron.Jobs())
	return c.JSON(http.StatusOK, resp)
}
