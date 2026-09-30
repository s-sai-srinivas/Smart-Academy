package utils

import (
	"testing"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		// Valid passwords
		{
			name:     "valid password with 6 chars",
			password: "abc123",
			wantErr:  false,
		},
		{
			name:     "valid password with letters only",
			password: "abcdef",
			wantErr:  false,
		},
		{
			name:     "valid password with mixed characters",
			password: "MyP@ssw0rd!",
			wantErr:  false,
		},

		// Invalid - too short
		{
			name:     "too short - 5 characters",
			password: "abc12",
			wantErr:  true,
		},
		{
			name:     "too short - empty",
			password: "",
			wantErr:  true,
		},

		// Invalid - too long
		{
			name:     "too long - 129 characters",
			password: "Password1" + string(make([]byte, 120)),
			wantErr:  true,
		},

		// Invalid - spaces
		{
			name:     "leading space",
			password: " MySecure123",
			wantErr:  true,
		},
		{
			name:     "trailing space",
			password: "MySecure123 ",
			wantErr:  true,
		},

		// Edge cases
		{
			name:     "exactly 6 characters",
			password: "Abc123",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword(%q) error = %v, wantErr %v", tt.password, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePasswordWithMessage(t *testing.T) {
	// Test that it returns a combined error message
	err := ValidatePasswordWithMessage("weak")
	if err == nil {
		t.Error("Expected error for weak password")
	}

	// The error message should be user-friendly
	err = ValidatePasswordWithMessage("MySecure123")
	if err != nil {
		t.Errorf("Expected no error for valid password, got: %v", err)
	}
}
