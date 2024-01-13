package fake

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cpuguy83/go-docker/errdefs"

	"github.com/ustclug/Yuki/pkg/docker"
)

type Client struct {
	mu         sync.Mutex
	containers map[string]docker.ContainerSummary
}

func (f *Client) RunContainer(ctx context.Context, config docker.RunContainerConfig) (id string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.containers[config.Name]
	if ok {
		return "", errdefs.Conflict("container already exists")
	}
	f.containers[config.Name] = docker.ContainerSummary{
		ID:     config.Name,
		Labels: config.Labels,
	}
	return config.Name, nil
}

func (f *Client) WaitContainerWithTimeout(id string, timeout time.Duration) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	_, ok := f.containers[id]
	if !ok {
		return 0, fmt.Errorf("container %s not found", id)
	}
	const delay = 5 * time.Second
	if timeout > 0 && timeout < delay {
		time.Sleep(timeout)
		return 0, context.DeadlineExceeded
	}
	time.Sleep(delay)
	return 0, nil
}

func (f *Client) RemoveContainerWithTimeout(id string, timeout time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.containers, id)
	return nil
}

func (f *Client) ListContainersWithTimeout(running bool, timeout time.Duration) ([]docker.ContainerSummary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	l := make([]docker.ContainerSummary, 0, len(f.containers))
	for _, ct := range f.containers {
		l = append(l, ct)
	}
	return l, nil
}

func (f *Client) UpgradeImages(refs []string) error {
	panic("not implemented")
}

func NewClient() docker.Client {
	return &Client{
		containers: make(map[string]docker.ContainerSummary),
	}
}
