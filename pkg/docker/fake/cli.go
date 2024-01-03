package fake

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/errdefs"

	"github.com/ustclug/Yuki/pkg/docker"
)

type Client struct {
	mu         sync.Mutex
	containers map[string]types.Container
}

func (f *Client) RunContainer(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (id string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.containers[containerName]
	if ok {
		return "", errdefs.Conflict(errors.New("container already exists"))
	}
	f.containers[containerName] = types.Container{
		ID:     containerName,
		Names:  []string{containerName},
		Labels: config.Labels,
		Status: "running",
	}
	return containerName, nil
}

func (f *Client) PullImage(ctx context.Context, image string) error {
	// TODO implement me
	panic("implement me")
}

func (f *Client) WaitContainerWithTimeout(id string, timeout time.Duration) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	ct, ok := f.containers[id]
	if !ok {
		return 0, fmt.Errorf("container %s not found", id)
	}
	const delay = 5 * time.Second
	if timeout > 0 && timeout < delay {
		time.Sleep(timeout)
		return 0, context.DeadlineExceeded
	}
	time.Sleep(delay)
	ct.Status = "exited"
	f.containers[id] = ct
	return 0, nil
}

func (f *Client) RemoveContainerWithTimeout(id string, timeout time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.containers, id)
	return nil
}

func (f *Client) ListContainersWithTimeout(running bool, timeout time.Duration) ([]types.Container, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	l := make([]types.Container, 0, len(f.containers))
	for _, ct := range f.containers {
		l = append(l, ct)
	}
	return l, nil
}

func (f *Client) RemoveDanglingImages() error {
	return nil
}

func NewClient() docker.Client {
	return &Client{
		containers: make(map[string]types.Container),
	}
}
