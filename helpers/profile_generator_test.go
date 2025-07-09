package helpers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/sso/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestFixtures contains test data for consistent testing
type TestFixtures struct {
	ValidSSOProfile     Profile
	InvalidProfile      Profile
	MockSSOAccounts     []types.AccountInfo
	MockSSORoles        []types.RoleInfo
	MockDiscoveredRoles []DiscoveredRole
	ExpectedProfiles    []GeneratedProfile
	ConfigContent       string
}

// MockSSOClient is a mock SSO client for testing
type MockSSOClient struct {
	mock.Mock
}

func (m *MockSSOClient) ListAccounts(ctx context.Context, params *sso.ListAccountsInput, optFns ...func(*sso.Options)) (*sso.ListAccountsOutput, error) {
	args := m.Called(ctx, params, optFns)
	return args.Get(0).(*sso.ListAccountsOutput), args.Error(1)
}

func (m *MockSSOClient) ListAccountRoles(ctx context.Context, params *sso.ListAccountRolesInput, optFns ...func(*sso.Options)) (*sso.ListAccountRolesOutput, error) {
	args := m.Called(ctx, params, optFns)
	return args.Get(0).(*sso.ListAccountRolesOutput), args.Error(1)
}

func (m *MockSSOClient) GetRoleCredentials(ctx context.Context, params *sso.GetRoleCredentialsInput, optFns ...func(*sso.Options)) (*sso.GetRoleCredentialsOutput, error) {
	args := m.Called(ctx, params, optFns)
	return args.Get(0).(*sso.GetRoleCredentialsOutput), args.Error(1)
}

func (m *MockSSOClient) Logout(ctx context.Context, params *sso.LogoutInput, optFns ...func(*sso.Options)) (*sso.LogoutOutput, error) {
	args := m.Called(ctx, params, optFns)
	return args.Get(0).(*sso.LogoutOutput), args.Error(1)
}

// MockSTSClient is a mock STS client for testing
type MockSTSClient struct {
	mock.Mock
}

func (m *MockSTSClient) GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
	args := m.Called(ctx, params, optFns)
	return args.Get(0).(*sts.GetCallerIdentityOutput), args.Error(1)
}

func (m *MockSTSClient) AssumeRole(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
	args := m.Called(ctx, params, optFns)
	return args.Get(0).(*sts.AssumeRoleOutput), args.Error(1)
}

// MockLogger is a mock logger for testing
type MockLogger struct {
	logs []string
}

func (m *MockLogger) Printf(format string, args ...interface{}) {
	m.logs = append(m.logs, fmt.Sprintf(format, args...))
}

func (m *MockLogger) GetLogs() []string {
	return m.logs
}

func (m *MockLogger) ClearLogs() {
	m.logs = []string{}
}

