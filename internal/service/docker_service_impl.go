package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/masmuss/micro-paas/internal/config"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// DockerServiceImpl implements [DockerService] for interacting with the Docker daemon.
type DockerServiceImpl struct {
	cli *client.Client
	log *slog.Logger
}

var _ DockerService = (*DockerServiceImpl)(nil)

func newDockerServiceImpl(cfg *config.Config, log *slog.Logger) (DockerService, error) {
	cli, err := client.New(
		client.WithHost("unix://"+cfg.DockerSocket),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
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
	env map[string]string,
) (string, error) {
	s.log.InfoContext(ctx, "Creating container", "name", containerName, "image", imageName)

	var envList []string
	for k, v := range env {
		envList = append(envList, fmt.Sprintf("%s=%s", k, v))
	}

	options := client.ContainerCreateOptions{
		Name: containerName,
		Config: &container.Config{
			Image: imageName,
			Env:   envList,
		},
	}

	resp, err := s.cli.ContainerCreate(ctx, options)
	if err != nil {
		return "", fmt.Errorf("create container %q: %w", containerName, err)
	}

	s.log.InfoContext(ctx, "Container created successfully", "id", resp.ID)

	return resp.ID, nil
}

// Ping checks the connectivity to the Docker daemon and logs the API version.
func (s *DockerServiceImpl) Ping(ctx context.Context) error {
	s.log.InfoContext(ctx, "Pinging Docker daemon")
	ping, err := s.cli.Ping(ctx, client.PingOptions{})
	if err != nil {
		return fmt.Errorf("ping docker daemon: %w", err)
	}
	s.log.InfoContext(ctx, "Docker daemon ping successful", "api_version", ping.APIVersion)
	return nil
}

// PullImage pulls the specified Docker image, logging progress and errors.
func (s *DockerServiceImpl) PullImage(ctx context.Context, imageName string) error {
	s.log.InfoContext(ctx, "Pulling image", "image", imageName)

	reader, imagePullErr := s.cli.ImagePull(ctx, imageName, client.ImagePullOptions{})
	if imagePullErr != nil {
		return fmt.Errorf("pull image %q: %w", imageName, imagePullErr)
	}
	defer reader.Close()

	_, copyErr := io.Copy(io.Discard, reader)
	if copyErr != nil {
		return fmt.Errorf("copy image pull progress: %w", copyErr)
	}

	s.log.InfoContext(ctx, "Image pulled successfully", "image", imageName)
	return nil
}

// StartContainer starts the Docker container with the given ID, logging the process and any errors.
func (s *DockerServiceImpl) StartContainer(ctx context.Context, containerID string) error {
	s.log.InfoContext(ctx, "Starting container", "id", containerID)

	if _, containerStartErr := s.cli.ContainerStart(
		ctx,
		containerID,
		client.ContainerStartOptions{},
	); containerStartErr != nil {
		return fmt.Errorf("start container %q: %w", containerID, containerStartErr)
	}

	s.log.InfoContext(ctx, "Container started successfully", "id", containerID)
	return nil
}

// StopContainer stops the Docker container with the given ID.
func (s *DockerServiceImpl) StopContainer(ctx context.Context, containerID string) error {
	s.log.InfoContext(ctx, "Stopping container", "id", containerID)

	timeout := 10
	if _, err := s.cli.ContainerStop(ctx, containerID, client.ContainerStopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("stop container %q: %w", containerID, err)
	}

	s.log.InfoContext(ctx, "Container stopped successfully", "id", containerID)
	return nil
}

// RemoveContainer removes the Docker container with the given ID.
func (s *DockerServiceImpl) RemoveContainer(ctx context.Context, containerID string) error {
	s.log.InfoContext(ctx, "Removing container", "id", containerID)

	if _, err := s.cli.ContainerRemove(ctx, containerID, client.ContainerRemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("remove container %q: %w", containerID, err)
	}

	s.log.InfoContext(ctx, "Container removed successfully", "id", containerID)
	return nil
}

// GetContainerStatus returns the current status of a container.
func (s *DockerServiceImpl) GetContainerStatus(ctx context.Context, containerID string) (string, error) {
	inspect, err := s.cli.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	if err != nil {
		return "", fmt.Errorf("inspect container %q: %w", containerID, err)
	}
	if inspect.Container.State == nil {
		return "removed", nil
	}
	return string(inspect.Container.State.Status), nil
}

// GetContainerLogs returns a stream of the container's logs.
func (s *DockerServiceImpl) GetContainerLogs(ctx context.Context, containerID string) (io.ReadCloser, error) {
	options := client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     false,
		Timestamps: true,
		Tail:       "100",
	}

	return s.cli.ContainerLogs(ctx, containerID, options)
}
