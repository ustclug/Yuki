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

func TestControlServerMetaRoutes(t *testing.T) {
	te := NewTestEnv(t)
	require.NoError(t, te.server.db.Create(&model.RepoMeta{Name: "example"}).Error)
	e := te.server.newEcho()
	te.server.registerControlAPIs(e)
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)
	cli := resty.New().SetBaseURL(srv.URL)

	for _, path := range []string{"/api/v1/metas", "/api/v1/metas/example"} {
		resp, err := cli.R().Get(path)
		require.NoError(t, err)
		require.NotEqual(t, http.StatusNotFound, resp.StatusCode(), path)
	}
}

func TestListenDoesNotRemoveExistingUnixSocketPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "yukid.sock")
	require.NoError(t, os.WriteFile(path, []byte("keep me"), 0o600))

	_, err := (&Server{}).listen(controlplane.Endpoint{
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

	_, err = (&Server{}).listen(controlplane.Endpoint{
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

	_, err = (&Server{}).listen(controlplane.Endpoint{
		Type:    controlplane.EndpointUnix,
		Address: path,
	})
	require.ErrorContains(t, err, "verify that no yukid process is running")
	require.FileExists(t, path)
}

func TestStartUsesOneControlListener(t *testing.T) {
	te := NewTestEnv(t)
	te.httpSrv.Close()
	s := te.server
	s.config.ListenAddr = availableTCPAddress(t)
	s.config.RepoConfigDir = []string{t.TempDir()}
	s.config.RepoLogsDir = t.TempDir()
	s.e = s.newEcho()
	s.registerControlAPIs(s.e)

	cancel, done := startTestServer(t, s)
	t.Cleanup(cancel)
	client := resty.New().SetBaseURL("http://" + s.config.ListenAddr)
	var resp *resty.Response
	require.Eventually(t, func() bool {
		var err error
		resp, err = client.R().Get("/api/v1/metas")
		return err == nil
	}, time.Second, 10*time.Millisecond)
	require.Equal(t, http.StatusOK, resp.StatusCode())
	resp, err := client.R().Get("/api/v1/repos")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode())

	cancel()
	require.NoError(t, <-done)
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
