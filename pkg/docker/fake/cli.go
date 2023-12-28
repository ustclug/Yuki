package fake

import (
	"context"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"

	"github.com/ustclug/Yuki/pkg/docker"
)

type fakeClient struct {
	containers []types.Container
}

func (f *fakeClient) RunContainer(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (id string, err error) {
	// TODO implement me
	panic("implement me")
}

func (f *fakeClient) PullImage(ctx context.Context, image string) error {
	// TODO implement me
	panic("implement me")
}

func (f *fakeClient) WaitContainer(ctx context.Context, id string) (int, error) {
	// TODO implement me
	panic("implement me")
}

func (f *fakeClient) RemoveContainerWithTimeout(id string, timeout time.Duration) error {
	// TODO implement me
	panic("implement me")
}

func (f *fakeClient) ListContainersWithTimeout(running bool, timeout time.Duration) ([]types.Container, error) {
	// TODO implement me
	panic("implement me")
}

func (f *fakeClient) RemoveDanglingImages() error {
	// TODO implement me
	panic("implement me")
}

func NewClient(containers []types.Container) docker.Client {
	return &fakeClient{containers}
}
