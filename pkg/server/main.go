package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/errdefs"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/core"
	"github.com/ustclug/Yuki/pkg/cron"
)

type Server struct {
	e          *echo.Echo
	c          *core.Core
	syncStatus sync.Map
	config     *Config
	cron       *cron.Cron
	db         *gorm.DB
	logger     *slog.Logger
	preSyncCh  chan api.PreSyncPayload
	postSyncCh chan api.PostSyncPayload

	getSize func(string) int64
}

func New() (*Server, error) {
	cfg, err := LoadConfig()
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

	// workaround a systemd bug.
	// See also https://github.com/ustclug/Yuki/issues/4
	logfile, err := os.OpenFile(filepath.Join(cfg.LogDir, "yukid.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	slogger := newSlogger(logfile, cfg.Debug, cfg.LogLevel)
	s := Server{
		e:          echo.New(),
		cron:       cron.New(),
		db:         db,
		logger:     slogger,
		config:     cfg,
		preSyncCh:  make(chan api.PreSyncPayload),
		postSyncCh: make(chan api.PostSyncPayload),

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

func (s *Server) schedRepos() {
	repos := s.c.ListAllRepositories()
	logrus.Infof("Scheduling %d repositories", len(repos))
	for _, r := range repos {
		if err := s.cron.AddJob(r.Name, r.Interval, s.newJob(r.Name)); err != nil {
			logrus.WithField("repo", r.Name).Errorln(err)
		}
	}
}

func (s *Server) newJob(name string) cron.FuncJob {
	return func() {
		// FIXME: determine the real context
		ct, err := s.c.Sync(context.TODO(), core.SyncOptions{
			LogDir:         s.config.LogDir,
			DefaultOwner:   s.config.Owner,
			NamePrefix:     s.config.NamePrefix,
			Name:           name,
			MountDir:       true,
			DefaultBindIP:  s.config.BindIP,
			SeccompProfile: s.config.SeccompProfile,
		})
		entry := logrus.WithField("repo", name)
		if err != nil {
			if errdefs.IsConflict(err) {
				entry.Warningln(err)
			} else {
				entry.Errorln(err)
			}
			return
		}
		logrus.Infof("Syncing %s", name)
		if err := s.waitForSync(ct); err != nil {
			entry.Warningln(err)
		}
	}
}

func (s *Server) Start(rootCtx context.Context) {
	l := s.logger
	ctx, cancel := context.WithCancelCause(rootCtx)
	defer cancel(context.Canceled)

	go func() {
		l.Info("Running HTTP server", slog.String("addr", s.config.ListenAddr))
		if err := s.e.Start(s.config.ListenAddr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			l.Error("Fail to run HTTP server", slogErrAttr(err))
			cancel(err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.onPreSync(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		s.onPostSync(ctx)
	}()

	/*
		 s.cleanDeadContainers(ctx)

		s.waitRunningContainers()

		s.schedRepos()
		s.c.InitMetas()
	*/

	err := s.cron.AddFunc(s.config.ImagesUpgradeInterval, func() {
		logrus.Info("Upgrading images")
		s.upgradeImages(ctx)
		logrus.Info("Cleaning images")
		s.cleanImages(ctx)
	})
	if err != nil {
		logrus.Fatalf("cron.AddFunc: %s", err)
	}

	<-ctx.Done()
	l.Info("Waiting for goroutines to exit")
	wg.Wait()
	l.Info("Shutting down HTTP server")
	_ = s.e.Shutdown(context.Background())
}

func (s *Server) upgradeImages(ctx context.Context) {
	var images []string
	err := s.c.FindRepository(nil).Distinct("image", &images)
	if err != nil {
		return
	}
	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(5)
	for _, i := range images {
		img := i
		eg.Go(func() error {
			pullCtx, cancel := context.WithTimeout(egCtx, time.Minute*5)
			defer cancel()
			err := s.c.PullImage(pullCtx, img)
			if err != nil {
				logrus.Warningf("pullImage: %s", err)
			}
			return nil
		})
	}
	_ = eg.Wait()
}

func (s *Server) cleanImages(rootCtx context.Context) {
	ctx, cancel := context.WithTimeout(rootCtx, time.Second*5)
	defer cancel()
	imgs, err := s.c.ListImages(ctx, map[string][]string{
		"label":    {"org.ustcmirror.images=true"},
		"dangling": {"true"},
	})
	if err != nil {
		logrus.Warningf("listImages: %s", err)
		return
	}
	for _, i := range imgs {
		ctx, cancel := context.WithTimeout(rootCtx, time.Second*5)
		err := s.c.RemoveImage(ctx, i.ID)
		cancel()
		if err != nil {
			logrus.Warningf("removeImage: %s", err)
		}
	}
}

func (s *Server) onPreSync(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case data := <-s.preSyncCh:
			err := s.c.UpdatePrevRun(data.Name)
			if err != nil {
				logrus.WithField("repo", data.Name).Errorf("failed to update prevRun: %s", err)
			}
			s.syncStatus.Store(data.Name, struct{}{})
		}
	}
}

func (s *Server) onPostSync(ctx context.Context) {
	cmds := s.config.PostSync
	for {
		select {
		case <-ctx.Done():
			return
		case data := <-s.postSyncCh:
			s.syncStatus.Delete(data.Name)
			rmCtx, cancel := context.WithTimeout(ctx, time.Second*20)
			err := s.c.RemoveContainer(rmCtx, data.ID)
			cancel()
			entry := logrus.WithField("repo", data.Name)
			if err != nil {
				entry.Errorf("failed to remove container: %s", err)
			}
			err = s.c.UpsertRepoMeta(data.Name, data.Dir, data.ExitCode)
			if err != nil {
				entry.Errorf("failed to upsert repo meta: %s", err)
			}
			envs := []string{
				fmt.Sprintf("ID=%s", data.ID),
				fmt.Sprintf("Name=%s", data.Name),
				fmt.Sprintf("Dir=%s", data.Dir),
				fmt.Sprintf("ExitCode=%d", data.ExitCode),
			}
			go func() {
				for _, cmd := range cmds {
					prog := exec.Command("sh", "-c", cmd)
					prog.Env = envs
					if err := prog.Run(); err != nil {
						logrus.WithFields(logrus.Fields{
							"command": cmd,
						}).Errorln(err)
					}
				}
			}()
		}
	}
}

// cleanDeadContainers removes containers which status are `created`, `exited` or `dead`.
func (s *Server) cleanDeadContainers(ctx context.Context) {
	logrus.Info("Cleaning dead containers")

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*10)
	defer cancel()
	cts, err := s.c.ListContainers(ctx, map[string][]string{
		"label":  {"org.ustcmirror.syncing=true"},
		"status": {"created", "exited", "dead"},
	})
	if err != nil {
		logrus.Errorf("listContainers: %s", err)
		return
	}
	for _, ct := range cts {
		rmCtx, cancel := context.WithTimeout(ctx, time.Second*20)
		err := s.c.RemoveContainer(rmCtx, ct.ID)
		cancel()
		if err != nil {
			logrus.WithField("container", ct.Names[0]).Errorf("removeContainer: %s", err)
		}
	}
}

// waitRunningContainers waits for all syncing containers to stop and remove them.
func (s *Server) waitRunningContainers() {
	cts, err := s.c.ListContainers(context.Background(), map[string][]string{
		"label":  {"org.ustcmirror.syncing=true"},
		"status": {"running"},
	})
	if err != nil {
		logrus.Errorf("listContainers: %s", err)
		return
	}
	for _, ct := range cts {
		go func(ct types.Container) {
			if err := s.waitForSync(&api.Container{
				ID:     ct.ID,
				Labels: ct.Labels,
			}); err != nil {
				logrus.WithField("container", ct.Names[0]).Warningf("waitForSync: %s", err)
			}
		}(ct)
	}
}

// waitForSync emits `SyncStart` event at first, then blocks until the container stops and emits the `SyncEnd` event.
func (s *Server) waitForSync(ct *api.Container) error {
	s.preSyncCh <- api.PreSyncPayload{
		Name: ct.Labels["org.ustcmirror.name"],
	}

	// FIXME: determine the real context
	ctx := context.TODO()
	if s.config.SyncTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.config.SyncTimeout)
		defer cancel()
	}

	code, err := s.c.WaitContainer(ctx, ct.ID)
	if err != nil {
		if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return err
		} else {
			// When the error is timeout, we expect that
			// container will be stopped and removed in onPostSync() goroutine
			// Here we set a special exit code to indicate that the container is timeout in meta.
			code = -2
		}
	}

	name, ok := ct.Labels["org.ustcmirror.name"]
	if !ok {
		return fmt.Errorf("missing label: org.ustcmirror.name")
	}
	dir, ok := ct.Labels["org.ustcmirror.storage-dir"]
	if !ok {
		return fmt.Errorf("missing label: org.ustcmirror.storage-dir")
	}

	s.postSyncCh <- api.PostSyncPayload{
		ID:       ct.ID,
		Name:     name,
		Dir:      dir,
		ExitCode: code,
	}
	// returns context.DeadlineExceeded when timeout
	// or nil when it succeeded
	return err
}
