package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProfileConflictDetector(t *testing.T) {
	configFile := &AWSConfigFile{
		Profiles: make(map[string]Profile),
		Sessions: make(map[string]SSOSession),
	}
	namingPattern, err := NewNamingPattern("{account_name}-{role_name}")
	require.NoError(t, err)

	detector := NewProfileConflictDetector(configFile, namingPattern)

	assert.NotNil(t, detector)
	assert.Equal(t, configFile, detector.configFile)
	assert.Equal(t, namingPattern, detector.namingPattern)
	assert.NotNil(t, detector.logger)
}

func TestProfileConflictDetector_DetectConflicts_EmptyRoles(t *testing.T) {
	configFile := &AWSConfigFile{
		Profiles: make(map[string]Profile),
		Sessions: make(map[string]SSOSession),
	}
	namingPattern, err := NewNamingPattern("{account_name}-{role_name}")
	require.NoError(t, err)

	detector := NewProfileConflictDetector(configFile, namingPattern)

	conflicts, err := detector.DetectConflicts([]DiscoveredRole{})
	assert.NoError(t, err)
	assert.Empty(t, conflicts)
}

func TestProfileConflictDetector_DetectConflicts_NoConflicts(t *testing.T) {
	configFile := &AWSConfigFile{
		Profiles: make(map[string]Profile),
		Sessions: make(map[string]SSOSession),
	}
	namingPattern, err := NewNamingPattern("{account_name}-{role_name}")
	require.NoError(t, err)

	detector := NewProfileConflictDetector(configFile, namingPattern)

	roles := []DiscoveredRole{
		{
			AccountID:         "123456789012",
			AccountName:       "production",
			RoleName:          "PowerUserAccess",
			PermissionSetName: "PowerUserAccess",
		},
	}

	conflicts, err := detector.DetectConflicts(roles)
	assert.NoError(t, err)
	assert.Empty(t, conflicts)
}

func TestProfileConflictDetector_DetectConflicts_WithSameRoleConflict(t *testing.T) {
	// Create config file with existing profile
	configFile := &AWSConfigFile{
		Profiles: map[string]Profile{
			"existing-profile": {
				Name:         "existing-profile",
				SSOAccountID: "123456789012",
				SSORoleName:  "PowerUserAccess",
				SSOStartURL:  "https://example.awsapps.com/start",
				SSORegion:    "us-east-1",
			},
		},
		Sessions: make(map[string]SSOSession),
	}

	namingPattern, err := NewNamingPattern("{account_name}-{role_name}")
	require.NoError(t, err)

	detector := NewProfileConflictDetector(configFile, namingPattern)

	roles := []DiscoveredRole{
		{
			AccountID:         "123456789012",
			AccountName:       "production",
			RoleName:          "PowerUserAccess",
			PermissionSetName: "PowerUserAccess",
		},
	}

	conflicts, err := detector.DetectConflicts(roles)
	assert.NoError(t, err)
	assert.Len(t, conflicts, 1)

	conflict := conflicts[0]
	assert.Equal(t, "production-PowerUserAccess", conflict.ProposedName)
	assert.Equal(t, ConflictSameRole, conflict.ConflictType)
	assert.Len(t, conflict.ExistingProfiles, 1)
	assert.Equal(t, "existing-profile", conflict.ExistingProfiles[0].Name)
}

func TestProfileConflictDetector_DetectConflicts_WithSameNameConflict(t *testing.T) {
	// Create config file with existing profile that has same name but different role
	configFile := &AWSConfigFile{
		Profiles: map[string]Profile{
			"production-PowerUserAccess": {
				Name:         "production-PowerUserAccess",
				SSOAccountID: "999888777666",   // Different account
				SSORoleName:  "ReadOnlyAccess", // Different role
				SSOStartURL:  "https://example.awsapps.com/start",
				SSORegion:    "us-east-1",
			},
		},
		Sessions: make(map[string]SSOSession),
	}

	namingPattern, err := NewNamingPattern("{account_name}-{role_name}")
	require.NoError(t, err)

	detector := NewProfileConflictDetector(configFile, namingPattern)

	roles := []DiscoveredRole{
		{
			AccountID:         "123456789012",
			AccountName:       "production",
			RoleName:          "PowerUserAccess",
			PermissionSetName: "PowerUserAccess",
		},
	}

	conflicts, err := detector.DetectConflicts(roles)
	assert.NoError(t, err)
	assert.Len(t, conflicts, 1)

	conflict := conflicts[0]
	assert.Equal(t, "production-PowerUserAccess", conflict.ProposedName)
	assert.Equal(t, ConflictSameName, conflict.ConflictType)
	assert.Len(t, conflict.ExistingProfiles, 1)
	assert.Equal(t, "production-PowerUserAccess", conflict.ExistingProfiles[0].Name)
}