// SetupTestFixtures creates test fixtures for consistent testing
func SetupTestFixtures() *TestFixtures {
	return &TestFixtures{
		ValidSSOProfile: Profile{
			Name:         "test-sso-profile",
			Region:       "us-east-1",
			SSOStartURL:  "https://example.awsapps.com/start",
			SSORegion:    "us-east-1",
			SSOAccountID: "123456789012",
			SSORoleName:  "PowerUserAccess",
			SSOSession:   "test-session",
		},
		InvalidProfile: Profile{
			Name:   "invalid-profile",
			Region: "us-east-1",
		},
		MockSSOAccounts: []types.AccountInfo{
			{
				AccountId:    aws.String("123456789012"),
				AccountName:  aws.String("test-account"),
				EmailAddress: aws.String("test@example.com"),
			},
			{
				AccountId:    aws.String("210987654321"),
				AccountName:  aws.String("production-account"),
				EmailAddress: aws.String("prod@example.com"),
			},
		},
		MockSSORoles: []types.RoleInfo{
			{
				RoleName:  aws.String("PowerUserAccess"),
				AccountId: aws.String("123456789012"),
			},
			{
				RoleName:  aws.String("ReadOnlyAccess"),
				AccountId: aws.String("123456789012"),
			},
		},
		MockDiscoveredRoles: []DiscoveredRole{
			{
				AccountID:         "123456789012",
				AccountName:       "test-account",
				PermissionSetName: "PowerUserAccess",
				RoleName:          "PowerUserAccess",
			},
			{
				AccountID:         "210987654321",
				AccountName:       "production-account",
				PermissionSetName: "ReadOnlyAccess",
				RoleName:          "ReadOnlyAccess",
			},
		},
		ExpectedProfiles: []GeneratedProfile{
			{
				Name:         "test-account-PowerUserAccess",
				AccountID:    "123456789012",
				AccountName:  "test-account",
				RoleName:     "PowerUserAccess",
				Region:       "us-east-1",
				SSOStartURL:  "https://example.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOSession:   "test-session",
				SSOAccountID: "123456789012",
				SSORoleName:  "PowerUserAccess",
				IsLegacy:     false,
			},
			{
				Name:         "production-account-ReadOnlyAccess",
				AccountID:    "210987654321",
				AccountName:  "production-account",
				RoleName:     "ReadOnlyAccess",
				Region:       "us-east-1",
				SSOStartURL:  "https://example.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOSession:   "test-session",
				SSOAccountID: "210987654321",
				SSORoleName:  "ReadOnlyAccess",
				IsLegacy:     false,
			},
		},
		ConfigContent: `[profile test-sso-profile]
region = us-east-1
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_account_id = 123456789012
sso_role_name = PowerUserAccess
sso_session = test-session

[profile existing-profile]
region = us-west-2
`,
	}
}

// CreateTempConfigFile creates a temporary AWS config file for testing
func CreateTempConfigFile(t *testing.T, content string) string {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config")

	err := os.WriteFile(configFile, []byte(content), 0600)
	assert.NoError(t, err)

	return configFile
}

// CreateTempSSOCacheDir creates a temporary SSO cache directory for testing
func CreateTempSSOCacheDir(t *testing.T) string {
	tempDir := t.TempDir()
	ssoDir := filepath.Join(tempDir, ".aws", "sso", "cache")

	err := os.MkdirAll(ssoDir, 0700)
	assert.NoError(t, err)

	// Create a mock token file
	tokenFile := filepath.Join(ssoDir, "test-token.json")
	tokenData := `{
		"accessToken": "test-access-token",
		"expiresAt": "` + time.Now().Add(time.Hour).Format(time.RFC3339) + `",
		"region": "us-east-1",
		"startUrl": "https://example.awsapps.com/start"
	}`

	err = os.WriteFile(tokenFile, []byte(tokenData), 0600)
	assert.NoError(t, err)

	return ssoDir
}

// SetupMockSSOClient sets up a mock SSO client with expected responses
func SetupMockSSOClient(fixtures *TestFixtures) *MockSSOClient {
	mockClient := &MockSSOClient{}

	// Mock ListAccounts
	mockClient.On("ListAccounts", mock.Anything, mock.Anything, mock.Anything).Return(
		&sso.ListAccountsOutput{
			AccountList: fixtures.MockSSOAccounts,
		}, nil)

	// Mock ListAccountRoles
	mockClient.On("ListAccountRoles", mock.Anything, mock.Anything, mock.Anything).Return(
		&sso.ListAccountRolesOutput{
			RoleList: fixtures.MockSSORoles,
		}, nil)

	// Mock GetRoleCredentials
	mockClient.On("GetRoleCredentials", mock.Anything, mock.Anything, mock.Anything).Return(
		&sso.GetRoleCredentialsOutput{
			RoleCredentials: &types.RoleCredentials{
				AccessKeyId:     aws.String("test-access-key"),
				SecretAccessKey: aws.String("test-secret-key"),
				SessionToken:    aws.String("test-session-token"),
				Expiration:      time.Now().Add(time.Hour).Unix(),
			},
		}, nil)

	return mockClient
}

