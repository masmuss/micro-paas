// Package model contains data models for the application.
package model

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

// Status represents the state of an instance.
type Status int

const (
	// StatusUnknown represents an unknown status.
	StatusUnknown Status = iota
	// StatusRunning represents a running status.
	StatusRunning
	// StatusStopped represents a stopped status.
	StatusStopped
	// StatusError represents an error status.
	StatusError
)

var statusStrings = map[Status]string{
	StatusUnknown: "unknown",
	StatusRunning: "running",
	StatusStopped: "stopped",
	StatusError:   "error",
}

var stringToStatus = map[string]Status{
	"unknown": StatusUnknown,
	"running": StatusRunning,
	"stopped": StatusStopped,
	"error":   StatusError,
}

func (s Status) String() string {
	str, ok := statusStrings[s]
	if !ok {
		return "unknown"
	}
	return str
}

// MarshalJSON implements the [json.Marshaler] interface for Status.
func (s Status) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s.String() + `"`), nil
}

// UnmarshalJSON implements the [json.Unmarshaler] interface for Status.
func (s *Status) UnmarshalJSON(data []byte) error {
	if len(data) < 3 {
		return fmt.Errorf("invalid status value: %s", data)
	}
	str := string(data[1 : len(data)-1])
	val, ok := stringToStatus[str]
	if !ok {
		return fmt.Errorf("unknown status: %s", str)
	}
	*s = val
	return nil
}

// Value implements the [driver.Valuer] interface for Status.
func (s Status) Value() (driver.Value, error) {
	return s.String(), nil
}

// Scan implements the [driver.Scanner] interface for Status.
func (s *Status) Scan(value any) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("expected string for status, got %T", value)
	}
	val, ok := stringToStatus[str]
	if !ok {
		return fmt.Errorf("unknown status: %s", str)
	}
	*s = val
	return nil
}

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
	// Port is the internal port the container is listening on.
	Port int `bun:"port,default:80" json:"port"`
	// Status is the current state of the instance.
	Status Status `bun:"status,default:'running'" json:"status"`
	// Env stores environment variables for the instance.
	Env map[string]string `bun:"env,type:jsonb" json:"env"`
	// CreatedAt is the timestamp when the instance was created.
	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
	// UpdatedAt is the timestamp when the instance was last updated.
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updated_at"`
}
