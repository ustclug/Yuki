package server

import (
	"github.com/knight42/Yuki/core"
	"gopkg.in/labstack/echo.v3"
)

type ServerConfig struct {
	*core.ManagerConfig
}

var (
	DefaultServerConfig = ServerConfig{
		ManagerConfig: &core.ManagerConfig{
			Debug:          true,
			DbURL:          "127.0.0.1:27017",
			DbName:         "mirror",
			DockerEndpoint: "unix:///var/run/docker.sock",
			NamePrefix:     "syncing-",
		},
	}
)

type Server struct {
	e *echo.Echo
	m core.Manager
}

func New() (*Server, error) {
	return NewWithConfig(DefaultServerConfig)
}

func NewWithConfig(cfg ServerConfig) (*Server, error) {
	managerCfg := core.ManagerConfig{
		Debug:          cfg.Debug,
		DbURL:          cfg.DbURL,
		DbName:         cfg.DbName,
		NamePrefix:     cfg.NamePrefix,
		DockerEndpoint: cfg.DockerEndpoint,
	}
	m, err := core.NewWithConfig(managerCfg)
	if err != nil {
		return nil, err
	}
	s := Server{
		e: echo.New(),
		m: m,
	}
	s.e.Debug = cfg.Debug
	s.e.HideBanner = !cfg.Debug

	g := s.e.Group("/api/v1/")
	s.registerAPIs(g)
	return &s, nil
}

func (s *Server) Start(addr string) error {
	return s.e.Start(addr)
}
