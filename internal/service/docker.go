// Package service provides application services that encapsulate business logic and interactions with external systems.
package service

import (
	"context"
	"log/slog"

	"github.com/masmuss/micro-paas/internal/config"
	"github.com/moby/moby/client"
)

// DockerService defines the interface for interacting with the Docker daemon.
type DockerService interface {
	Ping(ctx context.Context) error
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
