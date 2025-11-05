package config

import (
	"errors"
	"fmt"
	"strings"
)

// ValidationError represents multiple validation errors
type ValidationError struct {
	Errors []error
}

// NewValidationError creates a new ValidationError
func NewValidationError() *ValidationError {
	return &ValidationError{
		Errors: make([]error, 0),
	}
}

// Add adds an error to the validation error list
func (v *ValidationError) Add(err error) {
	if err != nil {
		v.Errors = append(v.Errors, err)
	}
}

// HasErrors returns true if there are any validation errors
func (v *ValidationError) HasErrors() bool {
	return len(v.Errors) > 0
}

// Error implements the error interface
func (v *ValidationError) Error() string {
	if len(v.Errors) == 0 {
		return ""
	}

	if len(v.Errors) == 1 {
		return v.Errors[0].Error()
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("found %d validation errors:\n", len(v.Errors)))
	for i, err := range v.Errors {
		sb.WriteString(fmt.Sprintf("  %d. %v\n", i+1, err))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// Is implements the errors.Is interface for single error case
func (v *ValidationError) Is(target error) bool {
	if len(v.Errors) == 1 {
		return errors.Is(v.Errors[0], target)
	}
	return false
}

// Unwrap implements the errors.Unwrap interface for single error case
func (v *ValidationError) Unwrap() error {
	if len(v.Errors) == 1 {
		return v.Errors[0]
	}
	return nil
}

// ErrorOrNil returns the error if there are any validation errors, otherwise nil
func (v *ValidationError) ErrorOrNil() error {
	if v.HasErrors() {
		return v
	}
	return nil
}
