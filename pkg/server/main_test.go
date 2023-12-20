package server

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/core"
	"github.com/ustclug/Yuki/pkg/cron"
	"github.com/ustclug/Yuki/pkg/model"
)

func getTestingServer(ctx context.Context, prefix string, name string, storageDir string) (*Server, error) {
	s, err := New()
	if err != nil {
		return nil, err
	}

	defer os.RemoveAll("/tmp/log_yuki")
	defer os.RemoveAll("/tmp/config_yuki")

	err = os.MkdirAll(storageDir, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(storageDir)
	// remove existing repository and container
	_ = s.c.RemoveRepository(name)
	_ = s.c.RemoveContainer(ctx, prefix+name)

	go s.onPreSync(context.TODO())
	go s.onPostSync(context.TODO())

	return s, nil
}

func TestWaitForSync(t *testing.T) {
	as := assert.New(t)
	req := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	viper.Reset()
	viper.SetConfigFile("../../test/fixtures/sync_timeout.toml")

	prefix := "syncing-"
	name := "yuki-wait-test-sync-repo"
	d := "/tmp/" + name
	cycle := 10

	s, err := getTestingServer(ctx, prefix, name, d)
	req.Nil(err)

	err = s.c.AddRepository(api.Repository{
		Name:        name,
		Interval:    "1 * * * *",
		Image:       "ustcmirror/test:latest",
		StorageDir:  d,
		LogRotCycle: &cycle,
		Envs:        map[string]string{"SLEEP_INFINITY": "1"},
	})
	as.Nil(err)

	ct, err := s.c.Sync(ctx, core.SyncOptions{
		MountDir:   false,
		Name:       name,
		NamePrefix: prefix,
	})
	as.Nil(err)
	defer s.c.RemoveContainer(ctx, ct.ID)
	err = s.waitForSync(ct)
	if err != context.DeadlineExceeded && err.Error() != "" {
		t.Fatalf("Expected error to be context.DeadlineExceeded, got %v", err)
	}
}

type TestEnv struct {
	t       *testing.T
	httpSrv *httptest.Server
	server  *Server

	ctx context.Context
}

func (t *TestEnv) RESTClient() *resty.Client {
	return resty.New().SetBaseURL(t.httpSrv.URL + "/api/v1")
}

func NewTestEnv(t *testing.T) *TestEnv {
	slogger := newSlogger(os.Stderr, true, slog.LevelInfo)

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		QueryFields: true,
	})
	require.NoError(t, err)
	require.NoError(t, model.AutoMigrate(db))

	s := &Server{
		e:      e,
		db:     db,
		cron:   cron.New(),
		logger: slogger,
	}
	s.e.Use(setLogger(slogger))
	s.e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus: true,
		LogMethod: true,
		LogURI:    true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			attrs := []slog.Attr{
				slog.Int("status", v.Status),
				slog.String("method", v.Method),
				slog.String("uri", v.URI),
			}
			l := getLogger(c)
			l.LogAttrs(context.Background(), slog.LevelInfo, "REQUEST", attrs...)
			return nil
		},
	}))
	s.registerAPIs(e)
	srv := httptest.NewServer(e)
	return &TestEnv{
		t:       t,
		httpSrv: srv,
		server:  s,
	}
}
