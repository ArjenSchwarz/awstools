package helpers

import (
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
