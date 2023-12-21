package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/model"
)

const suffixYAML = ".yaml"

func (s *Server) getDB(c echo.Context) *gorm.DB {
	return s.db.WithContext(c.Request().Context())
}

func getRequiredParamFromEchoContext(c echo.Context, name string) (string, error) {
	val := c.Param(name)
	if len(val) == 0 {
		return "", badRequest(name + " is required")
	}
	return val, nil
}

func (s *Server) convertModelRepoMetaToGetMetaResponse(in model.RepoMeta, jobs map[string]cron.Entry) api.GetMetaResponse {
	_, syncing := s.syncStatus.Load(in.Name)
	var nextRun int64
	job, ok := jobs[in.Name]
	if ok {
		nextRun = job.Next.Unix()
	}
	return api.GetMetaResponse{
		Name:        in.Name,
		Upstream:    in.Upstream,
		Syncing:     syncing,
		Size:        in.Size,
		ExitCode:    in.ExitCode,
		LastSuccess: in.LastSuccess,
		UpdatedAt:   in.UpdatedAt,
		PrevRun:     in.PrevRun,
		NextRun:     nextRun,
	}
}

func slogErrAttr(err error) slog.Attr {
	return slog.Any("err", err)
}

func bindAndValidate[T any](c echo.Context, input *T) error {
	err := c.Bind(input)
	if err != nil {
		return err
	}
	return c.Validate(input)
}

// Deprecated: use newHTTPError instead
func badRequest(msg string) error {
	return &echo.HTTPError{
		Code:    http.StatusBadRequest,
		Message: msg,
	}
}

// Deprecated: use newHTTPError instead
func notFound(msg string) error {
	return &echo.HTTPError{
		Code:    http.StatusNotFound,
		Message: msg,
	}
}

// Deprecated: use newHTTPError instead
func conflict(msg string) error {
	return &echo.HTTPError{
		Code:    http.StatusConflict,
		Message: msg,
	}
}

func newHTTPError(code int, msg string) error {
	return &echo.HTTPError{
		Code:    code,
		Message: msg,
	}
}

func (s *Server) waitForSyncV2(name, ctID, storageDir string) {
	var ctx context.Context
	if s.config.SyncTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), s.config.SyncTimeout)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	l := s.logger.With(slog.String("repo", name))
	code, err := s.dockerCli.WaitContainer(ctx, ctID)
	if err != nil {
		if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
			l.Error("Fail to wait for container", slogErrAttr(err))
			return
		} else {
			// When the error is timeout, we expect that
			// container will be stopped and removed in onPostSync() goroutine
			// Here we set a special exit code to indicate that the container is timeout in meta.
			code = -2
		}
	}
	s.syncStatus.Delete(name)
	err = s.dockerCli.RemoveContainer(ctID, time.Second*20)
	if err != nil {
		l.Error("Fail to remove container", slogErrAttr(err))
	}

	var lastSuccess int64
	if code == 0 {
		lastSuccess = time.Now().Unix()
	}

	err = s.db.
		Where(model.RepoMeta{Name: name}).
		Updates(&model.RepoMeta{
			Size:        s.getSize(storageDir),
			ExitCode:    code,
			LastSuccess: lastSuccess,
		}).Error
	if err != nil {
		l.Error("Fail to update RepoMeta", slogErrAttr(err))
	}

	go func() {
		envs := []string{
			fmt.Sprintf("ID=%s", ctID),
			fmt.Sprintf("Name=%s", name),
			fmt.Sprintf("Dir=%s", storageDir),
			fmt.Sprintf("ExitCode=%d", code),
		}
		for _, cmd := range s.config.PostSync {
			prog := exec.Command("sh", "-c", cmd)
			prog.Env = envs
			output, err := prog.CombinedOutput()
			if err != nil {
				l.Error("PostSync program exit abnormally",
					slog.String("output", string(output)),
					slog.String("command", cmd),
				)
			}
		}
	}()
}

