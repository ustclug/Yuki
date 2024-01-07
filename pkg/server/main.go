package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/ustclug/Yuki/pkg/docker"
	"github.com/ustclug/Yuki/pkg/fs"
	"github.com/ustclug/Yuki/pkg/model"
)

type Server struct {
	repoSchedules cmap.ConcurrentMap[string, cron.Schedule]

	e         *echo.Echo
	dockerCli docker.Client
	config    Config
	db        *gorm.DB
	logger    *slog.Logger
	getSize   func(string) int64
}

func New(configPath string) (*Server, error) {
	v := viper.New()
	v.SetConfigFile(configPath)
	err := v.ReadInConfig()
	if err != nil {
		return nil, err
	}
	cfg := DefaultConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	validate := validator.New()
	if err := validate.Struct(&cfg); err != nil {
		return nil, err
	}
	return NewWithConfig(cfg)
}

func NewWithConfig(cfg Config) (*Server, error) {
	// TODO: enforce shared cache mode?
	db, err := gorm.Open(sqlite.Open(cfg.DbURL), &gorm.Config{
		QueryFields:            true,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	dockerCli, err := docker.NewClient(cfg.DockerEndpoint)
	if err != nil {
		return nil, err
	}

	// workaround a systemd bug.
	// See also https://github.com/ustclug/Yuki/issues/4
	logfile, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	var logLvl slog.Level
	switch cfg.LogLevel {
	case "debug":
		logLvl = slog.LevelDebug
	case "warn":
		logLvl = slog.LevelWarn
	case "error":
		logLvl = slog.LevelError
	default:
		logLvl = slog.LevelInfo
	}

	slogger := newSlogger(logfile, cfg.Debug, logLvl)

	s := Server{
		e:             echo.New(),
		db:            db,
		logger:        slogger,
		dockerCli:     dockerCli,
		config:        cfg,
		repoSchedules: cmap.New[cron.Schedule](),
	}
	switch cfg.FileSystem {
	case "zfs":
		s.getSize = fs.New(fs.ZFS).GetSize
	case "xfs":
		s.getSize = fs.New(fs.XFS).GetSize
	default:
		s.getSize = fs.New(fs.DEFAULT).GetSize
	}

	validate := validator.New()
	s.e.Validator = echoValidator(validate.Struct)
	s.e.Debug = cfg.Debug
	s.e.HideBanner = true
	s.e.Logger.SetOutput(io.Discard)

	// Middlewares.
	// The order matters.
	s.e.Use(middleware.RequestID())
	s.e.Use(setLogger(slogger))
	s.e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:    true,
		LogLatency:   true,
		LogUserAgent: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			attrs := []slog.Attr{
				slog.Int("status", v.Status),
				slog.String("user_agent", v.UserAgent),
				slog.Duration("latency", v.Latency),
			}
			l := getLogger(c)
			l.LogAttrs(context.Background(), slog.LevelDebug, "REQUEST", attrs...)
			return nil
		},
	}))

	s.registerAPIs(s.e)

	return &s, nil
}

func (s *Server) Start(rootCtx context.Context) error {
	l := s.logger
	ctx, cancel := context.WithCancelCause(rootCtx)
	defer cancel(context.Canceled)

	l.Info("Initializing database")
	err := model.AutoMigrate(s.db)
	if err != nil {
		return fmt.Errorf("init db: %w", err)
	}

	l.Info("Initializing repo metas")
	err = s.initRepoMetas()
	if err != nil {
		return fmt.Errorf("init meta: %w", err)
	}

	l.Info("Cleaning dead containers")
	err = s.cleanDeadContainers()
	if err != nil {
		return fmt.Errorf("clean dead containers: %w", err)
	}

	l.Info("Waiting running containers")
	err = s.waitRunningContainers()
	if err != nil {
		return fmt.Errorf("wait running containers: %w", err)
	}

	l.Info("Scheduling tasks")
	s.scheduleTasks(ctx)

	go func() {
		l.Info("Running HTTP server", slog.String("addr", s.config.ListenAddr))
		if err := s.e.Start(s.config.ListenAddr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			l.Error("Fail to run HTTP server", slogErrAttr(err))
			cancel(err)
		}
	}()

	<-ctx.Done()
	l.Info("Shutting down HTTP server")
	_ = s.e.Shutdown(context.Background())

	caused := context.Cause(ctx)
	if errors.Is(caused, context.Canceled) {
		return nil
	}
	return caused
}

// ListenAddr returns the actual address the server is listening on.
// It is useful when the server is configured to listen on a random port.
func (s *Server) ListenAddr() string {
	return s.e.Listener.Addr().String()
}

func (s *Server) registerAPIs(e *echo.Echo) {
	v1API := e.Group("/api/v1/")

	// public APIs
	v1API.GET("metas", s.handlerListRepoMetas)
	v1API.GET("metas/:name", s.handlerGetRepoMeta)

	// private APIs
	v1API.GET("repos", s.handlerListRepos)
	v1API.GET("repos/:name", s.handlerGetRepo)
	v1API.DELETE("repos/:name", s.handlerRemoveRepo)
	v1API.POST("repos/:name", s.handlerReloadRepo)
	v1API.POST("repos", s.handlerReloadAllRepos)
	v1API.POST("repos/:name/sync", s.handlerSyncRepo)
}
