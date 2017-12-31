package core

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"
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
			c.PullImage(i, time.Second*10)
		}(i)
	}
	wg.Wait()
}

func (c *Core) CleanImages() {
	imgs, err := c.Docker.ListImages(docker.ListImagesOptions{
		All: true,
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

func (c *Core) PullImage(img string, timeout time.Duration) error {
	repo, tag := docker.ParseRepositoryTag(img)
	return c.Docker.PullImage(docker.PullImageOptions{
		InactivityTimeout: timeout,
		Repository:        repo,
		Tag:               tag,
	}, docker.AuthConfiguration{})
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

func (c *Core) WaitRunningContainers(prefix string) {
	opts := docker.ListContainersOptions{
		All: true,
		Filters: map[string][]string{
			"label":  {"ustcmirror.images"},
			"status": {"running"},
		},
	}
	cts, err := c.Docker.ListContainers(opts)
	if err != nil {
		return
	}
	for _, ct := range cts {
		go func(id string) {
			code, err := c.Docker.WaitContainer(id)
			if err != nil {
				return
			}

			c.RemoveContainer(id)
			name := strings.TrimPrefix(id, prefix)
			if r, err := c.GetRepository(name); err == nil {
				c.UpsertRepoMeta(r, code)
			}
		}(ct.Names[0][1:])
	}
}

func (c *Core) CleanDeadContainers() {
	opts := docker.ListContainersOptions{
		All: true,
		Filters: map[string][]string{
			"label":  {"ustcmirror.images"},
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
			if err = c.PullImage(r.Image, time.Second*5); err != nil {
				return err
			}
			ct, err = c.Docker.CreateContainer(createOpts)
		} else {
			return err
		}
	}

	if err := c.Docker.StartContainer(ct.ID, nil); err != nil {
		return err
	}

	go func() {
		var code int
		code, _ = c.Docker.WaitContainer(ct.ID)
		if code != 0 {
			for i := 0; i < r.Retry; i++ {
				c.Docker.StartContainer(ct.ID, nil)
				code, _ = c.Docker.WaitContainer(ct.ID)
				if code == 0 {
					break
				}
			}
		}
		c.RemoveContainer(ct.ID)
		c.UpsertRepoMeta(r, code)
	}()

	return nil
}
