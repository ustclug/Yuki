package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/cpuguy83/go-docker"
	"github.com/cpuguy83/go-docker/container"
	"github.com/cpuguy83/go-docker/container/containerapi"
	"github.com/cpuguy83/go-docker/container/containerapi/mount"
	"github.com/cpuguy83/go-docker/errdefs"
	"github.com/cpuguy83/go-docker/image"
	"github.com/cpuguy83/go-docker/image/imageapi"
	"github.com/cpuguy83/go-docker/transport"
	"golang.org/x/sync/errgroup"

	"github.com/ustclug/Yuki/pkg/api"
)

type RunContainerConfig struct {
	// ContainerConfig
	Labels map[string]string
	Env    []string
	Image  string
	Name   string

	// HostConfig
	Binds []string

	// NetworkingConfig
	Network string
}

type ContainerSummary struct {
	ID     string
	Labels map[string]string
}

type Client interface {
	// RunContainer creates and starts a container with the given config.
	// The specified image will be pulled automatically if it does not exist.
	RunContainer(ctx context.Context, config RunContainerConfig) (id string, err error)
	WaitContainerWithTimeout(id string, timeout time.Duration) (int, error)
	RemoveContainerWithTimeout(id string, timeout time.Duration) error
	ListContainersWithTimeout(running bool, timeout time.Duration) ([]ContainerSummary, error)
	UpgradeImages(refs []string) error
}

func NewClient(endpoint string) (Client, error) {
	tr, err := transport.FromConnectionString(endpoint)
	if err != nil {
		return nil, err
	}
	return &clientImpl{
		client: docker.NewClient(docker.WithTransport(tr)),
	}, nil
}

func getTimeoutContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout == 0 {
		return context.WithCancel(context.Background())
	}
	return context.WithTimeout(context.Background(), timeout)
}

type clientImpl struct {
	client *docker.Client
}

func (c *clientImpl) RunContainer(ctx context.Context, config RunContainerConfig) (id string, err error) {
	setCfg := func(cfg *container.CreateConfig) {
		cfg.Name = config.Name
		cfg.Spec.Config = containerapi.Config{
			Image:     config.Image,
			OpenStdin: true,
			Env:       config.Env,
			Labels:    config.Labels,
		}

		cfg.Spec.HostConfig = containerapi.HostConfig{
			Binds: config.Binds,
		}
		cfg.Spec.HostConfig.Mounts = []mount.Mount{
			{
				Type:   mount.TypeTmpfs,
				Target: "/tmp",
			},
		}

		cfg.Spec.NetworkConfig.EndpointsConfig = make(map[string]*containerapi.EndpointSettings)
		switch config.Network {
		case "host", "":
			cfg.Spec.HostConfig.NetworkMode = "host"
		default:
			cfg.Spec.NetworkConfig.EndpointsConfig[config.Network] = &containerapi.EndpointSettings{}
		}
	}
	ct, err := c.client.ContainerService().Create(ctx, "", setCfg)
	if err != nil {
		if errdefs.IsNotFound(err) {
			err = c.pullImage(ctx, config.Image)
			if err != nil {
				return "", fmt.Errorf("pull image: %w", err)
			}
			ct, err = c.client.ContainerService().Create(ctx, "", setCfg)
		}
		if err != nil {
			return "", fmt.Errorf("create container: %w", err)
		}
	}
	err = ct.Start(ctx)
	if err != nil {
		return "", fmt.Errorf("start container: %w", err)
	}
	return ct.ID(), nil
}

func (c *clientImpl) ListContainersWithTimeout(running bool, timeout time.Duration) ([]ContainerSummary, error) {
	ctx, cancel := getTimeoutContext(timeout)
	defer cancel()

	var statuses []string
	if running {
		statuses = append(statuses, "running")
	} else {
		statuses = append(statuses, "exited", "created", "dead")
	}

	cts, err := c.client.ContainerService().List(ctx, func(config *container.ListConfig) {
		config.Filter = container.ListFilter{
			Status: statuses,
			Label:  []string{api.LabelRepoName},
		}
	})
	if err != nil {
		return nil, err
	}

	result := make([]ContainerSummary, len(cts))
	for i, ct := range cts {
		result[i] = ContainerSummary{
			ID:     ct.ID,
			Labels: ct.Labels,
		}
	}
	return result, nil
}

func (c *clientImpl) RemoveContainerWithTimeout(id string, timeout time.Duration) error {
	ctx, cancel := getTimeoutContext(timeout)
	defer cancel()
	return c.client.ContainerService().Remove(ctx, id, func(cfg *container.RemoveConfig) {
		cfg.RemoveVolumes = true
		cfg.Force = true
	})
}

func (c *clientImpl) WaitContainerWithTimeout(id string, timeout time.Duration) (int, error) {
	ctx, cancel := getTimeoutContext(timeout)
	defer cancel()
	ct := c.client.ContainerService().NewContainer(ctx, id)
	status, err := ct.Wait(ctx, container.WithWaitCondition(container.WaitConditionNotRunning))
	if err != nil {
		return 0, fmt.Errorf("wait container: %w", err)
	}
	return status.ExitCode()
}

func (c *clientImpl) pullImage(ctx context.Context, ref string) error {
	remote, err := image.ParseRef(ref)
	if err != nil {
		return fmt.Errorf("invalid image ref: %w", err)
	}
	return c.client.ImageService().Pull(ctx, remote)
}

func (c *clientImpl) removeImage(id string, timeout time.Duration) error {
	ctx, cancel := getTimeoutContext(timeout)
	defer cancel()
	_, err := c.client.ImageService().Remove(ctx, id)
	return err
}

func (c *clientImpl) removeDanglingImages() error {
	images, err := c.listDanglingImages(time.Second * 5)
	if err != nil {
		return fmt.Errorf("list images: %w", err)
	}
	for _, img := range images {
		err = c.removeImage(img.ID, time.Second*20)
		if err != nil {
			return fmt.Errorf("remove image: %q: %w", img.ID, err)
		}
	}
	return nil
}

func (c *clientImpl) listDanglingImages(timeout time.Duration) ([]imageapi.Image, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.client.ImageService().List(ctx, func(cfg *image.ListConfig) {
		cfg.Filter = image.ListFilter{
			Label:    []string{api.LabelImages},
			Dangling: []string{"true"},
		}
	})
}

func (c *clientImpl) UpgradeImages(refs []string) error {
	eg, ctx := errgroup.WithContext(context.Background())
	eg.SetLimit(5)
	for _, ref := range refs {
		img := ref
		eg.Go(func() error {
			pullCtx, cancel := context.WithTimeout(ctx, time.Minute*10)
			defer cancel()
			return c.pullImage(pullCtx, img)
		})
	}
	err := eg.Wait()
	if err != nil {
		return err
	}

	return c.removeDanglingImages()
}