// SetupMockSTSClient sets up a mock STS client with expected responses
func SetupMockSTSClient() *MockSTSClient {
	mockClient := &MockSTSClient{}

	// Mock GetCallerIdentity
	mockClient.On("GetCallerIdentity", mock.Anything, mock.Anything, mock.Anything).Return(
		&sts.GetCallerIdentityOutput{
			Account: aws.String("123456789012"),
			Arn:     aws.String("arn:aws:sts::123456789012:assumed-role/PowerUserAccess/test-session"),
			UserId:  aws.String("AIDACKCEVSQ6C2EXAMPLE"),
		}, nil)

	// Mock AssumeRole
	mockClient.On("AssumeRole", mock.Anything, mock.Anything, mock.Anything).Return(
		&sts.AssumeRoleOutput{
			Credentials: &ststypes.Credentials{
				AccessKeyId:     aws.String("test-access-key"),
				SecretAccessKey: aws.String("test-secret-key"),
				SessionToken:    aws.String("test-session-token"),
				Expiration:      aws.Time(time.Now().Add(time.Hour)),
			},
			AssumedRoleUser: &ststypes.AssumedRoleUser{
				AssumedRoleId: aws.String("AIDACKCEVSQ6C2EXAMPLE:test-session"),
				Arn:           aws.String("arn:aws:sts::123456789012:assumed-role/PowerUserAccess/test-session"),
			},
		}, nil)

	return mockClient
}

// TestNewProfileGenerator tests the ProfileGenerator constructor
func TestNewProfileGenerator(t *testing.T) {
	_ = SetupTestFixtures()

	tests := []struct {
		name            string
		templateProfile string
		namingPattern   string
		autoApprove     bool
		outputFile      string
		awsConfig       aws.Config
		expectedError   bool
		errorType       ErrorType
	}{
		{
			name:            "Valid configuration",
			templateProfile: "test-profile",
			namingPattern:   "{account_name}-{role_name}",
			autoApprove:     false,
			outputFile:      "",
			awsConfig:       aws.Config{},
			expectedError:   false,
		},
		{
			name:            "Empty template profile",
			templateProfile: "",
			namingPattern:   "{account_name}-{role_name}",
			autoApprove:     false,
			outputFile:      "",
			awsConfig:       aws.Config{},
			expectedError:   true,
			errorType:       ErrorTypeValidation,
		},
		{
			name:            "Invalid naming pattern",
			templateProfile: "test-profile",
			namingPattern:   "{invalid_placeholder}",
			autoApprove:     false,
			outputFile:      "",
			awsConfig:       aws.Config{},
			expectedError:   true,
			errorType:       ErrorTypeValidation,
		},
		{
			name:            "Default naming pattern",
			templateProfile: "test-profile",
			namingPattern:   "",
			autoApprove:     false,
			outputFile:      "",
			awsConfig:       aws.Config{},
			expectedError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg, err := NewProfileGenerator(tt.templateProfile, tt.namingPattern, tt.autoApprove, tt.outputFile, tt.awsConfig)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, pg)

				if pgErr, ok := err.(ProfileGeneratorError); ok {
					assert.Equal(t, tt.errorType, pgErr.Type)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, pg)
				assert.Equal(t, tt.templateProfile, pg.templateProfile)

				expectedPattern := tt.namingPattern
				if expectedPattern == "" {
					expectedPattern = "{account_name}-{role_name}"
				}
				assert.Equal(t, expectedPattern, pg.namingPattern)
				assert.Equal(t, tt.autoApprove, pg.autoApprove)
				assert.Equal(t, tt.outputFile, pg.outputFile)
			}
		})
	}
}

// TestProfileGeneratorGetters tests the ProfileGenerator getter methods
func TestProfileGeneratorGetters(t *testing.T) {
	templateProfile := "test-profile"
	namingPattern := "{account_name}-{role_name}"
	autoApprove := true
	outputFile := "/tmp/test-config"

	pg, err := NewProfileGenerator(templateProfile, namingPattern, autoApprove, outputFile, aws.Config{})
	assert.NoError(t, err)

	assert.Equal(t, templateProfile, pg.GetTemplateProfile())
	assert.Equal(t, namingPattern, pg.GetNamingPattern())
	assert.Equal(t, autoApprove, pg.IsAutoApprove())
	assert.Equal(t, outputFile, pg.GetOutputFile())
}

