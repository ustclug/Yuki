package server

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/model"
)

func (s *Server) handlerListRepoMetas(c echo.Context) error {
	l := getLogger(c)
	l.Debug("Invoked")

	var metas []model.RepoMeta
	err := s.getDB(c).Find(&metas).Error
	if err != nil {
		const msg = "Fail to list RepoMetas"
		l.Error(msg, slogErrAttr(err))
		return &echo.HTTPError{
			Code:    http.StatusInternalServerError,
			Message: msg,
		}
	}
	resp := make(api.ListRepoMetasResponse, len(metas))
	jobs := s.cron.Jobs()
	for i, meta := range metas {
		resp[i] = s.convertModelRepoMetaToGetMetaResponse(meta, jobs)
	}
	return c.JSON(http.StatusOK, resp)
}

func (s *Server) handlerGetRepoMeta(c echo.Context) error {
	l := getLogger(c)
	l.Debug("Invoked")

	name, err := getRequiredParamFromEchoContext(c, "name")
	if err != nil {
		return err
	}

	var meta model.RepoMeta
	res := s.getDB(c).
		Where(model.RepoMeta{
			Name: name,
		}).
		Limit(1).
		Find(&meta)
	if res.Error != nil {
		const msg = "Fail to get RepoMeta"
		l.Error(msg, slogErrAttr(res.Error))
		return &echo.HTTPError{
			Code:    http.StatusInternalServerError,
			Message: msg,
		}
	}
	if res.RowsAffected == 0 {
		return &echo.HTTPError{
			Code:    http.StatusNotFound,
			Message: "RepoMeta not found",
		}
	}

	resp := s.convertModelRepoMetaToGetMetaResponse(meta, s.cron.Jobs())
	return c.JSON(http.StatusOK, resp)
}