func TestProfileConflictDetector_AnalyzeRole_ValidRole(t *testing.T) {
	configFile := &AWSConfigFile{
		Profiles: make(map[string]Profile),
		Sessions: make(map[string]SSOSession),
	}
	namingPattern, err := NewNamingPattern("{account_name}-{role_name}")
	require.NoError(t, err)

	detector := NewProfileConflictDetector(configFile, namingPattern)

	role := DiscoveredRole{
		AccountID:         "123456789012",
		AccountName:       "production",
		RoleName:          "PowerUserAccess",
		PermissionSetName: "PowerUserAccess",
	}

	conflict, err := detector.AnalyzeRole(role)
	assert.NoError(t, err)
	assert.Nil(t, conflict) // No conflicts expected
}

func TestProfileConflictDetector_AnalyzeRole_InvalidRole(t *testing.T) {
	configFile := &AWSConfigFile{
		Profiles: make(map[string]Profile),
		Sessions: make(map[string]SSOSession),
	}
	namingPattern, err := NewNamingPattern("{account_name}-{role_name}")
	require.NoError(t, err)

	detector := NewProfileConflictDetector(configFile, namingPattern)

	// Invalid role - missing account ID
	role := DiscoveredRole{
		AccountName: "production",
		RoleName:    "PowerUserAccess",
	}

	conflict, err := detector.AnalyzeRole(role)
	assert.Error(t, err)
	assert.Nil(t, conflict)
	assert.Contains(t, err.Error(), "account ID is required")
}

func TestProfileConflictDetector_ClassifyConflict_SameRole(t *testing.T) {
	configFile := &AWSConfigFile{
		Profiles: map[string]Profile{
			"existing-profile": {
				Name:         "existing-profile",
				SSOAccountID: "123456789012",
				SSORoleName:  "PowerUserAccess",
				SSOStartURL:  "https://example.awsapps.com/start",
				SSORegion:    "us-east-1",
			},
		},
		Sessions: make(map[string]SSOSession),
	}

	namingPattern, err := NewNamingPattern("{account_name}-{role_name}")
	require.NoError(t, err)

	detector := NewProfileConflictDetector(configFile, namingPattern)

	existingProfiles := []Profile{configFile.Profiles["existing-profile"]}
	role := DiscoveredRole{
		AccountID:         "123456789012",
		RoleName:          "PowerUserAccess",
		PermissionSetName: "PowerUserAccess",
	}

	conflictType := detector.ClassifyConflict(existingProfiles, "production-PowerUserAccess", role)
	assert.Equal(t, ConflictSameRole, conflictType)
}

func TestProfileConflictDetector_ClassifyConflict_SameName(t *testing.T) {
	configFile := &AWSConfigFile{
		Profiles: map[string]Profile{
			"production-PowerUserAccess": {
				Name:         "production-PowerUserAccess",
				SSOAccountID: "999888777666",   // Different account
				SSORoleName:  "ReadOnlyAccess", // Different role
				SSOStartURL:  "https://example.awsapps.com/start",
				SSORegion:    "us-east-1",
			},
		},
		Sessions: make(map[string]SSOSession),
	}

	namingPattern, err := NewNamingPattern("{account_name}-{role_name}")
	require.NoError(t, err)

	detector := NewProfileConflictDetector(configFile, namingPattern)

	existingProfiles := []Profile{configFile.Profiles["production-PowerUserAccess"]}
	role := DiscoveredRole{
		AccountID:         "123456789012",
		RoleName:          "PowerUserAccess",
		PermissionSetName: "PowerUserAccess",
	}

	conflictType := detector.ClassifyConflict(existingProfiles, "production-PowerUserAccess", role)
	assert.Equal(t, ConflictSameName, conflictType)
}

func TestProfileConflictDetector_GenerateConflictSummary_NoConflicts(t *testing.T) {
	configFile := &AWSConfigFile{
		Profiles: make(map[string]Profile),
		Sessions: make(map[string]SSOSession),
	}
	namingPattern, err := NewNamingPattern("{account_name}-{role_name}")
	require.NoError(t, err)

	detector := NewProfileConflictDetector(configFile, namingPattern)

	summary := detector.GenerateConflictSummary([]ProfileConflict{})
	assert.Equal(t, "No profile conflicts detected.", summary)
}