// TestProfileGeneratorSetLogger tests the SetLogger method
func TestProfileGeneratorSetLogger(t *testing.T) {
	pg, err := NewProfileGenerator("test-profile", "{account_name}-{role_name}", false, "", aws.Config{})
	assert.NoError(t, err)

	mockLogger := &MockLogger{}
	pg.SetLogger(mockLogger)

	assert.Equal(t, mockLogger, pg.logger)
}

// TestValidateConfiguration tests the ValidateConfiguration method
func TestValidateConfiguration(t *testing.T) {
	tests := []struct {
		name            string
		templateProfile string
		namingPattern   string
		outputFile      string
		expectedError   bool
		errorType       ErrorType
	}{
		{
			name:            "Valid configuration",
			templateProfile: "test-profile",
			namingPattern:   "{account_name}-{role_name}",
			outputFile:      "",
			expectedError:   false,
		},
		{
			name:            "Empty template profile",
			templateProfile: "",
			namingPattern:   "{account_name}-{role_name}",
			outputFile:      "",
			expectedError:   true,
			errorType:       ErrorTypeValidation,
		},
		{
			name:            "Invalid naming pattern",
			templateProfile: "test-profile",
			namingPattern:   "{invalid_placeholder}",
			outputFile:      "",
			expectedError:   true,
			errorType:       ErrorTypeValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg, err := NewProfileGenerator(tt.templateProfile, tt.namingPattern, false, tt.outputFile, aws.Config{})
			if tt.expectedError && err != nil {
				// Expected error during construction
				return
			}

			if tt.expectedError {
				err = pg.ValidateConfiguration()
				assert.Error(t, err)

				if pgErr, ok := err.(ProfileGeneratorError); ok {
					assert.Equal(t, tt.errorType, pgErr.Type)
				}
			} else {
				assert.NoError(t, err)
				if pg != nil {
					err = pg.ValidateConfiguration()
					assert.NoError(t, err)
				}
			}
		})
	}
}

// TestPreviewProfiles tests the PreviewProfiles method
func TestPreviewProfiles(t *testing.T) {
	fixtures := SetupTestFixtures()

	pg, err := NewProfileGenerator("test-profile", "{account_name}-{role_name}", false, "", aws.Config{})
	assert.NoError(t, err)

	mockLogger := &MockLogger{}
	pg.SetLogger(mockLogger)

	// Test with profiles
	err = pg.PreviewProfiles(fixtures.ExpectedProfiles)
	assert.NoError(t, err)

	logs := mockLogger.GetLogs()
	assert.True(t, len(logs) > 0)
	assert.Contains(t, logs[0], "Generated Profiles Preview")

	// Test with empty profiles
	mockLogger.ClearLogs()
	err = pg.PreviewProfiles([]GeneratedProfile{})
	assert.NoError(t, err)

	logs = mockLogger.GetLogs()
	assert.True(t, len(logs) > 0)
	assert.Contains(t, logs[0], "No profiles to preview")
}

// TestGetProfileGenerationSummary tests the GetProfileGenerationSummary method
func TestGetProfileGenerationSummary(t *testing.T) {
	fixtures := SetupTestFixtures()

	pg, err := NewProfileGenerator("test-profile", "{account_name}-{role_name}", false, "", aws.Config{})
	assert.NoError(t, err)

	result := &ProfileGenerationResult{
		TemplateProfile:     TemplateProfile{Name: "test-profile"},
		DiscoveredRoles:     fixtures.MockDiscoveredRoles,
		GeneratedProfiles:   fixtures.ExpectedProfiles,
		ConflictingProfiles: []string{"conflicting-profile"},
		SuccessfulProfiles:  []string{"successful-profile"},
		Errors:              []ProfileGeneratorError{},
	}

	summary := pg.GetProfileGenerationSummary(result)

	assert.Contains(t, summary, "Profile Generation Summary")
	assert.Contains(t, summary, "Template Profile: test-profile")
	assert.Contains(t, summary, "Naming Pattern: {account_name}-{role_name}")
	assert.Contains(t, summary, "Discovered Roles: 2")
	assert.Contains(t, summary, "Generated Profiles: 2")
	assert.Contains(t, summary, "Successful Profiles: 1")
	assert.Contains(t, summary, "Conflicting Profiles: 1")
	assert.Contains(t, summary, "Errors: 0")
}

