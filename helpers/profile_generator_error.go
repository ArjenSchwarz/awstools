package helpers

import (
	"fmt"
)

// unknownString is the default string for unknown enum values
const unknownString = "unknown"

// ErrorType represents the category of profile generator errors
type ErrorType int

// ErrorType constants represent different categories of profile generator errors
const (
	// ErrorTypeValidation represents validation errors
	ErrorTypeValidation ErrorType = iota
	// ErrorTypeAuth represents authentication errors
	ErrorTypeAuth
	// ErrorTypeAPI represents API errors
	ErrorTypeAPI
	// ErrorTypeFileSystem represents file system errors
	ErrorTypeFileSystem
	// ErrorTypeNetwork represents network errors
	ErrorTypeNetwork
	// ErrorTypeConflictResolution represents conflict resolution errors
	ErrorTypeConflictResolution
	// ErrorTypeBackup represents backup errors
	ErrorTypeBackup
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
	case ErrorTypeConflictResolution:
		return "conflict_resolution"
	case ErrorTypeBackup:
		return "backup"
	default:
		return unknownString
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

// ConflictResolutionError represents errors that occur during profile conflict resolution
type ConflictResolutionError struct {
	ProfileGeneratorError
	ConflictType ConflictType
	ProfileName  string
	RoleName     string
	AccountID    string
}

// NewConflictResolutionError creates a new conflict resolution error
func NewConflictResolutionError(message string, cause error, conflictType ConflictType) *ConflictResolutionError {
	return &ConflictResolutionError{
		ProfileGeneratorError: ProfileGeneratorError{
			Type:    ErrorTypeConflictResolution,
			Message: message,
			Cause:   cause,
			Context: make(map[string]any),
		},
		ConflictType: conflictType,
	}
}

// WithProfileContext adds profile-specific context to the conflict resolution error
func (cre *ConflictResolutionError) WithProfileContext(profileName, roleName, accountID string) *ConflictResolutionError {
	cre.ProfileName = profileName
	cre.RoleName = roleName
	cre.AccountID = accountID
	cre.Context["profile_name"] = profileName
	cre.Context["role_name"] = roleName
	cre.Context["account_id"] = accountID
	cre.Context["conflict_type"] = cre.ConflictType.String()
	return cre
}

// Error implements the error interface with enhanced context
func (cre *ConflictResolutionError) Error() string {
	baseError := cre.ProfileGeneratorError.Error()
	if cre.ProfileName != "" && cre.RoleName != "" {
		return fmt.Sprintf("%s (profile: %s, role: %s in account %s)",
			baseError, cre.ProfileName, cre.RoleName, cre.AccountID)
	}
	return baseError
}

// RecoveryGuidance provides guidance for recovering from conflict resolution errors
func (cre *ConflictResolutionError) RecoveryGuidance() string {
	switch cre.ConflictType {
	case ConflictSameRole:
		return "Try using --replace-existing to replace the existing profile, or --skip-existing to skip this role"
	case ConflictSameName:
		return "Consider using a different naming pattern to avoid profile name conflicts"
	default:
		return "Review the conflict details and choose an appropriate resolution strategy"
	}
}

// BackupError represents errors that occur during backup and recovery operations
type BackupError struct {
	ProfileGeneratorError
	BackupPath   string
	OriginalPath string
	Operation    string
}

// NewBackupError creates a new backup error
func NewBackupError(message string, cause error, backupPath, originalPath string) *BackupError {
	return &BackupError{
		ProfileGeneratorError: ProfileGeneratorError{
			Type:    ErrorTypeBackup,
			Message: message,
			Cause:   cause,
			Context: make(map[string]any),
		},
		BackupPath:   backupPath,
		OriginalPath: originalPath,
	}
}

// WithOperation adds operation context to the backup error
func (be *BackupError) WithOperation(operation string) *BackupError {
	be.Operation = operation
	be.Context["operation"] = operation
	be.Context["backup_path"] = be.BackupPath
	be.Context["original_path"] = be.OriginalPath
	return be
}

// Error implements the error interface with enhanced context
func (be *BackupError) Error() string {
	baseError := be.ProfileGeneratorError.Error()
	if be.Operation != "" {
		return fmt.Sprintf("%s during %s operation (backup: %s, original: %s)",
			baseError, be.Operation, be.BackupPath, be.OriginalPath)
	}
	return fmt.Sprintf("%s (backup: %s, original: %s)",
		baseError, be.BackupPath, be.OriginalPath)
}

// RecoveryGuidance provides guidance for recovering from backup errors
func (be *BackupError) RecoveryGuidance() string {
	switch be.Operation {
	case "create":
		return "Check file permissions and available disk space. Ensure the backup directory is writable"
	case "restore":
		return "Verify the backup file exists and is readable. Check file permissions on the original location"
	case "cleanup":
		return "Manual cleanup may be required. Check for temporary files in the backup location"
	default:
		return "Check file permissions and disk space. Verify backup and original file paths are accessible"
	}
}
