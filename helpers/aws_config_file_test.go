package helpers

import (
	"os"
	"path/filepath"
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
