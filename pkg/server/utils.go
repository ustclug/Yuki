package server

import (
	"github.com/labstack/echo/v4"
	"github.com/robfig/cron/v3"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/model"
)

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
