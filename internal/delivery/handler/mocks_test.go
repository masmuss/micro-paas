package handler

import (
	"context"
	"io"

	"github.com/masmuss/micro-paas/internal/model"
	"github.com/stretchr/testify/mock"
)

type mockDockerService struct {
	mock.Mock
}

func (m *mockDockerService) CreateContainer(
	ctx context.Context,
	imageName string,
	containerName string,
	env map[string]string,
) (string, error) {
	args := m.Called(ctx, imageName, containerName, env)
	return args.String(0), args.Error(1)
}

func (m *mockDockerService) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockDockerService) PullImage(ctx context.Context, imageName string) error {
	args := m.Called(ctx, imageName)
	return args.Error(0)
}

func (m *mockDockerService) StartContainer(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

func (m *mockDockerService) StopContainer(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

func (m *mockDockerService) RemoveContainer(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

func (m *mockDockerService) GetContainerStatus(ctx context.Context, containerID string) (string, error) {
	args := m.Called(ctx, containerID)
	return args.String(0), args.Error(1)
}

func (m *mockDockerService) GetContainerLogs(ctx context.Context, containerID string) (io.ReadCloser, error) {
	args := m.Called(ctx, containerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	rc, _ := args.Get(0).(io.ReadCloser)
	return rc, args.Error(1)
}

type mockInstanceRepository struct {
	mock.Mock
}

func (m *mockInstanceRepository) Create(ctx context.Context, instance *model.Instance) error {
	args := m.Called(ctx, instance)
	return args.Error(0)
}

func (m *mockInstanceRepository) GetByID(ctx context.Context, id int64) (*model.Instance, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	inst, _ := args.Get(0).(*model.Instance)
	return inst, args.Error(1)
}

func (m *mockInstanceRepository) GetByName(ctx context.Context, name string) (*model.Instance, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	inst, _ := args.Get(0).(*model.Instance)
	return inst, args.Error(1)
}

func (m *mockInstanceRepository) GetByContainerID(ctx context.Context, containerID string) (*model.Instance, error) {
	args := m.Called(ctx, containerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	inst, _ := args.Get(0).(*model.Instance)
	return inst, args.Error(1)
}

func (m *mockInstanceRepository) GetBySubdomain(ctx context.Context, subdomain string) (*model.Instance, error) {
	args := m.Called(ctx, subdomain)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	inst, _ := args.Get(0).(*model.Instance)
	return inst, args.Error(1)
}

func (m *mockInstanceRepository) Delete(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockInstanceRepository) Update(ctx context.Context, instance *model.Instance) error {
	args := m.Called(ctx, instance)
	return args.Error(0)
}

func (m *mockInstanceRepository) List(ctx context.Context) ([]*model.Instance, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	list, _ := args.Get(0).([]*model.Instance)
	return list, args.Error(1)
}
