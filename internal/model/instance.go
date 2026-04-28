// Package model contains data models for the application.
package model

import (
	"time"

	"github.com/uptrace/bun"
)

// Instance is a data model for an application instance.
type Instance struct {
	bun.BaseModel `bun:"table:instances,alias:instances"`

	// ID is the unique identifier for the instance (primary key).
	ID int64 `bun:"id,pk,autoincrement" json:"id"`
	// Name is the human-readable name of the instance (must be unique).
	Name string `bun:"name,unique,notnull" json:"name"`
	// ContainerID is the Docker container ID associated with this instance.
	ContainerID string `bun:"container_id,unique" json:"container_id"`
	// Subdomain is the unique subdomain assigned for routing to this instance.
	Subdomain string `bun:"subdomain,unique" json:"subdomain"`
	// Status is the current state of the instance (e.g. "running", "stopped").
	Status string `bun:"status,default:'running'" json:"status"`
	// CreatedAt is the timestamp when the instance was created.
	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
	// UpdatedAt is the timestamp when the instance was last updated.
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updated_at"`
}
