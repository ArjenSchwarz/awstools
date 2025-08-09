package helpers

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSOSession_Validate(t *testing.T) {
	tests := []struct {
		name      string
		session   SSOSession
		expectErr bool
	}{
		{
			name: "Valid SSO session",
			session: SSOSession{
				Name:        "test-session",
				SSOStartURL: "https://test.awsapps.com/start",
				SSORegion:   "us-east-1",
			},
			expectErr: false,
		},
		{
			name: "Missing name",
			session: SSOSession{
				Name:        "", // Invalid - required
				SSOStartURL: "https://test.awsapps.com/start",
				SSORegion:   "us-east-1",
			},
			expectErr: true,
		},
		{
			name: "Missing SSO start URL",
			session: SSOSession{
				Name:        "test-session",
				SSOStartURL: "", // Invalid - required
				SSORegion:   "us-east-1",
			},
			expectErr: true,
		},
		{
			name: "Missing SSO region",
			session: SSOSession{
				Name:        "test-session",
				SSOStartURL: "https://test.awsapps.com/start",
				SSORegion:   "", // Invalid - required
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.session.Validate()
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestResolvedSSOConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    ResolvedSSOConfig
		expectErr bool
	}{
		{
			name: "Valid resolved SSO config",
			config: ResolvedSSOConfig{
				StartURL:  "https://test.awsapps.com/start",
				Region:    "us-east-1",
				AccountID: "123456789012",
				RoleName:  "AdministratorAccess",
			},
			expectErr: false,
		},
		{
			name: "Missing start URL",
			config: ResolvedSSOConfig{
				StartURL:  "", // Invalid - required
				Region:    "us-east-1",
				AccountID: "123456789012",
				RoleName:  "AdministratorAccess",
			},
			expectErr: true,
		},
		{
			name: "Missing region",
			config: ResolvedSSOConfig{
				StartURL:  "https://test.awsapps.com/start",
				Region:    "", // Invalid - required
				AccountID: "123456789012",
				RoleName:  "AdministratorAccess",
			},
			expectErr: true,
		},
		{
			name: "Missing account ID",
			config: ResolvedSSOConfig{
				StartURL:  "https://test.awsapps.com/start",
				Region:    "us-east-1",
				AccountID: "", // Invalid - required
				RoleName:  "AdministratorAccess",
			},
			expectErr: true,
		},
		{
			name: "Missing role name",
			config: ResolvedSSOConfig{
				StartURL:  "https://test.awsapps.com/start",
				Region:    "us-east-1",
				AccountID: "123456789012",
				RoleName:  "", // Invalid - required
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAWSConfigFile_ResolveSSOSession(t *testing.T) {
	configFile := &AWSConfigFile{
		Sessions: map[string]SSOSession{
			"test-session": {
				Name:        "test-session",
				SSOStartURL: "https://test.awsapps.com/start",
				SSORegion:   "us-east-1",
			},
		},
	}

	tests := []struct {
		name        string
		sessionName string
		expectErr   bool
		expected    *SSOSession
	}{
		{
			name:        "Valid session resolution",
			sessionName: "test-session",
			expectErr:   false,
			expected: &SSOSession{
				Name:        "test-session",
				SSOStartURL: "https://test.awsapps.com/start",
				SSORegion:   "us-east-1",
			},
		},
		{
			name:        "Session not found",
			sessionName: "nonexistent-session",
			expectErr:   true,
			expected:    nil,
		},
		{
			name:        "Empty session name",
			sessionName: "",
			expectErr:   true,
			expected:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := configFile.ResolveSSOSession(tt.sessionName)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestAWSConfigFile_ResolveProfileSSOConfig(t *testing.T) {
	configFile := &AWSConfigFile{
		Sessions: map[string]SSOSession{
			"test-session": {
				Name:        "test-session",
				SSOStartURL: "https://test.awsapps.com/start",
				SSORegion:   "us-east-1",
			},
		},
	}

	tests := []struct {
		name      string
		profile   Profile
		expectErr bool
		expected  *ResolvedSSOConfig
	}{
		{
			name: "Legacy SSO format",
			profile: Profile{
				Name:         "legacy-profile",
				SSOStartURL:  "https://test.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "123456789012",
				SSORoleName:  "AdministratorAccess",
			},
			expectErr: false,
			expected: &ResolvedSSOConfig{
				StartURL:  "https://test.awsapps.com/start",
				Region:    "us-east-1",
				AccountID: "123456789012",
				RoleName:  "AdministratorAccess",
			},
		},
		{
			name: "SSO session format",
			profile: Profile{
				Name:         "session-profile",
				SSOSession:   "test-session",
				SSOAccountID: "123456789012",
				SSORoleName:  "AdministratorAccess",
			},
			expectErr: false,
			expected: &ResolvedSSOConfig{
				StartURL:  "https://test.awsapps.com/start",
				Region:    "us-east-1",
				AccountID: "123456789012",
				RoleName:  "AdministratorAccess",
			},
		},
		{
			name: "Invalid SSO session reference",
			profile: Profile{
				Name:         "invalid-session-profile",
				SSOSession:   "nonexistent-session",
				SSOAccountID: "123456789012",
				SSORoleName:  "AdministratorAccess",
			},
			expectErr: true,
			expected:  nil,
		},
		{
			name: "No SSO configuration",
			profile: Profile{
				Name:   "no-sso-profile",
				Region: "us-east-1",
			},
			expectErr: true,
			expected:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := configFile.ResolveProfileSSOConfig(tt.profile)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestAWSConfigFile_MatchesRole(t *testing.T) {
	configFile := &AWSConfigFile{
		Sessions: map[string]SSOSession{
			"test-session": {
				Name:        "test-session",
				SSOStartURL: "https://test.awsapps.com/start",
				SSORegion:   "us-east-1",
			},
		},
	}

	tests := []struct {
		name      string
		profile   Profile
		accountID string
		roleName  string
		startURL  string
		expectErr bool
		expected  bool
	}{
		{
			name: "Legacy format matches",
			profile: Profile{
				Name:         "legacy-profile",
				SSOStartURL:  "https://test.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "123456789012",
				SSORoleName:  "AdministratorAccess",
			},
			accountID: "123456789012",
			roleName:  "AdministratorAccess",
			startURL:  "https://test.awsapps.com/start",
			expectErr: false,
			expected:  true,
		},
		{
			name: "Session format matches",
			profile: Profile{
				Name:         "session-profile",
				SSOSession:   "test-session",
				SSOAccountID: "123456789012",
				SSORoleName:  "AdministratorAccess",
			},
			accountID: "123456789012",
			roleName:  "AdministratorAccess",
			startURL:  "https://test.awsapps.com/start",
			expectErr: false,
			expected:  true,
		},
		{
			name: "Different account ID",
			profile: Profile{
				Name:         "legacy-profile",
				SSOStartURL:  "https://test.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "123456789012",
				SSORoleName:  "AdministratorAccess",
			},
			accountID: "987654321098", // Different account ID
			roleName:  "AdministratorAccess",
			startURL:  "https://test.awsapps.com/start",
			expectErr: false,
			expected:  false,
		},
		{
			name: "Different role name",
			profile: Profile{
				Name:         "legacy-profile",
				SSOStartURL:  "https://test.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "123456789012",
				SSORoleName:  "AdministratorAccess",
			},
			accountID: "123456789012",
			roleName:  "ReadOnlyAccess", // Different role name
			startURL:  "https://test.awsapps.com/start",
			expectErr: false,
			expected:  false,
		},
		{
			name: "Different start URL",
			profile: Profile{
				Name:         "legacy-profile",
				SSOStartURL:  "https://test.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "123456789012",
				SSORoleName:  "AdministratorAccess",
			},
			accountID: "123456789012",
			roleName:  "AdministratorAccess",
			startURL:  "https://different.awsapps.com/start", // Different start URL
			expectErr: false,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := configFile.MatchesRole(tt.profile, tt.accountID, tt.roleName, tt.startURL)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestAWSConfigFile_FindProfilesForRole(t *testing.T) {
	configFile := &AWSConfigFile{
		Profiles: map[string]Profile{
			"legacy-profile": {
				Name:         "legacy-profile",
				SSOStartURL:  "https://test.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "123456789012",
				SSORoleName:  "AdministratorAccess",
			},
			"different-role": {
				Name:         "different-role",
				SSOStartURL:  "https://test.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "123456789012",
				SSORoleName:  "ReadOnlyAccess",
			},
			"different-account": {
				Name:         "different-account",
				SSOStartURL:  "https://test.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "987654321098",
				SSORoleName:  "AdministratorAccess",
			},
		},
		Sessions: map[string]SSOSession{
			"test-session": {
				Name:        "test-session",
				SSOStartURL: "https://test.awsapps.com/start",
				SSORegion:   "us-east-1",
			},
		},
	}

	tests := []struct {
		name      string
		accountID string
		roleName  string
		startURL  string
		expected  []string // Profile names that should match
	}{
		{
			name:      "Find matching profile",
			accountID: "123456789012",
			roleName:  "AdministratorAccess",
			startURL:  "https://test.awsapps.com/start",
			expected:  []string{"legacy-profile"},
		},
		{
			name:      "No matching profiles",
			accountID: "999999999999",
			roleName:  "AdministratorAccess",
			startURL:  "https://test.awsapps.com/start",
			expected:  []string{},
		},
		{
			name:      "Multiple matching profiles would be empty in this test",
			accountID: "123456789012",
			roleName:  "ReadOnlyAccess",
			startURL:  "https://test.awsapps.com/start",
			expected:  []string{"different-role"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := configFile.FindProfilesForRole(tt.accountID, tt.roleName, tt.startURL)
			assert.NoError(t, err)

			// Convert result to profile names for easier comparison
			var resultNames []string
			for _, profile := range result {
				resultNames = append(resultNames, profile.Name)
			}

			assert.ElementsMatch(t, tt.expected, resultNames)
		})
	}
}

func TestAWSConfigFile_ParseSSOSessions(t *testing.T) {
	// Create a temporary config file with SSO sessions
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")

	configContent := `[sso-session test-session]
sso_start_url = https://test.awsapps.com/start
sso_region = us-east-1

[sso-session another-session]
sso_start_url = https://another.awsapps.com/start
sso_region = us-west-2

[profile test-profile]
region = us-east-1
sso_session = test-session
sso_account_id = 123456789012
sso_role_name = AdministratorAccess

[profile legacy-profile]
region = us-east-1
sso_start_url = https://legacy.awsapps.com/start
sso_region = us-east-1
sso_account_id = 987654321098
sso_role_name = ReadOnlyAccess
`

	err := os.WriteFile(configPath, []byte(configContent), 0600)
	require.NoError(t, err)

	// Load the config file
	configFile, err := LoadAWSConfigFile(configPath)
	require.NoError(t, err)

	// Test SSO sessions were parsed correctly
	assert.Len(t, configFile.Sessions, 2)

	testSession, exists := configFile.Sessions["test-session"]
	assert.True(t, exists)
	assert.Equal(t, "test-session", testSession.Name)
	assert.Equal(t, "https://test.awsapps.com/start", testSession.SSOStartURL)
	assert.Equal(t, "us-east-1", testSession.SSORegion)

	anotherSession, exists := configFile.Sessions["another-session"]
	assert.True(t, exists)
	assert.Equal(t, "another-session", anotherSession.Name)
	assert.Equal(t, "https://another.awsapps.com/start", anotherSession.SSOStartURL)
	assert.Equal(t, "us-west-2", anotherSession.SSORegion)

	// Test profiles were parsed correctly
	assert.Len(t, configFile.Profiles, 2)

	testProfile, exists := configFile.Profiles["test-profile"]
	assert.True(t, exists)
	assert.Equal(t, "test-session", testProfile.SSOSession)
	assert.Equal(t, "123456789012", testProfile.SSOAccountID)
	assert.Equal(t, "AdministratorAccess", testProfile.SSORoleName)

	legacyProfile, exists := configFile.Profiles["legacy-profile"]
	assert.True(t, exists)
	assert.Equal(t, "https://legacy.awsapps.com/start", legacyProfile.SSOStartURL)
	assert.Equal(t, "987654321098", legacyProfile.SSOAccountID)
	assert.Equal(t, "ReadOnlyAccess", legacyProfile.SSORoleName)
}
func TestAWSConfigFile_FindProfilesByName(t *testing.T) {
	configFile := &AWSConfigFile{
		Profiles: map[string]Profile{
			"profile1": {Name: "profile1", Region: "us-east-1"},
			"profile2": {Name: "profile2", Region: "us-west-2"},
			"profile3": {Name: "profile3", Region: "eu-west-1"},
		},
	}

	tests := []struct {
		name         string
		profileNames []string
		expected     map[string]Profile
	}{
		{
			name:         "Find existing profiles",
			profileNames: []string{"profile1", "profile2"},
			expected: map[string]Profile{
				"profile1": {Name: "profile1", Region: "us-east-1"},
				"profile2": {Name: "profile2", Region: "us-west-2"},
			},
		},
		{
			name:         "Find non-existing profiles",
			profileNames: []string{"nonexistent1", "nonexistent2"},
			expected:     map[string]Profile{},
		},
		{
			name:         "Mixed existing and non-existing",
			profileNames: []string{"profile1", "nonexistent", "profile3"},
			expected: map[string]Profile{
				"profile1": {Name: "profile1", Region: "us-east-1"},
				"profile3": {Name: "profile3", Region: "eu-west-1"},
			},
		},
		{
			name:         "Empty input",
			profileNames: []string{},
			expected:     map[string]Profile{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := configFile.FindProfilesByName(tt.profileNames)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAWSConfigFile_HasProfileName(t *testing.T) {
	configFile := &AWSConfigFile{
		Profiles: map[string]Profile{
			"existing-profile": {Name: "existing-profile"},
		},
	}

	tests := []struct {
		name        string
		profileName string
		expected    bool
	}{
		{
			name:        "Existing profile",
			profileName: "existing-profile",
			expected:    true,
		},
		{
			name:        "Non-existing profile",
			profileName: "non-existing-profile",
			expected:    false,
		},
		{
			name:        "Empty profile name",
			profileName: "",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := configFile.HasProfileName(tt.profileName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAWSConfigFile_FindDuplicateProfiles(t *testing.T) {
	configFile := &AWSConfigFile{
		Profiles: map[string]Profile{
			"profile1": {
				Name:         "profile1",
				SSOStartURL:  "https://test.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "123456789012",
				SSORoleName:  "AdministratorAccess",
			},
			"profile2": {
				Name:         "profile2",
				SSOStartURL:  "https://test.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "123456789012",
				SSORoleName:  "AdministratorAccess", // Same SSO config as profile1
			},
			"profile3": {
				Name:         "profile3",
				SSOStartURL:  "https://test.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "987654321098",
				SSORoleName:  "ReadOnlyAccess", // Different SSO config
			},
			"non-sso-profile": {
				Name:   "non-sso-profile",
				Region: "us-east-1", // Not an SSO profile
			},
		},
		Sessions: make(map[string]SSOSession),
	}

	duplicates, err := configFile.FindDuplicateProfiles()
	assert.NoError(t, err)

	// Should find one duplicate group (profile1 and profile2)
	assert.Len(t, duplicates, 1)

	// Find the duplicate group
	var duplicateGroup []Profile
	for _, profiles := range duplicates {
		duplicateGroup = profiles
		break
	}

	assert.Len(t, duplicateGroup, 2)

	// Extract profile names for easier comparison
	var profileNames []string
	for _, profile := range duplicateGroup {
		profileNames = append(profileNames, profile.Name)
	}
	assert.ElementsMatch(t, []string{"profile1", "profile2"}, profileNames)
}

func TestAWSConfigFile_FindProfilesWithSSOConfig(t *testing.T) {
	configFile := &AWSConfigFile{
		Profiles: map[string]Profile{
			"matching-profile1": {
				Name:         "matching-profile1",
				SSOStartURL:  "https://test.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "123456789012",
				SSORoleName:  "AdministratorAccess",
			},
			"matching-profile2": {
				Name:         "matching-profile2",
				SSOStartURL:  "https://test.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "123456789012",
				SSORoleName:  "AdministratorAccess", // Same SSO config
			},
			"different-account": {
				Name:         "different-account",
				SSOStartURL:  "https://test.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "987654321098", // Different account
				SSORoleName:  "AdministratorAccess",
			},
			"different-role": {
				Name:         "different-role",
				SSOStartURL:  "https://test.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "123456789012",
				SSORoleName:  "ReadOnlyAccess", // Different role
			},
		},
		Sessions: make(map[string]SSOSession),
	}

	tests := []struct {
		name      string
		startURL  string
		region    string
		accountID string
		roleName  string
		expected  []string // Profile names that should match
	}{
		{
			name:      "Find matching profiles",
			startURL:  "https://test.awsapps.com/start",
			region:    "us-east-1",
			accountID: "123456789012",
			roleName:  "AdministratorAccess",
			expected:  []string{"matching-profile1", "matching-profile2"},
		},
		{
			name:      "No matching profiles",
			startURL:  "https://nonexistent.awsapps.com/start",
			region:    "us-east-1",
			accountID: "123456789012",
			roleName:  "AdministratorAccess",
			expected:  []string{},
		},
		{
			name:      "Different account",
			startURL:  "https://test.awsapps.com/start",
			region:    "us-east-1",
			accountID: "987654321098",
			roleName:  "AdministratorAccess",
			expected:  []string{"different-account"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := configFile.FindProfilesWithSSOConfig(tt.startURL, tt.region, tt.accountID, tt.roleName)
			assert.NoError(t, err)

			// Convert result to profile names for easier comparison
			var resultNames []string
			for _, profile := range result {
				resultNames = append(resultNames, profile.Name)
			}

			assert.ElementsMatch(t, tt.expected, resultNames)
		})
	}
}

func TestAWSConfigFile_GetProfileNameConflicts(t *testing.T) {
	configFile := &AWSConfigFile{
		Profiles: map[string]Profile{
			"existing-profile1": {Name: "existing-profile1"},
			"existing-profile2": {Name: "existing-profile2"},
		},
	}

	tests := []struct {
		name          string
		proposedNames []string
		expected      []string
	}{
		{
			name:          "No conflicts",
			proposedNames: []string{"new-profile1", "new-profile2"},
			expected:      []string{},
		},
		{
			name:          "Some conflicts",
			proposedNames: []string{"existing-profile1", "new-profile", "existing-profile2"},
			expected:      []string{"existing-profile1", "existing-profile2"},
		},
		{
			name:          "All conflicts",
			proposedNames: []string{"existing-profile1", "existing-profile2"},
			expected:      []string{"existing-profile1", "existing-profile2"},
		},
		{
			name:          "Empty input",
			proposedNames: []string{},
			expected:      []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := configFile.GetProfileNameConflicts(tt.proposedNames)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestAWSConfigFile_BuildProfileLookupIndex(t *testing.T) {
	configFile := &AWSConfigFile{
		Profiles: map[string]Profile{
			"sso-profile1": {
				Name:         "sso-profile1",
				SSOStartURL:  "https://test.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "123456789012",
				SSORoleName:  "AdministratorAccess",
			},
			"sso-profile2": {
				Name:         "sso-profile2",
				SSOStartURL:  "https://test.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "987654321098",
				SSORoleName:  "ReadOnlyAccess",
			},
			"non-sso-profile": {
				Name:   "non-sso-profile",
				Region: "us-east-1",
			},
		},
		Sessions: make(map[string]SSOSession),
	}

	index, err := configFile.BuildProfileLookupIndex()
	assert.NoError(t, err)
	assert.NotNil(t, index)

	// Test ByName index
	assert.Len(t, index.ByName, 3)
	assert.Equal(t, "sso-profile1", index.ByName["sso-profile1"].Name)
	assert.Equal(t, "non-sso-profile", index.ByName["non-sso-profile"].Name)

	// Test ByAccount index
	assert.Len(t, index.ByAccount, 2)
	assert.Len(t, index.ByAccount["123456789012"], 1)
	assert.Equal(t, "sso-profile1", index.ByAccount["123456789012"][0].Name)
	assert.Len(t, index.ByAccount["987654321098"], 1)
	assert.Equal(t, "sso-profile2", index.ByAccount["987654321098"][0].Name)

	// Test ByRole index
	assert.Len(t, index.ByRole, 2)
	assert.Len(t, index.ByRole["AdministratorAccess"], 1)
	assert.Equal(t, "sso-profile1", index.ByRole["AdministratorAccess"][0].Name)
	assert.Len(t, index.ByRole["ReadOnlyAccess"], 1)
	assert.Equal(t, "sso-profile2", index.ByRole["ReadOnlyAccess"][0].Name)

	// Test BySSO index
	assert.Len(t, index.BySSO, 2) // Two different SSO configurations
}

func TestProfileLookupIndex_FindBySSO(t *testing.T) {
	configFile := &AWSConfigFile{
		Profiles: map[string]Profile{
			"profile1": {
				Name:         "profile1",
				SSOStartURL:  "https://test.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "123456789012",
				SSORoleName:  "AdministratorAccess",
			},
			"profile2": {
				Name:         "profile2",
				SSOStartURL:  "https://test.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "123456789012",
				SSORoleName:  "AdministratorAccess", // Same SSO config
			},
		},
		Sessions: make(map[string]SSOSession),
	}

	index, err := configFile.BuildProfileLookupIndex()
	require.NoError(t, err)

	// Find profiles with specific SSO configuration
	profiles := index.FindBySSO("https://test.awsapps.com/start", "us-east-1", "123456789012", "AdministratorAccess")
	assert.Len(t, profiles, 2)

	var profileNames []string
	for _, profile := range profiles {
		profileNames = append(profileNames, profile.Name)
	}
	assert.ElementsMatch(t, []string{"profile1", "profile2"}, profileNames)

	// Find profiles with non-existing SSO configuration
	profiles = index.FindBySSO("https://nonexistent.awsapps.com/start", "us-east-1", "123456789012", "AdministratorAccess")
	assert.Len(t, profiles, 0)
}

func TestProfileLookupIndex_FindByAccount(t *testing.T) {
	configFile := &AWSConfigFile{
		Profiles: map[string]Profile{
			"profile1": {
				Name:         "profile1",
				SSOStartURL:  "https://test.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "123456789012",
				SSORoleName:  "AdministratorAccess",
			},
			"profile2": {
				Name:         "profile2",
				SSOStartURL:  "https://test.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "123456789012",
				SSORoleName:  "ReadOnlyAccess", // Same account, different role
			},
		},
		Sessions: make(map[string]SSOSession),
	}

	index, err := configFile.BuildProfileLookupIndex()
	require.NoError(t, err)

	// Find profiles for specific account
	profiles := index.FindByAccount("123456789012")
	assert.Len(t, profiles, 2)

	var profileNames []string
	for _, profile := range profiles {
		profileNames = append(profileNames, profile.Name)
	}
	assert.ElementsMatch(t, []string{"profile1", "profile2"}, profileNames)

	// Find profiles for non-existing account
	profiles = index.FindByAccount("999999999999")
	assert.Len(t, profiles, 0)
}

func TestProfileLookupIndex_FindByRole(t *testing.T) {
	configFile := &AWSConfigFile{
		Profiles: map[string]Profile{
			"profile1": {
				Name:         "profile1",
				SSOStartURL:  "https://test.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "123456789012",
				SSORoleName:  "AdministratorAccess",
			},
			"profile2": {
				Name:         "profile2",
				SSOStartURL:  "https://test.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "987654321098",
				SSORoleName:  "AdministratorAccess", // Same role, different account
			},
		},
		Sessions: make(map[string]SSOSession),
	}

	index, err := configFile.BuildProfileLookupIndex()
	require.NoError(t, err)

	// Find profiles for specific role
	profiles := index.FindByRole("AdministratorAccess")
	assert.Len(t, profiles, 2)

	var profileNames []string
	for _, profile := range profiles {
		profileNames = append(profileNames, profile.Name)
	}
	assert.ElementsMatch(t, []string{"profile1", "profile2"}, profileNames)

	// Find profiles for non-existing role
	profiles = index.FindByRole("NonExistentRole")
	assert.Len(t, profiles, 0)
}

func TestProfileLookupIndex_HasName(t *testing.T) {
	configFile := &AWSConfigFile{
		Profiles: map[string]Profile{
			"existing-profile": {Name: "existing-profile"},
		},
		Sessions: make(map[string]SSOSession),
	}

	index, err := configFile.BuildProfileLookupIndex()
	require.NoError(t, err)

	// Test existing profile
	assert.True(t, index.HasName("existing-profile"))

	// Test non-existing profile
	assert.False(t, index.HasName("non-existing-profile"))
}

func TestAWSConfigFile_ProfileSearchEdgeCases(t *testing.T) {
	// Test with SSO session format
	configFile := &AWSConfigFile{
		Profiles: map[string]Profile{
			"session-profile": {
				Name:         "session-profile",
				SSOSession:   "test-session",
				SSOAccountID: "123456789012",
				SSORoleName:  "AdministratorAccess",
			},
			"malformed-profile": {
				Name:       "malformed-profile",
				SSOSession: "nonexistent-session", // Invalid session reference
			},
		},
		Sessions: map[string]SSOSession{
			"test-session": {
				Name:        "test-session",
				SSOStartURL: "https://test.awsapps.com/start",
				SSORegion:   "us-east-1",
			},
		},
	}

	t.Run("FindDuplicateProfiles with session format", func(t *testing.T) {
		duplicates, err := configFile.FindDuplicateProfiles()
		assert.NoError(t, err)
		// Should not find duplicates due to malformed profile being skipped
		assert.Len(t, duplicates, 0)
	})

	t.Run("FindProfilesWithSSOConfig with session format", func(t *testing.T) {
		profiles, err := configFile.FindProfilesWithSSOConfig(
			"https://test.awsapps.com/start",
			"us-east-1",
			"123456789012",
			"AdministratorAccess")
		assert.NoError(t, err)
		assert.Len(t, profiles, 1)
		assert.Equal(t, "session-profile", profiles[0].Name)
	})

	t.Run("BuildProfileLookupIndex with malformed profiles", func(t *testing.T) {
		index, err := configFile.BuildProfileLookupIndex()
		assert.NoError(t, err)

		// Should index all profiles by name, but only valid SSO profiles in other indices
		assert.Len(t, index.ByName, 2)
		assert.Len(t, index.ByAccount, 1) // Only valid session-profile
		assert.Equal(t, "session-profile", index.ByAccount["123456789012"][0].Name)
	})
}
func TestAWSConfigFile_ReplaceProfile(t *testing.T) {
	tests := []struct {
		name           string
		initialProfile Profile
		oldName        string
		newName        string
		newProfile     Profile
		expectErr      bool
		expectedErrMsg string
	}{
		{
			name: "Replace profile with same name",
			initialProfile: Profile{
				Name:         "test-profile",
				Region:       "us-east-1",
				SSOStartURL:  "https://old.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "123456789012",
				SSORoleName:  "OldRole",
				Output:       "json",
				OtherProperties: map[string]string{
					"custom_prop": "old_value",
				},
			},
			oldName: "test-profile",
			newName: "test-profile",
			newProfile: Profile{
				Name:         "test-profile",
				SSOStartURL:  "https://new.awsapps.com/start",
				SSORegion:    "us-west-2",
				SSOAccountID: "123456789012",
				SSORoleName:  "NewRole",
				OtherProperties: map[string]string{
					"new_prop": "new_value",
				},
			},
			expectErr: false,
		},
		{
			name: "Replace profile with different name",
			initialProfile: Profile{
				Name:         "old-profile",
				Region:       "us-east-1",
				SSOStartURL:  "https://test.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "123456789012",
				SSORoleName:  "TestRole",
				Output:       "json",
				OtherProperties: map[string]string{
					"custom_prop": "value",
				},
			},
			oldName: "old-profile",
			newName: "new-profile",
			newProfile: Profile{
				Name:         "new-profile",
				SSOStartURL:  "https://test.awsapps.com/start",
				SSORegion:    "us-west-2",
				SSOAccountID: "123456789012",
				SSORoleName:  "NewRole",
			},
			expectErr: false,
		},
		{
			name: "Replace non-existent profile",
			initialProfile: Profile{
				Name: "existing-profile",
			},
			oldName: "non-existent",
			newName: "new-profile",
			newProfile: Profile{
				Name: "new-profile",
			},
			expectErr:      true,
			expectedErrMsg: "profile to replace does not exist",
		},
		{
			name: "Replace with existing new name",
			initialProfile: Profile{
				Name: "profile1",
			},
			oldName: "profile1",
			newName: "profile2",
			newProfile: Profile{
				Name: "profile2",
			},
			expectErr:      true,
			expectedErrMsg: "new profile name already exists",
		},
		{
			name:           "Empty old name",
			oldName:        "",
			newName:        "new-profile",
			newProfile:     Profile{Name: "new-profile"},
			expectErr:      true,
			expectedErrMsg: "old profile name cannot be empty",
		},
		{
			name:           "Empty new name",
			oldName:        "old-profile",
			newName:        "",
			newProfile:     Profile{},
			expectErr:      true,
			expectedErrMsg: "new profile name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf := &AWSConfigFile{
				Profiles: make(map[string]Profile),
			}

			// Set up initial profiles
			if tt.initialProfile.Name != "" {
				cf.Profiles[tt.initialProfile.Name] = tt.initialProfile
			}

			// Add a second profile for conflict testing
			if tt.name == "Replace with existing new name" {
				cf.Profiles["profile2"] = Profile{Name: "profile2"}
			}

			err := cf.ReplaceProfile(tt.oldName, tt.newName, tt.newProfile)

			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				require.NoError(t, err)

				// Verify the replacement
				if tt.oldName != tt.newName {
					// Old profile should be gone
					_, exists := cf.Profiles[tt.oldName]
					assert.False(t, exists)
				}

				// New profile should exist
				newProfile, exists := cf.Profiles[tt.newName]
				require.True(t, exists)
				assert.Equal(t, tt.newName, newProfile.Name)

				// Verify property preservation
				if tt.name == "Replace profile with same name" {
					// Custom properties should be preserved if not overridden
					assert.Equal(t, "old_value", newProfile.OtherProperties["custom_prop"])
					assert.Equal(t, "new_value", newProfile.OtherProperties["new_prop"])
					// Region and output should be preserved
					assert.Equal(t, "us-east-1", newProfile.Region)
					assert.Equal(t, "json", newProfile.Output)
				}

				if tt.name == "Replace profile with different name" {
					// Custom properties should be preserved
					assert.Equal(t, "value", newProfile.OtherProperties["custom_prop"])
					// Region and output should be preserved
					assert.Equal(t, "us-east-1", newProfile.Region)
					assert.Equal(t, "json", newProfile.Output)
				}
			}
		})
	}
}

func TestAWSConfigFile_RemoveProfile(t *testing.T) {
	tests := []struct {
		name            string
		initialProfile  Profile
		profileToRemove string
		expectErr       bool
		expectedErrMsg  string
	}{
		{
			name: "Remove existing profile",
			initialProfile: Profile{
				Name: "test-profile",
			},
			profileToRemove: "test-profile",
			expectErr:       false,
		},
		{
			name: "Remove non-existent profile",
			initialProfile: Profile{
				Name: "existing-profile",
			},
			profileToRemove: "non-existent",
			expectErr:       true,
			expectedErrMsg:  "profile to remove does not exist",
		},
		{
			name:            "Empty profile name",
			profileToRemove: "",
			expectErr:       true,
			expectedErrMsg:  "profile name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf := &AWSConfigFile{
				Profiles: make(map[string]Profile),
			}

			// Set up initial profile
			if tt.initialProfile.Name != "" {
				cf.Profiles[tt.initialProfile.Name] = tt.initialProfile
			}

			err := cf.RemoveProfile(tt.profileToRemove)

			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				require.NoError(t, err)

				// Verify the profile was removed
				_, exists := cf.Profiles[tt.profileToRemove]
				assert.False(t, exists)
			}
		})
	}
}

func TestAWSConfigFile_ValidateConfigIntegrity(t *testing.T) {
	tests := []struct {
		name           string
		profiles       map[string]Profile
		sessions       map[string]SSOSession
		expectErr      bool
		expectedErrMsg string
	}{
		{
			name: "Valid config",
			profiles: map[string]Profile{
				"profile1": {Name: "profile1", Region: "us-east-1"},
				"profile2": {Name: "profile2", Region: "us-west-2"},
			},
			sessions:  map[string]SSOSession{},
			expectErr: false,
		},
		{
			name: "Profile name mismatch",
			profiles: map[string]Profile{
				"profile1": {Name: "different-name", Region: "us-east-1"},
			},
			sessions:       map[string]SSOSession{},
			expectErr:      true,
			expectedErrMsg: "profile name mismatch",
		},
		{
			name: "Profile references non-existent SSO session",
			profiles: map[string]Profile{
				"profile1": {
					Name:         "profile1",
					SSOSession:   "non-existent-session",
					SSOAccountID: "123456789012",
					SSORoleName:  "TestRole",
				},
			},
			sessions:       map[string]SSOSession{},
			expectErr:      true,
			expectedErrMsg: "profile references non-existent SSO session",
		},
		{
			name:     "SSO session name mismatch",
			profiles: map[string]Profile{},
			sessions: map[string]SSOSession{
				"session1": {Name: "different-name", SSOStartURL: "https://test.awsapps.com/start", SSORegion: "us-east-1"},
			},
			expectErr:      true,
			expectedErrMsg: "SSO session name mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf := &AWSConfigFile{
				Profiles: tt.profiles,
				Sessions: tt.sessions,
			}

			err := cf.ValidateConfigIntegrity()

			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAWSConfigFile_CreateBackup(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")

	// Write test content to the file
	testContent := `[profile test]
region = us-east-1
sso_start_url = https://test.awsapps.com/start
sso_region = us-east-1
sso_account_id = 123456789012
sso_role_name = TestRole
`
	err := os.WriteFile(configPath, []byte(testContent), 0600)
	require.NoError(t, err)

	cf := &AWSConfigFile{
		FilePath: configPath,
	}

	// Test backup creation
	backupPath, err := cf.CreateBackup()
	require.NoError(t, err)
	require.NotEmpty(t, backupPath)

	// Verify backup file exists
	_, err = os.Stat(backupPath)
	require.NoError(t, err)

	// Verify backup content matches original
	backupContent, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, testContent, string(backupContent))

	// Verify backup has same permissions
	originalInfo, err := os.Stat(configPath)
	require.NoError(t, err)
	backupInfo, err := os.Stat(backupPath)
	require.NoError(t, err)
	assert.Equal(t, originalInfo.Mode(), backupInfo.Mode())

	// Clean up
	os.Remove(backupPath)
}

func TestAWSConfigFile_CreateBackup_NoFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "non-existent-config")

	cf := &AWSConfigFile{
		FilePath: configPath,
	}

	// Test backup creation for non-existent file
	backupPath, err := cf.CreateBackup()
	require.NoError(t, err)
	assert.Empty(t, backupPath) // Should return empty string for non-existent file
}

func TestAWSConfigFile_RestoreFromBackup(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")
	backupPath := filepath.Join(tempDir, "config.backup")

	// Write original content
	originalContent := `[profile original]
region = us-east-1
`
	err := os.WriteFile(configPath, []byte(originalContent), 0600)
	require.NoError(t, err)

	// Write backup content
	backupContent := `[profile backup]
region = us-west-2
`
	err = os.WriteFile(backupPath, []byte(backupContent), 0600)
	require.NoError(t, err)

	cf := &AWSConfigFile{
		FilePath: configPath,
		Profiles: map[string]Profile{
			"original": {Name: "original", Region: "us-east-1"},
		},
	}

	// Test restore from backup
	err = cf.RestoreFromBackup(backupPath)
	require.NoError(t, err)

	// Verify file content was restored
	restoredContent, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, backupContent, string(restoredContent))

	// Verify in-memory data was updated
	assert.Contains(t, cf.Profiles, "backup")
	assert.NotContains(t, cf.Profiles, "original")
}

func TestAWSConfigFile_RestoreFromBackup_Errors(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")

	cf := &AWSConfigFile{
		FilePath: configPath,
	}

	// Test empty backup path
	err := cf.RestoreFromBackup("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "backup path cannot be empty")

	// Test non-existent backup file
	err = cf.RestoreFromBackup(filepath.Join(tempDir, "non-existent-backup"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "backup file does not exist")
}

func TestIsSSORProperty(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		{"sso_start_url", true},
		{"sso_region", true},
		{"sso_account_id", true},
		{"sso_role_name", true},
		{"sso_session", true},
		{"region", false},
		{"output", false},
		{"custom_property", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := isSSORProperty(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProfile_PreserveCustomProperties(t *testing.T) {
	// Test that ReplaceProfile preserves custom properties correctly
	cf := &AWSConfigFile{
		Profiles: make(map[string]Profile),
	}

	oldProfile := Profile{
		Name:         "test-profile",
		Region:       "us-east-1",
		SSOStartURL:  "https://old.awsapps.com/start",
		SSORegion:    "us-east-1",
		SSOAccountID: "123456789012",
		SSORoleName:  "OldRole",
		Output:       "json",
		OtherProperties: map[string]string{
			"custom_prop1":   "value1",
			"custom_prop2":   "value2",
			"sso_start_url":  "should_not_preserve", // SSO property should not be preserved
			"another_custom": "value3",
		},
	}

	cf.Profiles["test-profile"] = oldProfile

	newProfile := Profile{
		Name:         "test-profile",
		SSOStartURL:  "https://new.awsapps.com/start",
		SSORegion:    "us-west-2",
		SSOAccountID: "123456789012",
		SSORoleName:  "NewRole",
		OtherProperties: map[string]string{
			"custom_prop2": "new_value2", // This should override the old value
			"new_custom":   "new_value",
		},
	}

	err := cf.ReplaceProfile("test-profile", "test-profile", newProfile)
	require.NoError(t, err)

	// Verify the result
	result := cf.Profiles["test-profile"]

	// SSO properties should be from new profile
	assert.Equal(t, "https://new.awsapps.com/start", result.SSOStartURL)
	assert.Equal(t, "us-west-2", result.SSORegion)
	assert.Equal(t, "NewRole", result.SSORoleName)

	// Region and output should be preserved from old profile
	assert.Equal(t, "us-east-1", result.Region)
	assert.Equal(t, "json", result.Output)

	// Custom properties should be handled correctly
	assert.Equal(t, "value1", result.OtherProperties["custom_prop1"])     // Preserved from old
	assert.Equal(t, "new_value2", result.OtherProperties["custom_prop2"]) // Overridden by new
	assert.Equal(t, "value3", result.OtherProperties["another_custom"])   // Preserved from old
	assert.Equal(t, "new_value", result.OtherProperties["new_custom"])    // Added from new

	// SSO property should not be preserved in OtherProperties
	_, exists := result.OtherProperties["sso_start_url"]
	assert.False(t, exists)
}
func TestAWSConfigFile_AtomicWriteToFile(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")

	cf := &AWSConfigFile{
		FilePath: configPath,
		Profiles: map[string]Profile{
			"test-profile": {
				Name:         "test-profile",
				Region:       "us-east-1",
				SSOStartURL:  "https://test.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "123456789012",
				SSORoleName:  "TestRole",
			},
		},
		Sessions: map[string]SSOSession{
			"test-session": {
				Name:        "test-session",
				SSOStartURL: "https://test.awsapps.com/start",
				SSORegion:   "us-east-1",
			},
		},
	}

	// Test atomic write
	err := cf.AtomicWriteToFile()
	require.NoError(t, err)

	// Verify file exists and has correct permissions
	fileInfo, err := os.Stat(configPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), fileInfo.Mode().Perm())

	// Verify content
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "[sso-session test-session]")
	assert.Contains(t, contentStr, "[profile test-profile]")
	assert.Contains(t, contentStr, "sso_start_url = https://test.awsapps.com/start")
}

func TestAWSConfigFile_AtomicReplaceProfile(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")

	// Write initial content
	initialContent := `[profile old-profile]
region = us-east-1
sso_start_url = https://test.awsapps.com/start
sso_region = us-east-1
sso_account_id = 123456789012
sso_role_name = OldRole
`
	err := os.WriteFile(configPath, []byte(initialContent), 0600)
	require.NoError(t, err)

	// Load the config file
	cf, err := LoadAWSConfigFile(configPath)
	require.NoError(t, err)

	newProfile := Profile{
		Name:            "new-profile",
		Region:          "us-west-2",
		SSOStartURL:     "https://test.awsapps.com/start",
		SSORegion:       "us-west-2",
		SSOAccountID:    "123456789012",
		SSORoleName:     "NewRole",
		OtherProperties: make(map[string]string),
	}

	// Test atomic replace
	err = cf.AtomicReplaceProfile("old-profile", "new-profile", newProfile)
	require.NoError(t, err)

	// Verify the change was persisted
	updatedCf, err := LoadAWSConfigFile(configPath)
	require.NoError(t, err)

	// Old profile should be gone
	_, exists := updatedCf.Profiles["old-profile"]
	assert.False(t, exists)

	// New profile should exist
	profile, exists := updatedCf.Profiles["new-profile"]
	require.True(t, exists)
	assert.Equal(t, "new-profile", profile.Name)
	assert.Equal(t, "us-west-2", profile.Region)
	assert.Equal(t, "NewRole", profile.SSORoleName)
}

func TestAWSConfigFile_AtomicRemoveProfile(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")

	// Write initial content
	initialContent := `[profile profile1]
region = us-east-1

[profile profile2]
region = us-west-2
`
	err := os.WriteFile(configPath, []byte(initialContent), 0600)
	require.NoError(t, err)

	// Load the config file
	cf, err := LoadAWSConfigFile(configPath)
	require.NoError(t, err)

	// Test atomic remove
	err = cf.AtomicRemoveProfile("profile1")
	require.NoError(t, err)

	// Verify the change was persisted
	updatedCf, err := LoadAWSConfigFile(configPath)
	require.NoError(t, err)

	// profile1 should be gone
	_, exists := updatedCf.Profiles["profile1"]
	assert.False(t, exists)

	// profile2 should still exist
	_, exists = updatedCf.Profiles["profile2"]
	assert.True(t, exists)
}

func TestAWSConfigFile_AtomicOperations_Rollback(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")

	// Write initial content
	initialContent := `[profile test-profile]
region = us-east-1
`
	err := os.WriteFile(configPath, []byte(initialContent), 0600)
	require.NoError(t, err)

	// Load the config file
	cf, err := LoadAWSConfigFile(configPath)
	require.NoError(t, err)

	// Create an invalid profile that will cause validation to fail
	invalidProfile := Profile{
		Name: "", // Invalid - empty name
	}

	// Test that atomic replace rolls back on validation failure
	err = cf.AtomicReplaceProfile("test-profile", "invalid-profile", invalidProfile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation")

	// Verify original profile is still there
	profile, exists := cf.Profiles["test-profile"]
	require.True(t, exists)
	assert.Equal(t, "test-profile", profile.Name)

	// Verify file content wasn't corrupted
	updatedCf, err := LoadAWSConfigFile(configPath)
	require.NoError(t, err)
	_, exists = updatedCf.Profiles["test-profile"]
	assert.True(t, exists)
}

func TestAWSConfigFile_ConcurrentAccess(t *testing.T) {
	// This test simulates concurrent access scenarios
	// Note: Full concurrent testing would require more complex setup

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")

	// Write initial content
	initialContent := `[profile test-profile]
region = us-east-1
`
	err := os.WriteFile(configPath, []byte(initialContent), 0600)
	require.NoError(t, err)

	cf, err := LoadAWSConfigFile(configPath)
	require.NoError(t, err)

	// Test that atomic write completes successfully
	// In a real concurrent scenario, this would test file locking
	err = cf.AtomicWriteToFile()
	require.NoError(t, err)

	// Verify file integrity
	updatedCf, err := LoadAWSConfigFile(configPath)
	require.NoError(t, err)
	assert.Len(t, updatedCf.Profiles, 1)
}

func TestAWSConfigFile_BackupCleanup(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")

	// Write initial content
	initialContent := `[profile test-profile]
region = us-east-1
`
	err := os.WriteFile(configPath, []byte(initialContent), 0600)
	require.NoError(t, err)

	cf := &AWSConfigFile{
		FilePath: configPath,
		Profiles: map[string]Profile{
			"test-profile": {
				Name:   "test-profile",
				Region: "us-east-1",
			},
		},
		Sessions: make(map[string]SSOSession),
	}

	// Test that backup is created and cleaned up on successful write
	err = cf.AtomicWriteToFile()
	require.NoError(t, err)

	// Check that no backup files remain (they should be cleaned up)
	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)

	backupCount := 0
	for _, file := range files {
		if strings.Contains(file.Name(), ".backup.") {
			backupCount++
		}
	}
	assert.Equal(t, 0, backupCount, "Backup files should be cleaned up after successful write")
}

func TestAWSConfigFile_PermissionPreservation(t *testing.T) {
	// Create a temporary config file with specific permissions
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")

	// Write initial content with 0600 permissions
	err := os.WriteFile(configPath, []byte("[profile test]\nregion = us-east-1\n"), 0600)
	require.NoError(t, err)

	// Create backup
	cf := &AWSConfigFile{FilePath: configPath}
	backupPath, err := cf.CreateBackup()
	require.NoError(t, err)
	require.NotEmpty(t, backupPath)

	// Verify backup has same permissions
	originalInfo, err := os.Stat(configPath)
	require.NoError(t, err)
	backupInfo, err := os.Stat(backupPath)
	require.NoError(t, err)
	assert.Equal(t, originalInfo.Mode(), backupInfo.Mode())

	// Test restore preserves permissions
	// Change original file permissions
	err = os.Chmod(configPath, 0644)
	require.NoError(t, err)

	// Restore from backup
	err = cf.RestoreFromBackup(backupPath)
	require.NoError(t, err)

	// Verify permissions were restored
	restoredInfo, err := os.Stat(configPath)
	require.NoError(t, err)
	assert.Equal(t, originalInfo.Mode(), restoredInfo.Mode())

	// Clean up
	os.Remove(backupPath)
}
func TestAWSConfigFile_ParseConfigFileWithRecovery(t *testing.T) {
	tests := []struct {
		name           string
		configContent  string
		expectProfiles int
		expectSessions int
		expectError    bool
		errorType      ErrorType
	}{
		{
			name: "valid config with profiles and sessions",
			configContent: `[sso-session dev]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1

[profile dev-admin]
sso_session = dev
sso_account_id = 123456789012
sso_role_name = AdministratorAccess
region = us-east-1

[profile legacy-profile]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_account_id = 123456789012
sso_role_name = ReadOnlyAccess
region = us-east-1`,
			expectProfiles: 2,
			expectSessions: 1,
			expectError:    false,
		},
		{
			name: "malformed config with partial recovery",
			configContent: `[sso-session dev]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1

[profile dev-admin]
sso_session = dev
sso_account_id = 123456789012
sso_role_name = AdministratorAccess
region = us-east-1

[profile invalid-profile]
malformed_line_without_equals
sso_account_id = 123456789012

[profile valid-profile]
sso_session = dev
sso_account_id = 987654321098
sso_role_name = ReadOnlyAccess
region = us-west-2`,
			expectProfiles: 3, // dev-admin, invalid-profile, and valid-profile should be parsed
			expectSessions: 1,
			expectError:    true,
			errorType:      ErrorTypeValidation,
		},
		{
			name: "empty profile name",
			configContent: `[profile ]
sso_session = dev
sso_account_id = 123456789012`,
			expectProfiles: 0,
			expectSessions: 0,
			expectError:    true,
			errorType:      ErrorTypeValidation,
		},
		{
			name: "property outside section",
			configContent: `sso_start_url = https://example.awsapps.com/start

[profile dev-admin]
sso_account_id = 123456789012`,
			expectProfiles: 1,
			expectSessions: 0,
			expectError:    true,
			errorType:      ErrorTypeValidation,
		},
		{
			name: "unknown SSO session property",
			configContent: `[sso-session dev]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
unknown_property = value

[profile dev-admin]
sso_session = dev
sso_account_id = 123456789012
sso_role_name = AdministratorAccess`,
			expectProfiles: 1,
			expectSessions: 1,
			expectError:    true,
			errorType:      ErrorTypeValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile, err := os.CreateTemp("", "aws-config-test-*.conf")
			require.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			// Write test content
			_, err = tmpFile.WriteString(tt.configContent)
			require.NoError(t, err)
			tmpFile.Close()

			// Create config file instance
			configFile := &AWSConfigFile{
				FilePath: tmpFile.Name(),
				Profiles: make(map[string]Profile),
				Sessions: make(map[string]SSOSession),
			}

			// Open file for parsing
			file, err := os.Open(tmpFile.Name())
			require.NoError(t, err)
			defer file.Close()

			// Parse with recovery
			err = configFile.parseConfigFileWithRecovery(file)

			if tt.expectError {
				assert.Error(t, err)
				if pgErr, ok := err.(ProfileGeneratorError); ok {
					assert.Equal(t, tt.errorType, pgErr.Type)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectProfiles, len(configFile.Profiles))
			assert.Equal(t, tt.expectSessions, len(configFile.Sessions))
		})
	}
}

func TestValidateFilePermissionsForWrite(t *testing.T) {
	t.Run("writable file", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "aws-config-test-*.conf")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		// Set writable permissions
		err = os.Chmod(tmpFile.Name(), 0600)
		require.NoError(t, err)

		err = validateFilePermissionsForWrite(tmpFile.Name())
		assert.NoError(t, err)
	})

	t.Run("read-only file", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "aws-config-test-*.conf")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		// Set read-only permissions
		err = os.Chmod(tmpFile.Name(), 0400)
		require.NoError(t, err)

		err = validateFilePermissionsForWrite(tmpFile.Name())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not writable")
	})

	t.Run("non-existent file with writable directory", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "aws-config-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		nonExistentFile := filepath.Join(tmpDir, "config")
		err = validateFilePermissionsForWrite(nonExistentFile)
		assert.NoError(t, err)
	})

	t.Run("non-existent file with read-only directory", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "aws-config-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Set directory to read-only
		err = os.Chmod(tmpDir, 0500)
		require.NoError(t, err)

		nonExistentFile := filepath.Join(tmpDir, "config")
		err = validateFilePermissionsForWrite(nonExistentFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not writable")
	})
}

func TestWithFileLock(t *testing.T) {
	t.Run("successful lock and operation", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "aws-config-test-*.conf")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		executed := false
		err = withFileLock(tmpFile.Name(), func(file *os.File) error {
			executed = true
			// Write some test data
			_, writeErr := file.WriteString("test data")
			return writeErr
		})

		assert.NoError(t, err)
		assert.True(t, executed)

		// Verify data was written
		content, err := os.ReadFile(tmpFile.Name())
		require.NoError(t, err)
		assert.Equal(t, "test data", string(content))
	})

	t.Run("function returns error", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "aws-config-test-*.conf")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		testError := errors.New("test error")
		err = withFileLock(tmpFile.Name(), func(file *os.File) error {
			return testError
		})

		assert.Equal(t, testError, err)
	})

	t.Run("non-existent file", func(t *testing.T) {
		err := withFileLock("/non/existent/file", func(file *os.File) error {
			return nil
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open file for locking")
	})
}

func TestAWSConfigFile_WriteToFileWithLocking(t *testing.T) {
	t.Run("successful write with backup", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "aws-config-test-*.conf")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		// Write initial content
		initialContent := "[profile initial]\nregion = us-east-1\n"
		_, err = tmpFile.WriteString(initialContent)
		require.NoError(t, err)
		tmpFile.Close()

		// Create config file instance
		configFile := &AWSConfigFile{
			FilePath: tmpFile.Name(),
			Profiles: map[string]Profile{
				"test-profile": {
					Name:         "test-profile",
					Region:       "us-west-2",
					SSOStartURL:  "https://example.awsapps.com/start",
					SSORegion:    "us-east-1",
					SSOAccountID: "123456789012",
					SSORoleName:  "AdministratorAccess",
				},
			},
			Sessions: map[string]SSOSession{
				"dev": {
					Name:        "dev",
					SSOStartURL: "https://example.awsapps.com/start",
					SSORegion:   "us-east-1",
				},
			},
		}

		err = configFile.WriteToFile()
		assert.NoError(t, err)

		// Verify content was written
		content, err := os.ReadFile(tmpFile.Name())
		require.NoError(t, err)
		assert.Contains(t, string(content), "[sso-session dev]")
		assert.Contains(t, string(content), "[profile test-profile]")

		// Verify backup was created
		backupFiles, err := filepath.Glob(tmpFile.Name() + ".backup.*")
		require.NoError(t, err)
		assert.Len(t, backupFiles, 1)

		// Verify backup contains original content
		backupContent, err := os.ReadFile(backupFiles[0])
		require.NoError(t, err)
		assert.Equal(t, initialContent, string(backupContent))

		// Cleanup backup
		os.Remove(backupFiles[0])
	})

	t.Run("write failure with backup restore", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "aws-config-test-*.conf")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		// Write initial content
		initialContent := "[profile initial]\nregion = us-east-1\n"
		_, err = tmpFile.WriteString(initialContent)
		require.NoError(t, err)
		tmpFile.Close()

		// Make file read-only to cause write failure
		err = os.Chmod(tmpFile.Name(), 0400)
		require.NoError(t, err)

		configFile := &AWSConfigFile{
			FilePath: tmpFile.Name(),
			Profiles: map[string]Profile{
				"test-profile": {
					Name:   "test-profile",
					Region: "us-west-2",
				},
			},
			Sessions: make(map[string]SSOSession),
		}

		err = configFile.WriteToFile()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not writable")
	})
}

func TestAWSConfigFile_AppendToFileWithLocking(t *testing.T) {
	t.Run("successful append with backup", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "aws-config-test-*.conf")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		// Write initial content
		initialContent := "[profile initial]\nregion = us-east-1\n\n"
		_, err = tmpFile.WriteString(initialContent)
		require.NoError(t, err)
		tmpFile.Close()

		configFile := &AWSConfigFile{
			FilePath: tmpFile.Name(),
			Profiles: make(map[string]Profile),
			Sessions: make(map[string]SSOSession),
		}

		profiles := []GeneratedProfile{
			{
				Name:         "test-profile",
				Region:       "us-west-2",
				SSOStartURL:  "https://example.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "123456789012",
				SSORoleName:  "AdministratorAccess",
			},
		}

		err = configFile.AppendToFile(profiles)
		assert.NoError(t, err)

		// Verify content was appended
		content, err := os.ReadFile(tmpFile.Name())
		require.NoError(t, err)
		assert.Contains(t, string(content), "[profile initial]")
		assert.Contains(t, string(content), "[profile test-profile]")

		// Verify backup was created
		backupFiles, err := filepath.Glob(tmpFile.Name() + ".backup.*")
		require.NoError(t, err)
		assert.Len(t, backupFiles, 1)

		// Cleanup backup
		os.Remove(backupFiles[0])
	})

	t.Run("append failure with permission error", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "aws-config-test-*.conf")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		// Make file read-only to cause append failure
		err = os.Chmod(tmpFile.Name(), 0400)
		require.NoError(t, err)

		configFile := &AWSConfigFile{
			FilePath: tmpFile.Name(),
			Profiles: make(map[string]Profile),
			Sessions: make(map[string]SSOSession),
		}

		profiles := []GeneratedProfile{
			{
				Name:   "test-profile",
				Region: "us-west-2",
			},
		}

		err = configFile.AppendToFile(profiles)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not writable")
	})
}

func TestCopyFileWithPermissions(t *testing.T) {
	t.Run("successful copy with permission preservation", func(t *testing.T) {
		// Create source file with specific permissions
		srcFile, err := os.CreateTemp("", "src-*.conf")
		require.NoError(t, err)
		defer os.Remove(srcFile.Name())

		content := "test content"
		_, err = srcFile.WriteString(content)
		require.NoError(t, err)
		srcFile.Close()

		// Set specific permissions
		err = os.Chmod(srcFile.Name(), 0640)
		require.NoError(t, err)

		// Create destination path
		dstFile, err := os.CreateTemp("", "dst-*.conf")
		require.NoError(t, err)
		dstPath := dstFile.Name()
		dstFile.Close()
		os.Remove(dstPath) // Remove so we can test creation

		// Copy file
		err = copyFileWithPermissions(srcFile.Name(), dstPath)
		assert.NoError(t, err)
		defer os.Remove(dstPath)

		// Verify content
		dstContent, err := os.ReadFile(dstPath)
		require.NoError(t, err)
		assert.Equal(t, content, string(dstContent))

		// Verify permissions
		srcInfo, err := os.Stat(srcFile.Name())
		require.NoError(t, err)
		dstInfo, err := os.Stat(dstPath)
		require.NoError(t, err)
		assert.Equal(t, srcInfo.Mode(), dstInfo.Mode())
	})
}
func TestTransaction_BasicOperations(t *testing.T) {
	// Create a temporary config file
	tmpFile, err := os.CreateTemp("", "aws-config-test-*.conf")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Write initial content
	initialContent := `[profile existing]
region = us-east-1
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_account_id = 123456789012
sso_role_name = ReadOnlyAccess
`
	_, err = tmpFile.WriteString(initialContent)
	require.NoError(t, err)
	tmpFile.Close()

	// Load config file
	configFile, err := LoadAWSConfigFile(tmpFile.Name())
	require.NoError(t, err)

	t.Run("successful transaction with add, update, remove", func(t *testing.T) {
		// Begin transaction
		tx, err := configFile.BeginTransaction()
		require.NoError(t, err)

		// Add a new profile
		newProfile := Profile{
			Name:         "new-profile",
			Region:       "us-west-2",
			SSOStartURL:  "https://example.awsapps.com/start",
			SSORegion:    "us-east-1",
			SSOAccountID: "987654321098",
			SSORoleName:  "AdministratorAccess",
		}
		err = tx.AddProfile("new-profile", newProfile)
		assert.NoError(t, err)

		// Update existing profile
		updatedProfile := Profile{
			Name:         "existing",
			Region:       "us-west-1", // Changed region
			SSOStartURL:  "https://example.awsapps.com/start",
			SSORegion:    "us-east-1",
			SSOAccountID: "123456789012",
			SSORoleName:  "ReadOnlyAccess",
		}
		err = tx.UpdateProfile("existing", updatedProfile)
		assert.NoError(t, err)

		// Verify operations are recorded
		operations := tx.GetOperations()
		assert.Len(t, operations, 2)
		assert.Equal(t, OpAdd, operations[0].Type)
		assert.Equal(t, "new-profile", operations[0].ProfileName)
		assert.Equal(t, OpUpdate, operations[1].Type)
		assert.Equal(t, "existing", operations[1].ProfileName)

		// Commit transaction
		err = tx.Commit()
		assert.NoError(t, err)

		// Verify changes were applied
		assert.True(t, configFile.HasProfile("new-profile"))
		existingProfile, exists := configFile.GetProfile("existing")
		assert.True(t, exists)
		assert.Equal(t, "us-west-1", existingProfile.Region)

		// Verify file was updated
		content, err := os.ReadFile(tmpFile.Name())
		require.NoError(t, err)
		assert.Contains(t, string(content), "[profile new-profile]")
		assert.Contains(t, string(content), "region = us-west-1")
	})
}

func TestTransaction_Rollback(t *testing.T) {
	// Create a temporary config file
	tmpFile, err := os.CreateTemp("", "aws-config-test-*.conf")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Write initial content
	initialContent := `[profile existing]
region = us-east-1
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_account_id = 123456789012
sso_role_name = ReadOnlyAccess
`
	_, err = tmpFile.WriteString(initialContent)
	require.NoError(t, err)
	tmpFile.Close()

	// Load config file
	configFile, err := LoadAWSConfigFile(tmpFile.Name())
	require.NoError(t, err)

	t.Run("rollback after failed operation", func(t *testing.T) {
		// Begin transaction
		tx, err := configFile.BeginTransaction()
		require.NoError(t, err)

		// Add a valid profile
		validProfile := Profile{
			Name:         "valid-profile",
			Region:       "us-west-2",
			SSOStartURL:  "https://example.awsapps.com/start",
			SSORegion:    "us-east-1",
			SSOAccountID: "987654321098",
			SSORoleName:  "AdministratorAccess",
		}
		err = tx.AddProfile("valid-profile", validProfile)
		assert.NoError(t, err)

		// Try to add an invalid profile (should fail validation)
		invalidProfile := Profile{
			Name: "", // Invalid - empty name
		}
		err = tx.AddProfile("invalid-profile", invalidProfile)
		assert.Error(t, err)

		// Rollback the transaction
		err = tx.Rollback()
		assert.NoError(t, err)

		// Verify original state is restored
		assert.False(t, configFile.HasProfile("valid-profile"))
		assert.True(t, configFile.HasProfile("existing"))

		// Verify file content is unchanged
		content, err := os.ReadFile(tmpFile.Name())
		require.NoError(t, err)
		assert.NotContains(t, string(content), "[profile valid-profile]")
		assert.Contains(t, string(content), "[profile existing]")
	})

	t.Run("automatic rollback on commit failure", func(t *testing.T) {
		// Begin transaction first (while file is still writable)
		tx, err := configFile.BeginTransaction()
		require.NoError(t, err)

		// Add a profile
		newProfile := Profile{
			Name:         "test-profile",
			Region:       "us-west-2",
			SSOStartURL:  "https://example.awsapps.com/start",
			SSORegion:    "us-east-1",
			SSOAccountID: "987654321098",
			SSORoleName:  "AdministratorAccess",
		}
		err = tx.AddProfile("test-profile", newProfile)
		assert.NoError(t, err)

		// Verify profile was added to in-memory config
		assert.True(t, configFile.HasProfile("test-profile"))

		// Make file read-only to cause commit failure
		err = os.Chmod(tmpFile.Name(), 0400)
		require.NoError(t, err)
		defer os.Chmod(tmpFile.Name(), 0600) // Restore permissions

		// Commit should fail and trigger automatic rollback
		err = tx.Commit()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config file is not writable")

		// Verify profile was removed from in-memory config after rollback
		assert.False(t, configFile.HasProfile("test-profile"))
	})
}

func TestTransaction_ReplaceProfile(t *testing.T) {
	// Create a temporary config file
	tmpFile, err := os.CreateTemp("", "aws-config-test-*.conf")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Write initial content
	initialContent := `[profile old-name]
region = us-east-1
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_account_id = 123456789012
sso_role_name = ReadOnlyAccess
`
	_, err = tmpFile.WriteString(initialContent)
	require.NoError(t, err)
	tmpFile.Close()

	// Load config file
	configFile, err := LoadAWSConfigFile(tmpFile.Name())
	require.NoError(t, err)

	t.Run("successful profile replacement", func(t *testing.T) {
		// Begin transaction
		tx, err := configFile.BeginTransaction()
		require.NoError(t, err)

		// Replace profile with new name
		newProfile := Profile{
			Name:         "new-name",
			Region:       "us-west-2", // Changed region
			SSOStartURL:  "https://example.awsapps.com/start",
			SSORegion:    "us-east-1",
			SSOAccountID: "123456789012",
			SSORoleName:  "AdministratorAccess", // Changed role
		}
		err = tx.ReplaceProfile("old-name", "new-name", newProfile)
		assert.NoError(t, err)

		// Verify operations are recorded (should be remove + add)
		operations := tx.GetOperations()
		assert.Len(t, operations, 2)
		assert.Equal(t, OpRemove, operations[0].Type)
		assert.Equal(t, "old-name", operations[0].ProfileName)
		assert.Equal(t, OpAdd, operations[1].Type)
		assert.Equal(t, "new-name", operations[1].ProfileName)

		// Commit transaction
		err = tx.Commit()
		assert.NoError(t, err)

		// Verify changes were applied
		assert.False(t, configFile.HasProfile("old-name"))
		assert.True(t, configFile.HasProfile("new-name"))

		newProfileResult, exists := configFile.GetProfile("new-name")
		assert.True(t, exists)
		assert.Equal(t, "us-west-2", newProfileResult.Region)
		assert.Equal(t, "AdministratorAccess", newProfileResult.SSORoleName)
	})
}

func TestExecuteAtomicProfileOperations(t *testing.T) {
	// Create a temporary config file
	tmpFile, err := os.CreateTemp("", "aws-config-test-*.conf")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Write initial content
	initialContent := `[profile existing]
region = us-east-1
`
	_, err = tmpFile.WriteString(initialContent)
	require.NoError(t, err)
	tmpFile.Close()

	// Load config file
	configFile, err := LoadAWSConfigFile(tmpFile.Name())
	require.NoError(t, err)

	t.Run("successful atomic operations", func(t *testing.T) {
		operations := []func(*Transaction) error{
			func(tx *Transaction) error {
				return tx.AddProfile("profile1", Profile{
					Name:   "profile1",
					Region: "us-west-1",
				})
			},
			func(tx *Transaction) error {
				return tx.AddProfile("profile2", Profile{
					Name:   "profile2",
					Region: "us-west-2",
				})
			},
			func(tx *Transaction) error {
				return tx.UpdateProfile("existing", Profile{
					Name:   "existing",
					Region: "eu-west-1", // Changed region
				})
			},
		}

		err = configFile.ExecuteAtomicProfileOperations(operations)
		assert.NoError(t, err)

		// Verify all operations were applied
		assert.True(t, configFile.HasProfile("profile1"))
		assert.True(t, configFile.HasProfile("profile2"))

		existingProfile, exists := configFile.GetProfile("existing")
		assert.True(t, exists)
		assert.Equal(t, "eu-west-1", existingProfile.Region)
	})

	t.Run("failed atomic operations with rollback", func(t *testing.T) {
		// Record initial state
		initialProfileCount := len(configFile.Profiles)

		operations := []func(*Transaction) error{
			func(tx *Transaction) error {
				return tx.AddProfile("temp-profile1", Profile{
					Name:   "temp-profile1",
					Region: "us-east-1",
				})
			},
			func(tx *Transaction) error {
				return tx.AddProfile("temp-profile2", Profile{
					Name:   "temp-profile2",
					Region: "us-east-2",
				})
			},
			func(tx *Transaction) error {
				// This operation should fail - invalid profile
				return tx.AddProfile("invalid", Profile{
					Name: "", // Invalid - empty name
				})
			},
		}

		err = configFile.ExecuteAtomicProfileOperations(operations)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "operation 3 failed")

		// Verify no operations were applied (rollback successful)
		assert.False(t, configFile.HasProfile("temp-profile1"))
		assert.False(t, configFile.HasProfile("temp-profile2"))
		assert.False(t, configFile.HasProfile("invalid"))
		assert.Equal(t, initialProfileCount, len(configFile.Profiles))
	})
}

func TestTransaction_OperationSummary(t *testing.T) {
	// Create a temporary config file
	tmpFile, err := os.CreateTemp("", "aws-config-test-*.conf")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Load config file
	configFile, err := LoadAWSConfigFile(tmpFile.Name())
	require.NoError(t, err)

	t.Run("operation summary", func(t *testing.T) {
		// Begin transaction
		tx, err := configFile.BeginTransaction()
		require.NoError(t, err)

		// Add some operations
		err = tx.AddProfile("profile1", Profile{Name: "profile1", Region: "us-east-1"})
		assert.NoError(t, err)

		err = tx.AddProfile("profile2", Profile{Name: "profile2", Region: "us-west-1"})
		assert.NoError(t, err)

		// Get summary
		summary := tx.GetOperationSummary()
		assert.Contains(t, summary, "Transaction with 2 operations:")
		assert.Contains(t, summary, "1. add profile profile1")
		assert.Contains(t, summary, "2. add profile profile2")
		assert.Contains(t, summary, "Status: Pending")

		// Commit and check summary again
		err = tx.Commit()
		assert.NoError(t, err)

		summary = tx.GetOperationSummary()
		assert.Contains(t, summary, "Status: Committed")
	})

	t.Run("empty transaction summary", func(t *testing.T) {
		tx, err := configFile.BeginTransaction()
		require.NoError(t, err)

		summary := tx.GetOperationSummary()
		assert.Equal(t, "No operations in transaction", summary)
	})
}

func TestTransaction_ErrorHandling(t *testing.T) {
	// Create a temporary config file
	tmpFile, err := os.CreateTemp("", "aws-config-test-*.conf")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Load config file
	configFile, err := LoadAWSConfigFile(tmpFile.Name())
	require.NoError(t, err)

	t.Run("operations on completed transaction", func(t *testing.T) {
		// Begin and commit transaction
		tx, err := configFile.BeginTransaction()
		require.NoError(t, err)

		err = tx.Commit()
		assert.NoError(t, err)

		// Try to add profile to committed transaction
		err = tx.AddProfile("test", Profile{Name: "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transaction is already completed")

		// Try to rollback committed transaction
		err = tx.Rollback()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot rollback committed transaction")
	})

	t.Run("operations on rolled back transaction", func(t *testing.T) {
		// Begin and rollback transaction
		tx, err := configFile.BeginTransaction()
		require.NoError(t, err)

		err = tx.Rollback()
		assert.NoError(t, err)

		// Try to add profile to rolled back transaction
		err = tx.AddProfile("test", Profile{Name: "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transaction is already completed")

		// Try to rollback again
		err = tx.Rollback()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transaction is already rolled back")
	})

	t.Run("duplicate profile addition", func(t *testing.T) {
		tx, err := configFile.BeginTransaction()
		require.NoError(t, err)

		// Add profile
		err = tx.AddProfile("duplicate", Profile{Name: "duplicate", Region: "us-east-1"})
		assert.NoError(t, err)

		// Try to add same profile again
		err = tx.AddProfile("duplicate", Profile{Name: "duplicate", Region: "us-west-1"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "profile already exists")

		// Rollback to clean up
		err = tx.Rollback()
		assert.NoError(t, err)
	})

	t.Run("update non-existent profile", func(t *testing.T) {
		tx, err := configFile.BeginTransaction()
		require.NoError(t, err)

		// Try to update non-existent profile
		err = tx.UpdateProfile("non-existent", Profile{Name: "non-existent"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "profile does not exist")

		// Rollback to clean up
		err = tx.Rollback()
		assert.NoError(t, err)
	})

	t.Run("remove non-existent profile", func(t *testing.T) {
		tx, err := configFile.BeginTransaction()
		require.NoError(t, err)

		// Try to remove non-existent profile
		err = tx.RemoveProfile("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "profile does not exist")

		// Rollback to clean up
		err = tx.Rollback()
		assert.NoError(t, err)
	})
}
