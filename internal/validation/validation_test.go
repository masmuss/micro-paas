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
