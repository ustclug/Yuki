package server

import (
	"os"

	"github.com/knight42/Yuki/core"
	"github.com/knight42/Yuki/events"
	"gopkg.in/labstack/echo.v3"
)

type Config struct {
	*core.Config
	Owner      string
	LogDir     string
	NamePrefix string
}

var (
	DefaultServerConfig = Config{
		Config: &core.Config{
			Debug:          true,
			DbURL:          "127.0.0.1:27017",
			DbName:         "mirror",
			DockerEndpoint: "unix:///var/run/docker.sock",
		},
		Owner:      "0:0",
		LogDir:     "/var/log/ustcmirror/",
		NamePrefix: "syncing-",
	}
)

type Server struct {
	e       *echo.Echo
	c       *core.Core
	emitter *events.Emitter
	config  *Config
}

func New() (*Server, error) {
	return NewWithConfig(DefaultServerConfig)
}

func NewWithConfig(cfg Config) (*Server, error) {
	if err := os.MkdirAll(cfg.LogDir, os.ModePerm); err != nil {
		return nil, err
	}
	coreCfg := core.Config{
		Debug:          cfg.Debug,
		DbURL:          cfg.DbURL,
		DbName:         cfg.DbName,
		DockerEndpoint: cfg.DockerEndpoint,
	}
	c, err := core.NewWithConfig(coreCfg)
	if err != nil {
		return nil, err
	}
	s := Server{
		c:       c,
		e:       echo.New(),
		config:  &cfg,
		emitter: events.NewEmitter(),
	}
	s.e.Debug = cfg.Debug
	s.e.HideBanner = !cfg.Debug

	s.e.GET("/", func(c echo.Context) error {
		return c.String(200, "Hello World!")
	})
	g := s.e.Group("/api/v1/")
	s.RegisterAPIs(g)
	return &s, nil
}

func (s *Server) Start(addr string) error {
	return s.e.Start(addr)
}