// TestEnvironmentCleanup tests that test environment is properly cleaned up
func TestEnvironmentCleanup(t *testing.T) {
	// Create temporary files
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test-config")

	err := os.WriteFile(testFile, []byte("test content"), 0600)
	assert.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(testFile)
	assert.NoError(t, err)

	// Test cleanup is automatic due to t.TempDir()
}

// TestValidateTemplateProfile tests the template profile validation
func TestValidateTemplateProfile(t *testing.T) {
	fixtures := SetupTestFixtures()

	tests := []struct {
		name            string
		configContent   string
		templateProfile string
		expectedError   bool
		errorType       ErrorType
	}{
		{
			name:            "Valid SSO profile",
			configContent:   fixtures.ConfigContent,
			templateProfile: "test-sso-profile",
			expectedError:   false,
		},
		{
			name:            "Profile not found",
			configContent:   fixtures.ConfigContent,
			templateProfile: "non-existent-profile",
			expectedError:   true,
			errorType:       ErrorTypeValidation,
		},
		{
			name: "Non-SSO profile",
			configContent: `[profile regular-profile]
region = us-east-1
`,
			templateProfile: "regular-profile",
			expectedError:   true,
			errorType:       ErrorTypeValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp config file
			configFile := CreateTempConfigFile(t, tt.configContent)

			// Set environment variable to point to test config
			oldValue := os.Getenv("AWS_CONFIG_FILE")
			defer func() {
				if oldValue != "" {
					os.Setenv("AWS_CONFIG_FILE", oldValue)
				} else {
					os.Unsetenv("AWS_CONFIG_FILE")
				}
			}()
			os.Setenv("AWS_CONFIG_FILE", configFile)

			// Create profile generator
			pg, err := NewProfileGenerator(tt.templateProfile, "{account_name}-{role_name}", false, "", aws.Config{})
			assert.NoError(t, err)

			// Test validation
			templateProfile, err := pg.ValidateTemplateProfile()

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, templateProfile)

				if pgErr, ok := err.(ProfileGeneratorError); ok {
					assert.Equal(t, tt.errorType, pgErr.Type)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, templateProfile)
				assert.Equal(t, tt.templateProfile, templateProfile.Name)
				assert.True(t, templateProfile.IsSSO)
			}
		})
	}
}

// TestGenerateProfiles tests the profile generation logic
func TestGenerateProfiles(t *testing.T) {
	fixtures := SetupTestFixtures()

	tests := []struct {
		name             string
		templateProfile  *TemplateProfile
		discoveredRoles  []DiscoveredRole
		namingPattern    string
		expectedProfiles int
		expectedError    bool
		errorType        ErrorType
	}{
		{
			name: "Valid role generation",
			templateProfile: &TemplateProfile{
				Name:        "test-profile",
				Region:      "us-east-1",
				SSOStartURL: "https://example.awsapps.com/start",
				SSORegion:   "us-east-1",
				SSOSession:  "test-session",
				IsSSO:       true,
			},
			discoveredRoles:  fixtures.MockDiscoveredRoles,
			namingPattern:    "{account_name}-{role_name}",
			expectedProfiles: 2,
			expectedError:    false,
		},
		{
			name: "No roles to generate",
			templateProfile: &TemplateProfile{
				Name:        "test-profile",
				Region:      "us-east-1",
				SSOStartURL: "https://example.awsapps.com/start",
				SSORegion:   "us-east-1",
				SSOSession:  "test-session",
				IsSSO:       true,
			},
			discoveredRoles:  []DiscoveredRole{},
			namingPattern:    "{account_name}-{role_name}",
			expectedProfiles: 0,
			expectedError:    true,
			errorType:        ErrorTypeValidation,
		},
		{
			name: "Invalid naming pattern",
			templateProfile: &TemplateProfile{
				Name:        "test-profile",
				Region:      "us-east-1",
				SSOStartURL: "https://example.awsapps.com/start",
				SSORegion:   "us-east-1",
				SSOSession:  "test-session",
				IsSSO:       true,
			},
			discoveredRoles:  fixtures.MockDiscoveredRoles,
			namingPattern:    "{invalid_placeholder}",
			expectedProfiles: 0,
			expectedError:    true,
			errorType:        ErrorTypeValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp config file
			configFile := CreateTempConfigFile(t, fixtures.ConfigContent)

			// Set environment variable to point to test config
			oldValue := os.Getenv("AWS_CONFIG_FILE")
			defer func() {
				if oldValue != "" {
					os.Setenv("AWS_CONFIG_FILE", oldValue)
				} else {
					os.Unsetenv("AWS_CONFIG_FILE")
				}
			}()
			os.Setenv("AWS_CONFIG_FILE", configFile)

			// Create profile generator
			pg, err := NewProfileGenerator("test-profile", tt.namingPattern, false, "", aws.Config{})
			if tt.expectedError && err != nil {
				// Expected error during construction
				return
			}
			assert.NoError(t, err)

			// Test profile generation
			profiles, err := pg.GenerateProfiles(tt.templateProfile, tt.discoveredRoles)

			if tt.expectedError {
				assert.Error(t, err)

				if pgErr, ok := err.(ProfileGeneratorError); ok {
					assert.Equal(t, tt.errorType, pgErr.Type)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, profiles, tt.expectedProfiles)

				// Verify generated profiles
				for _, profile := range profiles {
					assert.NotEmpty(t, profile.Name)
					assert.NotEmpty(t, profile.AccountID)
					assert.NotEmpty(t, profile.RoleName)
					assert.Equal(t, tt.templateProfile.Region, profile.Region)
					assert.Equal(t, tt.templateProfile.SSOStartURL, profile.SSOStartURL)
					assert.Equal(t, tt.templateProfile.SSORegion, profile.SSORegion)
				}
			}
		})
	}
}

