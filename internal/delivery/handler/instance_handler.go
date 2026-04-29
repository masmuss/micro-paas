package handler

import (
	"log/slog"
	"net/http"

	"github.com/masmuss/micro-paas/internal/delivery/dto/req"
	"github.com/masmuss/micro-paas/internal/delivery/dto/res"
	"github.com/masmuss/micro-paas/internal/model"
	"github.com/masmuss/micro-paas/internal/repository"
	"github.com/masmuss/micro-paas/internal/service"
	"github.com/masmuss/micro-paas/pkg/response"
)

type InstanceHandler struct {
	dockerSvc service.DockerService
	repo      repository.InstanceRepository
	logger    *slog.Logger
}

func NewInstanceHandler(
	dockerSvc service.DockerService,
	repo repository.InstanceRepository,
	logger *slog.Logger,
) *InstanceHandler {
	return &InstanceHandler{
		dockerSvc: dockerSvc,
		repo:      repo,
		logger:    logger,
	}
}

func (h *InstanceHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var reqBody req.CreateInstance

	if err := response.DecodeJSON(r, &reqBody); err != nil {
		h.logger.ErrorContext(ctx, "failed to decode request", "error", err)
		_ = response.WriteJSON(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	instance := &model.Instance{
		Name:      reqBody.Name,
		Subdomain: reqBody.Subdomain,
		Status:    "running",
	}

	if pullErr := h.dockerSvc.PullImage(ctx, reqBody.Image); pullErr != nil {
		h.logger.ErrorContext(ctx, "Failed to pull image", "error", pullErr)
		_ = response.WriteJSON(w, http.StatusInternalServerError, "Failed to pull image", nil)
		return
	}

	cID, createContainerErr := h.dockerSvc.CreateContainer(ctx, reqBody.Image, instance.Name)
	if createContainerErr != nil {
		h.logger.ErrorContext(ctx, "Failed to create container", "error", createContainerErr)
		_ = response.WriteJSON(w, http.StatusInternalServerError, "Failed to create container", nil)
		return
	}

	if startContainerErr := h.dockerSvc.StartContainer(ctx, cID); startContainerErr != nil {
		h.logger.ErrorContext(ctx, "Failed to start container", "error", startContainerErr)
		_ = response.WriteJSON(w, http.StatusInternalServerError, "Failed to start container", nil)
		return
	}

	instance.ContainerID = cID

	if err := h.repo.Create(ctx, instance); err != nil {
		h.logger.ErrorContext(ctx, "Failed to create instance", "error", err)
		_ = response.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	instanceRes := toResponse(instance)
	_ = response.WriteJSON(w, http.StatusCreated, "Instance created successfully", instanceRes)
}

func (h *InstanceHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	instances, listErr := h.repo.List(ctx)

	if listErr != nil {
		h.logger.ErrorContext(ctx, "Failed to list instances", "error", listErr)
		_ = response.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	instancesRes := toListResponse(instances)
	_ = response.WriteJSON(w, http.StatusOK, "Instances retrieved successfully", instancesRes)
}

func toResponse(m *model.Instance) *res.Instance {
	return &res.Instance{
		ID:        m.ID,
		Name:      m.Name,
		Subdomain: m.Subdomain,
		Status:    m.Status,
	}
}

func toListResponse(instances []*model.Instance) res.Instances {
	result := make(res.Instances, len(instances))
	for i, m := range instances {
		result[i] = toResponse(m)
	}
	return result
}
