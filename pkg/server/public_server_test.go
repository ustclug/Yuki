package server

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"

	"github.com/ustclug/Yuki/pkg/controlplane"
	"github.com/ustclug/Yuki/pkg/model"
)

func TestPublicServerRoutes(t *testing.T) {
	te := NewTestEnv(t)
	require.NoError(t, te.server.db.Create(&model.RepoMeta{Name: "example"}).Error)
	e := te.server.newEcho()
	te.server.registerPublicAPIs(e)
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)
	cli := resty.New().SetBaseURL(srv.URL)

	for _, path := range []string{"/api/v1/metas", "/api/v1/metas/example"} {
		resp, err := cli.R().Get(path)
		require.NoError(t, err)
		require.NotEqual(t, http.StatusNotFound, resp.StatusCode(), path)
	}

	privateRequests := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/v1/repos"},
		{method: http.MethodPost, path: "/api/v1/repos"},
		{method: http.MethodPost, path: "/api/v1/repos/example/sync"},
		{method: http.MethodDelete, path: "/api/v1/repos/example"},
	}
	for _, req := range privateRequests {
		resp, err := cli.R().Execute(req.method, req.path)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, resp.StatusCode(), "%s %s", req.method, req.path)
	}
}

func TestListenDoesNotRemoveExistingUnixSocketPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "yukid.sock")
	require.NoError(t, os.WriteFile(path, []byte("keep me"), 0o600))

	_, err := (&Server{}).listen("control", controlplane.Endpoint{
		Type:    controlplane.EndpointUnix,
		Address: path,
	})
	require.ErrorContains(t, err, "systemd cleans the default /run/yuki directory")
	require.FileExists(t, path)
	contents, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	require.Equal(t, []byte("keep me"), contents)
}

func TestListenRejectsActiveUnixSocket(t *testing.T) {
	path := filepath.Join(t.TempDir(), "yukid.sock")
	ln, err := net.Listen("unix", path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	_, err = (&Server{}).listen("control", controlplane.Endpoint{
		Type:    controlplane.EndpointUnix,
		Address: path,
	})
	require.Error(t, err)
	require.FileExists(t, path)
}

func TestListenDoesNotRemoveStaleUnixSocket(t *testing.T) {
	path := filepath.Join(t.TempDir(), "yukid.sock")
	ln, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	require.NoError(t, err)
	ln.SetUnlinkOnClose(false)
	require.NoError(t, ln.Close())

	_, err = (&Server{}).listen("control", controlplane.Endpoint{
		Type:    controlplane.EndpointUnix,
		Address: path,
	})
	require.ErrorContains(t, err, "verify that no yukid process is running")
	require.FileExists(t, path)
}

func TestPublicListenAddrValidation(t *testing.T) {
	cfg := DefaultConfig
	cfg.DbURL = ":memory:"
	tmpDir := t.TempDir()
	cfg.RepoLogsDir = tmpDir
	cfg.RepoConfigDir = []string{tmpDir}
	validate := InitValidator()

	for _, address := range []string{"", "127.0.0.1:9999", "localhost:9999"} {
		cfg.PublicListenAddr = address
		require.NoError(t, validate.Struct(cfg), address)
	}

	for _, address := range []string{"/run/yuki/public.sock", "http://127.0.0.1:9999/", "localhost"} {
		cfg.PublicListenAddr = address
		require.Error(t, validate.Struct(cfg), address)
	}
}

func TestStartSeparatesPublicAndControlListeners(t *testing.T) {
	te := NewTestEnv(t)
	te.httpSrv.Close()
	s := te.server
	s.config.ListenAddr = filepath.Join(t.TempDir(), "yukid.sock")
	s.config.PublicListenAddr = availableTCPAddress(t)
	s.config.RepoConfigDir = []string{t.TempDir()}
	s.config.RepoLogsDir = t.TempDir()
	s.e = s.newEcho()
	s.publicE = s.newEcho()
	s.registerControlAPIs(s.e)
	s.registerPublicAPIs(s.publicE)

	cancel, done := startTestServer(t, s)
	t.Cleanup(cancel)
	publicClient := resty.New().SetBaseURL("http://" + s.config.PublicListenAddr)
	var resp *resty.Response
	require.Eventually(t, func() bool {
		var err error
		resp, err = publicClient.R().Get("/api/v1/metas")
		return err == nil
	}, time.Second, 10*time.Millisecond)
	require.Equal(t, http.StatusOK, resp.StatusCode())
	resp, err := publicClient.R().Get("/api/v1/repos")
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode())

	cancel()
	require.NoError(t, <-done)
}

func TestStartMergesIdenticalTCPListeners(t *testing.T) {
	te := NewTestEnv(t)
	te.httpSrv.Close()
	s := te.server
	address := availableTCPAddress(t)
	s.config.ListenAddr = address
	s.config.PublicListenAddr = address
	s.config.RepoConfigDir = []string{t.TempDir()}
	s.config.RepoLogsDir = t.TempDir()
	s.e = s.newEcho()
	s.publicE = s.newEcho()
	s.registerControlAPIs(s.e)
	s.registerPublicAPIs(s.publicE)

	cancel, done := startTestServer(t, s)
	t.Cleanup(cancel)
	client := resty.New().SetBaseURL("http://" + address)
	var resp *resty.Response
	require.Eventually(t, func() bool {
		var err error
		resp, err = client.R().Get("/api/v1/repos")
		return err == nil
	}, time.Second, 10*time.Millisecond)
	require.Equal(t, http.StatusOK, resp.StatusCode())

	cancel()
	require.NoError(t, <-done)
}

func TestStartFailsWhenPublicListenerCannotBind(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = occupied.Close() })

	te := NewTestEnv(t)
	te.httpSrv.Close()
	s := te.server
	socketPath := filepath.Join(t.TempDir(), "yukid.sock")
	s.config.ListenAddr = socketPath
	s.config.PublicListenAddr = occupied.Addr().String()
	s.config.RepoConfigDir = []string{t.TempDir()}
	s.config.RepoLogsDir = t.TempDir()
	s.e = s.newEcho()
	s.publicE = s.newEcho()
	s.registerControlAPIs(s.e)
	s.registerPublicAPIs(s.publicE)

	err = s.Start(context.Background())
	require.ErrorContains(t, err, "listen on public endpoint")
	require.NoFileExists(t, socketPath)
}

func startTestServer(t *testing.T, s *Server) (context.CancelFunc, <-chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Start(ctx) }()
	return cancel, done
}

func availableTCPAddress(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := ln.Addr().String()
	require.NoError(t, ln.Close())
	return address
}
