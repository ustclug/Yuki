package fake

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/errdefs"

	"github.com/ustclug/Yuki/pkg/docker"
)

type Client struct {
	containers []types.Container
}

func (f *Client) RunContainer(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (id string, err error) {
	for _, ct := range f.containers {
		if ct.Names[0] == containerName {
			return "", errdefs.Conflict(errors.New("container already exists"))
		}
	}
	id = fmt.Sprintf("fake-%d", len(f.containers))
	f.containers = append(f.containers, types.Container{
		ID:     id,
		Names:  []string{containerName},
		Labels: config.Labels,
		Status: "running",
	})
	return id, nil
}

func (f *Client) PullImage(ctx context.Context, image string) error {
	// TODO implement me
	panic("implement me")
}

func (f *Client) WaitContainer(ctx context.Context, id string) (int, error) {
	for i, ct := range f.containers {
		if ct.ID == id {
			time.Sleep(5 * time.Second)
			f.containers[i].Status = "exited"
			return 0, nil
		}
	}
	return 0, fmt.Errorf("container %s not found", id)
}

func (f *Client) RemoveContainerWithTimeout(id string, timeout time.Duration) error {
	cts := make([]types.Container, 0, len(f.containers))
	for _, ct := range f.containers {
		if ct.ID != id {
			cts = append(cts, ct)
		}
	}
	f.containers = cts
	return nil
}

func (f *Client) ListContainersWithTimeout(running bool, timeout time.Duration) ([]types.Container, error) {
	return f.containers, nil
}

func (f *Client) RemoveDanglingImages() error {
	return nil
}

func NewClient() docker.Client {
	return &Client{}
}
