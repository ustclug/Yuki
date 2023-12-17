package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func (s *Server) getDB(c echo.Context) *gorm.DB {
	return s.db.WithContext(c.Request().Context())
}

func (s *Server) handlerListMetas(c echo.Context) error {
	ms := s.c.ListAllMetas()
	jobs := s.cron.Jobs()
	for i := 0; i < len(ms); i++ {
		job, ok := jobs[ms[i].Name]
		if ok {
			ms[i].NextRun = job.Next.Unix()
		}
		s.updateSyncStatus(&ms[i])
	}
	return c.JSON(http.StatusOK, ms)
}

func (s *Server) handlerGetMeta(c echo.Context) error {
	name := c.Param("name")
	m, err := s.c.GetMeta(name)
	if err != nil {
		return notFound(err)
	}
	s.updateSyncStatus(m)
	jobs := s.cron.Jobs()
	job, ok := jobs[m.Name]
	if ok {
		m.NextRun = job.Next.Unix()
	}
	return c.JSON(http.StatusOK, m)
}
