package validation

import (
	"testing"
)

func TestValidateCreateInstance(t *testing.T) {
	tests := []struct {
		name       string
		nameField  string
		image      string
		subdomain  string
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:      "valid input",
			nameField: "test-app",
			image:     "nginx:latest",
			subdomain: "test",
			wantErr:   false,
		},
		{
			name:       "missing name",
			nameField:  "",
			image:      "nginx:latest",
			subdomain:  "test",
			wantErr:    true,
			wantErrMsg: ErrRequiredName.Error(),
		},
		{
			name:       "missing image",
			nameField:  "test-app",
			image:      "",
			subdomain:  "test",
			wantErr:    true,
			wantErrMsg: ErrRequiredImage.Error(),
		},
		{
			name:       "missing subdomain",
			nameField:  "test-app",
			image:      "nginx:latest",
			subdomain:  "",
			wantErr:    true,
			wantErrMsg: ErrRequiredSubdomain.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCreateInstance(tt.nameField, tt.image, tt.subdomain)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantErrMsg != "" && err.Error() != tt.wantErrMsg {
				t.Errorf("error = %v, want %v", err.Error(), tt.wantErrMsg)
			}
		})
	}
}

func TestValidateSubdomain(t *testing.T) {
	tests := []struct {
		name       string
		subdomain  string
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:      "valid lowercase alphanumeric",
			subdomain: "myapp",
			wantErr:   false,
		},
		{
			name:      "valid with hyphens",
			subdomain: "my-app-123",
			wantErr:   false,
		},
		{
			name:       "uppercase letters",
			subdomain:  "MyApp",
			wantErr:    true,
			wantErrMsg: "subdomain must be lowercase alphanumeric and may contain hyphens",
		},
		{
			name:       "contains underscore",
			subdomain:  "my_app",
			wantErr:    true,
			wantErrMsg: "subdomain must be lowercase alphanumeric and may contain hyphens",
		},
		{
			name:       "contains dot",
			subdomain:  "my.app",
			wantErr:    true,
			wantErrMsg: "subdomain must be lowercase alphanumeric and may contain hyphens",
		},
		{
			name:       "contains space",
			subdomain:  "my app",
			wantErr:    true,
			wantErrMsg: "subdomain must be lowercase alphanumeric and may contain hyphens",
		},
		{
			name:       "empty string",
			subdomain:  "",
			wantErr:    true,
			wantErrMsg: "subdomain must be lowercase alphanumeric and may contain hyphens",
		},
		{
			name:      "single character",
			subdomain: "a",
			wantErr:   false,
		},
		{
			name:      "numbers only",
			subdomain: "123",
			wantErr:   false,
		},
		{
			name:      "starts and ends with hyphen",
			subdomain: "-test-",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSubdomain(tt.subdomain)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantErrMsg != "" && err != nil && err.Error() != tt.wantErrMsg {
				t.Errorf("error = %v, want %v", err.Error(), tt.wantErrMsg)
			}
		})
	}
}
