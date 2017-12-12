package server

import (
	"net/http"
	"encoding/json"
	"gopkg.in/mgo.v2/bson"
	"gopkg.in/labstack/echo.v3"
	"github.com/knight42/Yuki/core"
)

func (s *Server) listRepos(c echo.Context) error {
	return c.JSON(http.StatusOK, s.m.ListRepositories(nil, bson.M{
		"interval": 1,
		"image": 1,
	}))
}

func (s *Server) addRepo(c echo.Context) error {
	repo := core.Repository{}
	if err := c.Bind(&repo); err != nil {
		return err
	}
	return s.m.AddRepository(&repo)
}

func (s *Server) getRepo(c echo.Context) error {
	name := c.Param("name")
	repo, err := s.m.GetRepository(name)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, repo)
}

func (s *Server) updateRepo(c echo.Context) error {
	t := bson.M{}
	decoder := json.NewDecoder(c.Request().Body)
	if err := decoder.Decode(&t); err != nil {
		return err
	}
	name := c.Param("name")
	if err := s.m.UpdateRepository(name, t); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) removeRepo(c echo.Context) error {
	name := c.Param("name")
	err := s.m.RemoveRepository(name)
	if err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) registerAPIs(g *echo.Group) {
	g.GET("/repositories", s.listRepos)
	g.POST("/repositories", s.addRepo)
	g.GET("/repositories/:name", s.getRepo)
	g.PUT("/repositories/:name", s.updateRepo)
	g.DELETE("/repositories/:name", s.removeRepo)
}

