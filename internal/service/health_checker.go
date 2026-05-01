package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/masmuss/micro-paas/internal/model"
	"github.com/masmuss/micro-paas/internal/repository"
)

// HealthChecker periodically syncs container statuses from Docker to the database.
type HealthChecker struct {
	dockerSvc DockerService
	repo      repository.InstanceRepository
	logger    *slog.Logger
	interval  time.Duration
	stopCh    chan struct{}
}

// NewHealthChecker creates a new HealthChecker.
func NewHealthChecker(
	dockerSvc DockerService,
	repo repository.InstanceRepository,
	logger *slog.Logger,
	interval time.Duration,
) *HealthChecker {
	return &HealthChecker{
		dockerSvc: dockerSvc,
		repo:      repo,
		logger:    logger.With("component", "health_checker"),
		interval:  interval,
		stopCh:    make(chan struct{}),
	}
}

// Start begins the periodic health checking.
func (h *HealthChecker) Start(ctx context.Context) {
	h.logger.InfoContext(ctx, "Starting health checker", "interval", h.interval)
	go h.run(ctx)
}

// Stop stops the health checker.
func (h *HealthChecker) Stop() {
	h.logger.Info("Stopping health checker")
	close(h.stopCh)
}

func (h *HealthChecker) run(ctx context.Context) {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	h.checkAll(ctx)

	for {
		select {
		case <-ticker.C:
			h.checkAll(ctx)
		case <-h.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (h *HealthChecker) checkAll(ctx context.Context) {
	instances, err := h.repo.List(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to list instances", "error", err)
		return
	}

	for _, instance := range instances {
		h.checkInstance(ctx, instance)
	}
}

func (h *HealthChecker) checkInstance(ctx context.Context, instance *model.Instance) {
	status, err := h.dockerSvc.GetContainerStatus(ctx, instance.ContainerID)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to get container status", "container_id", instance.ContainerID, "error", err)
		return
	}

	var newStatus model.Status
	switch status {
	case "running":
		newStatus = model.StatusRunning
	case "exited", "stopped":
		newStatus = model.StatusStopped
	default:
		newStatus = model.StatusError
	}

	if newStatus != instance.Status {
		h.logger.InfoContext(ctx, "Status changed",
			"instance_id", instance.ID,
			"old", instance.Status,
			"new", newStatus,
		)
		instance.Status = newStatus
		if updateErr := h.repo.Update(ctx, instance); updateErr != nil {
			h.logger.ErrorContext(ctx, "Failed to update instance status", "error", updateErr)
		}
	}
}
