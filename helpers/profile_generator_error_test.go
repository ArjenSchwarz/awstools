package helpers

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorType_String(t *testing.T) {
	tests := []struct {
		name      string
		errorType ErrorType
		expected  string
	}{
		{"validation", ErrorTypeValidation, "validation"},
		{"auth", ErrorTypeAuth, "authentication"},
		{"api", ErrorTypeAPI, "api"},
		{"filesystem", ErrorTypeFileSystem, "filesystem"},
		{"network", ErrorTypeNetwork, "network"},
		{"conflict_resolution", ErrorTypeConflictResolution, "conflict_resolution"},
		{"backup", ErrorTypeBackup, "backup"},
		{"unknown", ErrorType(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.errorType.String())
		})
	}
}

func TestProfileGeneratorError(t *testing.T) {
	t.Run("basic error", func(t *testing.T) {
		err := ProfileGeneratorError{
			Type:    ErrorTypeValidation,
			Message: "test error",
			Cause:   nil,
			Context: make(map[string]any),
		}

		assert.Equal(t, "validation error: test error", err.Error())
		assert.Nil(t, err.Unwrap())
	})

	t.Run("error with cause", func(t *testing.T) {
		cause := errors.New("underlying error")
		err := ProfileGeneratorError{
			Type:    ErrorTypeAPI,
			Message: "test error",
			Cause:   cause,
			Context: make(map[string]any),
		}

		assert.Equal(t, "api error: test error (caused by: underlying error)", err.Error())
		assert.Equal(t, cause, err.Unwrap())
	})

	t.Run("with context", func(t *testing.T) {
		err := ProfileGeneratorError{
			Type:    ErrorTypeValidation,
			Message: "test error",
			Context: make(map[string]any),
		}

		enriched := err.WithContext("key", "value")
		assert.Equal(t, "value", enriched.Context["key"])
	})
}

