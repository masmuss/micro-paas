// Package handler provides HTTP handlers for the instance resource.
package handler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/masmuss/micro-paas/internal/delivery/response"
	"github.com/masmuss/micro-paas/internal/model"
	"github.com/masmuss/micro-paas/internal/repository"
	"github.com/masmuss/micro-paas/internal/service"
	"github.com/masmuss/micro-paas/internal/validation"
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
	Name      string            `json:"name"`
	Image     string            `json:"image"`
	Subdomain string            `json:"subdomain"`
	Port      int               `json:"port"`
	Env       map[string]string `json:"env"`
}

type instanceRes struct {
	ID        int64             `json:"id"`
	Name      string            `json:"name"`
	Subdomain string            `json:"subdomain"`
	Port      int               `json:"port"`
	Status    string            `json:"status"`
	Env       map[string]string `json:"env"`
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

	if err := validation.ValidateCreateInstance(reqBody.Name, reqBody.Image, reqBody.Subdomain); err != nil {
		if writeErr := response.WriteJSON(w, http.StatusBadRequest, err.Error(), nil); writeErr != nil {
			h.logger.ErrorContext(ctx, "failed to write error response", "error", writeErr)
		}
		return
	}

	if _, err := h.repo.GetBySubdomain(ctx, reqBody.Subdomain); err == nil {
		if writeErr := response.WriteJSON(w, http.StatusConflict, "subdomain already in use", nil); writeErr != nil {
			h.logger.ErrorContext(ctx, "failed to write error response", "error", writeErr)
		}
		return
	}

	if _, err := h.repo.GetByName(ctx, reqBody.Name); err == nil {
		if writeErr := response.WriteJSON(w, http.StatusConflict, "name already in use", nil); writeErr != nil {
			h.logger.ErrorContext(ctx, "failed to write error response", "error", writeErr)
		}
		return
	}

	instance := &model.Instance{
		Name:      reqBody.Name,
		Subdomain: reqBody.Subdomain,
		Port:      reqBody.Port,
		Status:    model.StatusRunning,
		Env:       reqBody.Env,
	}

	if instance.Port == 0 {
		instance.Port = 80
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

	cID, createContainerErr := h.dockerSvc.CreateContainer(ctx, reqBody.Image, instance.Name, instance.Env)
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
		h.cleanupContainer(ctx, cID)
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
		h.cleanupContainer(ctx, cID)
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

	for _, instance := range instances {
		h.syncStatus(ctx, instance)
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
		Port:      m.Port,
		Status:    m.Status.String(),
		Env:       m.Env,
	}
}

func (h *InstanceHandler) cleanupContainer(ctx context.Context, containerID string) {
	if stopErr := h.dockerSvc.StopContainer(ctx, containerID); stopErr != nil {
		h.logger.ErrorContext(ctx, "Failed to stop container during cleanup", "error", stopErr)
	}
	if removeErr := h.dockerSvc.RemoveContainer(ctx, containerID); removeErr != nil {
		h.logger.ErrorContext(ctx, "Failed to remove container during cleanup", "error", removeErr)
	}
}

// Delete handles the deletion of an instance and its container.
func (h *InstanceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		if writeErr := response.WriteJSON(w, http.StatusBadRequest, "Invalid ID", nil); writeErr != nil {
			h.logger.ErrorContext(ctx, "failed to write error response", "error", writeErr)
		}
		return
	}

	instance, err := h.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			if writeErr := response.WriteJSON(w, http.StatusNotFound, "Instance not found", nil); writeErr != nil {
				h.logger.ErrorContext(ctx, "failed to write error response", "error", writeErr)
			}
			return
		}
		h.logger.ErrorContext(ctx, "Failed to get instance", "error", err)
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

	h.cleanupContainer(ctx, instance.ContainerID)

	if deleteContainerErr := h.repo.Delete(ctx, id); deleteContainerErr != nil {
		h.logger.ErrorContext(
			ctx,
			"Failed to delete instance",
			"error",
			deleteContainerErr,
		)
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

	if writeErr := response.WriteJSON(w, http.StatusOK, "Instance deleted successfully", nil); writeErr != nil {
		h.logger.ErrorContext(ctx, "failed to write success response", "error", writeErr)
	}
}

func toListResponse(instances []*model.Instance) instancesRes {
	result := make(instancesRes, len(instances))
	for i, m := range instances {
		result[i] = toResponse(m)
	}
	return result
}

func (h *InstanceHandler) syncStatus(ctx context.Context, instance *model.Instance) {
	status, err := h.dockerSvc.GetContainerStatus(ctx, instance.ContainerID)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to get container status", "error", err)
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
		instance.Status = newStatus
		if updateInstanceErr := h.repo.Update(ctx, instance); updateInstanceErr != nil {
			h.logger.ErrorContext(
				ctx,
				"Failed to update instance status",
				"error", updateInstanceErr,
			)
		}
	}
}

// GetByID handles retrieving a single instance by ID.
func (h *InstanceHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		if writeErr := response.WriteJSON(w, http.StatusBadRequest, "Invalid ID", nil); writeErr != nil {
			h.logger.ErrorContext(ctx, "failed to write error response", "error", writeErr)
		}
		return
	}

	instance, err := h.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			if writeErr := response.WriteJSON(w, http.StatusNotFound, "Instance not found", nil); writeErr != nil {
				h.logger.ErrorContext(ctx, "failed to write error response", "error", writeErr)
			}
			return
		}
		h.logger.ErrorContext(ctx, "Failed to get instance", "error", err)
		if writeErr := response.WriteJSON(
			w,
			http.StatusInternalServerError,
			"Internal server error",
			nil,
		); writeErr != nil {
			h.logger.ErrorContext(ctx,
				"failed to write error response",
				"error", writeErr,
			)
		}
		return
	}

	h.syncStatus(ctx, instance)

	res := toResponse(instance)
	if writeErr := response.WriteJSON(w, http.StatusOK, "Instance retrieved successfully", res); writeErr != nil {
		h.logger.ErrorContext(ctx, "failed to write success response", "error", writeErr)
	}
}

// Logs handles retrieving the logs for an instance.
func (h *InstanceHandler) Logs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	instance, err := h.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			http.Error(w, "Instance not found", http.StatusNotFound)
			return
		}
		h.logger.ErrorContext(ctx, "Failed to get instance", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	reader, err := h.dockerSvc.GetContainerLogs(ctx, instance.ContainerID)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to get logs", "error", err)
		http.Error(w, "Failed to get logs", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "text/plain")
	if _, ioCopyErr := io.Copy(w, reader); ioCopyErr != nil {
		h.logger.ErrorContext(ctx, "Failed to copy logs to response", "error", ioCopyErr)
	}
}