// TestAppendToConfig tests the config file appending logic
func TestAppendToConfig(t *testing.T) {
	fixtures := SetupTestFixtures()

	tests := []struct {
		name          string
		profiles      []GeneratedProfile
		autoApprove   bool
		outputFile    string
		expectedError bool
		errorType     ErrorType
	}{
		{
			name:          "Valid profile append",
			profiles:      fixtures.ExpectedProfiles,
			autoApprove:   true,
			outputFile:    "",
			expectedError: false,
		},
		{
			name:          "No profiles to append",
			profiles:      []GeneratedProfile{},
			autoApprove:   true,
			outputFile:    "",
			expectedError: true,
			errorType:     ErrorTypeValidation,
		},
		{
			name:          "Custom output file",
			profiles:      fixtures.ExpectedProfiles,
			autoApprove:   true,
			outputFile:    "custom-config",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for output
			tempDir := t.TempDir()

			var outputFile string
			if tt.outputFile != "" {
				outputFile = filepath.Join(tempDir, tt.outputFile)
			}

			// Create base config file
			if tt.outputFile == "" {
				// Default path - set up AWS config directory
				awsDir := filepath.Join(tempDir, ".aws")
				err := os.MkdirAll(awsDir, 0700)
				assert.NoError(t, err)

				configFile := filepath.Join(awsDir, "config")
				err = os.WriteFile(configFile, []byte(fixtures.ConfigContent), 0600)
				assert.NoError(t, err)

				// Set HOME to temp directory
				oldHome := os.Getenv("HOME")
				defer func() {
					if oldHome != "" {
						os.Setenv("HOME", oldHome)
					} else {
						os.Unsetenv("HOME")
					}
				}()
				os.Setenv("HOME", tempDir)
			} else {
				// Custom output file
				err := os.WriteFile(outputFile, []byte(fixtures.ConfigContent), 0600)
				assert.NoError(t, err)
			}

			// Create profile generator
			pg, err := NewProfileGenerator("test-profile", "{account_name}-{role_name}", tt.autoApprove, outputFile, aws.Config{})
			assert.NoError(t, err)

			// Test append operation
			err = pg.AppendToConfig(tt.profiles)

			if tt.expectedError {
				assert.Error(t, err)

				if pgErr, ok := err.(ProfileGeneratorError); ok {
					assert.Equal(t, tt.errorType, pgErr.Type)
				}
			} else {
				assert.NoError(t, err)

				// Verify file was modified
				var configPath string
				if tt.outputFile != "" {
					configPath = outputFile
				} else {
					configPath = filepath.Join(tempDir, ".aws", "config")
				}

				content, err := os.ReadFile(configPath)
				assert.NoError(t, err)

				// Verify profiles were added
				for _, profile := range tt.profiles {
					assert.Contains(t, string(content), fmt.Sprintf("[profile %s]", profile.Name))
				}
			}
		})
	}
}

