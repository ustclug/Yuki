package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/ustclug/Yuki/pkg/cron"
	"github.com/ustclug/Yuki/pkg/docker"
	"github.com/ustclug/Yuki/pkg/model"
)

type Server struct {
	syncingContainers sync.Map

	e         *echo.Echo
	dockerCli docker.Client
	config    *Config
	cron      *cron.Cron
	db        *gorm.DB
	logger    *slog.Logger
	getSize   func(string) int64
}

func New(configPath string) (*Server, error) {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return nil, err
	}
	return NewWithConfig(cfg)
}

func newSlogger(writer io.Writer, addSource bool, level slog.Leveler) *slog.Logger {
	return slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{
		AddSource: addSource,
		Level:     level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Taken from https://gist.github.com/HalCanary/6bd335057c65f3b803088cc55b9ebd2b
			if a.Key == slog.SourceKey {
				source, _ := a.Value.Any().(*slog.Source)
				if source != nil {
					_, after, _ := strings.Cut(source.File, "Yuki")
					source.File = after
				}
			}
			return a
		},
	}))
}

func NewWithConfig(cfg *Config) (*Server, error) {
	db, err := gorm.Open(sqlite.Open(cfg.DbURL), &gorm.Config{
		QueryFields: true,
	})
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := os.MkdirAll(cfg.LogDir, os.ModePerm); err != nil {
		return nil, err
	}
	for _, dir := range cfg.RepoConfigDir {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return nil, err
		}
	}

	dockerCli, err := docker.NewClient(cfg.DockerEndpoint)
	if err != nil {
		return nil, err
	}

	// workaround a systemd bug.
	// See also https://github.com/ustclug/Yuki/issues/4
	logfile, err := os.OpenFile(filepath.Join(cfg.LogDir, "yukid.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	slogger := newSlogger(logfile, cfg.Debug, cfg.LogLevel)

	s := Server{
		e:         echo.New(),
		cron:      cron.New(),
		db:        db,
		logger:    slogger,
		dockerCli: dockerCli,
		config:    cfg,

		getSize: cfg.GetSizer.GetSize,
	}

	v := validator.New()
	s.e.Validator = echoValidator(v.Struct)
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
			l.LogAttrs(context.Background(), slog.LevelInfo, "REQUEST", attrs...)
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

	err = s.cron.AddFunc(s.config.ImagesUpgradeInterval, s.upgradeImages)
	if err != nil {
		return fmt.Errorf("add cronjob to upgrade images: %w", err)
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

	l.Info("Scheduling repos")
	err = s.scheduleRepos()
	if err != nil {
		return fmt.Errorf("schedule repos: %w", err)
	}

	l.Info("Initializing repo metas")
	err = s.initRepoMetas()
	if err != nil {
		return fmt.Errorf("init meta: %w", err)
	}

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
