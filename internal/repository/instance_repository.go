package repository

import (
	"context"

	"github.com/masmuss/micro-paas/internal/model"
)

// InstanceRepository abstracts data access for Instance.
type InstanceRepository interface {
	Create(ctx context.Context, instance *model.Instance) error
	GetByID(ctx context.Context, id int64) (*model.Instance, error)
	GetByName(ctx context.Context, name string) (*model.Instance, error)
	GetByContainerID(ctx context.Context, containerID string) (*model.Instance, error)
	GetBySubdomain(ctx context.Context, subdomain string) (*model.Instance, error)
	Update(ctx context.Context, instance *model.Instance) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]*model.Instance, error)
}