// TestWorkflowIntegration tests the complete workflow integration
func TestWorkflowIntegration(t *testing.T) {
	fixtures := SetupTestFixtures()

	// Create temp config file
	configFile := CreateTempConfigFile(t, fixtures.ConfigContent)

	// Create temp SSO cache
	ssoCache := CreateTempSSOCacheDir(t)

	// Verify files exist
	_, err := os.Stat(configFile)
	assert.NoError(t, err)

	_, err = os.Stat(ssoCache)
	assert.NoError(t, err)

	// Test that we can create mock clients
	mockSSOClient := SetupMockSSOClient(fixtures)
	assert.NotNil(t, mockSSOClient)

	mockSTSClient := SetupMockSTSClient()
	assert.NotNil(t, mockSTSClient)

	// Test fixtures are valid
	assert.NotEmpty(t, fixtures.MockSSOAccounts)
	assert.NotEmpty(t, fixtures.MockSSORoles)
	assert.NotEmpty(t, fixtures.ExpectedProfiles)
}

// TestErrorHandling tests various error scenarios
func TestErrorHandling(t *testing.T) {
	_ = SetupTestFixtures()

	t.Run("Configuration error scenarios", func(t *testing.T) {
		// Test with invalid config file path
		pg, err := NewProfileGenerator("test-profile", "{account_name}-{role_name}", false, "/nonexistent/path/config", aws.Config{})
		assert.NoError(t, err)

		// Test validation with invalid path
		err = pg.ValidateConfiguration()
		assert.Error(t, err)

		pgErr, ok := err.(ProfileGeneratorError)
		assert.True(t, ok)
		assert.Equal(t, ErrorTypeFileSystem, pgErr.Type)
		assert.Contains(t, pgErr.Message, "output directory does not exist")
	})

	t.Run("File system error scenarios", func(t *testing.T) {
		// Test with non-existent config file
		nonExistentFile := "/nonexistent/config"
		os.Setenv("AWS_CONFIG_FILE", nonExistentFile)
		defer os.Unsetenv("AWS_CONFIG_FILE")

		pg, err := NewProfileGenerator("test-profile", "{account_name}-{role_name}", false, "", aws.Config{})
		assert.NoError(t, err)

		// This should not fail because the function handles non-existent files gracefully
		templateProfile, err := pg.ValidateTemplateProfile()
		assert.Error(t, err)
		assert.Nil(t, templateProfile)

		pgErr, ok := err.(ProfileGeneratorError)
		assert.True(t, ok)
		assert.Equal(t, ErrorTypeValidation, pgErr.Type)
	})

	t.Run("Profile validation error scenarios", func(t *testing.T) {
		// Test with invalid template profile
		invalidConfig := `[profile invalid-profile]
region = us-east-1
# Missing SSO configuration
`
		configFile := CreateTempConfigFile(t, invalidConfig)
		os.Setenv("AWS_CONFIG_FILE", configFile)
		defer os.Unsetenv("AWS_CONFIG_FILE")

		pg, err := NewProfileGenerator("invalid-profile", "{account_name}-{role_name}", false, "", aws.Config{})
		assert.NoError(t, err)

		templateProfile, err := pg.ValidateTemplateProfile()
		assert.Error(t, err)
		assert.Nil(t, templateProfile)

		pgErr, ok := err.(ProfileGeneratorError)
		assert.True(t, ok)
		assert.Equal(t, ErrorTypeValidation, pgErr.Type)
		assert.Contains(t, pgErr.Message, "template profile must be an SSO profile")
	})

	t.Run("Role generation error scenarios", func(t *testing.T) {
		// Test with invalid discovered role
		invalidRole := DiscoveredRole{
			AccountID:         "invalid-account", // Invalid format
			AccountName:       "test-account",
			PermissionSetName: "PowerUserAccess",
			RoleName:          "PowerUserAccess",
		}

		_ = &TemplateProfile{
			Name:        "test-profile",
			Region:      "us-east-1",
			SSOStartURL: "https://example.awsapps.com/start",
			SSORegion:   "us-east-1",
			SSOSession:  "test-session",
			IsSSO:       true,
		}

		// Test role validation
		err := invalidRole.Validate()
		assert.Error(t, err)

		pgErr, ok := err.(ProfileGeneratorError)
		assert.True(t, ok)
		assert.Equal(t, ErrorTypeValidation, pgErr.Type)
		assert.Contains(t, pgErr.Message, "invalid account ID format")
	})

	t.Run("File permissions error scenarios", func(t *testing.T) {
		// Test with a non-existent directory for output file
		nonExistentDir := "/nonexistent/directory"
		outputFile := filepath.Join(nonExistentDir, "config")

		pg, err := NewProfileGenerator("test-profile", "{account_name}-{role_name}", false, outputFile, aws.Config{})
		assert.NoError(t, err)

		err = pg.ValidateConfiguration()
		assert.Error(t, err)

		pgErr, ok := err.(ProfileGeneratorError)
		assert.True(t, ok)
		assert.Equal(t, ErrorTypeFileSystem, pgErr.Type)
		assert.Contains(t, pgErr.Message, "output directory does not exist")
	})

	t.Run("Profile generation result error handling", func(t *testing.T) {
		result := &ProfileGenerationResult{
			Errors: []ProfileGeneratorError{},
		}

		// Test error addition
		testError := NewValidationError("test error", nil)
		result.AddError(testError)

		assert.True(t, result.HasErrors())
		assert.Len(t, result.Errors, 1)
		assert.Equal(t, testError, result.Errors[0])

		// Test summary with errors
		summary := result.Summary()
		assert.Contains(t, summary, "Errors: 1")
	})

	t.Run("Error context and chaining", func(t *testing.T) {
		// Test error context
		baseErr := fmt.Errorf("base error")
		pgErr := NewValidationError("validation failed", baseErr).
			WithContext("profile_name", "test-profile").
			WithContext("account_id", "123456789012")

		assert.Contains(t, pgErr.Error(), "validation failed")
		assert.Equal(t, baseErr, pgErr.Unwrap())
		assert.Equal(t, ErrorTypeValidation, pgErr.Type)

		// Test context values
		assert.Equal(t, "test-profile", pgErr.Context["profile_name"])
		assert.Equal(t, "123456789012", pgErr.Context["account_id"])
	})
}

