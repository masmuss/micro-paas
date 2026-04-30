// Package handler provides HTTP handlers for the instance resource.
package handler

import (
	"log/slog"
	"net/http"

	"github.com/masmuss/micro-paas/internal/delivery/response"
	"github.com/masmuss/micro-paas/internal/model"
	"github.com/masmuss/micro-paas/internal/repository"
	"github.com/masmuss/micro-paas/internal/service"
)

// InstanceHandler handles HTTP requests for the instance resource.
type InstanceHandler struct {
	dockerSvc service.DockerService
	repo      repository.InstanceRepository
	logger    *slog.Logger
}

// NewInstanceHandler creates a new InstanceHandler.
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

type createInstanceReq struct {
	Name      string `json:"name"`
	Image     string `json:"image"`
	Subdomain string `json:"subdomain"`
}

type instanceRes struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Subdomain string `json:"subdomain"`
	Status    string `json:"status"`
}

type instancesRes []*instanceRes

// Create handles the creation of a new instance.
func (h *InstanceHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var reqBody createInstanceReq

	if err := response.DecodeJSON(r, &reqBody); err != nil {
		h.logger.ErrorContext(ctx, "failed to decode request", "error", err)
		if writeErr := response.WriteJSON(w, http.StatusBadRequest, "Invalid request body", nil); writeErr != nil {
			h.logger.ErrorContext(ctx, "failed to write error response", "error", writeErr)
		}
		return
	}

	instance := &model.Instance{
		Name:      reqBody.Name,
		Subdomain: reqBody.Subdomain,
		Status:    model.StatusRunning,
	}

	if pullErr := h.dockerSvc.PullImage(ctx, reqBody.Image); pullErr != nil {
		h.logger.ErrorContext(ctx, "Failed to pull image", "error", pullErr)
		if writeErr := response.WriteJSON(
			w,
			http.StatusInternalServerError,
			"Failed to pull image",
			nil,
		); writeErr != nil {
			h.logger.ErrorContext(ctx, "failed to write error response", "error", writeErr)
		}
		return
	}

	cID, createContainerErr := h.dockerSvc.CreateContainer(ctx, reqBody.Image, instance.Name)
	if createContainerErr != nil {
		h.logger.ErrorContext(ctx, "Failed to create container", "error", createContainerErr)
		if writeErr := response.WriteJSON(
			w,
			http.StatusInternalServerError,
			"Failed to create container",
			nil,
		); writeErr != nil {
			h.logger.ErrorContext(ctx, "failed to write error response", "error", writeErr)
		}
		return
	}

	if startContainerErr := h.dockerSvc.StartContainer(ctx, cID); startContainerErr != nil {
		h.logger.ErrorContext(ctx, "Failed to start container", "error", startContainerErr)
		if writeErr := response.WriteJSON(
			w,
			http.StatusInternalServerError,
			"Failed to start container",
			nil,
		); writeErr != nil {
			h.logger.ErrorContext(ctx, "failed to write error response", "error", writeErr)
		}
		return
	}

	instance.ContainerID = cID

	if err := h.repo.Create(ctx, instance); err != nil {
		h.logger.ErrorContext(ctx, "Failed to create instance", "error", err)
		if writeErr := response.WriteJSON(
			w,
			http.StatusInternalServerError,
			"Internal server error",
			nil,
		); writeErr != nil {
			h.logger.ErrorContext(ctx, "failed to write error response", "error", writeErr)
		}
		return
	}

	res := toResponse(instance)
	if writeErr := response.WriteJSON(w, http.StatusCreated, "Instance created successfully", res); writeErr != nil {
		h.logger.ErrorContext(ctx, "failed to write success response", "error", writeErr)
	}
}

// List handles the retrieval of all instances.
func (h *InstanceHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	instances, listErr := h.repo.List(ctx)

	if listErr != nil {
		h.logger.ErrorContext(ctx, "Failed to list instances", "error", listErr)
		if writeErr := response.WriteJSON(
			w,
			http.StatusInternalServerError,
			"Internal server error",
			nil,
		); writeErr != nil {
			h.logger.ErrorContext(ctx, "failed to write error response", "error", writeErr)
		}
		return
	}

	res := toListResponse(instances)
	if writeErr := response.WriteJSON(w, http.StatusOK, "Instances retrieved successfully", res); writeErr != nil {
		h.logger.ErrorContext(ctx, "failed to write success response", "error", writeErr)
	}
}

func toResponse(m *model.Instance) *instanceRes {
	return &instanceRes{
		ID:        m.ID,
		Name:      m.Name,
		Subdomain: m.Subdomain,
		Status:    m.Status.String(),
	}
}

func toListResponse(instances []*model.Instance) instancesRes {
	result := make(instancesRes, len(instances))
	for i, m := range instances {
		result[i] = toResponse(m)
	}
	return result
}
