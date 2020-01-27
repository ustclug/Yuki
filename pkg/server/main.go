package server

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"reflect"
	"sync"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/core"
	"github.com/ustclug/Yuki/pkg/cron"
	"github.com/ustclug/Yuki/pkg/gpool"
)

type Server struct {
	e          *echo.Echo
	c          *core.Core
	gpool      gpool.Pool
	config     *Config
	cron       *cron.Cron
	syncStatus *sync.Map
	quit       chan struct{}

	preSyncCh  chan api.PreSyncPayload
	postSyncCh chan api.PostSyncPayload
}

func init() {
	viper.SetEnvPrefix("YUKI")
	viper.SetConfigFile("/etc/yuki/daemon.toml")
}

func New() (*Server, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	return NewWithConfig(cfg)
}

type varAndDefVal struct {
	Var    interface{}
	DefVal interface{}
}

func setDefault(us []varAndDefVal) {
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
	setDefault([]varAndDefVal{
		{&cfg.DbURL, DefaultServerConfig.DbURL},
		{&cfg.DbName, DefaultServerConfig.DbName},
		{&cfg.DockerEndpoint, DefaultServerConfig.DockerEndpoint},

		{&cfg.Owner, DefaultServerConfig.Owner},
		{&cfg.LogDir, DefaultServerConfig.LogDir},
		{&cfg.ListenAddr, DefaultServerConfig.ListenAddr},
		{&cfg.NamePrefix, DefaultServerConfig.NamePrefix},
		{&cfg.ImagesUpgradeInterval, DefaultServerConfig.ImagesUpgradeInterval},
	})

	if err := os.MkdirAll(cfg.LogDir, os.ModePerm); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.RepoConfigDir, os.ModePerm); err != nil {
		return nil, err
	}
	coreCfg := core.Config{
		Debug:          cfg.Debug,
		DbURL:          cfg.DbURL,
		DbName:         cfg.DbName,
		GetSizer:       cfg.GetSizer,
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
		syncStatus: new(sync.Map),

		quit:       make(chan struct{}),
		preSyncCh:  make(chan api.PreSyncPayload),
		postSyncCh: make(chan api.PostSyncPayload),
	}
	s.e.Validator = &myValidator{NewValidator()}

	logrus.SetLevel(cfg.LogLevel)

	s.e.Debug = cfg.Debug
	s.e.HideBanner = true

	s.e.HTTPErrorHandler = s.HTTPErrorHandler
	s.e.Logger.SetOutput(ioutil.Discard)

	logfile, err := os.OpenFile(path.Join(cfg.LogDir, "yukid.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	logrus.SetFormatter(new(logrus.TextFormatter))
	logrus.SetOutput(logfile)

	logrus.Info("Cleaning dead containers")
	s.cleanDeadContainers()

	s.waitRunningContainers()

	s.schedRepos()
	s.c.InitMetas()

	_, err = s.cron.AddFunc(cfg.ImagesUpgradeInterval, func() {
		logrus.Info("Upgrading images")
		s.upgradeImages()
		logrus.Info("Cleaning images")
		s.cleanImages()
	})
	if err != nil {
		return nil, err
	}

	// middlewares
	s.e.Use(middleware.BodyLimit("2M"))

	g := s.e.Group("/api/v1/")
	s.registerAPIs(g)

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
		ct, err := s.c.Sync(core.SyncOptions{
			LogDir:        s.config.LogDir,
			DefaultOwner:  s.config.Owner,
			NamePrefix:    s.config.NamePrefix,
			Name:          name,
			MountDir:      true,
			DefaultBindIP: s.config.BindIP,
		})
		if err != nil {
			logrus.WithField("repo", name).Errorln(err)
			return
		}
		logrus.Infof("Syncing %s", name)
		if err := s.waitForSync(ct); err != nil {
			logrus.WithField("repo", name).Errorln(err)
		}
	}
}

