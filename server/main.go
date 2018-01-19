package server

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/knight42/Yuki/core"
	"github.com/knight42/Yuki/cron"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	log "github.com/sirupsen/logrus"
)

type Server struct {
	e      *echo.Echo
	c      *core.Core
	config *Config
	cron   *cron.Cron
	logger *log.Logger
}

func New() (*Server, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	return NewWithConfig(*cfg)
}

func NewWithConfig(cfg Config) (*Server, error) {
	if cfg.DbURL == "" {
		cfg.DbURL = DefaultServerConfig.DbURL
	}
	if cfg.DbName == "" {
		cfg.DbName = DefaultServerConfig.DbName
	}
	if cfg.FileSystem == "" {
		cfg.FileSystem = DefaultServerConfig.FileSystem
	}
	if cfg.DockerEndpoint == "" {
		cfg.DockerEndpoint = DefaultServerConfig.DockerEndpoint
	}

	if cfg.Owner == "" {
		cfg.Owner = DefaultServerConfig.Owner
	}
	if cfg.LogDir == "" {
		cfg.LogDir = DefaultServerConfig.LogDir
	}
	if cfg.NamePrefix == "" {
		cfg.NamePrefix = DefaultServerConfig.NamePrefix
	}
	if len(cfg.AllowOrigins) == 0 {
		cfg.AllowOrigins = DefaultServerConfig.AllowOrigins
	}
	if cfg.ImagesUpgradeInterval == "" {
		cfg.ImagesUpgradeInterval = DefaultServerConfig.ImagesUpgradeInterval
	}

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
		c:      c,
		e:      echo.New(),
		cron:   cron.New(),
		config: &cfg,
		logger: log.New(),
	}
	s.e.Validator = &myValidator{NewValidator()}

	s.logger.SetLevel(cfg.LogLevel)

	s.e.Debug = cfg.Debug
	s.e.HideBanner = true

	s.e.HTTPErrorHandler = s.HTTPErrorHandler
	s.e.Logger.SetOutput(ioutil.Discard)

	logfile, err := os.OpenFile(path.Join(cfg.LogDir, "yukid.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	s.logger.Formatter = new(log.TextFormatter)
	s.logger.Out = logfile

	s.logger.Info("Cleaning dead containers")
	s.c.CleanDeadContainers()

	s.c.WaitRunningContainers()

	s.schedRepos()
	s.c.InitMetas()
	s.registerPostSync()

	s.cron.AddFunc(cfg.ImagesUpgradeInterval, func() {
		s.logger.Info("Upgrading images")
		s.c.UpgradeImages()
		s.logger.Info("Cleaning images")
		s.c.CleanImages()
	})

	g := s.e.Group("/api/v1/")
	s.registerAPIs(g)

	// middlewares
	secureCfg := middleware.DefaultSecureConfig
	secureCfg.HSTSMaxAge = 31536000
	s.e.Use(middleware.SecureWithConfig(secureCfg))

	corsCfg := middleware.DefaultCORSConfig
	corsCfg.AllowOrigins = cfg.AllowOrigins
	s.e.Use(middleware.CORSWithConfig(corsCfg))

	return &s, nil
}

func (s *Server) schedRepos() {
	repos := s.c.ListRepositories(nil, nil)
	s.logger.Infof("Scheduling %d repositories", len(repos))
	for _, r := range repos {
		if err := s.cron.AddJob(r.Name, r.Interval, s.newJob(&r)); err != nil {
			s.logger.Errorln(err)
		}
	}
}

func (s *Server) newJob(r *core.Repository) cron.FuncJob {
	return func() {
		name := r.Name
		s.logger.Infof("Syncing %s", name)
		ct, err := s.c.Sync(core.SyncOptions{
			LogDir:     s.config.LogDir,
			Owner:      s.config.Owner,
			NamePrefix: s.config.NamePrefix,
			Name:       name,
			MountDir:   !IsTest,
		})
		if err != nil {
			s.logger.Errorln(err)
			return
		}
		if err := s.c.WaitForSync(*ct, r.Retry); err != nil {
			s.logger.Errorln(err)
		}
	}
}

func (s *Server) Start() error {
	s.logger.Infof("Listening at %s", s.config.ListenAddr)
	return s.e.Start(s.config.ListenAddr)
}