func TestNewConflictResolutionError(t *testing.T) {
	t.Run("basic conflict resolution error", func(t *testing.T) {
		cause := errors.New("underlying error")
		err := NewConflictResolutionError("conflict detected", cause, ConflictSameRole)

		assert.Equal(t, ErrorTypeConflictResolution, err.Type)
		assert.Equal(t, "conflict detected", err.Message)
		assert.Equal(t, cause, err.Cause)
		assert.Equal(t, ConflictSameRole, err.ConflictType)
		assert.NotNil(t, err.Context)
	})

	t.Run("with profile context", func(t *testing.T) {
		err := NewConflictResolutionError("conflict detected", nil, ConflictSameRole)
		enriched := err.WithProfileContext("test-profile", "AdminRole", "123456789012")

		assert.Equal(t, "test-profile", enriched.ProfileName)
		assert.Equal(t, "AdminRole", enriched.RoleName)
		assert.Equal(t, "123456789012", enriched.AccountID)
		assert.Equal(t, "test-profile", enriched.Context["profile_name"])
		assert.Equal(t, "AdminRole", enriched.Context["role_name"])
		assert.Equal(t, "123456789012", enriched.Context["account_id"])
		assert.Equal(t, "same_role", enriched.Context["conflict_type"])
	})

	t.Run("error message with context", func(t *testing.T) {
		err := NewConflictResolutionError("conflict detected", nil, ConflictSameRole)
		err.WithProfileContext("test-profile", "AdminRole", "123456789012")

		expected := "conflict_resolution error: conflict detected (profile: test-profile, role: AdminRole in account 123456789012)"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("error message without context", func(t *testing.T) {
		err := NewConflictResolutionError("conflict detected", nil, ConflictSameRole)
		expected := "conflict_resolution error: conflict detected"
		assert.Equal(t, expected, err.Error())
	})
}

func TestConflictResolutionError_RecoveryGuidance(t *testing.T) {
	tests := []struct {
		name         string
		conflictType ConflictType
		expected     string
	}{
		{
			name:         "same role conflict",
			conflictType: ConflictSameRole,
			expected:     "Try using --replace-existing to replace the existing profile, or --skip-existing to skip this role",
		},
		{
			name:         "same name conflict",
			conflictType: ConflictSameName,
			expected:     "Consider using a different naming pattern to avoid profile name conflicts",
		},
		{
			name:         "unknown conflict",
			conflictType: ConflictType(999),
			expected:     "Review the conflict details and choose an appropriate resolution strategy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewConflictResolutionError("test error", nil, tt.conflictType)
			assert.Equal(t, tt.expected, err.RecoveryGuidance())
		})
	}
}

func TestNewBackupError(t *testing.T) {
	t.Run("basic backup error", func(t *testing.T) {
		cause := errors.New("underlying error")
		err := NewBackupError("backup failed", cause, "/tmp/backup.conf", "/home/user/.aws/config")

		assert.Equal(t, ErrorTypeBackup, err.Type)
		assert.Equal(t, "backup failed", err.Message)
		assert.Equal(t, cause, err.Cause)
		assert.Equal(t, "/tmp/backup.conf", err.BackupPath)
		assert.Equal(t, "/home/user/.aws/config", err.OriginalPath)
		assert.NotNil(t, err.Context)
	})

	t.Run("with operation context", func(t *testing.T) {
		err := NewBackupError("backup failed", nil, "/tmp/backup.conf", "/home/user/.aws/config")
		enriched := err.WithOperation("create")

		assert.Equal(t, "create", enriched.Operation)
		assert.Equal(t, "create", enriched.Context["operation"])
		assert.Equal(t, "/tmp/backup.conf", enriched.Context["backup_path"])
		assert.Equal(t, "/home/user/.aws/config", enriched.Context["original_path"])
	})

	t.Run("error message with operation", func(t *testing.T) {
		err := NewBackupError("backup failed", nil, "/tmp/backup.conf", "/home/user/.aws/config")
		err.WithOperation("create")

		expected := "backup error: backup failed during create operation (backup: /tmp/backup.conf, original: /home/user/.aws/config)"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("error message without operation", func(t *testing.T) {
		err := NewBackupError("backup failed", nil, "/tmp/backup.conf", "/home/user/.aws/config")
		expected := "backup error: backup failed (backup: /tmp/backup.conf, original: /home/user/.aws/config)"
		assert.Equal(t, expected, err.Error())
	})
}

func TestBackupError_RecoveryGuidance(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		expected  string
	}{
		{
			name:      "create operation",
			operation: "create",
			expected:  "Check file permissions and available disk space. Ensure the backup directory is writable",
		},
		{
			name:      "restore operation",
			operation: "restore",
			expected:  "Verify the backup file exists and is readable. Check file permissions on the original location",
		},
		{
			name:      "cleanup operation",
			operation: "cleanup",
			expected:  "Manual cleanup may be required. Check for temporary files in the backup location",
		},
		{
			name:      "unknown operation",
			operation: "unknown",
			expected:  "Check file permissions and disk space. Verify backup and original file paths are accessible",
		},
		{
			name:      "no operation",
			operation: "",
			expected:  "Check file permissions and disk space. Verify backup and original file paths are accessible",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewBackupError("test error", nil, "/tmp/backup", "/original")
			if tt.operation != "" {
				err.WithOperation(tt.operation)
			}
			assert.Equal(t, tt.expected, err.RecoveryGuidance())
		})
	}
}

func TestErrorConstructors(t *testing.T) {
	t.Run("NewValidationError", func(t *testing.T) {
		cause := errors.New("cause")
		err := NewValidationError("validation failed", cause)

		assert.Equal(t, ErrorTypeValidation, err.Type)
		assert.Equal(t, "validation failed", err.Message)
		assert.Equal(t, cause, err.Cause)
		assert.NotNil(t, err.Context)
	})

	t.Run("NewAuthError", func(t *testing.T) {
		cause := errors.New("cause")
		err := NewAuthError("auth failed", cause)

		assert.Equal(t, ErrorTypeAuth, err.Type)
		assert.Equal(t, "auth failed", err.Message)
		assert.Equal(t, cause, err.Cause)
		assert.NotNil(t, err.Context)
	})

	t.Run("NewAPIError", func(t *testing.T) {
		cause := errors.New("cause")
		err := NewAPIError("api failed", cause)

		assert.Equal(t, ErrorTypeAPI, err.Type)
		assert.Equal(t, "api failed", err.Message)
		assert.Equal(t, cause, err.Cause)
		assert.NotNil(t, err.Context)
	})

	t.Run("NewFileSystemError", func(t *testing.T) {
		cause := errors.New("cause")
		err := NewFileSystemError("filesystem failed", cause)

		assert.Equal(t, ErrorTypeFileSystem, err.Type)
		assert.Equal(t, "filesystem failed", err.Message)
		assert.Equal(t, cause, err.Cause)
		assert.NotNil(t, err.Context)
	})

	t.Run("NewNetworkError", func(t *testing.T) {
		cause := errors.New("cause")
		err := NewNetworkError("network failed", cause)

		assert.Equal(t, ErrorTypeNetwork, err.Type)
		assert.Equal(t, "network failed", err.Message)
		assert.Equal(t, cause, err.Cause)
		assert.NotNil(t, err.Context)
	})
}

func TestErrorIntegration(t *testing.T) {
	t.Run("conflict resolution error in result", func(t *testing.T) {
		result := &ProfileGenerationResult{
			Errors: []ProfileGeneratorError{},
		}

		conflictErr := NewConflictResolutionError("profile conflict", nil, ConflictSameRole)
		conflictErr.WithProfileContext("test-profile", "AdminRole", "123456789012")

		result.AddError(conflictErr.ProfileGeneratorError)

		require.True(t, result.HasErrors())
		require.Len(t, result.Errors, 1)
		assert.Equal(t, ErrorTypeConflictResolution, result.Errors[0].Type)
	})

	t.Run("backup error in result", func(t *testing.T) {
		result := &ProfileGenerationResult{
			Errors: []ProfileGeneratorError{},
		}

		backupErr := NewBackupError("backup failed", nil, "/tmp/backup", "/original")
		backupErr.WithOperation("create")

		result.AddError(backupErr.ProfileGeneratorError)

		require.True(t, result.HasErrors())
		require.Len(t, result.Errors, 1)
		assert.Equal(t, ErrorTypeBackup, result.Errors[0].Type)
	})
}
