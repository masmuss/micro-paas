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

// DockerServiceImpl is the concrete implementation of DockerService interface.
type DockerServiceImpl struct {
	cli *client.Client
	log *slog.Logger
}

func newDockerServiceImpl(cfg *config.Config, log *slog.Logger) (DockerService, error) {
	cli, err := client.New(
		client.WithHost("unix://"+cfg.DockerSocket),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, err
	}
	return &DockerServiceImpl{
		cli: cli,
		log: log.With("component", "docker_service"),
	}, nil
}

// CreateContainer creates a new Docker container with the specified image and name, returning the container ID or an error.
func (s *DockerServiceImpl) CreateContainer(
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

// Ping checks the connectivity to the Docker daemon and logs the API version.
func (s *DockerServiceImpl) Ping(ctx context.Context) error {
	s.log.InfoContext(ctx, "Pinging Docker daemon...")
	ping, err := s.cli.Ping(ctx, client.PingOptions{})
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to ping Docker daemon", "error", err)
		return err
	}
	s.log.InfoContext(ctx, "Docker daemon ping successful", "api_version", ping.APIVersion)
	return nil
}

// PullImage pulls the specified Docker image, logging progress and errors.
func (s *DockerServiceImpl) PullImage(ctx context.Context, imageName string) error {
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

// StartContainer starts the Docker container with the given ID, logging the process and any errors.
func (s *DockerServiceImpl) StartContainer(ctx context.Context, containerID string) error {
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
