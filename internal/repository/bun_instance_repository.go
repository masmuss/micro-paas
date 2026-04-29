// Package repository implements data access for instances.
package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/masmuss/micro-paas/internal/model"
	"github.com/uptrace/bun"
)

// bunInstanceRepository implements InstanceRepository using Bun ORM.
type bunInstanceRepository struct {
	db     *bun.DB
	logger *slog.Logger
}

// NewInstanceRepository returns a Bun-based InstanceRepository.
func NewInstanceRepository(db *bun.DB, logger *slog.Logger) InstanceRepository {
	return &bunInstanceRepository{
		db:     db,
		logger: logger,
	}
}

// Create implements [InstanceRepository].
func (r *bunInstanceRepository) Create(ctx context.Context, instance *model.Instance) error {
	r.logger.DebugContext(ctx, "creating instance", "name", instance.Name)

	_, err := r.db.NewInsert().Model(instance).Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert instance: %w", err)
	}

	return nil
}

// Delete implements [InstanceRepository].
func (r *bunInstanceRepository) Delete(ctx context.Context, id int64) error {
	r.logger.DebugContext(ctx, "deleting instance", "id", id)

	_, err := r.db.NewDelete().Model((*model.Instance)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete instance %d: %w", id, err)
	}

	return nil
}

// GetByContainerID implements [InstanceRepository].
func (r *bunInstanceRepository) GetByContainerID(ctx context.Context, containerID string) (*model.Instance, error) {
	r.logger.DebugContext(ctx, "getting instance by container ID", "containerID", containerID)

	instance := new(model.Instance)
	err := r.db.NewSelect().Model(instance).Where("container_id = ?", containerID).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return instance, nil
}

// GetByID implements [InstanceRepository].
func (r *bunInstanceRepository) GetByID(ctx context.Context, id int64) (*model.Instance, error) {
	r.logger.DebugContext(ctx, "getting instance by ID", "id", id)

	instance := new(model.Instance)
	err := r.db.NewSelect().Model(instance).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("get instance %d: %w", id, err)
	}

	return instance, nil
}

// GetByName implements [InstanceRepository].
func (r *bunInstanceRepository) GetByName(ctx context.Context, name string) (*model.Instance, error) {
	r.logger.DebugContext(ctx, "getting instance by name", "name", name)

	instance := new(model.Instance)
	err := r.db.NewSelect().Model(instance).Where("name = ?", name).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("get instance by name %q: %w", name, err)
	}

	return instance, nil
}

// GetBySubdomain implements [InstanceRepository].
func (r *bunInstanceRepository) GetBySubdomain(ctx context.Context, subdomain string) (*model.Instance, error) {
	r.logger.DebugContext(ctx, "getting instance by subdomain", "subdomain", subdomain)

	instance := new(model.Instance)
	err := r.db.NewSelect().Model(instance).Where("subdomain = ?", subdomain).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("get instance by subdomain %q: %w", subdomain, err)
	}

	return instance, nil
}

// List implements [InstanceRepository].
func (r *bunInstanceRepository) List(ctx context.Context) ([]*model.Instance, error) {
	r.logger.DebugContext(ctx, "listing instances")

	var instances []*model.Instance
	err := r.db.NewSelect().Model(&instances).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list instances: %w", err)
	}

	return instances, nil
}

// Update implements [InstanceRepository].
func (r *bunInstanceRepository) Update(ctx context.Context, instance *model.Instance) error {
	r.logger.DebugContext(ctx, "updating instance", "id", instance.ID, "name", instance.Name)

	_, err := r.db.NewUpdate().Model(instance).Where("id = ?", instance.ID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("update instance %d: %w", instance.ID, err)
	}

	return nil
}
