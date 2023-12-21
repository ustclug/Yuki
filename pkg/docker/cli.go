package docker

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
)

type Client interface {
	// RunContainer creates and starts a container with the given config.
	// The specified image will be pulled automatically if it does not exist.
	RunContainer(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (id string, err error)
	PullImage(ctx context.Context, image string) error
	WaitContainer(ctx context.Context, id string) (int, error)
	RemoveContainer(id string, timeout time.Duration) error
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

func (c *clientImpl) RemoveContainer(id string, timeout time.Duration) error {
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
