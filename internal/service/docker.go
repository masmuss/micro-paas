// Package service provides application services that encapsulate business logic and interactions with external systems.
package service

import (
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/masmuss/micro-paas/internal/config"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// DockerService defines the interface for interacting with the Docker daemon.
type DockerService interface {
	CreateContainer(ctx context.Context, imageName string, containerName string) (string, error)
	Ping(ctx context.Context) error
	PullImage(ctx context.Context, imageName string) error
	StartContainer(ctx context.Context, containerID string) error
}

type dockerServiceImpl struct {
	cli *client.Client
	log *slog.Logger
}

// NewDockerService creates a new DockerService using the provided configuration and logger.
func NewDockerService(cfg *config.Config, log *slog.Logger) (DockerService, error) {
	cli, err := client.New(
		client.WithHost("unix://"+cfg.DockerSocket),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, err
	}
	return &dockerServiceImpl{
		cli: cli,
		log: log.With("component", "docker_service"),
	}, nil
}

// CreateContainer pulls the specified image and creates a new container with the given name, then starts it.
func (s *dockerServiceImpl) CreateContainer(
	ctx context.Context,
	imageName string,
	containerName string,
) (string, error) {
	s.log.InfoContext(ctx, "Creating container...", "name", containerName)

	resp, containerCreateErr := s.cli.ContainerCreate(ctx, client.ContainerCreateOptions{
		Image:      imageName,
		HostConfig: &container.HostConfig{},
		Name:       containerName,
	})

	if containerCreateErr != nil {
		s.log.ErrorContext(ctx, "Failed to create container", "error", containerCreateErr)
		return "", containerCreateErr
	}

	s.log.InfoContext(ctx, "Container created successfully", "id", resp.ID)

	return resp.ID, nil
}

// Ping checks the connectivity to the Docker daemon by sending a ping request and logging the response.
func (s *dockerServiceImpl) Ping(ctx context.Context) error {
	s.log.InfoContext(ctx, "Pinging Docker daemon...")
	ping, err := s.cli.Ping(ctx, client.PingOptions{})
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to ping Docker daemon", "error", err)
		return err
	}
	s.log.InfoContext(ctx, "Docker daemon ping successful", "api_version", ping.APIVersion)
	return nil
}

// PullImage pulls the specified Docker image from the registry and logs the progress.
func (s *dockerServiceImpl) PullImage(ctx context.Context, imageName string) error {
	s.log.InfoContext(ctx, "Pulling image", "image", imageName)

	reader, imagePullErr := s.cli.ImagePull(ctx, imageName, client.ImagePullOptions{})
	if imagePullErr != nil {
		s.log.ErrorContext(ctx, "Failed to pull image", "error", imagePullErr)
		return imagePullErr
	}
	defer reader.Close()

	_, copyErr := io.Copy(os.Stdout, reader)
	if copyErr != nil {
		s.log.ErrorContext(ctx, "Failed to copy image pull progress", "error", copyErr)
		return copyErr
	}

	s.log.InfoContext(ctx, "Image pulled successfully", "image", imageName)
	return nil
}

// StartContainer starts the container with the specified ID and logs the outcome.
func (s *dockerServiceImpl) StartContainer(ctx context.Context, containerID string) error {
	s.log.InfoContext(ctx, "Starting container...", "id", containerID)

	if _, containerStartErr := s.cli.ContainerStart(
		ctx,
		containerID,
		client.ContainerStartOptions{},
	); containerStartErr != nil {
		s.log.ErrorContext(ctx, "Failed to start container", "error", containerStartErr)
		return containerStartErr
	}

	s.log.InfoContext(ctx, "Container started successfully", "id", containerID)
	return nil
}
