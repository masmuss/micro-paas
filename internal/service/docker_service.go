// Package service provides application services that encapsulate business logic and interactions with external systems.
package service

import (
	"context"
	"log/slog"

	"github.com/masmuss/micro-paas/internal/config"
)

// DockerService defines the interface for interacting with the Docker daemon.
type DockerService interface {
	CreateContainer(ctx context.Context, imageName string, containerName string) (string, error)
	Ping(ctx context.Context) error
	PullImage(ctx context.Context, imageName string) error
	StartContainer(ctx context.Context, containerID string) error
	StopContainer(ctx context.Context, containerID string) error
	RemoveContainer(ctx context.Context, containerID string) error
	GetContainerStatus(ctx context.Context, containerID string) (string, error)
}

// NewDockerService creates a new DockerService using the provided configuration and logger.
func NewDockerService(cfg *config.Config, log *slog.Logger) (DockerService, error) {
	return newDockerServiceImpl(cfg, log)
}
