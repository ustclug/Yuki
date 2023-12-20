package server

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/model"
)

const suffixYAML = ".yaml"

func (s *Server) getDB(c echo.Context) *gorm.DB {
	return s.db.WithContext(c.Request().Context())
}

func getRequiredParamFromEchoContext(c echo.Context, name string) (string, error) {
	val := c.Param(name)
	if len(val) == 0 {
		return "", badRequest(name + " is required")
	}
	return val, nil
}

func (s *Server) convertModelRepoMetaToGetMetaResponse(in model.RepoMeta, jobs map[string]cron.Entry) api.GetMetaResponse {
	_, syncing := s.syncStatus.Load(in.Name)
	var nextRun int64
	job, ok := jobs[in.Name]
	if ok {
		nextRun = job.Next.Unix()
	}
	return api.GetMetaResponse{
		Name:        in.Name,
		Upstream:    in.Upstream,
		Syncing:     syncing,
		Size:        in.Size,
		ExitCode:    in.ExitCode,
		LastSuccess: in.LastSuccess,
		UpdatedAt:   in.UpdatedAt,
		PrevRun:     in.PrevRun,
		NextRun:     nextRun,
	}
}

func slogErrAttr(err error) slog.Attr {
	return slog.Any("err", err)
}

func bindAndValidate[T any](c echo.Context, input *T) error {
	err := c.Bind(input)
	if err != nil {
		return err
	}
	return c.Validate(input)
}

func badRequest(msg string) error {
	return &echo.HTTPError{
		Code:    http.StatusBadRequest,
		Message: msg,
	}
}

func notFound(msg string) error {
	return &echo.HTTPError{
		Code:    http.StatusNotFound,
		Message: msg,
	}
}

func conflict(msg string) error {
	return &echo.HTTPError{
		Code:    http.StatusConflict,
		Message: msg,
	}
}

func newHTTPError(code int, msg string) error {
	return &echo.HTTPError{
		Code:    code,
		Message: msg,
	}
}