func TestProfileConflictDetector_GenerateConflictSummary_WithConflicts(t *testing.T) {
	configFile := &AWSConfigFile{
		Profiles: make(map[string]Profile),
		Sessions: make(map[string]SSOSession),
	}
	namingPattern, err := NewNamingPattern("{account_name}-{role_name}")
	require.NoError(t, err)

	detector := NewProfileConflictDetector(configFile, namingPattern)

	conflicts := []ProfileConflict{
		{
			DiscoveredRole: DiscoveredRole{
				AccountID:         "123456789012",
				AccountName:       "production",
				RoleName:          "PowerUserAccess",
				PermissionSetName: "PowerUserAccess",
			},
			ExistingProfiles: []Profile{
				{
					Name:         "existing-profile",
					SSOAccountID: "123456789012",
					SSORoleName:  "PowerUserAccess",
				},
			},
			ProposedName: "production-PowerUserAccess",
			ConflictType: ConflictSameRole,
		},
	}

	summary := detector.GenerateConflictSummary(conflicts)
	assert.Contains(t, summary, "Profile Conflicts Detected: 1")
	assert.Contains(t, summary, "Proposed Profile: production-PowerUserAccess")
	assert.Contains(t, summary, "Account: production (123456789012)")
	assert.Contains(t, summary, "Role: PowerUserAccess")
	assert.Contains(t, summary, "Conflict Type: same_role")
	assert.Contains(t, summary, "- existing-profile")
}

func TestProfileConflictDetector_DetectConflicts_WithSSOSessionFormat(t *testing.T) {
	// Create config file with SSO session format
	configFile := &AWSConfigFile{
		Profiles: map[string]Profile{
			"existing-profile": {
				Name:         "existing-profile",
				SSOSession:   "my-sso",
				SSOAccountID: "123456789012",
				SSORoleName:  "PowerUserAccess",
			},
		},
		Sessions: map[string]SSOSession{
			"my-sso": {
				Name:        "my-sso",
				SSOStartURL: "https://example.awsapps.com/start",
				SSORegion:   "us-east-1",
			},
		},
	}

	namingPattern, err := NewNamingPattern("{account_name}-{role_name}")
	require.NoError(t, err)

	detector := NewProfileConflictDetector(configFile, namingPattern)

	roles := []DiscoveredRole{
		{
			AccountID:         "123456789012",
			AccountName:       "production",
			RoleName:          "PowerUserAccess",
			PermissionSetName: "PowerUserAccess",
		},
	}

	conflicts, err := detector.DetectConflicts(roles)
	assert.NoError(t, err)
	assert.Len(t, conflicts, 1)

	conflict := conflicts[0]
	assert.Equal(t, "production-PowerUserAccess", conflict.ProposedName)
	assert.Equal(t, ConflictSameRole, conflict.ConflictType)
	assert.Len(t, conflict.ExistingProfiles, 1)
	assert.Equal(t, "existing-profile", conflict.ExistingProfiles[0].Name)
}

func TestProfileConflictDetector_DetectConflicts_MultipleConflicts(t *testing.T) {
	// Create config file with multiple existing profiles
	configFile := &AWSConfigFile{
		Profiles: map[string]Profile{
			"existing-profile-1": {
				Name:         "existing-profile-1",
				SSOAccountID: "123456789012",
				SSORoleName:  "PowerUserAccess",
				SSOStartURL:  "https://example.awsapps.com/start",
				SSORegion:    "us-east-1",
			},
			"production-ReadOnlyAccess": {
				Name:         "production-ReadOnlyAccess",
				SSOAccountID: "999888777666", // Different account, same name pattern
				SSORoleName:  "ReadOnlyAccess",
				SSOStartURL:  "https://example.awsapps.com/start",
				SSORegion:    "us-east-1",
			},
		},
		Sessions: make(map[string]SSOSession),
	}

	namingPattern, err := NewNamingPattern("{account_name}-{role_name}")
	require.NoError(t, err)

	detector := NewProfileConflictDetector(configFile, namingPattern)

	roles := []DiscoveredRole{
		{
			AccountID:         "123456789012",
			AccountName:       "production",
			RoleName:          "PowerUserAccess",
			PermissionSetName: "PowerUserAccess",
		},
		{
			AccountID:         "123456789012",
			AccountName:       "production",
			RoleName:          "ReadOnlyAccess",
			PermissionSetName: "ReadOnlyAccess",
		},
	}

	conflicts, err := detector.DetectConflicts(roles)
	assert.NoError(t, err)
	assert.Len(t, conflicts, 2)

	// First conflict should be same role
	assert.Equal(t, ConflictSameRole, conflicts[0].ConflictType)
	assert.Equal(t, "production-PowerUserAccess", conflicts[0].ProposedName)

	// Second conflict should be same name
	assert.Equal(t, ConflictSameName, conflicts[1].ConflictType)
	assert.Equal(t, "production-ReadOnlyAccess", conflicts[1].ProposedName)
}
