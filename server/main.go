package server

import (
	"os"
	"path"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/knight42/Yuki/core"
	"github.com/knight42/Yuki/cron"
	"github.com/knight42/Yuki/events"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"
	"gopkg.in/mgo.v2/bson"
)

type Config struct {
	*core.Config
	Owner                 string   `json:"owner,omitempty" toml:"owner,omitempty"`
	LogDir                string   `json:"log_dir,omitempty" toml:"log_dir,omitempty"`
	BindIP                string   `json:"bind_ip,omitempty" toml:"bind_ip,omitempty"`
	NamePrefix            string   `json:"name_prefix,omitempty" toml:"name_prefix,omitempty"`
	LogLevel              log.Lvl  `json:"log_level,omitempty" toml:"log_level,omitempty"`
	AllowOrigins          []string `json:"allow_origins,omitempty" toml:"allow_origins,omitempty"`
	ImagesUpgradeInterval string   `json:"images_upgrade_interval,omitempty" toml:"images_upgrade_interval,omitempty"`
}

var (
	DefaultServerConfig = Config{
		Config: &core.Config{
			Debug:          true,
			DbURL:          "127.0.0.1:27017",
			DbName:         "mirror",
			DockerEndpoint: "unix:///var/run/docker.sock",
		},
		Owner:        "0:0",
		LogDir:       "/var/log/ustcmirror/",
		NamePrefix:   "syncing-",
		LogLevel:     log.INFO,
		AllowOrigins: []string{"*"},
	}
)

type Server struct {
	e       *echo.Echo
	c       *core.Core
	config  *Config
	cron    *cron.Cron
	emitter *events.Emitter
	logger  echo.Logger
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
		cron:    cron.New(),
		config:  &cfg,
		emitter: events.NewEmitter(),
	}
	s.logger = s.e.Logger
	s.logger.SetLevel(cfg.LogLevel)
	s.e.Debug = cfg.Debug
	s.e.HideBanner = true

	logfile, err := os.OpenFile(path.Join(cfg.LogDir, "yukid.log"), os.O_APPEND | os.O_CREATE | os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	s.logger.SetOutput(logfile)

	s.cleanContainers()
	s.schedRepos()

	s.e.GET("/", func(c echo.Context) error {
		return c.String(200, "Hello World!")
	})
	g := s.e.Group("/api/v1/")
	s.RegisterAPIs(g)

	// middlewares
	secureCfg := middleware.DefaultSecureConfig
	secureCfg.HSTSMaxAge = 31536000
	s.e.Use(middleware.SecureWithConfig(secureCfg))

	corsCfg := middleware.DefaultCORSConfig
	corsCfg.AllowOrigins = cfg.AllowOrigins
	s.e.Use(middleware.CORSWithConfig(corsCfg))
	return &s, nil
}

func (s *Server) cleanContainers() {
	s.e.Logger.Info("Cleaning dead containers.")
	opts := docker.ListContainersOptions{
		All: true,
		Filters: map[string][]string{
			"label":  {"ustcmirror.images"},
			"status": {"created", "exited", "dead"},
		},
	}
	cts, err := s.c.Docker.ListContainers(opts)
	if err != nil {
		s.logger.Error(err)
		return
	}
	for _, ct := range cts {
		go func(id string) {
			if err = s.c.RemoveContainer(ct.ID); err != nil {
				s.logger.Error(err)
			}
		}(ct.ID)
	}
}

func (s *Server) schedRepos() {
	repos := s.c.ListRepositories(nil, bson.M{"interval": 1})
	s.logger.Infof("Scheduling %d repositories.", len(repos))
	for _, r := range repos {
		s.cron.AddJob(r.Name, r.Interval, s.newJob(r.Name))
	}
}

func (s *Server) newJob(name string) cron.FuncJob {
	return func() {
		if err := s.c.Sync(core.SyncOptions{
			LogDir:     s.config.LogDir,
			Owner:      s.config.Owner,
			NamePrefix: s.config.NamePrefix,
			Name:       name,
			MountDir:   !IsTest,
		}); err != nil {
			s.logger.Error(err)
		}
	}
}

func (s *Server) Start(addr string) error {
	s.logger.Infof("Listening at %s.", addr)
	return s.e.Start(addr)
}
