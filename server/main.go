package server

import (
	"context"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"reflect"
	"sync"
	"time"

	"github.com/gorilla/sessions"
	"github.com/knight42/Yuki/core"
	"github.com/knight42/Yuki/cron"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/middleware"
	"github.com/sirupsen/logrus"
)

var ctxLogFieldKey = "logfields"

type Server struct {
	e          *echo.Echo
	c          *core.Core
	config     *Config
	cron       *cron.Cron
	syncStatus *sync.Map
	quit       chan struct{}
	logger     *logrus.Logger
}

func New() (*Server, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	return NewWithConfig(cfg)
}

type valTuple struct {
	Var    interface{}
	DefVal interface{}
}

func setDefault(us []valTuple) {
	for _, u := range us {
		val := reflect.ValueOf(u.Var)
		dstType := val.Type().Elem()
		dstVal := reflect.Indirect(val)
		if reflect.DeepEqual(dstVal.Interface(), reflect.Zero(dstType).Interface()) {
			dstVal.Set(reflect.ValueOf(u.DefVal))
		}
	}
}

func NewWithConfig(cfg *Config) (*Server, error) {
	setDefault([]valTuple{
		{&cfg.DbURL, DefaultServerConfig.DbURL},
		{&cfg.DbName, DefaultServerConfig.DbName},
		{&cfg.SessionAge, DefaultServerConfig.SessionAge},
		{&cfg.DockerEndpoint, DefaultServerConfig.DockerEndpoint},

		{&cfg.Owner, DefaultServerConfig.Owner},
		{&cfg.LogDir, DefaultServerConfig.LogDir},
		{&cfg.ListenAddr, DefaultServerConfig.ListenAddr},
		{&cfg.NamePrefix, DefaultServerConfig.NamePrefix},
		{&cfg.AllowOrigins, DefaultServerConfig.AllowOrigins},
		{&cfg.ImagesUpgradeInterval, DefaultServerConfig.ImagesUpgradeInterval},
	})

	if err := os.MkdirAll(cfg.LogDir, os.ModePerm); err != nil {
		return nil, err
	}
	coreCfg := core.Config{
		Debug:          cfg.Debug,
		DbURL:          cfg.DbURL,
		DbName:         cfg.DbName,
		GetSizer:       cfg.GetSizer,
		SessionAge:     cfg.SessionAge,
		DockerEndpoint: cfg.DockerEndpoint,
	}
	c, err := core.NewWithConfig(coreCfg)
	if err != nil {
		return nil, err
	}
	s := Server{
		c:          c,
		e:          echo.New(),
		cron:       cron.New(),
		config:     cfg,
		logger:     logrus.New(),
		syncStatus: new(sync.Map),
		quit:       make(chan struct{}),
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
	s.logger.Formatter = new(logrus.TextFormatter)
	s.logger.Out = logfile

	s.hookAllEvents()

	s.logger.Info("Cleaning dead containers")
	s.c.CleanDeadContainers()

	s.c.WaitRunningContainers()

	s.schedRepos()
	s.c.InitMetas()

	s.cron.AddFunc(cfg.ImagesUpgradeInterval, func() {
		s.logger.Info("Upgrading images")
		s.c.UpgradeImages()
		s.logger.Info("Cleaning images")
		s.c.CleanImages()
	})

	// middlewares
	secureCfg := middleware.DefaultSecureConfig
	secureCfg.HSTSMaxAge = 31536000
	s.e.Use(middleware.SecureWithConfig(secureCfg))

	corsCfg := middleware.DefaultCORSConfig
	corsCfg.AllowOrigins = cfg.AllowOrigins
	s.e.Use(middleware.CORSWithConfig(corsCfg))

	s.e.Use(middleware.BodyLimit("2M"))
	s.e.Use(session.Middleware(sessions.NewCookieStore([]byte(s.config.CookieKey))))
	s.e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(ctxLogFieldKey, logrus.Fields{})
			return next(c)
		}
	})

	g := s.e.Group("/api/v1/")
	s.registerAPIs(g)

	return &s, nil
}

func (s *Server) schedRepos() {
	repos := s.c.ListAllRepositories()
	s.logger.Infof("Scheduling %d repositories", len(repos))
	for _, r := range repos {
		if err := s.cron.AddJob(r.Name, r.Interval, s.newJob(r.Name)); err != nil {
			s.logger.WithField("repo", r.Name).Errorln(err)
		}
	}
}

func (s *Server) newJob(name string) cron.FuncJob {
	return func() {
		ct, err := s.c.Sync(core.SyncOptions{
			LogDir:        s.config.LogDir,
			DefaultOwner:  s.config.Owner,
			NamePrefix:    s.config.NamePrefix,
			Name:          name,
			MountDir:      true,
			DefaultBindIP: s.config.BindIP,
		})
		if err != nil {
			s.logger.WithField("repo", name).Errorln(err)
			return
		}
		s.logger.Infof("Syncing %s", name)
		if err := s.c.WaitForSync(*ct); err != nil {
			s.logger.WithField("repo", name).Errorln(err)
		}
	}
}

func (s *Server) Start() {
	s.logger.Infof("Listening at %s", s.config.ListenAddr)
	go func() {
		if err := s.e.Start(s.config.ListenAddr); err != nil {
			s.logger.Warnf("Shutting down the server: %v", err)
		}
	}()

	go func() {
		c := time.Tick(time.Second * 20)
		fail := 0
		const threshold int = 3
		for range c {
			if err := s.c.MgoSess.Ping(); err != nil {
				fail++
				if fail > threshold {
					s.logger.Errorln("Failed to connect to MongoDB, exit...")
					s.quit <- struct{}{}
					return
				}
				s.logger.Warnf("Failed to connect to MongoDB: %d", fail)
			} else if fail != 0 {
				s.logger.Warnln("Reconnected to MongoDB")
				fail = 0
			}
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 10 seconds.
	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt)

	select {
	case <-signals:
	case <-s.quit:
	}
	s.teardown()
}

func (s *Server) teardown() {
	s.config.Authenticator.Cleanup()
	s.c.MgoSess.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if err := s.e.Shutdown(ctx); err != nil {
		s.logger.Errorln(err)
	}
}