func (s *Server) Start() {
	logrus.Infof("Listening at %s", s.config.ListenAddr)
	go func() {
		if err := s.e.Start(s.config.ListenAddr); err != nil {
			logrus.Warnf("Shutting down the server: %v", err)
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
					logrus.Errorln("Failed to connect to MongoDB, exit...")
					s.quit <- struct{}{}
					return
				}
				logrus.Warnf("Failed to connect to MongoDB: %d", fail)
			} else if fail != 0 {
				logrus.Warnln("Reconnected to MongoDB")
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
	s.c.MgoSess.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if err := s.e.Shutdown(ctx); err != nil {
		logrus.Errorln(err)
	}
}

func (s *Server) upgradeImages() {
	var images []string
	err := s.c.FindRepository(nil).Distinct("image", &images)
	if err != nil {
		return
	}
	var wg sync.WaitGroup
	wg.Add(len(images))
	for _, i := range images {
		img := i
		s.gpool.Submit(func() {
			err := s.c.PullImage(img)
			wg.Done()
			if err != nil {
				logrus.Errorf("pullImage: %s", err)
			}
		})
	}
	wg.Wait()
}

func (s *Server) cleanImages() {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*5)
	defer cancel()
	imgs, err := s.c.Docker.ListImages(docker.ListImagesOptions{
		All:     true,
		Context: ctx,
		Filters: map[string][]string{
			"dangling": {"true"},
			"label":    {"org.ustcmirror.images=true"},
		},
	})
	if err != nil {
		return
	}
	for _, i := range imgs {
		err := s.c.Docker.RemoveImage(i.ID)
		if err != nil {
			logrus.Errorf("removeImage: %s", err)
		}
	}
}

func (s *Server) onPreSync() {
	for {
		select {
		case <-s.quit:
			return
		case data := <-s.preSyncCh:
			s.syncStatus.Store(data.Name, struct{}{})
		}
	}
}

func (s *Server) onPostSync() {
	cmds := s.config.PostSync
	for {
		select {
		case <-s.quit:
			return
		case data := <-s.postSyncCh:
			s.syncStatus.Delete(data.ID)
			err := s.c.RemoveContainer(data.ID)
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
func (s *Server) cleanDeadContainers() {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*10)
	defer cancel()
	opts := docker.ListContainersOptions{
		Context: ctx,
		All:     true,
		Filters: map[string][]string{
			"label":  {"org.ustcmirror.syncing=true"},
			"status": {"created", "exited", "dead"},
		},
	}
	cts, err := s.c.Docker.ListContainers(opts)
	if err != nil {
		logrus.Errorf("listContainers: %s", err)
		return
	}
	for _, ct := range cts {
		err := s.c.RemoveContainer(ct.ID)
		if err != nil {
			logrus.WithField("container", ct.Names[0]).Errorf("removeContainer: %s", err)
		}
	}
}

// waitRunningContainers waits for all syncing containers to stop and remove them.
func (s *Server) waitRunningContainers() {
	opts := docker.ListContainersOptions{
		All: true,
		Filters: map[string][]string{
			"label":  {"org.ustcmirror.syncing=true"},
			"status": {"running"},
		},
	}
	cts, err := s.c.Docker.ListContainers(opts)
	if err != nil {
		logrus.Errorf("listContainers: %s", err)
		return
	}
	for _, ct := range cts {
		go func(ct docker.APIContainers) {
			if err := s.waitForSync(&api.Container{
				ID:     ct.ID,
				Labels: ct.Labels,
			}); err != nil {
				logrus.WithField("container", ct.Names[0]).Errorf("waitForSync: %s", err)
			}
		}(ct)
	}
}

// waitForSync emits `SyncStart` event at first, then blocks until the container stops and emits the `SyncEnd` event.
func (s *Server) waitForSync(ct *api.Container) error {
	s.preSyncCh <- api.PreSyncPayload{
		Name: ct.Labels["org.ustcmirror.name"],
	}

	code, err := s.c.Docker.WaitContainer(ct.ID)
	if err != nil {
		return err
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
	return nil
}

// initMetas creates metadata for each Repository.
// func (s *Server) initMetas() {
// 	repos := s.c.ListAllRepositories()
// 	now := time.Now().Unix()
// 	for _, r := range repos {
// 		size := s.c.GetSize(r.StorageDir)
// 		_, _ = c.metaColl.UpsertId(r.Name, bson.M{
// 			"$set": bson.M{
// 				"size": size,
// 			},
// 			"$setOnInsert": bson.M{
// 				"createdAt": now,
// 				"exitCode":  -1,
// 			},
// 		})
// 	}
// }
