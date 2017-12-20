package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/knight42/Yuki/core"
	"github.com/labstack/echo"
	"gopkg.in/mgo.v2/bson"
)

func (s *Server) listRepos(c echo.Context) error {
	return c.JSON(http.StatusOK, s.c.ListRepositories(nil, bson.M{
		"interval": 1,
		"image":    1,
	}))
}

func (s *Server) addRepo(c echo.Context) error {
	repo := core.Repository{}
	if err := c.Bind(&repo); err != nil {
		return err
	}
	return s.c.AddRepository(&repo)
}

func (s *Server) getRepo(c echo.Context) error {
	name := c.Param("name")
	repo, err := s.c.GetRepository(name)
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
	if err := s.c.UpdateRepository(name, t); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) removeRepo(c echo.Context) error {
	name := c.Param("name")
	if err := s.c.RemoveRepository(name); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) listCts(c echo.Context) error {
	opts := docker.ListContainersOptions{
		All: true,
	}
	//FIXME: filter containers
	cts, err := s.c.Docker.ListContainers(opts)
	if err != nil {
		return err
	}
	//FIXME: filter fields
	return c.JSON(http.StatusOK, cts)
}

func (s *Server) sync(c echo.Context) error {
	name := c.Param("name")

	repo, err := s.c.GetRepository(name)
	if err != nil {
		return err
	}
	envs := []string{}
	for k, v := range repo.Envs {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}
	binds := []string{}
	for k, v := range repo.Volumes {
		binds = append(binds, fmt.Sprintf("%s:%s", k, v))
	}
	if !IsTest {
		logdir := path.Join(s.config.LogDir, name)
		if err := os.MkdirAll(logdir, os.ModePerm); err != nil {
			return err
		}
		binds = append(binds, fmt.Sprintf("%s:/data/", repo.StorageDir), fmt.Sprintf("%s:/log/", logdir))
	}
	opts := docker.CreateContainerOptions{
		Name: s.config.NamePrefix + name,
		Config: &docker.Config{
			Image:     repo.Image,
			OpenStdin: true,
			User:      s.config.Owner,
			Env:       envs,
		},
		HostConfig: &docker.HostConfig{
			Binds:         binds,
			RestartPolicy: docker.RestartUnlessStopped(),
		},
	}
	ct, err := s.c.Docker.CreateContainer(opts)
	if err != nil {
		return err
	}
	if err := s.c.Docker.StartContainer(ct.ID, nil); err != nil {
		return err
	}
	return c.String(http.StatusCreated, "Created")
}

func (s *Server) stopCt(c echo.Context) error {
	name := c.Param("name")
	return s.c.Docker.StopContainer(name, 100)
}

func (s *Server) removeCt(c echo.Context) error {
	name := c.Param("name")
	opts := docker.RemoveContainerOptions{
		RemoveVolumes: true,
		Force:         true,
		ID:            s.config.NamePrefix + name,
	}
	return s.c.Docker.RemoveContainer(opts)
}

func (s *Server) RegisterAPIs(g *echo.Group) {
	g.GET("/repositories", s.listRepos)
	g.POST("/repositories/:name", s.addRepo)
	g.GET("/repositories/:name", s.getRepo)
	g.PUT("/repositories/:name", s.updateRepo)
	g.DELETE("/repositories/:name", s.removeRepo)

	g.GET("/containers", s.listCts)
	g.POST("/containers/:name", s.sync)
	g.POST("/containers/:name/stop", s.stopCt)
	g.DELETE("/containers/:name", s.removeCt)
}
