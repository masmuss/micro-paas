// Package handler provides HTTP handlers for instance management.
package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/masmuss/micro-paas/internal/model"
	"github.com/masmuss/micro-paas/internal/repository"
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

	if err := json.NewDecoder(r.Body).Decode(&instance); err != nil {
		h.logger.Error("failed to decode instance", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.repo.Create(ctx, &instance); err != nil {
		h.logger.Error("Failed to create instance", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(instance); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

// List handles HTTP GET /instances to list all instances.
// Returns a JSON array of all instances or 500 on error.
func (h *InstanceHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	instances, listErr := h.repo.List(ctx)
	if listErr != nil {
		h.logger.Error("Failed to list instances", "error", listErr)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if encodeErr := json.NewEncoder(w).Encode(instances); encodeErr != nil {
		h.logger.Error("failed to encode response", "error", encodeErr)
	}
}
