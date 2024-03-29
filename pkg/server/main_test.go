package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/go-playground/validator/v10"
	"github.com/go-resty/resty/v2"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	fakedocker "github.com/ustclug/Yuki/pkg/docker/fake"
	"github.com/ustclug/Yuki/pkg/fs"
	"github.com/ustclug/Yuki/pkg/model"
)

type TestEnv struct {
	t       *testing.T
	httpSrv *httptest.Server
	server  *Server
}

func (t *TestEnv) RESTClient() *resty.Client {
	return resty.New().SetBaseURL(t.httpSrv.URL + "/api/v1")
}

func (t *TestEnv) RandomString() string {
	var buf [6]byte
	_, _ = rand.Read(buf[:])
	suffix := base64.RawURLEncoding.EncodeToString(buf[:])
	return t.t.Name() + suffix
}

func NewTestEnv(t *testing.T) *TestEnv {
	slogger := newSlogger(os.Stderr, true, slog.LevelInfo)

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	v := validator.New()
	e.Validator = echoValidator(v.Struct)

	dbFile, err := os.CreateTemp("", "yukid*.db")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = dbFile.Close()
		_ = os.Remove(dbFile.Name())
	})
	// Switch to WAL mode to avoid "database is locked" error.
	db, err := gorm.Open(sqlite.Open(dbFile.Name()), &gorm.Config{
		QueryFields: true,
	})
	require.NoError(t, err)
	require.NoError(t, model.AutoMigrate(db))

	s := &Server{
		e:         e,
		db:        db,
		logger:    slogger,
		dockerCli: fakedocker.NewClient(),
		getSize:   fs.New(fs.DEFAULT).GetSize,

		repoSchedules: cmap.New[cron.Schedule](),
	}
	s.e.Use(setLogger(slogger))
	s.e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus: true,
		LogMethod: true,
		LogURI:    true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			l := getLogger(c)
			l.LogAttrs(context.Background(), slog.LevelInfo, "REQUEST", slog.Int("status", v.Status))
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
