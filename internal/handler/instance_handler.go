// Package handler provides HTTP handlers for instance management.
package handler

import (
	"log/slog"
	"net/http"

	"github.com/masmuss/micro-paas/internal/model"
	"github.com/masmuss/micro-paas/internal/repository"
	"github.com/masmuss/micro-paas/pkg/response"
)

// InstanceHandler provides HTTP handlers for managing application instances.
// It depends on an InstanceRepository for data access and a logger for structured logging.
type InstanceHandler struct {
	// repo is the data access layer for instances.
	repo repository.InstanceRepository
	// logger is used for logging errors and info.
	logger *slog.Logger
}

// NewInstanceHandler constructs an InstanceHandler with the given repository and logger.
// Use this to register instance-related HTTP endpoints.
func NewInstanceHandler(repo repository.InstanceRepository, logger *slog.Logger) *InstanceHandler {
	return &InstanceHandler{
		repo:   repo,
		logger: logger,
	}
}

// Create handles HTTP POST /instances to create a new instance.
// It expects a JSON body representing an instance, validates and stores it, and returns the created instance as JSON.
// Responds with 400 for invalid input, 500 for server errors, 201 for success.
func (h *InstanceHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var instance model.Instance
	if err := response.DecodeJSON(r, &instance); err != nil {
		h.logger.ErrorContext(ctx, "failed to decode instance", "error", err)
		writeJSONErr := response.WriteJSON(w, http.StatusBadRequest, "Invalid request body", nil)
		if writeJSONErr != nil {
			h.logger.ErrorContext(ctx, "failed to encode response", "error", writeJSONErr)
		}
		return
	}

	if err := h.repo.Create(ctx, &instance); err != nil {
		h.logger.ErrorContext(ctx, "Failed to create instance", "error", err)
		writeJSONErr := response.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
		if writeJSONErr != nil {
			h.logger.ErrorContext(ctx, "failed to encode response", "error", writeJSONErr)
		}
		return
	}

	if writeJSONErr := response.WriteJSON(
		w,
		http.StatusCreated,
		"Instance created successfully",
		instance,
	); writeJSONErr != nil {
		h.logger.ErrorContext(ctx, "failed to encode response", "error", writeJSONErr)
	}
}

// List handles HTTP GET /instances to list all instances.
// Returns a JSON array of all instances or 500 on error.
func (h *InstanceHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	instances, listErr := h.repo.List(ctx)

	if listErr != nil {
		h.logger.ErrorContext(ctx, "Failed to list instances", "error", listErr)
		err := response.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
		if err != nil {
			h.logger.ErrorContext(ctx, "failed to encode response", "error", err)
		}
		return
	}

	if err := response.WriteJSON(w, http.StatusOK, "Instances retrieved successfully", instances); err != nil {
		h.logger.ErrorContext(ctx, "failed to encode response", "error", err)
	}
}
