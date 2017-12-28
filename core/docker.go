package core

import (
	"fmt"
	"io"
	"os"
	"path"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/knight42/Yuki/common"
)

type SyncOptions struct {
	Name       string
	NamePrefix string
	LogDir     string
	Owner      string
	MountDir   bool
	Debug      bool
	BindIP     string
}

type LogsOptions struct {
	ID     string
	Stream io.Writer
	Tail   string
	Follow bool
}

func (c *Core) GetContainerLogs(opts LogsOptions) error {
	return c.Docker.Logs(docker.LogsOptions{
		Stdout:       true,
		Stderr:       true,
		Container:    opts.ID,
		OutputStream: opts.Stream,
		ErrorStream:  opts.Stream,
		Tail:         opts.Tail,
		Follow:       opts.Follow,
	})
}

func (c *Core) StopContainer(id string) error {
	return c.Docker.StopContainer(id, 10)
}

func (c *Core) RemoveContainer(id string) error {
	opts := docker.RemoveContainerOptions{
		Force:         true,
		ID:            id,
		RemoveVolumes: true,
	}
	return c.Docker.RemoveContainer(opts)
}

func (c *Core) Sync(opts SyncOptions) error {
	r, err := c.GetRepository(opts.Name)
	if err != nil {
		return fmt.Errorf("could not find %s in DB", opts.Name)
	}

	envs := docker.Env{}
	for k, v := range r.Envs {
		envs.Set(k, v)
	}
	if r.BindIp == "" {
		r.BindIp = opts.BindIP
	}
	if r.User == "" {
		r.User = opts.Owner
	}
	envs.Set("REPO", r.Name)
	envs.Set("OWNER", r.User)
	envs.Set("BIND_ADDRESS", r.BindIp)
	envs.SetInt("LOG_ROTATE_CYCLE", int(r.LogRotCycle))
	if opts.Debug {
		envs.Set("DEBUG", "true")
	} else {
		envs.Set("DEBUG", "false")
	}

	binds := []string{}
	for k, v := range r.Volumes {
		binds = append(binds, fmt.Sprintf("%s:%s", k, v))
	}

	if opts.MountDir {
		logdir := path.Join(opts.LogDir, opts.Name)
		if err := os.MkdirAll(logdir, os.ModePerm); err != nil {
			return fmt.Errorf("not a directory: %s", logdir)
		}
		if !common.DirExists(r.StorageDir) {
			return fmt.Errorf("not a directory: %s", r.StorageDir)
		}
		binds = append(binds, fmt.Sprintf("%s:/data/", r.StorageDir), fmt.Sprintf("%s:/log/", logdir))
	}

	createOpts := docker.CreateContainerOptions{
		Name: opts.NamePrefix + opts.Name,
		Config: &docker.Config{
			Image:     r.Image,
			OpenStdin: true,
			User:      r.User,
			Env:       envs,
		},
		HostConfig: &docker.HostConfig{
			Binds:       binds,
			NetworkMode: "host",
		},
	}

	var ct *docker.Container
	ct, err = c.Docker.CreateContainer(createOpts)
	if err != nil {
		if err == docker.ErrNoSuchImage {
			repo, tag := docker.ParseRepositoryTag(r.Image)
			err := c.Docker.PullImage(docker.PullImageOptions{
				InactivityTimeout: time.Second * 5,
				Repository:        repo,
				Tag:               tag,
			}, docker.AuthConfiguration{})
			if err != nil {
				return err
			}
			ct, err = c.Docker.CreateContainer(createOpts)
		} else {
			return err
		}
	}

	return c.Docker.StartContainer(ct.ID, nil)
}
