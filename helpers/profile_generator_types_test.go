package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConflictResolutionStrategy_String(t *testing.T) {
	tests := []struct {
		name     string
		strategy ConflictResolutionStrategy
		expected string
	}{
		{
			name:     "ConflictPrompt",
			strategy: ConflictPrompt,
			expected: "prompt",
		},
		{
			name:     "ConflictReplace",
			strategy: ConflictReplace,
			expected: "replace",
		},
		{
			name:     "ConflictSkip",
			strategy: ConflictSkip,
			expected: "skip",
		},
		{
			name:     "Invalid strategy",
			strategy: ConflictResolutionStrategy(999),
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.strategy.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConflictResolutionStrategy_Validate(t *testing.T) {
	tests := []struct {
		name      string
		strategy  ConflictResolutionStrategy
		expectErr bool
	}{
		{
			name:      "Valid ConflictPrompt",
			strategy:  ConflictPrompt,
			expectErr: false,
		},
		{
			name:      "Valid ConflictReplace",
			strategy:  ConflictReplace,
			expectErr: false,
		},
		{
			name:      "Valid ConflictSkip",
			strategy:  ConflictSkip,
			expectErr: false,
		},
		{
			name:      "Invalid strategy",
			strategy:  ConflictResolutionStrategy(999),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.strategy.Validate()
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConflictType_String(t *testing.T) {
	tests := []struct {
		name         string
		conflictType ConflictType
		expected     string
	}{
		{
			name:         "ConflictSameRole",
			conflictType: ConflictSameRole,
			expected:     "same_role",
		},
		{
			name:         "ConflictSameName",
			conflictType: ConflictSameName,
			expected:     "same_name",
		},
		{
			name:         "Invalid conflict type",
			conflictType: ConflictType(999),
			expected:     "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.conflictType.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProfileConflict_Validate(t *testing.T) {
	validRole := DiscoveredRole{
		AccountID:         "123456789012",
		AccountName:       "test-account",
		PermissionSetName: "AdministratorAccess",
		RoleName:          "AdministratorAccess",
	}

	validProfile := Profile{
		Name:         "existing-profile",
		Region:       "us-east-1",
		SSOStartURL:  "https://test.awsapps.com/start",
		SSORegion:    "us-east-1",
		SSOAccountID: "123456789012",
		SSORoleName:  "AdministratorAccess",
	}

	tests := []struct {
		name      string
		conflict  ProfileConflict
		expectErr bool
	}{
		{
			name: "Valid profile conflict",
			conflict: ProfileConflict{
				DiscoveredRole:   validRole,
				ExistingProfiles: []Profile{validProfile},
				ProposedName:     "new-profile",
				ConflictType:     ConflictSameRole,
			},
			expectErr: false,
		},
		{
			name: "Invalid discovered role",
			conflict: ProfileConflict{
				DiscoveredRole:   DiscoveredRole{}, // Invalid - missing required fields
				ExistingProfiles: []Profile{validProfile},
				ProposedName:     "new-profile",
				ConflictType:     ConflictSameRole,
			},
			expectErr: true,
		},
		{
			name: "No existing profiles",
			conflict: ProfileConflict{
				DiscoveredRole:   validRole,
				ExistingProfiles: []Profile{}, // Invalid - must have at least one
				ProposedName:     "new-profile",
				ConflictType:     ConflictSameRole,
			},
			expectErr: true,
		},
		{
			name: "Empty proposed name",
			conflict: ProfileConflict{
				DiscoveredRole:   validRole,
				ExistingProfiles: []Profile{validProfile},
				ProposedName:     "", // Invalid - required
				ConflictType:     ConflictSameRole,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.conflict.Validate()
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestActionType_String(t *testing.T) {
	tests := []struct {
		name       string
		actionType ActionType
		expected   string
	}{
		{
			name:       "ActionReplace",
			actionType: ActionReplace,
			expected:   "replace",
		},
		{
			name:       "ActionSkip",
			actionType: ActionSkip,
			expected:   "skip",
		},
		{
			name:       "ActionCreate",
			actionType: ActionCreate,
			expected:   "create",
		},
		{
			name:       "Invalid action type",
			actionType: ActionType(999),
			expected:   "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.actionType.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConflictAction_Validate(t *testing.T) {
	validConflict := ProfileConflict{
		DiscoveredRole: DiscoveredRole{
			AccountID:         "123456789012",
			AccountName:       "test-account",
			PermissionSetName: "AdministratorAccess",
			RoleName:          "AdministratorAccess",
		},
		ExistingProfiles: []Profile{{
			Name:         "existing-profile",
			Region:       "us-east-1",
			SSOStartURL:  "https://test.awsapps.com/start",
			SSORegion:    "us-east-1",
			SSOAccountID: "123456789012",
			SSORoleName:  "AdministratorAccess",
		}},
		ProposedName: "new-profile",
		ConflictType: ConflictSameRole,
	}

	tests := []struct {
		name      string
		action    ConflictAction
		expectErr bool
	}{
		{
			name: "Valid replace action",
			action: ConflictAction{
				Conflict: validConflict,
				Action:   ActionReplace,
				NewName:  "new-profile",
				OldName:  "old-profile",
			},
			expectErr: false,
		},
		{
			name: "Valid skip action",
			action: ConflictAction{
				Conflict: validConflict,
				Action:   ActionSkip,
				OldName:  "old-profile",
			},
			expectErr: false,
		},
		{
			name: "Valid create action",
			action: ConflictAction{
				Conflict: validConflict,
				Action:   ActionCreate,
				NewName:  "new-profile",
			},
			expectErr: false,
		},
		{
			name: "Replace action missing new name",
			action: ConflictAction{
				Conflict: validConflict,
				Action:   ActionReplace,
				NewName:  "", // Invalid - required for replace
				OldName:  "old-profile",
			},
			expectErr: true,
		},
		{
			name: "Replace action missing old name",
			action: ConflictAction{
				Conflict: validConflict,
				Action:   ActionReplace,
				NewName:  "new-profile",
				OldName:  "", // Invalid - required for replace
			},
			expectErr: true,
		},
		{
			name: "Skip action missing old name",
			action: ConflictAction{
				Conflict: validConflict,
				Action:   ActionSkip,
				OldName:  "", // Invalid - required for skip
			},
			expectErr: true,
		},
		{
			name: "Create action missing new name",
			action: ConflictAction{
				Conflict: validConflict,
				Action:   ActionCreate,
				NewName:  "", // Invalid - required for create
			},
			expectErr: true,
		},
		{
			name: "Invalid action type",
			action: ConflictAction{
				Conflict: validConflict,
				Action:   ActionType(999), // Invalid action type
				NewName:  "new-profile",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.action.Validate()
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProfileReplacement_Validate(t *testing.T) {
	validOldProfile := Profile{
		Name:         "old-profile",
		Region:       "us-east-1",
		SSOStartURL:  "https://test.awsapps.com/start",
		SSORegion:    "us-east-1",
		SSOAccountID: "123456789012",
		SSORoleName:  "AdministratorAccess",
	}

	validNewProfile := GeneratedProfile{
		Name:         "new-profile",
		AccountID:    "123456789012",
		AccountName:  "test-account",
		RoleName:     "AdministratorAccess",
		Region:       "us-east-1",
		SSOStartURL:  "https://test.awsapps.com/start",
		SSORegion:    "us-east-1",
		SSOAccountID: "123456789012",
		SSORoleName:  "AdministratorAccess",
		IsLegacy:     true,
	}

	tests := []struct {
		name        string
		replacement ProfileReplacement
		expectErr   bool
	}{
		{
			name: "Valid profile replacement",
			replacement: ProfileReplacement{
				OldProfile: validOldProfile,
				NewProfile: validNewProfile,
				OldName:    "old-profile",
				NewName:    "new-profile",
			},
			expectErr: false,
		},
		{
			name: "Invalid old profile",
			replacement: ProfileReplacement{
				OldProfile: Profile{}, // Invalid - missing required fields
				NewProfile: validNewProfile,
				OldName:    "old-profile",
				NewName:    "new-profile",
			},
			expectErr: true,
		},
		{
			name: "Invalid new profile",
			replacement: ProfileReplacement{
				OldProfile: validOldProfile,
				NewProfile: GeneratedProfile{}, // Invalid - missing required fields
				OldName:    "old-profile",
				NewName:    "new-profile",
			},
			expectErr: true,
		},
		{
			name: "Missing old name",
			replacement: ProfileReplacement{
				OldProfile: validOldProfile,
				NewProfile: validNewProfile,
				OldName:    "", // Invalid - required
				NewName:    "new-profile",
			},
			expectErr: true,
		},
		{
			name: "Missing new name",
			replacement: ProfileReplacement{
				OldProfile: validOldProfile,
				NewProfile: validNewProfile,
				OldName:    "old-profile",
				NewName:    "", // Invalid - required
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.replacement.Validate()
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
