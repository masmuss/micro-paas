package handler

import (
	"bufio"
	"html/template"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/masmuss/micro-paas/internal/model"
	"github.com/masmuss/micro-paas/internal/repository"
	"github.com/masmuss/micro-paas/internal/service"
)

// UIHandler handles requests for the web dashboard and HTMX fragments.
type UIHandler struct {
	repo      repository.InstanceRepository
	dockerSvc service.DockerService
	tmpl      *template.Template
}

// NewUIHandler creates a new UIHandler and parses templates from the filesystem.
func NewUIHandler(repo repository.InstanceRepository, dockerSvc service.DockerService) *UIHandler {
	// Parse all templates in the html directory
	tmpl := template.Must(template.ParseGlob(filepath.Join("internal", "delivery", "html", "*.html")))

	return &UIHandler{
		repo:      repo,
		dockerSvc: dockerSvc,
		tmpl:      tmpl,
	}
}

// Dashboard renders the main dashboard layout.
func (h *UIHandler) Dashboard(w http.ResponseWriter, _ *http.Request) {
	err := h.tmpl.ExecuteTemplate(w, "layout", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// InstancesTable renders the instance list table fragment for HTMX requests.
func (h *UIHandler) InstancesTable(w http.ResponseWriter, r *http.Request) {
	instances, err := h.repo.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Instances interface{}
	}{
		Instances: instances,
	}

	err = h.tmpl.ExecuteTemplate(w, "instances-table", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// StartInstance handles the UI request to start a container and returns the updated table.
func (h *UIHandler) StartInstance(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	ctx := r.Context()
	inst, err := h.repo.GetByID(ctx, id)
	if err == nil {
		_ = h.dockerSvc.StartContainer(ctx, inst.ContainerID)
		inst.Status = model.StatusRunning
		_ = h.repo.Update(ctx, inst)
	}

	// After action, send back the updated table
	h.InstancesTable(w, r)
}

// StopInstance handles the UI request to stop a container and returns the updated table.
func (h *UIHandler) StopInstance(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	ctx := r.Context()
	inst, err := h.repo.GetByID(ctx, id)
	if err == nil {
		_ = h.dockerSvc.StopContainer(ctx, inst.ContainerID)
		inst.Status = model.StatusStopped
		_ = h.repo.Update(ctx, inst)
	}

	h.InstancesTable(w, r)
}

// LogsModal renders the logs content fragment for the UI modal.
func (h *UIHandler) LogsModal(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	inst, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Instance not found", http.StatusNotFound)
		return
	}

	// Fetch last 100 lines of logs synchronously for the modal
	reader, err := h.dockerSvc.GetContainerLogs(r.Context(), inst.ContainerID)
	var logs string
	if err == nil {
		defer reader.Close()
		scanner := bufio.NewScanner(reader)
		count := 0
		for scanner.Scan() && count < 100 {
			line := scanner.Text()
			if len(line) > 8 {
				logs += line[8:] + "\n"
			}
			count++
		}
	}

	data := struct {
		Name string
		Logs string
	}{
		Name: inst.Name,
		Logs: logs,
	}

	err = h.tmpl.ExecuteTemplate(w, "logs-content", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
