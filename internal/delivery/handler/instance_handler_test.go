package handler

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/masmuss/micro-paas/internal/model"
	"github.com/masmuss/micro-paas/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockDockerService struct {
	mock.Mock
}

func (m *mockDockerService) CreateContainer(
	ctx context.Context,
	imageName string,
	containerName string,
) (string, error) {
	args := m.Called(ctx, imageName, containerName)
	return args.String(0), args.Error(1)
}

func (m *mockDockerService) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockDockerService) PullImage(ctx context.Context, imageName string) error {
	args := m.Called(ctx, imageName)
	return args.Error(0)
}

func (m *mockDockerService) StartContainer(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

func (m *mockDockerService) StopContainer(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

func (m *mockDockerService) RemoveContainer(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

func (m *mockDockerService) GetContainerStatus(ctx context.Context, containerID string) (string, error) {
	args := m.Called(ctx, containerID)
	return args.String(0), args.Error(1)
}

type mockInstanceRepository struct {
	mock.Mock
}

func (m *mockInstanceRepository) Create(ctx context.Context, instance *model.Instance) error {
	args := m.Called(ctx, instance)
	return args.Error(0)
}

func (m *mockInstanceRepository) GetByID(ctx context.Context, id int64) (*model.Instance, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	inst, _ := args.Get(0).(*model.Instance)
	return inst, args.Error(1)
}

func (m *mockInstanceRepository) GetByName(ctx context.Context, name string) (*model.Instance, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	inst, _ := args.Get(0).(*model.Instance)
	return inst, args.Error(1)
}

func (m *mockInstanceRepository) GetByContainerID(ctx context.Context, containerID string) (*model.Instance, error) {
	args := m.Called(ctx, containerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	inst, _ := args.Get(0).(*model.Instance)
	return inst, args.Error(1)
}

func (m *mockInstanceRepository) GetBySubdomain(ctx context.Context, subdomain string) (*model.Instance, error) {
	args := m.Called(ctx, subdomain)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	inst, _ := args.Get(0).(*model.Instance)
	return inst, args.Error(1)
}

func (m *mockInstanceRepository) Delete(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockInstanceRepository) Update(ctx context.Context, instance *model.Instance) error {
	args := m.Called(ctx, instance)
	return args.Error(0)
}

func (m *mockInstanceRepository) List(ctx context.Context) ([]*model.Instance, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	list, _ := args.Get(0).([]*model.Instance)
	return list, args.Error(1)
}

func TestCreateInstance_Success(t *testing.T) {
	mockDocker := new(mockDockerService)
	mockRepo := new(mockInstanceRepository)
	handler := NewInstanceHandler(mockDocker, mockRepo, nil)

	mockDocker.On("PullImage", mock.Anything, "nginx:latest").Return(nil)
	mockDocker.On("CreateContainer", mock.Anything, "nginx:latest", "test-app").Return("container-123", nil)
	mockDocker.On("StartContainer", mock.Anything, "container-123").Return(nil)
	mockRepo.On("GetBySubdomain", mock.Anything, "test").Return(nil, repository.ErrNotFound)
	mockRepo.On("GetByName", mock.Anything, "test-app").Return(nil, repository.ErrNotFound)
	mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	body := `{"name":"test-app","image":"nginx:latest","subdomain":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/instances", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Create(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockDocker.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestCreateInstance_ValidationError(t *testing.T) {
	mockDocker := new(mockDockerService)
	mockRepo := new(mockInstanceRepository)
	handler := NewInstanceHandler(mockDocker, mockRepo, nil)

	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{"missing name", `{"image":"nginx:latest","subdomain":"test"}`, http.StatusBadRequest},
		{"missing image", `{"name":"test-app","subdomain":"test"}`, http.StatusBadRequest},
		{"missing subdomain", `{"name":"test-app","image":"nginx:latest"}`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/instances", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			handler.Create(w, req)
			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}

func TestCreateInstance_DuplicateSubdomain(t *testing.T) {
	mockDocker := new(mockDockerService)
	mockRepo := new(mockInstanceRepository)
	handler := NewInstanceHandler(mockDocker, mockRepo, nil)

	mockRepo.On("GetBySubdomain", mock.Anything, "test").Return(&model.Instance{ID: 1}, nil)

	body := `{"name":"test-app","image":"nginx:latest","subdomain":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/instances", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Create(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestCreateInstance_DockerFailure_Rollback(t *testing.T) {
	mockDocker := new(mockDockerService)
	mockRepo := new(mockInstanceRepository)
	handler := NewInstanceHandler(mockDocker, mockRepo, slog.Default())

	mockDocker.On("PullImage", mock.Anything, "nginx:latest").Return(nil)
	mockDocker.On("CreateContainer", mock.Anything, "nginx:latest", "test-app").Return("container-123", nil)
	mockDocker.On("StartContainer", mock.Anything, "container-123").Return(assert.AnError)
	mockDocker.On("StopContainer", mock.Anything, "container-123").Return(nil)
	mockDocker.On("RemoveContainer", mock.Anything, "container-123").Return(nil)
	mockRepo.On("GetBySubdomain", mock.Anything, "test").Return(nil, repository.ErrNotFound)
	mockRepo.On("GetByName", mock.Anything, "test-app").Return(nil, repository.ErrNotFound)

	body := `{"name":"test-app","image":"nginx:latest","subdomain":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/instances", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Create(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockDocker.AssertExpectations(t)
}

func TestListInstances(t *testing.T) {
	mockDocker := new(mockDockerService)
	mockRepo := new(mockInstanceRepository)
	handler := NewInstanceHandler(mockDocker, mockRepo, nil)

	instances := []*model.Instance{
		{ID: 1, Name: "app1", Subdomain: "sub1", Status: model.StatusRunning},
		{ID: 2, Name: "app2", Subdomain: "sub2", Status: model.StatusRunning},
	}
	mockRepo.On("List", mock.Anything).Return(instances, nil)
	mockDocker.On("GetContainerStatus", mock.Anything, mock.Anything).Return("running", nil)

	req := httptest.NewRequest(http.MethodGet, "/api/instances", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestGetByID_Success(t *testing.T) {
	mockDocker := new(mockDockerService)
	mockRepo := new(mockInstanceRepository)
	handler := NewInstanceHandler(mockDocker, mockRepo, slog.Default())

	instance := &model.Instance{ID: 1, Name: "app1", Subdomain: "sub1", Status: model.StatusRunning, ContainerID: "c1"}
	mockRepo.On("GetByID", mock.Anything, int64(1)).Return(instance, nil)
	mockDocker.On("GetContainerStatus", mock.Anything, "c1").Return("running", nil)

	req := httptest.NewRequest(http.MethodGet, "/api/instances/1", nil)
	ctx := chi.NewRouteContext()
	ctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, ctx))
	w := httptest.NewRecorder()

	handler.GetByID(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
	mockDocker.AssertExpectations(t)
}

func TestGetByID_NotFound(t *testing.T) {
	mockDocker := new(mockDockerService)
	mockRepo := new(mockInstanceRepository)
	handler := NewInstanceHandler(mockDocker, mockRepo, nil)

	mockRepo.On("GetByID", mock.Anything, int64(999)).Return(nil, repository.ErrNotFound)

	req := httptest.NewRequest(http.MethodGet, "/api/instances/999", nil)
	ctx := chi.NewRouteContext()
	ctx.URLParams.Add("id", "999")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, ctx))
	w := httptest.NewRecorder()

	handler.GetByID(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestDelete_Success(t *testing.T) {
	mockDocker := new(mockDockerService)
	mockRepo := new(mockInstanceRepository)
	handler := NewInstanceHandler(mockDocker, mockRepo, nil)

	instance := &model.Instance{ID: 1, Name: "app1", ContainerID: "c1"}
	mockRepo.On("GetByID", mock.Anything, int64(1)).Return(instance, nil)
	mockDocker.On("StopContainer", mock.Anything, "c1").Return(nil)
	mockDocker.On("RemoveContainer", mock.Anything, "c1").Return(nil)
	mockRepo.On("Delete", mock.Anything, int64(1)).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/instances/1", nil)
	ctx := chi.NewRouteContext()
	ctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, ctx))
	w := httptest.NewRecorder()

	handler.Delete(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
	mockDocker.AssertExpectations(t)
}

func TestDelete_NotFound(t *testing.T) {
	mockDocker := new(mockDockerService)
	mockRepo := new(mockInstanceRepository)
	handler := NewInstanceHandler(mockDocker, mockRepo, nil)

	mockRepo.On("GetByID", mock.Anything, int64(999)).Return(nil, repository.ErrNotFound)

	req := httptest.NewRequest(http.MethodDelete, "/api/instances/999", nil)
	ctx := chi.NewRouteContext()
	ctx.URLParams.Add("id", "999")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, ctx))
	w := httptest.NewRecorder()

	handler.Delete(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockRepo.AssertExpectations(t)
}
