package core

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	docker "github.com/fsouza/go-dockerclient"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/utils"
)

// SyncOptions provides params to the Sync function.
type SyncOptions struct {
	Name          string
	LogDir        string
	DefaultOwner  string
	DefaultBindIP string
	NamePrefix    string
	Debug         bool
	MountDir      bool
}

// LogsOptions provides params to the GetContainerLogs function.
type LogsOptions struct {
	ID          string
	Stream      io.Writer
	Tail        string
	Follow      bool
	CloseNotify <-chan struct{}
}

// GetContainerLogs gets all stdout and stderr logs from the given container.
func (c *Core) GetContainerLogs(opts LogsOptions) error {
	finished := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(c.ctx)

	go func() {
		select {
		case <-opts.CloseNotify:
		case <-finished:
		}
		cancel()
	}()

	err := c.Docker.Logs(docker.LogsOptions{
		Stdout:       true,
		Stderr:       true,
		Context:      ctx,
		Container:    opts.ID,
		OutputStream: opts.Stream,
		ErrorStream:  opts.Stream,
		Tail:         opts.Tail,
		Follow:       opts.Follow,
	})
	close(finished)
	if err != context.Canceled {
		return err
	}
	return nil
}

// PullImage pulls an image from remote registry.
func (c *Core) PullImage(img string) error {
	repo, tag := docker.ParseRepositoryTag(img)
	return c.Docker.PullImage(docker.PullImageOptions{
		Tag:               tag,
		Repository:        repo,
		InactivityTimeout: time.Second * 10,
	}, docker.AuthConfiguration{})
}

// StopContainer stops the given container.
func (c *Core) StopContainer(id string) error {
	return c.Docker.StopContainer(id, 10)
}

// RemoveContainer removes the given container.
func (c *Core) RemoveContainer(id string) error {
	ctx, cancel := context.WithTimeout(c.ctx, time.Second*20)
	defer cancel()
	opts := docker.RemoveContainerOptions{
		Context:       ctx,
		Force:         true,
		ID:            id,
		RemoveVolumes: true,
	}
	return c.Docker.RemoveContainer(opts)
}

// Sync creates and starts a predefined container to sync local repository.
func (c *Core) Sync(opts SyncOptions) (*api.Container, error) {
	r, err := c.GetRepository(opts.Name)
	if err != nil {
		return nil, fmt.Errorf("cannot find <%s> in the DB", opts.Name)
	}

	envs := docker.Env{}
	for k, v := range r.Envs {
		envs.Set(k, v)
	}
	if len(r.BindIP) == 0 {
		r.BindIP = opts.DefaultBindIP
	}
	if len(r.User) == 0 {
		r.User = opts.DefaultOwner
	}
	if r.LogRotCycle == nil {
		ten := 10
		r.LogRotCycle = &ten
	}
	envs.Set("REPO", r.Name)
	envs.Set("OWNER", r.User)
	envs.Set("BIND_ADDRESS", r.BindIP)
	envs.SetInt("RETRY", r.Retry)
	envs.SetInt("LOG_ROTATE_CYCLE", *r.LogRotCycle)
	if opts.Debug {
		envs.Set("DEBUG", "true")
	} else {
		envs.Set("DEBUG", "false")
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
	createOpts := docker.CreateContainerOptions{
		Name: opts.NamePrefix + opts.Name,
		Config: &docker.Config{
			Image:     r.Image,
			OpenStdin: true,
			Env:       envs,
			Labels:    labels,
		},
		HostConfig: &docker.HostConfig{
			Binds:       binds,
			NetworkMode: "host",
		},
	}

	ct, err := c.Docker.CreateContainer(createOpts)
	if err != nil {
		if err == docker.ErrNoSuchImage {
			if err = c.PullImage(r.Image); err == nil {
				ct, err = c.Docker.CreateContainer(createOpts)
			}
		}
		if err != nil {
			return nil, err
		}
	}

	if err = c.Docker.StartContainer(ct.ID, nil); err != nil {
		return nil, err
	}

	return &api.Container{
		ID:     ct.ID,
		Labels: labels,
	}, nil
}
