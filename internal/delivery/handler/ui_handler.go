package handler

import (
	"bufio"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/masmuss/micro-paas/internal/config"
	"github.com/masmuss/micro-paas/internal/model"
	"github.com/masmuss/micro-paas/internal/repository"
	"github.com/masmuss/micro-paas/internal/service"
)

// UIHandler handles requests for the web dashboard and HTMX fragments.
type UIHandler struct {
	repo      repository.InstanceRepository
	dockerSvc service.DockerService
	tmpl      *template.Template
	cfg       *config.Config
}

// NewUIHandler creates a new UIHandler and parses templates from the filesystem.
func NewUIHandler(repo repository.InstanceRepository, dockerSvc service.DockerService, cfg *config.Config) *UIHandler {
	// Parse all templates in the html directory
	tmpl := template.Must(template.ParseGlob(filepath.Join("internal", "delivery", "html", "*.html")))

	return &UIHandler{
		repo:      repo,
		dockerSvc: dockerSvc,
		tmpl:      tmpl,
		cfg:       cfg,
	}
}

// Dashboard renders the main dashboard layout.
func (h *UIHandler) Dashboard(w http.ResponseWriter, _ *http.Request) {
	data := struct {
		MainDomain string
	}{
		MainDomain: h.cfg.MainDomain,
	}
	err := h.tmpl.ExecuteTemplate(w, "layout", data)
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
		Instances  interface{}
		MainDomain string
	}{
		Instances:  instances,
		MainDomain: h.cfg.MainDomain,
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
		err = h.dockerSvc.StartContainer(ctx, inst.ContainerID)
		if err != nil {
			w.Header().Set("HX-Trigger", fmt.Sprintf(`{"showToast": "Failed to start: %s"}`, err.Error()))
		} else {
			inst.Status = model.StatusRunning
			_ = h.repo.Update(ctx, inst)
			w.Header().Set("HX-Trigger", `{"showToast": "Instance started"}`)
		}
	}

	// Setelah aksi, kirim balik tabel yang sudah diupdate
	h.InstancesTable(w, r)
}

// StopInstance handles the UI request to stop a container and returns the updated table.
func (h *UIHandler) StopInstance(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	ctx := r.Context()
	inst, err := h.repo.GetByID(ctx, id)
	if err == nil {
		err = h.dockerSvc.StopContainer(ctx, inst.ContainerID)
		if err != nil {
			w.Header().Set("HX-Trigger", fmt.Sprintf(`{"showToast": "Failed to stop: %s"}`, err.Error()))
		} else {
			inst.Status = model.StatusStopped
			_ = h.repo.Update(ctx, inst)
			w.Header().Set("HX-Trigger", `{"showToast": "Instance stopped"}`)
		}
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

// DeleteInstance handles the UI request to delete an instance and returns the updated table.
func (h *UIHandler) DeleteInstance(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	ctx := r.Context()
	inst, err := h.repo.GetByID(ctx, id)
	if err == nil {
		_ = h.dockerSvc.StopContainer(ctx, inst.ContainerID)
		_ = h.dockerSvc.RemoveContainer(ctx, inst.ContainerID)
		_ = h.repo.Delete(ctx, id)
		w.Header().Set("HX-Trigger", `{"showToast": "Instance deleted"}`)
	}

	h.InstancesTable(w, r)
}

// CreateInstance handles the form submission for a new instance.
func (h *UIHandler) CreateInstance(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.Header().Set("HX-Trigger", `{"showToast": "Failed to parse form"}`)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	port, _ := strconv.Atoi(r.FormValue("port"))
	if port == 0 {
		port = 80
	}

	image := r.FormValue("image")
	name := r.FormValue("name")

	// Parse ENV from textarea (KEY=VALUE per line)
	envStr := r.FormValue("env")
	env := make(map[string]string)
	lines := strings.Split(envStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			env[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	inst := &model.Instance{
		Name:      name,
		Subdomain: r.FormValue("subdomain"),
		Port:      port,
		Status:    model.StatusRunning,
		Env:       env,
	}

	// 1. Pull Image
	err = h.dockerSvc.PullImage(r.Context(), image)
	if err != nil {
		w.Header().Set("HX-Trigger", fmt.Sprintf(`{"showToast": "Failed to pull image: %s"}`, err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. Create container
	containerID, err := h.dockerSvc.CreateContainer(r.Context(), image, name, inst.Env)
	if err != nil {
		w.Header().Set("HX-Trigger", fmt.Sprintf(`{"showToast": "Failed to create container: %s"}`, err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 3. Start container
	err = h.dockerSvc.StartContainer(r.Context(), containerID)
	if err != nil {
		inst.Status = model.StatusError
		w.Header().Set(
			"HX-Trigger",
			fmt.Sprintf(
				`{"showToast": "Container created but failed to start: %s"}`,
				err.Error(),
			),
		)
	}

	inst.ContainerID = containerID
	err = h.repo.Create(r.Context(), inst)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"showToast": "Failed to save instance to DB"}`)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"showToast": "Instance created and started!"}`)
	h.InstancesTable(w, r)
}
