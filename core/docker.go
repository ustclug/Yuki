package core

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"sync"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/knight42/Yuki/common"
)

// SyncOptions provides params to the Sync function.
type SyncOptions struct {
	Name       string
	LogDir     string
	Owner      string
	BindIP     string
	NamePrefix string
	Debug      bool
	MountDir   bool
	// FIXME: Not sure whether we should add this param. If a container timed out and got removed, the problem may be hidden.
	Timeout time.Duration
}

// LogsOptions provides params to the GetContainerLogs function.
type LogsOptions struct {
	ID     string
	Stream io.Writer
	Tail   string
	Follow bool
}

// GetContainerLogs gets all stdout and stderr logs from the given container.
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

// UpgradeImages pulls all in use Docker images.
func (c *Core) UpgradeImages() {
	var images []string
	err := c.repoColl.Find(nil).Distinct("image", &images)
	if err != nil {
		return
	}
	var wg sync.WaitGroup
	wg.Add(len(images))
	for _, i := range images {
		go func(i string) {
			defer wg.Done()
			c.PullImage(i)
		}(i)
	}
	wg.Wait()
}

// CleanImages remove unused Docker images with `ustcmirror.images` label.
func (c *Core) CleanImages() {
	ctx, cancel := context.WithTimeout(c.ctx, time.Second*10)
	defer cancel()
	imgs, err := c.Docker.ListImages(docker.ListImagesOptions{
		All:     true,
		Context: ctx,
		Filters: map[string][]string{
			"dangling": {"true"},
			"label":    {"ustcmirror.images"},
		},
	})
	if err != nil {
		return
	}
	for _, i := range imgs {
		go func(id string) {
			c.Docker.RemoveImage(i.ID)
		}(i.ID)
	}
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

// WaitRunningContainers waits for all syncing containers to stop and remove them.
func (c *Core) WaitRunningContainers() {
	ctx, cancel := context.WithTimeout(c.ctx, time.Second*10)
	defer cancel()
	opts := docker.ListContainersOptions{
		All:     true,
		Context: ctx,
		Filters: map[string][]string{
			"label":  {"org.ustcmirror.syncing=true"},
			"status": {"running"},
		},
	}
	cts, err := c.Docker.ListContainers(opts)
	if err != nil {
		return
	}
	for _, ct := range cts {
		name, ok := ct.Labels["org.ustcmirror.name"]
		if !ok {
			continue
		}
		dir, ok := ct.Labels["org.ustcmirror.storage-dir"]
		if !ok {
			continue
		}
		go func(id, name, dir string) {
			code, err := c.Docker.WaitContainer(id)
			if err != nil {
				return
			}
			c.UpsertRepoMeta(name, dir, code)
			c.RemoveContainer(id)
		}(ct.ID, name, dir)
	}
}

// CleanDeadContainers removes containers which status are `created`, `exited` or `dead`.
func (c *Core) CleanDeadContainers() {
	ctx, cancel := context.WithTimeout(c.ctx, time.Second*10)
	defer cancel()
	opts := docker.ListContainersOptions{
		Context: ctx,
		All:     true,
		Filters: map[string][]string{
			"label":  {"org.ustcmirror.syncing=true"},
			"status": {"created", "exited", "dead"},
		},
	}
	cts, err := c.Docker.ListContainers(opts)
	if err != nil {
		return
	}
	for _, ct := range cts {
		go func(id string) {
			c.RemoveContainer(id)
		}(ct.ID)
	}
}

// Sync creates and starts a predefined container to sync local repository.
func (c *Core) Sync(opts SyncOptions) error {
	r, err := c.GetRepository(opts.Name)
	if err != nil {
		return fmt.Errorf("could not find %s in DB", opts.Name)
	}

	envs := docker.Env{}
	for k, v := range r.Envs {
		envs.Set(k, v)
	}
	if r.BindIP == "" {
		r.BindIP = opts.BindIP
	}
	if r.User == "" {
		r.User = opts.Owner
	}
	envs.Set("REPO", r.Name)
	envs.Set("OWNER", r.User)
	envs.Set("BIND_ADDRESS", r.BindIP)
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
		Name:    opts.NamePrefix + opts.Name,
		Config: &docker.Config{
			Image:     r.Image,
			OpenStdin: true,
			Env:       envs,
			Labels: M{
				"org.ustcmirror.syncing":    "true",
				"org.ustcmirror.name":       r.Name,
				"org.ustcmirror.storage-dir": r.StorageDir,
			},
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
			if err = c.PullImage(r.Image); err == nil {
				ct, err = c.Docker.CreateContainer(createOpts)
			}
		}
		if err != nil {
			return err
		}
	}

	if err = c.Docker.StartContainer(ct.ID, nil); err != nil {
		return err
	}

	go func(id, name, dir string, retry int) {
		code, err := c.Docker.WaitContainer(id)
		if err != nil {
			return
		}
		if code != 0 {
			for i := 0; i < retry; i++ {
				c.Docker.StartContainer(id, nil)
				code, err = c.Docker.WaitContainer(id)
				if err != nil {
					return
				}
				if code == 0 {
					break
				}
			}
		}
		c.RemoveContainer(id)
		c.UpsertRepoMeta(name, dir, code)
	}(ct.ID, r.Name, r.StorageDir, r.Retry)

	return nil
}
