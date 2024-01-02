package docker

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"

	"github.com/ustclug/Yuki/pkg/api"
)

type Client interface {
	// RunContainer creates and starts a container with the given config.
	// The specified image will be pulled automatically if it does not exist.
	RunContainer(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (id string, err error)
	PullImage(ctx context.Context, image string) error
	// TODO: rename to WaitContainerWithTimeout
	WaitContainer(ctx context.Context, id string) (int, error)
	RemoveContainerWithTimeout(id string, timeout time.Duration) error
	ListContainersWithTimeout(running bool, timeout time.Duration) ([]types.Container, error)
	RemoveDanglingImages() error
}

func NewClient(endpoint string) (Client, error) {
	d, err := client.NewClientWithOpts(
		client.WithHost(endpoint),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to the Docker daemon at %s", endpoint)
	}
	return &clientImpl{d}, nil
}

type clientImpl struct {
	client *client.Client
}

func (c *clientImpl) listImages(timeout time.Duration) ([]types.ImageSummary, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.client.ImageList(ctx, types.ImageListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("label", api.LabelImages),
			filters.Arg("dangling", "true"),
		),
	})
}

func (c *clientImpl) removeImage(id string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_, err := c.client.ImageRemove(ctx, id, types.ImageRemoveOptions{
		PruneChildren: true,
	})
	return err
}

func (c *clientImpl) RemoveDanglingImages() error {
	images, err := c.listImages(time.Second * 5)
	if err != nil {
		return fmt.Errorf("list images: %w", err)
	}
	for _, image := range images {
		err = c.removeImage(image.ID, time.Second*20)
		if err != nil {
			return fmt.Errorf("remove image: %q: %w", image.ID, err)
		}
	}
	return nil
}

func (c *clientImpl) ListContainersWithTimeout(running bool, timeout time.Duration) ([]types.Container, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	args := filters.NewArgs(filters.Arg("label", api.LabelRepoName))
	var statuses []string
	if running {
		statuses = append(statuses, "running")
	} else {
		statuses = append(statuses, "exited", "created", "dead")
	}
	for _, status := range statuses {
		args.Add("status", status)
	}

	return c.client.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: args,
	})
}

func (c *clientImpl) RemoveContainerWithTimeout(id string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.client.ContainerRemove(ctx, id, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})
}

func (c *clientImpl) WaitContainer(ctx context.Context, id string) (int, error) {
	stream, errCh := c.client.ContainerWait(ctx, id, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		return -1, err
	case resp := <-stream:
		return int(resp.StatusCode), nil
	}
}

func (c *clientImpl) PullImage(ctx context.Context, image string) error {
	stream, err := c.client.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("invoke ImagePull: %w", err)
	}
	defer stream.Close()
	_, err = io.Copy(io.Discard, stream)
	if err != nil {
		return fmt.Errorf("read stream: %w", err)
	}
	return err
}

func (c *clientImpl) RunContainer(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (string, error) {
	ct, err := c.client.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, containerName)
	if err != nil {
		if errdefs.IsNotFound(err) {
			err = c.PullImage(ctx, config.Image)
			if err != nil {
				return "", fmt.Errorf("pull image: %w", err)
			}
			ct, err = c.client.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, containerName)
		}
		if err != nil {
			return "", fmt.Errorf("create container: %w", err)
		}
	}

	err = c.client.ContainerStart(ctx, ct.ID, types.ContainerStartOptions{})
	if err != nil {
		return "", fmt.Errorf("start container: %w", err)
	}

	return ct.ID, nil
}
