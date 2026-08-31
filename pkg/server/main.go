package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
	"gorm.io/gorm"

	"github.com/ustclug/Yuki/pkg/controlplane"
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
	return NewWithConfig(cfg)
}

func NewWithConfig(cfg Config) (*Server, error) {
	validate := InitValidator()
	if err := validate.Struct(&cfg); err != nil {
		return nil, err
	}

	db, err := gorm.Open(sqlite.Open(cfg.DbURL), &gorm.Config{
		QueryFields: true,
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

	s.e = s.newEcho()
	s.registerControlAPIs(s.e)

	return &s, nil
}

func (s *Server) newEcho() *echo.Echo {
	e := echo.New()
	validate := InitValidator()
	e.Validator = echoValidator(validate.Struct)
	e.Debug = s.config.Debug
	e.HideBanner = true
	e.Logger.SetOutput(io.Discard)

	// Middlewares.
	// The order matters.
	e.Use(middleware.RequestID())
	e.Use(setLogger(s.logger))
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
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
	return e
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

	controlEndpoint, err := controlplane.ParseEndpoint(s.config.ListenAddr)
	if err != nil {
		return fmt.Errorf("parse control endpoint: %w", err)
	}
	controlLn, err := s.listen(controlEndpoint)
	if err != nil {
		return err
	}
	s.e.Listener = controlLn

	l.Info("Scheduling tasks")
	s.scheduleTasks(ctx)

	go func() {
		l.Info("Running HTTP server", slog.String("endpoint", s.config.ListenAddr))
		if err := s.e.Start(""); err != nil && !errors.Is(err, http.ErrServerClosed) {
			wrapped := fmt.Errorf("run HTTP server: %w", err)
			l.Error("Fail to run HTTP server", slogErrAttr(err))
			cancel(wrapped)
		}
	}()

	<-ctx.Done()
	l.Info("Shutting down HTTP servers")
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

func (s *Server) listen(endpoint controlplane.Endpoint) (net.Listener, error) {
	if endpoint.Type == controlplane.EndpointUnix {
		if err := os.MkdirAll(filepath.Dir(endpoint.Address), 0755); err != nil {
			return nil, fmt.Errorf("create Unix socket directory %q: %w; the bundled systemd unit uses RuntimeDirectory=yuki, while manual starts require a writable parent directory", filepath.Dir(endpoint.Address), err)
		}
	}

	ln, err := net.Listen(string(endpoint.Type), endpoint.Address)
	if err != nil {
		if endpoint.Type == controlplane.EndpointUnix {
			return nil, fmt.Errorf("listen on Unix socket %q: %w; systemd cleans the default /run/yuki directory, while custom or manual deployments must verify that no yukid process is running before removing a stale socket", endpoint.Address, err)
		}
		return nil, fmt.Errorf("listen on endpoint %q: %w", endpoint.Address, err)
	}
	return ln, nil
}

func (s *Server) registerControlAPIs(e *echo.Echo) {
	v1API := e.Group("/api/v1/")
	v1API.GET("metas", s.handlerListRepoMetas)
	v1API.GET("metas/:name", s.handlerGetRepoMeta)

	v1API.GET("repos", s.handlerListRepos)
	v1API.GET("repos/:name", s.handlerGetRepo)
	v1API.DELETE("repos/:name", s.handlerRemoveRepo)
	v1API.POST("repos/:name", s.handlerReloadRepo)
	v1API.POST("repos", s.handlerReloadAllRepos)
	v1API.POST("repos/:name/sync", s.handlerSyncRepo)
}
