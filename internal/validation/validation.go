// Package validation provides request validation logic.
package validation

import (
	"errors"
	"regexp"
)

var (
	// ErrRequiredName is returned when the name field is required.
	ErrRequiredName = errors.New("name is required")
	// ErrRequiredImage is returned when the image field is required.
	ErrRequiredImage = errors.New("image is required")
	// ErrRequiredSubdomain is returned when the subdomain field is required.
	ErrRequiredSubdomain = errors.New("subdomain is required")
)

// ValidateCreateInstance validates required fields for instance creation.
func ValidateCreateInstance(name, image, subdomain string) error {
	if name == "" {
		return ErrRequiredName
	}
	if image == "" {
		return ErrRequiredImage
	}
	if subdomain == "" {
		return ErrRequiredSubdomain
	}
	return nil
}

// ValidateSubdomain validates the subdomain field.
func ValidateSubdomain(s string) error {
	if matched, _ := regexp.MatchString(`^[a-z0-9-]+$`, s); !matched {
		return errors.New("subdomain must be lowercase alphanumeric and may contain hyphens")
	}
	return nil
}
