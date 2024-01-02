package server

import (
	"context"
	"errors"
	"fmt"
	"io"
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

func getRepoNameFromRoute(c echo.Context) (string, error) {
	val := c.Param("name")
	if len(val) == 0 {
		return "", newHTTPError(http.StatusBadRequest, "Missing required repo name")
	}
	return val, nil
}

func (s *Server) convertModelRepoMetaToGetMetaResponse(in model.RepoMeta) api.GetRepoMetaResponse {
	return api.GetRepoMetaResponse{
		Name:        in.Name,
		Upstream:    in.Upstream,
		Syncing:     in.Syncing,
		Size:        in.Size,
		ExitCode:    in.ExitCode,
		LastSuccess: in.LastSuccess,
		UpdatedAt:   in.UpdatedAt,
		PrevRun:     in.PrevRun,
		NextRun:     in.NextRun,
	}
}

func slogErrAttr(err error) slog.Attr {
	return slog.Any("err", err)
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
	err = s.dockerCli.RemoveContainerWithTimeout(ctID, time.Second*20)
	if err != nil {
		l.Error("Fail to remove container", slogErrAttr(err))
	}

	var lastSuccess int64
	if code == 0 {
		lastSuccess = time.Now().Unix()
	}

	err = s.db.
		Model(&model.RepoMeta{}).
		Where(model.RepoMeta{Name: name}).
		Updates(map[string]any{
			"size":         s.getSize(storageDir),
			"exit_code":    code,
			"last_success": lastSuccess,
			"syncing":      false,
		}).Error
	if err != nil {
		l.Error("Fail to update RepoMeta", slogErrAttr(err))
	}

	if len(s.config.PostSync) == 0 {
		return
	}
	go func() {
		envs := []string{
			fmt.Sprintf("NAME=%s", name),
			fmt.Sprintf("DIR=%s", storageDir),
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
		return fmt.Sprintf("rsync://%s/%s", envs["RSYNC_HOST"], envs["RSYNC_PATH"])
	case "aptsync", "apt-sync":
		return envs["APTSYNC_URL"]
	case "crates-io-index":
		return "https://github.com/rust-lang/crates.io-index"
	case "debian-cd":
		return fmt.Sprintf("rsync://%s/%s", envs["RSYNC_HOST"], envs["RSYNC_MODULE"])
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
		return "https://android.googlesource.com/mirror/manifest"
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
		err := s.db.
			Where(model.RepoMeta{Name: name}).
			Updates(&model.RepoMeta{Syncing: true}).
			Error
		if err != nil {
			s.logger.Error("Fail to set syncing to true", slogErrAttr(err), slog.String("repo", name))
		}
		go s.waitForSync(name, ctID, dir)
	}
	return nil
}

func (s *Server) upgradeImages() {
	db := s.db
	logger := s.logger
	logger.Debug("Upgrading images")

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

	logger.Debug("Removing dangling images")

	err = s.dockerCli.RemoveDanglingImages()
	if err != nil {
		logger.Error("Fail to remove dangling images", slogErrAttr(err))
	}
}

func (s *Server) scheduleTasks(ctx context.Context) {
	// sync repos
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				var metas []model.RepoMeta
				s.db.Select("name").Where("next_run <= ?", now.Unix()).Find(&metas)
				for _, meta := range metas {
					name := meta.Name
					go func() {
						l := s.logger.With(slog.String("repo", name))
						err := s.syncRepo(context.Background(), name, false)
						if err != nil {
							if errdefs.IsConflict(err) {
								l.Warn("Still syncing")
							} else {
								l.Error("Fail to sync", slogErrAttr(err))
							}
						}
					}()
				}
			}
		}
	}()

	// upgrade images
	if s.config.ImagesUpgradeInterval > 0 {
		go func() {
			ticker := time.NewTicker(s.config.ImagesUpgradeInterval)
			defer ticker.Stop()
			for {
				// fire immediately
				s.upgradeImages()
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
			}
		}()
	}
}

func (s *Server) initRepoMetas() error {
	db := s.db
	var repos []model.Repo
	return db.Select("name", "storage_dir", "cron").
		FindInBatches(&repos, 10, func(*gorm.DB, int) error {
			for _, repo := range repos {
				schedule, _ := cron.ParseStandard(repo.Cron)
				s.repoSchedules.Set(repo.Name, schedule)
				nextRun := schedule.Next(time.Now()).Unix()
				size := s.getSize(repo.StorageDir)
				err := db.Clauses(clause.OnConflict{
					DoUpdates: clause.Assignments(map[string]any{
						"size":     size,
						"syncing":  false,
						"next_run": nextRun,
					}),
				}).Create(&model.RepoMeta{
					Name:     repo.Name,
					Size:     size,
					NextRun:  nextRun,
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

	// Update next_run unconditionally
	logger := s.logger.With(slog.String("repo", name))
	now := time.Now()
	var nextRun int64
	schedule, ok := s.repoSchedules.Get(repo.Name)
	if ok {
		nextRun = schedule.Next(now).Unix()
	} else {
		logger.Warn("No schedule found for repo. Fallback to 1 hour")
		nextRun = now.Add(time.Hour).Unix()
	}
	err := db.
		Where(model.RepoMeta{Name: name}).
		Updates(&model.RepoMeta{NextRun: nextRun}).Error
	if err != nil {
		logger.Error("Fail to update next_run", slogErrAttr(err))
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

	envMap := repo.Envs
	if len(envMap) == 0 {
		envMap = make(map[string]string)
	}
	envMap["REPO"] = repo.Name
	envMap["OWNER"] = repo.User
	envMap["BIND_ADDRESS"] = repo.BindIP
	envMap["RETRY"] = strconv.Itoa(repo.Retry)
	envMap["LOG_ROTATE_CYCLE"] = strconv.Itoa(repo.LogRotCycle)
	if debug {
		envMap["DEBUG"] = "true"
	}

	envs := make([]string, 0, len(envMap))
	for k, v := range envMap {
		envs = append(envs, k+"="+v)
	}

	binds := []string{
		repo.StorageDir + ":/data",
		filepath.Join(s.config.RepoLogsDir, name) + ":/log",
	}
	for k, v := range repo.Volumes {
		binds = append(binds, k+":"+v)
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
		// NOTE: difference between "-v" and "--mount":
		// https://docs.docker.com/storage/bind-mounts/#choose-the--v-or---mount-flag
		Mounts: []mount.Mount{
			{
				// TODO: make it configurable?
				Type:   mount.TypeTmpfs,
				Target: "/tmp",
			},
		},
		Binds: binds,
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

	err = db.
		Where(model.RepoMeta{Name: name}).
		Updates(&model.RepoMeta{
			Upstream: getUpstream(repo.Image, repo.Envs),
			PrevRun:  now.Unix(),
			Syncing:  true,
		}).Error
	if err != nil {
		logger.Error("Fail to update RepoMeta", slogErrAttr(err))
	}
	go s.waitForSync(name, ctID, repo.StorageDir)

	return nil
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
