package helpers

import (
	"fmt"
)

// ErrorType represents the category of profile generator errors
type ErrorType int

const (
	ErrorTypeValidation ErrorType = iota
	ErrorTypeAuth
	ErrorTypeAPI
	ErrorTypeFileSystem
	ErrorTypeNetwork
)

// String returns the string representation of ErrorType
func (et ErrorType) String() string {
	switch et {
	case ErrorTypeValidation:
		return "validation"
	case ErrorTypeAuth:
		return "authentication"
	case ErrorTypeAPI:
		return "api"
	case ErrorTypeFileSystem:
		return "filesystem"
	case ErrorTypeNetwork:
		return "network"
	default:
		return "unknown"
	}
}

// ProfileGeneratorError represents a structured error for profile generation operations
type ProfileGeneratorError struct {
	Type    ErrorType
	Message string
	Cause   error
	Context map[string]any
}

// Error implements the error interface
func (e ProfileGeneratorError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s error: %s (caused by: %s)", e.Type.String(), e.Message, e.Cause.Error())
	}
	return fmt.Sprintf("%s error: %s", e.Type.String(), e.Message)
}

// Unwrap implements the error unwrapping interface
func (e ProfileGeneratorError) Unwrap() error {
	return e.Cause
}

// WithContext adds context information to the error
func (e ProfileGeneratorError) WithContext(key string, value any) ProfileGeneratorError {
	if e.Context == nil {
		e.Context = make(map[string]any)
	}
	e.Context[key] = value
	return e
}

// NewValidationError creates a new validation error
func NewValidationError(message string, cause error) ProfileGeneratorError {
	return ProfileGeneratorError{
		Type:    ErrorTypeValidation,
		Message: message,
		Cause:   cause,
		Context: make(map[string]any),
	}
}

// NewAuthError creates a new authentication error
func NewAuthError(message string, cause error) ProfileGeneratorError {
	return ProfileGeneratorError{
		Type:    ErrorTypeAuth,
		Message: message,
		Cause:   cause,
		Context: make(map[string]any),
	}
}

// NewAPIError creates a new API error
func NewAPIError(message string, cause error) ProfileGeneratorError {
	return ProfileGeneratorError{
		Type:    ErrorTypeAPI,
		Message: message,
		Cause:   cause,
		Context: make(map[string]any),
	}
}

// NewFileSystemError creates a new filesystem error
func NewFileSystemError(message string, cause error) ProfileGeneratorError {
	return ProfileGeneratorError{
		Type:    ErrorTypeFileSystem,
		Message: message,
		Cause:   cause,
		Context: make(map[string]any),
	}
}

// NewNetworkError creates a new network error
func NewNetworkError(message string, cause error) ProfileGeneratorError {
	return ProfileGeneratorError{
		Type:    ErrorTypeNetwork,
		Message: message,
		Cause:   cause,
		Context: make(map[string]any),
	}
}