func getUpstream(image string, envs model.StringMap) (upstream string) {
	image = strings.SplitN(image, ":", 2)[0]
	parts := strings.Split(image, "/")
	t := parts[len(parts)-1]

	var ok bool
	if upstream, ok = envs["$UPSTREAM"]; ok {
		return upstream
	}
	if upstream, ok = envs["UPSTREAM"]; ok {
		return upstream
	}
	switch t {
	case "archvsync", "rsync":
		return fmt.Sprintf("rsync://%s/%s/", envs["RSYNC_HOST"], envs["RSYNC_PATH"])
	case "aptsync", "apt-sync":
		return envs["APTSYNC_URL"]
	case "crates-io-index":
		return "https://github.com/rust-lang/crates.io-index"
	case "debian-cd":
		return fmt.Sprintf("rsync://%s/%s/", envs["RSYNC_HOST"], envs["RSYNC_MODULE"])
	case "docker-ce":
		return "https://download.docker.com/"
	case "fedora":
		remote, ok := envs["REMOTE"]
		if !ok {
			remote = "rsync://dl.fedoraproject.org"
		}
		return fmt.Sprintf("%s/%s", remote, envs["MODULE"])
	case "freebsd-pkg":
		if upstream, ok = envs["FBSD_PKG_UPSTREAM"]; !ok {
			return "http://pkg.freebsd.org/"
		}
	case "freebsd-ports":
		if upstream, ok = envs["FBSD_PORTS_DISTFILES_UPSTREAM"]; !ok {
			return "http://distcache.freebsd.org/ports-distfiles/"
		}
	case "ghcup":
		return "https://www.haskell.org/ghcup/"
	case "github-release":
		return "https://github.com"
	case "gitsync":
		return envs["GITSYNC_URL"]
	case "google-repo":
		if upstream, ok = envs["UPSTREAM"]; !ok {
			return "https://android.googlesource.com/mirror/manifest"
		}
	case "gsutil-rsync":
		return envs["GS_URL"]
	case "hackage":
		if upstream, ok = envs["HACKAGE_BASE_URL"]; !ok {
			return "https://hackage.haskell.org/"
		}
	case "homebrew-bottles":
		if upstream, ok = envs["HOMEBREW_BOTTLE_DOMAIN"]; !ok {
			return "https://ghcr.io/v2/homebrew/"
		}
	case "julia-storage":
		return "https://us-east.storage.juliahub.com, https://kr.storage.juliahub.com"
	case "nix-channels":
		if upstream, ok = envs["NIX_MIRROR_UPSTREAM"]; !ok {
			return "https://nixos.org/channels"
		}
	case "lftpsync":
		return fmt.Sprintf("%s/%s", envs["LFTPSYNC_HOST"], envs["LFTPSYNC_PATH"])
	case "nodesource":
		return "https://nodesource.com/"
	case "pypi":
		return "https://pypi.python.org/"
	case "rclone":
		remoteType := envs["RCLONE_CONFIG_REMOTE_TYPE"]
		path := envs["RCLONE_PATH"]
		domain := ""
		if remoteType == "swift" {
			domain = envs["RCLONE_SWIFT_STORAGE_URL"] + "/"
		} else if remoteType == "http" {
			domain = envs["RCLONE_CONFIG_REMOTE_URL"]
		} else if remoteType == "s3" {
			domain = envs["RCLONE_CONFIG_REMOTE_ENDPOINT"]
		}
		return fmt.Sprintf("%s%s", domain, path)
	case "rubygems":
		return "http://rubygems.org/"
	case "stackage":
		upstream = "https://github.com/commercialhaskell/"
	case "tsumugu":
		return envs["UPSTREAM"]
	case "winget-source":
		if upstream, ok = envs["WINGET_REPO_URL"]; !ok {
			return "https://cdn.winget.microsoft.com/cache"
		}
	case "yum-sync":
		return envs["YUMSYNC_URL"]
	}
	return
}
