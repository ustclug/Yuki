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

	db := s.getDB(c)
	var metas []model.RepoMeta
	err := db.Order("name").Find(&metas).Error
	if err != nil {
		const msg = "Fail to list RepoMetas"
		l.Error(msg, slogErrAttr(err))
		return &echo.HTTPError{
			Code:    http.StatusInternalServerError,
			Message: msg,
		}
	}
	var repos []model.Repo
	err = db.Select("name", "mirrorz").Find(&repos).Error
	if err != nil {
		const msg = "Fail to list Repo MirrorZ mappings"
		l.Error(msg, slogErrAttr(err))
		return newHTTPError(http.StatusInternalServerError, msg)
	}
	mirrorzByRepo := make(map[string][]model.MirrorzRepo, len(repos))
	for _, repo := range repos {
		mirrorzByRepo[repo.Name] = repo.EffectiveMirrorz()
	}
	resp := make(api.ListRepoMetasResponse, len(metas))
	for i, meta := range metas {
		resp[i] = s.convertModelRepoMetaToGetMetaResponse(meta, mirrorzByRepo[meta.Name])
	}
	return c.JSON(http.StatusOK, resp)
}

func (s *Server) handlerGetRepoMeta(c echo.Context) error {
	l := getLogger(c)
	l.Debug("Invoked")

	name, err := getRepoNameFromRoute(c)
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

	var repo model.Repo
	repoRes := s.getDB(c).
		Select("name", "mirrorz").
		Where(model.Repo{Name: name}).
		Limit(1).
		Find(&repo)
	if repoRes.Error != nil {
		const msg = "Fail to get Repo MirrorZ mapping"
		l.Error(msg, slogErrAttr(repoRes.Error))
		return newHTTPError(http.StatusInternalServerError, msg)
	}
	var mirrorz []model.MirrorzRepo
	if repoRes.RowsAffected > 0 {
		mirrorz = repo.EffectiveMirrorz()
	}

	resp := s.convertModelRepoMetaToGetMetaResponse(meta, mirrorz)
	return c.JSON(http.StatusOK, resp)
}