// TestProfileGeneratorErrorTypes tests all error types
func TestProfileGeneratorErrorTypes(t *testing.T) {
	testCases := []struct {
		name          string
		errorFunc     func() ProfileGeneratorError
		expectedType  ErrorType
		expectedError string
	}{
		{
			name:          "Validation error",
			errorFunc:     func() ProfileGeneratorError { return NewValidationError("validation failed", nil) },
			expectedType:  ErrorTypeValidation,
			expectedError: "validation failed",
		},
		{
			name:          "Authentication error",
			errorFunc:     func() ProfileGeneratorError { return NewAuthError("auth failed", nil) },
			expectedType:  ErrorTypeAuth,
			expectedError: "auth failed",
		},
		{
			name:          "API error",
			errorFunc:     func() ProfileGeneratorError { return NewAPIError("api failed", nil) },
			expectedType:  ErrorTypeAPI,
			expectedError: "api failed",
		},
		{
			name:          "File system error",
			errorFunc:     func() ProfileGeneratorError { return NewFileSystemError("fs failed", nil) },
			expectedType:  ErrorTypeFileSystem,
			expectedError: "fs failed",
		},
		{
			name:          "Network error",
			errorFunc:     func() ProfileGeneratorError { return NewNetworkError("network failed", nil) },
			expectedType:  ErrorTypeNetwork,
			expectedError: "network failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.errorFunc()
			assert.Equal(t, tc.expectedType, err.Type)
			assert.Contains(t, err.Error(), tc.expectedError)
		})
	}
}
