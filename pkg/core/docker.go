package core

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/utils"
)

// SyncOptions provides params to the Sync function.
type SyncOptions struct {
	Name           string
	LogDir         string
	DefaultOwner   string
	DefaultBindIP  string
	SeccompProfile string
	NamePrefix     string
	Debug          bool
	MountDir       bool
}

// LogsOptions provides params to the GetContainerLogs function.
type LogsOptions struct {
	ID     string
	Stream io.Writer
	Tail   string
	Follow bool
}

// GetContainerLogs gets all stdout and stderr logs from the given container.
func (c *Core) GetContainerLogs(ctx context.Context, opts LogsOptions) error {
	output, err := c.docker.ContainerLogs(ctx, opts.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       opts.Tail,
		Follow:     opts.Follow,
	})
	if err != nil {
		return err
	}
	defer output.Close()
	_, err = io.Copy(opts.Stream, output)
	if err != context.Canceled {
		return err
	}
	return nil
}

func makeFilterArgs(filter map[string][]string) filters.Args {
	args := filters.NewArgs()
	for k, l := range filter {
		for _, i := range l {
			args.Add(k, i)
		}
	}
	return args
}

// ListImages returns a list of docker images.
func (c *Core) ListImages(ctx context.Context, filter map[string][]string) ([]types.ImageSummary, error) {
	return c.docker.ImageList(ctx, types.ImageListOptions{
		All:     true,
		Filters: makeFilterArgs(filter),
	})
}

// PullImage pulls an image from remote registry.
func (c *Core) PullImage(ctx context.Context, img string) error {
	// ctx, cancel := context.WithTimeout(ctx, time.Minute*5)
	// defer cancel()
	stream, err := c.docker.ImagePull(ctx, img, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer stream.Close()
	_, err = io.Copy(ioutil.Discard, stream)
	return err
}

// RemoveImage removes the given docker image.
func (c *Core) RemoveImage(ctx context.Context, id string) error {
	_, err := c.docker.ImageRemove(ctx, id, types.ImageRemoveOptions{
		PruneChildren: true,
	})
	return err
}

// StopContainer stops the given container.
func (c *Core) StopContainer(ctx context.Context, id string) error {
	return c.docker.ContainerStop(ctx, id, nil)
}

// ListContainers returns a list of containers.
func (c *Core) ListContainers(ctx context.Context, filter map[string][]string) ([]types.Container, error) {
	return c.docker.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Latest:  true,
		Filters: makeFilterArgs(filter),
	})
}

// WaitContainer blocks until the given container exits.
func (c *Core) WaitContainer(ctx context.Context, id string) (int, error) {
	stream, errCh := c.docker.ContainerWait(ctx, id, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return -1, err
		}
		// unreachable
		return -1, fmt.Errorf("unreachable")
	case resp := <-stream:
		return int(resp.StatusCode), nil
	}
}

// RemoveContainer removes the given container.
func (c *Core) RemoveContainer(ctx context.Context, id string) error {
	// ctx, cancel := context.WithTimeout(c.ctx, time.Second*20)
	// defer cancel()
	return c.docker.ContainerRemove(ctx, id, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})
}

// Sync creates and starts a predefined container to sync local repository.
func (c *Core) Sync(ctx context.Context, opts SyncOptions) (*api.Container, error) {
	r, err := c.GetRepository(opts.Name)
	if err != nil {
		return nil, fmt.Errorf("cannot find <%s> in the DB", opts.Name)
	}

	envMap := map[string]string{}
	for k, v := range r.Envs {
		envMap[k] = v
	}
	if len(r.BindIP) == 0 {
		r.BindIP = opts.DefaultBindIP
	}
	if len(r.User) == 0 {
		r.User = opts.DefaultOwner
	}

	seccomp_item := ""
	security_opt := []string{}
	if len(opts.SeccompProfile) > 0 {
		seccomp_item = "seccomp=" + opts.SeccompProfile
		security_opt = append(security_opt, seccomp_item)
	}

	if r.LogRotCycle == nil {
		ten := 10
		r.LogRotCycle = &ten
	}
	envMap["REPO"] = r.Name
	envMap["OWNER"] = r.User
	envMap["BIND_ADDRESS"] = r.BindIP
	envMap["RETRY"] = strconv.FormatInt(int64(r.Retry), 10)
	envMap["LOG_ROTATE_CYCLE"] = strconv.FormatInt(int64(*r.LogRotCycle), 10)
	if opts.Debug {
		envMap["DEBUG"] = "true"
	} else if envMap["DEBUG"] == "" {
		envMap["DEBUG"] = "false"
	}
	envs := make([]string, 0, len(envMap))
	for k, v := range envMap {
		envs = append(envs, k+"="+v)
	}

	var binds []string
	for k, v := range r.Volumes {
		binds = append(binds, fmt.Sprintf("%s:%s", k, v))
	}

	if opts.MountDir {
		logdir := path.Join(opts.LogDir, opts.Name)
		if err = os.MkdirAll(logdir, os.ModePerm); err != nil {
			return nil, fmt.Errorf("not a directory: %s", logdir)
		}
		if !utils.DirExists(r.StorageDir) {
			return nil, fmt.Errorf("not a directory: %s", r.StorageDir)
		}
		binds = append(binds, fmt.Sprintf("%s:/data/", r.StorageDir), fmt.Sprintf("%s:/log/", logdir))
	}
	labels := api.M{
		"org.ustcmirror.name":        r.Name,
		"org.ustcmirror.syncing":     "true",
		"org.ustcmirror.storage-dir": r.StorageDir,
	}
	containerConfig := &container.Config{
		Image:     r.Image,
		OpenStdin: true,
		Env:       envs,
		Labels:    labels,
	}
	hostConfig := &container.HostConfig{
		Binds:       binds,
		SecurityOpt: security_opt,
		NetworkMode: "host",
	}
	ctName := opts.NamePrefix + opts.Name

	ct, err := c.docker.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		nil,
		nil,
		ctName,
	)
	if err != nil {
		if client.IsErrNotFound(err) {
			if err = c.PullImage(ctx, r.Image); err == nil {
				ct, err = c.docker.ContainerCreate(
					ctx,
					containerConfig,
					hostConfig,
					nil,
					nil,
					ctName,
				)
			}
		}
		if err != nil {
			return nil, err
		}
	}

	if err = c.docker.ContainerStart(ctx, ct.ID, types.ContainerStartOptions{}); err != nil {
		return nil, err
	}

	return &api.Container{
		ID:     ct.ID,
		Labels: labels,
	}, nil
}
