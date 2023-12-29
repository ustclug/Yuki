package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/errdefs"
	"github.com/labstack/echo/v4"
	"github.com/robfig/cron/v3"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/model"
)

const suffixYAML = ".yaml"

var errNotFound = errors.New("not found")

func (s *Server) getDB(c echo.Context) *gorm.DB {
	return s.db.WithContext(c.Request().Context())
}

//nolint:unparam
func getRequiredParamFromEchoContext(c echo.Context, name string) (string, error) {
	val := c.Param(name)
	if len(val) == 0 {
		return "", newHTTPError(http.StatusBadRequest, "Missing required parameter: "+name)
	}
	return val, nil
}

func (s *Server) convertModelRepoMetaToGetMetaResponse(in model.RepoMeta, jobs map[string]cron.Entry) api.GetRepoMetaResponse {
	_, syncing := s.syncingContainers.Load(in.Name)
	var nextRun int64
	job, ok := jobs[in.Name]
	if ok {
		nextRun = job.Next.Unix()
	}
	return api.GetRepoMetaResponse{
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

func newHTTPError(code int, msg string) error {
	return &echo.HTTPError{
		Code:    code,
		Message: msg,
	}
}

func (s *Server) waitForSync(name, ctID, storageDir string) {
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
	s.syncingContainers.Delete(name)
	err = s.dockerCli.RemoveContainerWithTimeout(ctID, time.Second*20)
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

	if len(s.config.PostSync) == 0 {
		return
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
		switch remoteType {
		case "swift":
			domain = envs["RCLONE_SWIFT_STORAGE_URL"] + "/"
		case "http":
			domain = envs["RCLONE_CONFIG_REMOTE_URL"]
		case "s3":
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

// cleanDeadContainers removes containers which status are `created`, `exited` or `dead`.
func (s *Server) cleanDeadContainers() error {
	cts, err := s.dockerCli.ListContainersWithTimeout(false, time.Second*10)
	if err != nil {
		return fmt.Errorf("list containers: %w", err)
	}

	for _, ct := range cts {
		err := s.dockerCli.RemoveContainerWithTimeout(ct.ID, time.Second*20)
		if err != nil {
			return fmt.Errorf("remove container %q: %w", ct.ID, err)
		}
	}
	return nil
}

// waitRunningContainers waits for all syncing containers to stop and remove them.
func (s *Server) waitRunningContainers() error {
	cts, err := s.dockerCli.ListContainersWithTimeout(true, time.Second*10)
	if err != nil {
		// logger.Error("Fail to list containers", slogErrAttr(err))
		return fmt.Errorf("list containers: %w", err)
	}
	for _, ct := range cts {
		name := ct.Labels[api.LabelRepoName]
		dir := ct.Labels[api.LabelStorageDir]
		ctID := ct.ID
		s.syncingContainers.Store(name, struct{}{})
		go s.waitForSync(name, ctID, dir)
	}
	return nil
}

func (s *Server) upgradeImages() {
	db := s.db
	logger := s.logger
	logger.Info("Upgrading images")

	var images []string
	err := db.Model(&model.Repo{}).
		Distinct("image").
		Pluck("image", &images).Error
	if err != nil {
		logger.Error("Fail to query images", slogErrAttr(err))
		return
	}
	eg, egCtx := errgroup.WithContext(context.Background())
	eg.SetLimit(5)
	for _, i := range images {
		img := i
		eg.Go(func() error {
			pullCtx, cancel := context.WithTimeout(egCtx, time.Minute*5)
			defer cancel()
			err := s.dockerCli.PullImage(pullCtx, img)
			if err != nil {
				logger.Error("Fail to pull image", slogErrAttr(err), slog.String("image", img))
			}
			return nil
		})
	}
	_ = eg.Wait()

	logger.Info("Removing dangling images")

	err = s.dockerCli.RemoveDanglingImages()
	if err != nil {
		logger.Error("Fail to remove dangling images", slogErrAttr(err))
	}
}

func (s *Server) newJob(name string) cron.FuncJob {
	l := s.logger.With(slog.String("repo", name))
	return func() {
		err := s.syncRepo(context.Background(), name, false)
		if err != nil {
			if errdefs.IsConflict(err) {
				l.Warn("Still syncing")
			} else {
				l.Error("Fail to sync", slogErrAttr(err))
			}
		}
	}
}

func (s *Server) scheduleRepos() error {
	var repos []model.Repo
	err := s.db.Select("name", "interval").Find(&repos).Error
	if err != nil {
		return fmt.Errorf("list repos: %w", err)
	}
	for _, r := range repos {
		err = s.cron.AddJob(r.Name, r.Interval, s.newJob(r.Name))
		if err != nil {
			return fmt.Errorf("add job for repo %q: %w", r.Name, err)
		}
	}
	return nil
}

func (s *Server) initRepoMetas() error {
	db := s.db
	var repos []model.Repo
	return db.Select("name", "storage_dir").
		FindInBatches(&repos, 10, func(*gorm.DB, int) error {
			for _, repo := range repos {
				size := s.getSize(repo.StorageDir)
				err := db.Clauses(clause.OnConflict{
					DoUpdates: clause.Assignments(map[string]any{
						"size": size,
					}),
				}).Create(&model.RepoMeta{
					Name:     repo.Name,
					Size:     size,
					ExitCode: -1,
				}).Error
				if err != nil {
					return fmt.Errorf("init meta for repo %q: %w", repo.Name, err)
				}
			}
			return nil
		}).Error
}

func (s *Server) syncRepo(ctx context.Context, name string, debug bool) error {
	db := s.db.WithContext(ctx)
	var repo model.Repo
	res := db.Where(model.Repo{Name: name}).Limit(1).Find(&repo)
	if res.Error != nil {
		return fmt.Errorf("get repo %q: %w", name, res.Error)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("get repo %q: %w", name, errNotFound)
	}

	envMap := map[string]string{}
	for k, v := range repo.Envs {
		envMap[k] = v
	}
	if len(repo.BindIP) == 0 {
		repo.BindIP = s.config.BindIP
	}
	if len(repo.User) == 0 {
		repo.User = s.config.Owner
	}

	var securityOpt []string
	if len(s.config.SeccompProfile) > 0 {
		securityOpt = append(securityOpt, "seccomp="+s.config.SeccompProfile)
	}

	envMap["REPO"] = repo.Name
	envMap["OWNER"] = repo.User
	envMap["BIND_ADDRESS"] = repo.BindIP
	envMap["RETRY"] = strconv.Itoa(repo.Retry)
	envMap["LOG_ROTATE_CYCLE"] = strconv.Itoa(repo.LogRotCycle)
	if debug {
		envMap["DEBUG"] = "true"
	} else if envMap["DEBUG"] == "" {
		envMap["DEBUG"] = "false"
	}

	envs := make([]string, 0, len(envMap))
	for k, v := range envMap {
		envs = append(envs, k+"="+v)
	}

	mounts := []mount.Mount{
		{
			// TODO: make it configurable?
			Type:   mount.TypeTmpfs,
			Target: "/tmp",
		},
		{
			Type:   mount.TypeBind,
			Source: repo.StorageDir,
			Target: "/data",
		},
		{
			Type:   mount.TypeBind,
			Source: filepath.Join(s.config.LogDir, name),
			Target: "/log",
		},
	}
	for k, v := range repo.Volumes {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: k,
			Target: v,
		})
	}

	containerConfig := &container.Config{
		Image:     repo.Image,
		OpenStdin: true,
		Env:       envs,
		Labels: map[string]string{
			api.LabelRepoName:   repo.Name,
			api.LabelStorageDir: repo.StorageDir,
		},
	}
	hostConfig := &container.HostConfig{
		SecurityOpt: securityOpt,
		Mounts:      mounts,
	}
	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: make(map[string]*network.EndpointSettings, 1),
	}
	switch repo.Network {
	case "", "host":
		hostConfig.NetworkMode = "host"
	default:
		// https://github.com/moby/moby/blob/master/daemon/create_test.go#L15
		networkingConfig.EndpointsConfig[repo.Network] = &network.EndpointSettings{}
	}
	ctName := s.config.NamePrefix + name

	ctID, err := s.dockerCli.RunContainer(
		ctx,
		containerConfig,
		hostConfig,
		networkingConfig,
		ctName,
	)
	if err != nil {
		return fmt.Errorf("run container: %w", err)
	}

	s.syncingContainers.Store(name, struct{}{})
	err = db.
		Where(model.RepoMeta{Name: name}).
		Updates(&model.RepoMeta{
			Upstream: getUpstream(repo.Image, repo.Envs),
			PrevRun:  time.Now().Unix(),
		}).Error
	if err != nil {
		s.logger.Error("Fail to update RepoMeta", slogErrAttr(err), slog.String("repo", name))
	}
	go s.waitForSync(name, ctID, repo.StorageDir)

	return nil
}
