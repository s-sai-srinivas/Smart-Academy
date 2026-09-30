package utils

import (
	"errors"
	"strings"
)

// Password validation errors
var (
	ErrPasswordTooShort = errors.New("password must be at least 6 characters")
	ErrPasswordTooLong  = errors.New("password must not exceed 128 characters")
	ErrPasswordSpaces   = errors.New("password cannot start or end with spaces")
)

// ValidatePassword validates password format.
// Requirements:
// - 6-128 characters
// - No leading/trailing spaces
func ValidatePassword(password string) error {
	// Check length
	if len(password) < 6 {
		return ErrPasswordTooShort
	}
	if len(password) > 128 {
		return ErrPasswordTooLong
	}

	// Check for leading/trailing spaces
	if strings.TrimSpace(password) != password {
		return ErrPasswordSpaces
	}

	return nil
}

// ValidatePasswordWithMessage returns a user-friendly error message
func ValidatePasswordWithMessage(password string) error {
	err := ValidatePassword(password)
	if err != nil {
		return errors.New("password must be 6-128 characters")
	}
	return nil
}
