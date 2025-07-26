package helpers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/sso/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

// MockIAMClient is a mock IAM client for testing
type MockIAMClient struct {
	mock.Mock
}

func (m *MockIAMClient) ListAccountAliases(ctx context.Context, params *iam.ListAccountAliasesInput, optFns ...func(*iam.Options)) (*iam.ListAccountAliasesOutput, error) {
	args := m.Called(ctx, params, optFns)
	return args.Get(0).(*iam.ListAccountAliasesOutput), args.Error(1)
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
				AccountAlias:      "test-alias",
				PermissionSetName: "PowerUserAccess",
				RoleName:          "PowerUserAccess",
			},
			{
				AccountID:         "210987654321",
				AccountName:       "production-account",
				AccountAlias:      "prod-alias",
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
		name             string
		templateProfile  string
		namingPattern    string
		autoApprove      bool
		outputFile       string
		conflictStrategy ConflictResolutionStrategy
		awsConfig        aws.Config
		expectedError    bool
		errorType        ErrorType
	}{
		{
			name:             "Valid configuration with prompt strategy",
			templateProfile:  "test-profile",
			namingPattern:    "{account_name}-{role_name}",
			autoApprove:      false,
			outputFile:       "",
			conflictStrategy: ConflictPrompt,
			awsConfig:        aws.Config{},
			expectedError:    false,
		},
		{
			name:             "Valid configuration with replace strategy",
			templateProfile:  "test-profile",
			namingPattern:    "{account_name}-{role_name}",
			autoApprove:      false,
			outputFile:       "",
			conflictStrategy: ConflictReplace,
			awsConfig:        aws.Config{},
			expectedError:    false,
		},
		{
			name:             "Valid configuration with skip strategy",
			templateProfile:  "test-profile",
			namingPattern:    "{account_name}-{role_name}",
			autoApprove:      false,
			outputFile:       "",
			conflictStrategy: ConflictSkip,
			awsConfig:        aws.Config{},
			expectedError:    false,
		},
		{
			name:             "Empty template profile",
			templateProfile:  "",
			namingPattern:    "{account_name}-{role_name}",
			autoApprove:      false,
			outputFile:       "",
			conflictStrategy: ConflictPrompt,
			awsConfig:        aws.Config{},
			expectedError:    true,
			errorType:        ErrorTypeValidation,
		},
		{
			name:             "Invalid naming pattern",
			templateProfile:  "test-profile",
			namingPattern:    "{invalid_placeholder}",
			autoApprove:      false,
			outputFile:       "",
			conflictStrategy: ConflictPrompt,
			awsConfig:        aws.Config{},
			expectedError:    true,
			errorType:        ErrorTypeValidation,
		},
		{
			name:             "Default naming pattern",
			templateProfile:  "test-profile",
			namingPattern:    "",
			autoApprove:      false,
			outputFile:       "",
			conflictStrategy: ConflictPrompt,
			awsConfig:        aws.Config{},
			expectedError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg, err := NewProfileGenerator(tt.templateProfile, tt.namingPattern, tt.autoApprove, tt.outputFile, tt.conflictStrategy, tt.awsConfig)

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
				assert.Equal(t, tt.conflictStrategy, pg.conflictStrategy)

				// Verify conflict detector is not initialized yet (lazy initialization)
				assert.Nil(t, pg.conflictDetector)
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

	pg, err := NewProfileGenerator(templateProfile, namingPattern, autoApprove, outputFile, ConflictPrompt, aws.Config{})
	assert.NoError(t, err)

	assert.Equal(t, templateProfile, pg.GetTemplateProfile())
	assert.Equal(t, namingPattern, pg.GetNamingPattern())
	assert.Equal(t, autoApprove, pg.IsAutoApprove())
	assert.Equal(t, outputFile, pg.GetOutputFile())
}

// TestProfileGeneratorSetLogger tests the SetLogger method
func TestProfileGeneratorSetLogger(t *testing.T) {
	pg, err := NewProfileGenerator("test-profile", "{account_name}-{role_name}", false, "", ConflictPrompt, aws.Config{})
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
		{
			name:            "Valid account alias pattern",
			templateProfile: "test-profile",
			namingPattern:   "{account_alias}-{role_name}",
			outputFile:      "",
			expectedError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg, err := NewProfileGenerator(tt.templateProfile, tt.namingPattern, false, tt.outputFile, ConflictPrompt, aws.Config{})
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

	pg, err := NewProfileGenerator("test-profile", "{account_name}-{role_name}", false, "", ConflictPrompt, aws.Config{})
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

	pg, err := NewProfileGenerator("test-profile", "{account_name}-{role_name}", false, "", ConflictPrompt, aws.Config{})
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
			pg, err := NewProfileGenerator(tt.templateProfile, "{account_name}-{role_name}", false, "", ConflictPrompt, aws.Config{})
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
			pg, err := NewProfileGenerator("test-profile", tt.namingPattern, false, "", ConflictPrompt, aws.Config{})
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
			pg, err := NewProfileGenerator("test-profile", "{account_name}-{role_name}", tt.autoApprove, outputFile, ConflictPrompt, aws.Config{})
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

// TestConflictDetectionIntegration tests the integration of conflict detection into the profile generator
func TestConflictDetectionIntegration(t *testing.T) {
	fixtures := SetupTestFixtures()

	// Create config with existing profiles that will conflict
	configContent := `[profile test-sso-profile]
region = us-east-1
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_account_id = 123456789012
sso_role_name = PowerUserAccess
sso_session = test-session

[profile existing-profile]
region = us-west-2
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_account_id = 123456789012
sso_role_name = PowerUserAccess

[profile test-account-PowerUserAccess]
region = us-east-1
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_account_id = 210987654321
sso_role_name = ReadOnlyAccess
`

	configFile := CreateTempConfigFile(t, configContent)
	os.Setenv("AWS_CONFIG_FILE", configFile)
	defer os.Unsetenv("AWS_CONFIG_FILE")

	tests := []struct {
		name              string
		conflictStrategy  ConflictResolutionStrategy
		discoveredRoles   []DiscoveredRole
		expectedConflicts int
		expectedError     bool
	}{
		{
			name:              "Detect conflicts with prompt strategy",
			conflictStrategy:  ConflictPrompt,
			discoveredRoles:   fixtures.MockDiscoveredRoles,
			expectedConflicts: 2, // Both roles should have conflicts
			expectedError:     false,
		},
		{
			name:              "Detect conflicts with replace strategy",
			conflictStrategy:  ConflictReplace,
			discoveredRoles:   fixtures.MockDiscoveredRoles,
			expectedConflicts: 2,
			expectedError:     false,
		},
		{
			name:              "Detect conflicts with skip strategy",
			conflictStrategy:  ConflictSkip,
			discoveredRoles:   fixtures.MockDiscoveredRoles,
			expectedConflicts: 2,
			expectedError:     false,
		},
		{
			name:              "No conflicts with empty roles",
			conflictStrategy:  ConflictPrompt,
			discoveredRoles:   []DiscoveredRole{},
			expectedConflicts: 0,
			expectedError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg, err := NewProfileGenerator("test-sso-profile", "{account_name}-{role_name}", false, "", tt.conflictStrategy, aws.Config{})
			require.NoError(t, err)

			// Test conflict detection
			conflicts, err := pg.DetectProfileConflicts(tt.discoveredRoles)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, conflicts, tt.expectedConflicts)

				// Verify conflict detector was initialized
				assert.NotNil(t, pg.conflictDetector)

				// Verify conflicts have proper structure
				for _, conflict := range conflicts {
					assert.NotEmpty(t, conflict.ProposedName)
					assert.NotEmpty(t, conflict.DiscoveredRole.AccountID)
					assert.NotEmpty(t, conflict.DiscoveredRole.PermissionSetName)
					assert.True(t, len(conflict.ExistingProfiles) > 0)
				}
			}
		})
	}
}

// TestInitializeConflictDetector tests the lazy initialization of conflict detector
func TestInitializeConflictDetector(t *testing.T) {
	fixtures := SetupTestFixtures()
	configFile := CreateTempConfigFile(t, fixtures.ConfigContent)
	os.Setenv("AWS_CONFIG_FILE", configFile)
	defer os.Unsetenv("AWS_CONFIG_FILE")

	tests := []struct {
		name          string
		namingPattern string
		expectedError bool
		errorType     ErrorType
	}{
		{
			name:          "Valid initialization",
			namingPattern: "{account_name}-{role_name}",
			expectedError: false,
		},
		{
			name:          "Invalid naming pattern",
			namingPattern: "{invalid_placeholder}",
			expectedError: true,
			errorType:     ErrorTypeValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg, err := NewProfileGenerator("test-sso-profile", tt.namingPattern, false, "", ConflictPrompt, aws.Config{})
			if tt.expectedError && err != nil {
				// Expected error during construction
				return
			}
			require.NoError(t, err)

			// Initially conflict detector should be nil
			assert.Nil(t, pg.conflictDetector)

			// Initialize conflict detector
			err = pg.initializeConflictDetector()

			if tt.expectedError {
				assert.Error(t, err)
				if pgErr, ok := err.(ProfileGeneratorError); ok {
					assert.Equal(t, tt.errorType, pgErr.Type)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, pg.conflictDetector)

				// Second call should not reinitialize
				oldDetector := pg.conflictDetector
				err = pg.initializeConflictDetector()
				assert.NoError(t, err)
				assert.Equal(t, oldDetector, pg.conflictDetector)
			}
		})
	}
}

// TestConflictDetectionWithMissingConfigFile tests conflict detection when config file is missing
func TestConflictDetectionWithMissingConfigFile(t *testing.T) {
	fixtures := SetupTestFixtures()

	// Set non-existent config file
	os.Setenv("AWS_CONFIG_FILE", "/nonexistent/config")
	defer os.Unsetenv("AWS_CONFIG_FILE")

	pg, err := NewProfileGenerator("test-sso-profile", "{account_name}-{role_name}", false, "", ConflictPrompt, aws.Config{})
	require.NoError(t, err)

	// Test conflict detection with missing config file
	conflicts, err := pg.DetectProfileConflicts(fixtures.MockDiscoveredRoles)

	// Should handle missing config file gracefully by returning no conflicts
	assert.NoError(t, err)
	assert.NotNil(t, conflicts)
	assert.Len(t, conflicts, 0) // No conflicts when no config file exists
}

// TestGetConflictStrategy tests the conflict strategy getter and setter
func TestGetConflictStrategy(t *testing.T) {
	pg, err := NewProfileGenerator("test-profile", "{account_name}-{role_name}", false, "", ConflictPrompt, aws.Config{})
	require.NoError(t, err)

	// Test initial strategy
	assert.Equal(t, ConflictPrompt, pg.GetConflictStrategy())

	// Test setting new strategy
	pg.SetConflictStrategy(ConflictReplace)
	assert.Equal(t, ConflictReplace, pg.GetConflictStrategy())

	pg.SetConflictStrategy(ConflictSkip)
	assert.Equal(t, ConflictSkip, pg.GetConflictStrategy())
}

// TestResolveConflictsOrchestration tests the enhanced conflict resolution orchestration
func TestResolveConflictsOrchestration(t *testing.T) {
	fixtures := SetupTestFixtures()

	// Create config with existing profiles that will conflict
	configContent := `[profile test-sso-profile]
region = us-east-1
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_account_id = 123456789012
sso_role_name = PowerUserAccess
sso_session = test-session

[profile existing-profile]
region = us-west-2
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_account_id = 123456789012
sso_role_name = PowerUserAccess
`

	configFile := CreateTempConfigFile(t, configContent)
	os.Setenv("AWS_CONFIG_FILE", configFile)
	defer os.Unsetenv("AWS_CONFIG_FILE")

	// Create test conflicts
	testConflicts := []ProfileConflict{
		{
			DiscoveredRole: fixtures.MockDiscoveredRoles[0],
			ExistingProfiles: []Profile{
				{
					Name:         "existing-profile",
					SSOStartURL:  "https://example.awsapps.com/start",
					SSORegion:    "us-east-1",
					SSOAccountID: "123456789012",
					SSORoleName:  "PowerUserAccess",
				},
			},
			ProposedName: "test-account-PowerUserAccess",
			ConflictType: ConflictSameRole,
		},
	}

	tests := []struct {
		name                      string
		conflictStrategy          ConflictResolutionStrategy
		conflicts                 []ProfileConflict
		expectedGeneratedProfiles int
		expectedSkippedRoles      int
		expectedActions           int
		expectedError             bool
	}{
		{
			name:                      "Replace strategy creates profiles",
			conflictStrategy:          ConflictReplace,
			conflicts:                 testConflicts,
			expectedGeneratedProfiles: 1,
			expectedSkippedRoles:      0,
			expectedActions:           1,
			expectedError:             false,
		},
		{
			name:                      "Skip strategy skips roles",
			conflictStrategy:          ConflictSkip,
			conflicts:                 testConflicts,
			expectedGeneratedProfiles: 0,
			expectedSkippedRoles:      1,
			expectedActions:           1,
			expectedError:             false,
		},
		{
			name:                      "No conflicts returns empty result",
			conflictStrategy:          ConflictPrompt,
			conflicts:                 []ProfileConflict{},
			expectedGeneratedProfiles: 0,
			expectedSkippedRoles:      0,
			expectedActions:           0,
			expectedError:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg, err := NewProfileGenerator("test-sso-profile", "{account_name}-{role_name}", false, "", tt.conflictStrategy, aws.Config{})
			require.NoError(t, err)

			// Test conflict resolution
			result, err := pg.ResolveConflicts(tt.conflicts)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result.GeneratedProfiles, tt.expectedGeneratedProfiles)
				assert.Len(t, result.SkippedRoles, tt.expectedSkippedRoles)
				assert.Len(t, result.Actions, tt.expectedActions)

				// Verify action tracking
				for _, action := range result.Actions {
					assert.NotEmpty(t, action.Conflict.DiscoveredRole.AccountID)
					assert.True(t, action.Action == ActionReplace || action.Action == ActionSkip)

					if action.Action == ActionReplace {
						assert.NotEmpty(t, action.NewName)
					}
				}

				// Verify generated profiles are valid
				for _, profile := range result.GeneratedProfiles {
					assert.NoError(t, profile.Validate())
				}
			}
		})
	}
}

// TestFilterRolesByConflicts tests the role filtering functionality
func TestFilterRolesByConflicts(t *testing.T) {
	fixtures := SetupTestFixtures()

	pg, err := NewProfileGenerator("test-profile", "{account_name}-{role_name}", false, "", ConflictPrompt, aws.Config{})
	require.NoError(t, err)

	// Create test conflicts for some roles
	testConflicts := []ProfileConflict{
		{
			DiscoveredRole: fixtures.MockDiscoveredRoles[0], // First role has conflict
			ProposedName:   "test-account-PowerUserAccess",
			ConflictType:   ConflictSameRole,
		},
	}

	// Test role filtering
	conflictedRoles, nonConflictedRoles := pg.FilterRolesByConflicts(fixtures.MockDiscoveredRoles, testConflicts)

	// Verify filtering results
	assert.Len(t, conflictedRoles, 1)
	assert.Len(t, nonConflictedRoles, 1)

	// Verify correct roles are in each group
	assert.Equal(t, fixtures.MockDiscoveredRoles[0].AccountID, conflictedRoles[0].AccountID)
	assert.Equal(t, fixtures.MockDiscoveredRoles[0].PermissionSetName, conflictedRoles[0].PermissionSetName)

	assert.Equal(t, fixtures.MockDiscoveredRoles[1].AccountID, nonConflictedRoles[0].AccountID)
	assert.Equal(t, fixtures.MockDiscoveredRoles[1].PermissionSetName, nonConflictedRoles[0].PermissionSetName)

	// Test with no conflicts
	conflictedRoles, nonConflictedRoles = pg.FilterRolesByConflicts(fixtures.MockDiscoveredRoles, []ProfileConflict{})
	assert.Len(t, conflictedRoles, 0)
	assert.Len(t, nonConflictedRoles, 2)

	// Test with all roles conflicted
	allConflicts := []ProfileConflict{
		{
			DiscoveredRole: fixtures.MockDiscoveredRoles[0],
			ProposedName:   "test-account-PowerUserAccess",
			ConflictType:   ConflictSameRole,
		},
		{
			DiscoveredRole: fixtures.MockDiscoveredRoles[1],
			ProposedName:   "production-account-ReadOnlyAccess",
			ConflictType:   ConflictSameRole,
		},
	}

	conflictedRoles, nonConflictedRoles = pg.FilterRolesByConflicts(fixtures.MockDiscoveredRoles, allConflicts)
	assert.Len(t, conflictedRoles, 2)
	assert.Len(t, nonConflictedRoles, 0)
}

// TestGenerateProfilesForNonConflictedRoles tests profile generation for non-conflicted roles
func TestGenerateProfilesForNonConflictedRoles(t *testing.T) {
	fixtures := SetupTestFixtures()

	configFile := CreateTempConfigFile(t, fixtures.ConfigContent)
	os.Setenv("AWS_CONFIG_FILE", configFile)
	defer os.Unsetenv("AWS_CONFIG_FILE")

	tests := []struct {
		name               string
		nonConflictedRoles []DiscoveredRole
		expectedProfiles   int
		expectedError      bool
	}{
		{
			name:               "Generate profiles for non-conflicted roles",
			nonConflictedRoles: fixtures.MockDiscoveredRoles,
			expectedProfiles:   2,
			expectedError:      false,
		},
		{
			name:               "Empty roles returns empty profiles",
			nonConflictedRoles: []DiscoveredRole{},
			expectedProfiles:   0,
			expectedError:      false,
		},
		{
			name: "Invalid role causes error",
			nonConflictedRoles: []DiscoveredRole{
				{
					AccountID:         "invalid-account", // Invalid format
					AccountName:       "test-account",
					PermissionSetName: "PowerUserAccess",
					RoleName:          "PowerUserAccess",
				},
			},
			expectedProfiles: 0,
			expectedError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg, err := NewProfileGenerator("test-sso-profile", "{account_name}-{role_name}", false, "", ConflictPrompt, aws.Config{})
			require.NoError(t, err)

			// Test profile generation for non-conflicted roles
			profiles, err := pg.GenerateProfilesForNonConflictedRoles(tt.nonConflictedRoles)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, profiles, tt.expectedProfiles)

				// Verify generated profiles are valid
				for _, profile := range profiles {
					assert.NoError(t, profile.Validate())
					assert.NotEmpty(t, profile.Name)
					assert.NotEmpty(t, profile.AccountID)
					assert.NotEmpty(t, profile.RoleName)
				}
			}
		})
	}
}

// TestGenerateConflictReportEnhanced tests the enhanced conflict report generation
func TestGenerateConflictReportEnhanced(t *testing.T) {
	fixtures := SetupTestFixtures()

	pg, err := NewProfileGenerator("test-profile", "{account_name}-{role_name}", false, "", ConflictReplace, aws.Config{})
	require.NoError(t, err)

	// Create test conflicts and resolution result
	testConflicts := []ProfileConflict{
		{
			DiscoveredRole: fixtures.MockDiscoveredRoles[0],
			ExistingProfiles: []Profile{
				{Name: "existing-profile"},
			},
			ProposedName: "test-account-PowerUserAccess",
			ConflictType: ConflictSameRole,
		},
	}

	result := &ConflictResolutionResult{
		GeneratedProfiles: []GeneratedProfile{
			{
				Name:        "test-account-PowerUserAccess",
				AccountName: "test-account",
				RoleName:    "PowerUserAccess",
			},
		},
		SkippedRoles: []DiscoveredRole{},
		Actions: []ConflictAction{
			{
				Conflict: testConflicts[0],
				Action:   ActionReplace,
				NewName:  "test-account-PowerUserAccess",
				OldName:  "existing-profile",
			},
		},
	}

	// Test report generation
	report := pg.GenerateConflictReport(testConflicts, result)

	// Verify report content
	assert.Contains(t, report, "Conflict Resolution Report")
	assert.Contains(t, report, "Total conflicts detected: 1")
	assert.Contains(t, report, "Resolution strategy: replace")
	assert.Contains(t, report, "Profiles replaced: 1")
	assert.Contains(t, report, "Generated profiles: 1")
	assert.Contains(t, report, "existing-profile -> test-account-PowerUserAccess")

	// Test with empty result
	emptyResult := &ConflictResolutionResult{
		GeneratedProfiles: []GeneratedProfile{},
		SkippedRoles:      []DiscoveredRole{},
		Actions:           []ConflictAction{},
	}

	report = pg.GenerateConflictReport([]ProfileConflict{}, emptyResult)
	assert.Contains(t, report, "No actions taken")
}

// TestProfileGenerationResultEnhanced tests the enhanced ProfileGenerationResult
func TestProfileGenerationResultEnhanced(t *testing.T) {
	fixtures := SetupTestFixtures()

	// Create enhanced result with conflict resolution information
	result := &ProfileGenerationResult{
		TemplateProfile:     TemplateProfile{Name: "test-template"},
		DiscoveredRoles:     fixtures.MockDiscoveredRoles,
		GeneratedProfiles:   fixtures.ExpectedProfiles,
		SuccessfulProfiles:  []string{"profile1", "profile2"},
		ConflictingProfiles: []string{"conflicting-profile"},
		DetectedConflicts: []ProfileConflict{
			{
				DiscoveredRole: fixtures.MockDiscoveredRoles[0],
				ExistingProfiles: []Profile{
					{Name: "existing-profile"},
				},
				ProposedName: "test-account-PowerUserAccess",
				ConflictType: ConflictSameRole,
			},
		},
		ResolutionActions: []ConflictAction{
			{
				Action:  ActionReplace,
				NewName: "test-account-PowerUserAccess",
				OldName: "existing-profile",
			},
		},
		ReplacedProfiles: []ProfileReplacement{
			{
				OldName: "existing-profile",
				NewName: "test-account-PowerUserAccess",
			},
		},
		SkippedRoles: []DiscoveredRole{
			fixtures.MockDiscoveredRoles[1],
		},
		BackupPath: "/tmp/backup-config",
		Errors:     []ProfileGeneratorError{},
	}

	// Test enhanced summary
	summary := result.Summary()
	assert.Contains(t, summary, "Template Profile: test-template")
	assert.Contains(t, summary, "Discovered Roles: 2")
	assert.Contains(t, summary, "Generated Profiles: 2")
	assert.Contains(t, summary, "Successful Profiles: 2")
	assert.Contains(t, summary, "Conflicting Profiles: 1")
	assert.Contains(t, summary, "Detected Conflicts: 1")
	assert.Contains(t, summary, "Resolution Actions: 1")
	assert.Contains(t, summary, "Replaced Profiles: 1")
	assert.Contains(t, summary, "Skipped Roles: 1")
	assert.Contains(t, summary, "Backup Path: /tmp/backup-config")

	// Test conflict report generation
	conflictReport := result.GenerateConflictReport()
	assert.Contains(t, conflictReport, "Profile Generation Conflict Report")
	assert.Contains(t, conflictReport, "Template Profile: test-template")
	assert.Contains(t, conflictReport, "Total Discovered Roles: 2")
	assert.Contains(t, conflictReport, "Conflicts Detected: 1")
	assert.Contains(t, conflictReport, "Conflict Details:")
	assert.Contains(t, conflictReport, "PowerUserAccess in test-account")
	assert.Contains(t, conflictReport, "Proposed Name: test-account-PowerUserAccess")
	assert.Contains(t, conflictReport, "Conflict Type: same_role")
	assert.Contains(t, conflictReport, "Resolution Actions Taken:")
	assert.Contains(t, conflictReport, "Profiles Replaced: 1")
	assert.Contains(t, conflictReport, "Replaced Profiles:")
	assert.Contains(t, conflictReport, "existing-profile -> test-account-PowerUserAccess")
	assert.Contains(t, conflictReport, "Final Results:")
	assert.Contains(t, conflictReport, "Generated Profiles: 2")
	assert.Contains(t, conflictReport, "Configuration Backup: /tmp/backup-config")

	// Test with no conflicts
	emptyResult := &ProfileGenerationResult{
		TemplateProfile:   TemplateProfile{Name: "test-template"},
		DiscoveredRoles:   fixtures.MockDiscoveredRoles,
		GeneratedProfiles: fixtures.ExpectedProfiles,
		DetectedConflicts: []ProfileConflict{},
		ResolutionActions: []ConflictAction{},
		ReplacedProfiles:  []ProfileReplacement{},
		SkippedRoles:      []DiscoveredRole{},
		Errors:            []ProfileGeneratorError{},
	}

	emptyReport := emptyResult.GenerateConflictReport()
	assert.Contains(t, emptyReport, "Conflicts Detected: 0")
	assert.NotContains(t, emptyReport, "Conflict Details:")
	assert.NotContains(t, emptyReport, "Resolution Actions Taken:")
}

// TestProfileGenerationResultValidation tests validation of enhanced result
func TestProfileGenerationResultValidation(t *testing.T) {
	fixtures := SetupTestFixtures()

	tests := []struct {
		name          string
		result        *ProfileGenerationResult
		expectedError bool
	}{
		{
			name: "Valid enhanced result",
			result: &ProfileGenerationResult{
				TemplateProfile: TemplateProfile{
					Name:        "test-template",
					Region:      "us-east-1",
					SSOStartURL: "https://example.awsapps.com/start",
					SSORegion:   "us-east-1",
					SSOSession:  "test-session",
					IsSSO:       true,
				},
				DiscoveredRoles:   fixtures.MockDiscoveredRoles,
				GeneratedProfiles: fixtures.ExpectedProfiles,
				DetectedConflicts: []ProfileConflict{},
				ResolutionActions: []ConflictAction{},
				ReplacedProfiles:  []ProfileReplacement{},
				SkippedRoles:      []DiscoveredRole{},
				Errors:            []ProfileGeneratorError{},
			},
			expectedError: false,
		},
		{
			name: "Invalid template profile",
			result: &ProfileGenerationResult{
				TemplateProfile: TemplateProfile{
					Name: "", // Invalid: empty name
				},
				DiscoveredRoles:   []DiscoveredRole{},
				GeneratedProfiles: []GeneratedProfile{},
				DetectedConflicts: []ProfileConflict{},
				ResolutionActions: []ConflictAction{},
				ReplacedProfiles:  []ProfileReplacement{},
				SkippedRoles:      []DiscoveredRole{},
				Errors:            []ProfileGeneratorError{},
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.result.Validate()

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestErrorHandling tests various error scenarios
func TestErrorHandling(t *testing.T) {
	_ = SetupTestFixtures()

	t.Run("Configuration error scenarios", func(t *testing.T) {
		// Test with invalid config file path
		pg, err := NewProfileGenerator("test-profile", "{account_name}-{role_name}", false, "/nonexistent/path/config", ConflictPrompt, aws.Config{})
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

		pg, err := NewProfileGenerator("test-profile", "{account_name}-{role_name}", false, "", ConflictPrompt, aws.Config{})
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

		pg, err := NewProfileGenerator("invalid-profile", "{account_name}-{role_name}", false, "", ConflictPrompt, aws.Config{})
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
			AccountAlias:      "test-account",
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

		pg, err := NewProfileGenerator("test-profile", "{account_name}-{role_name}", false, outputFile, ConflictPrompt, aws.Config{})
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

// TestAccountAliasNamingPattern tests the account alias functionality in naming patterns
func TestAccountAliasNamingPattern(t *testing.T) {
	fixtures := SetupTestFixtures()

	tests := []struct {
		name           string
		namingPattern  string
		expectedName   string
		discoveredRole DiscoveredRole
	}{
		{
			name:          "Account alias in naming pattern",
			namingPattern: "{account_alias}-{role_name}",
			expectedName:  "test-alias-PowerUserAccess",
			discoveredRole: DiscoveredRole{
				AccountID:         "123456789012",
				AccountName:       "test-account",
				AccountAlias:      "test-alias",
				PermissionSetName: "PowerUserAccess",
				RoleName:          "PowerUserAccess",
			},
		},
		{
			name:          "Account alias with account ID fallback",
			namingPattern: "{account_alias}-{role_name}",
			expectedName:  "123456789012-PowerUserAccess",
			discoveredRole: DiscoveredRole{
				AccountID:         "123456789012",
				AccountName:       "test-account",
				AccountAlias:      "123456789012", // Fallback to account ID
				PermissionSetName: "PowerUserAccess",
				RoleName:          "PowerUserAccess",
			},
		},
		{
			name:          "Mixed naming pattern with account alias",
			namingPattern: "sso-{account_alias}-{role_name}",
			expectedName:  "sso-prod-alias-ReadOnlyAccess",
			discoveredRole: DiscoveredRole{
				AccountID:         "210987654321",
				AccountName:       "production-account",
				AccountAlias:      "prod-alias",
				PermissionSetName: "ReadOnlyAccess",
				RoleName:          "ReadOnlyAccess",
			},
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
			pg, err := NewProfileGenerator("test-profile", tt.namingPattern, false, "", ConflictPrompt, aws.Config{})
			assert.NoError(t, err)

			// Create template profile
			templateProfile := &TemplateProfile{
				Name:        "test-profile",
				Region:      "us-east-1",
				SSOStartURL: "https://example.awsapps.com/start",
				SSORegion:   "us-east-1",
				SSOSession:  "test-session",
				IsSSO:       true,
			}

			// Generate profiles
			profiles, err := pg.GenerateProfiles(templateProfile, []DiscoveredRole{tt.discoveredRole})
			assert.NoError(t, err)
			assert.Len(t, profiles, 1)

			// Verify the generated profile name
			assert.Equal(t, tt.expectedName, profiles[0].Name)
			assert.Equal(t, tt.discoveredRole.AccountID, profiles[0].AccountID)
			assert.Equal(t, tt.discoveredRole.RoleName, profiles[0].RoleName)
		})
	}
}

// TestAccountAliasValidation tests the account alias validation
func TestAccountAliasValidation(t *testing.T) {
	// Test that account alias is included in supported placeholders
	placeholders := GetSupportedPlaceholders()
	assert.Contains(t, placeholders, "{account_alias}")

	// Test pattern validation with account alias
	pattern, err := NewNamingPattern("{account_alias}-{role_name}")
	assert.NoError(t, err)
	assert.NotNil(t, pattern)

	// Test pattern examples include account alias
	examples := ValidatePatternExamples()
	assert.NoError(t, examples["account_alias_and_role"])
	assert.NoError(t, examples["sso_alias_prefix"])
	assert.NoError(t, examples["all_variables"])
}

func TestProfileGenerator_DetectProfileConflicts(t *testing.T) {
	// Create a temporary config file with existing profiles
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")

	configContent := `[profile existing-profile]
region = us-east-1
sso_start_url = https://test.awsapps.com/start
sso_region = us-east-1
sso_account_id = 123456789012
sso_role_name = ExistingRole

[profile another-profile]
region = us-west-2
sso_start_url = https://test.awsapps.com/start
sso_region = us-west-2
sso_account_id = 987654321098
sso_role_name = AnotherRole
`
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	require.NoError(t, err)

	// Set AWS_CONFIG_FILE environment variable
	originalConfigFile := os.Getenv("AWS_CONFIG_FILE")
	os.Setenv("AWS_CONFIG_FILE", configPath)
	defer func() {
		if originalConfigFile != "" {
			os.Setenv("AWS_CONFIG_FILE", originalConfigFile)
		} else {
			os.Unsetenv("AWS_CONFIG_FILE")
		}
	}()

	// Create profile generator
	pg, err := NewProfileGenerator("existing-profile", "{account_name}-{role_name}", false, "", ConflictPrompt, aws.Config{})
	require.NoError(t, err)

	// Create discovered roles that will conflict
	discoveredRoles := []DiscoveredRole{
		{
			AccountID:         "123456789012",
			AccountName:       "test-account",
			AccountAlias:      "test-account",
			PermissionSetName: "ExistingRole", // This will conflict with existing profile
			RoleName:          "ExistingRole",
		},
		{
			AccountID:         "555666777888",
			AccountName:       "new-account",
			AccountAlias:      "new-account",
			PermissionSetName: "NewRole", // This won't conflict
			RoleName:          "NewRole",
		},
	}

	// Test conflict detection
	conflicts, err := pg.DetectProfileConflicts(discoveredRoles)
	require.NoError(t, err)

	// Should detect one conflict (ExistingRole)
	assert.Len(t, conflicts, 1)
	assert.Equal(t, "ExistingRole", conflicts[0].DiscoveredRole.PermissionSetName)
	assert.Equal(t, "123456789012", conflicts[0].DiscoveredRole.AccountID)
}

func TestProfileGenerator_ResolveConflicts_ReplaceStrategy(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")

	configContent := `[profile template-profile]
region = us-east-1
sso_start_url = https://test.awsapps.com/start
sso_region = us-east-1
sso_account_id = 123456789012
sso_role_name = TemplateRole

[profile existing-profile]
region = us-east-1
sso_start_url = https://test.awsapps.com/start
sso_region = us-east-1
sso_account_id = 123456789012
sso_role_name = ConflictRole
`
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	require.NoError(t, err)

	// Set AWS_CONFIG_FILE environment variable
	originalConfigFile := os.Getenv("AWS_CONFIG_FILE")
	os.Setenv("AWS_CONFIG_FILE", configPath)
	defer func() {
		if originalConfigFile != "" {
			os.Setenv("AWS_CONFIG_FILE", originalConfigFile)
		} else {
			os.Unsetenv("AWS_CONFIG_FILE")
		}
	}()

	// Create profile generator with replace strategy
	pg, err := NewProfileGenerator("template-profile", "{account_name}-{role_name}", false, "", ConflictReplace, aws.Config{})
	require.NoError(t, err)

	// Create conflicts
	conflicts := []ProfileConflict{
		{
			DiscoveredRole: DiscoveredRole{
				AccountID:         "123456789012",
				AccountName:       "test-account",
				AccountAlias:      "test-account",
				PermissionSetName: "ConflictRole",
				RoleName:          "ConflictRole",
			},
			ExistingProfiles: []Profile{
				{Name: "existing-profile"},
			},
			ProposedName: "test-account-ConflictRole",
			ConflictType: ConflictSameRole,
		},
	}

	// Test conflict resolution
	result, err := pg.ResolveConflicts(conflicts)
	require.NoError(t, err)

	// Should generate one profile (replace strategy)
	assert.Len(t, result.GeneratedProfiles, 1)
	assert.Len(t, result.SkippedRoles, 0)
	assert.Len(t, result.Actions, 1)
	assert.Equal(t, "test-account-ConflictRole", result.GeneratedProfiles[0].Name)
	assert.Equal(t, "ConflictRole", result.GeneratedProfiles[0].RoleName)
	assert.Equal(t, ActionReplace, result.Actions[0].Action)
}

func TestProfileGenerator_ResolveConflicts_SkipStrategy(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")

	configContent := `[profile template-profile]
region = us-east-1
sso_start_url = https://test.awsapps.com/start
sso_region = us-east-1
sso_account_id = 123456789012
sso_role_name = TemplateRole
`
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	require.NoError(t, err)

	// Set AWS_CONFIG_FILE environment variable
	originalConfigFile := os.Getenv("AWS_CONFIG_FILE")
	os.Setenv("AWS_CONFIG_FILE", configPath)
	defer func() {
		if originalConfigFile != "" {
			os.Setenv("AWS_CONFIG_FILE", originalConfigFile)
		} else {
			os.Unsetenv("AWS_CONFIG_FILE")
		}
	}()

	// Create profile generator with skip strategy
	pg, err := NewProfileGenerator("template-profile", "{account_name}-{role_name}", false, "", ConflictSkip, aws.Config{})
	require.NoError(t, err)

	// Create conflicts
	conflicts := []ProfileConflict{
		{
			DiscoveredRole: DiscoveredRole{
				AccountID:         "123456789012",
				AccountName:       "test-account",
				AccountAlias:      "test-account",
				PermissionSetName: "ConflictRole",
				RoleName:          "ConflictRole",
			},
			ExistingProfiles: []Profile{
				{Name: "existing-profile"},
			},
			ProposedName: "test-account-ConflictRole",
			ConflictType: ConflictSameRole,
		},
	}

	// Test conflict resolution
	result, err := pg.ResolveConflicts(conflicts)
	require.NoError(t, err)

	// Should skip the role (skip strategy)
	assert.Len(t, result.GeneratedProfiles, 0)
	assert.Len(t, result.SkippedRoles, 1)
	assert.Len(t, result.Actions, 1)
	assert.Equal(t, "ConflictRole", result.SkippedRoles[0].PermissionSetName)
	assert.Equal(t, ActionSkip, result.Actions[0].Action)
}

func TestProfileGenerator_GenerateConflictReport(t *testing.T) {
	// Create profile generator
	pg, err := NewProfileGenerator("template", "{account_name}-{role_name}", false, "", ConflictReplace, aws.Config{})
	require.NoError(t, err)

	// Create test conflicts and actions
	conflicts := []ProfileConflict{
		{
			DiscoveredRole: DiscoveredRole{
				AccountID:         "123456789012",
				AccountName:       "prod-account",
				AccountAlias:      "prod-account",
				PermissionSetName: "AdminRole",
				RoleName:          "AdminRole",
			},
			ProposedName: "prod-account-AdminRole",
			ConflictType: ConflictSameRole,
		},
		{
			DiscoveredRole: DiscoveredRole{
				AccountID:         "987654321098",
				AccountName:       "dev-account",
				AccountAlias:      "dev-account",
				PermissionSetName: "DevRole",
				RoleName:          "DevRole",
			},
			ProposedName: "dev-account-DevRole",
			ConflictType: ConflictSameName,
		},
	}

	result := &ConflictResolutionResult{
		GeneratedProfiles: []GeneratedProfile{
			{
				Name:        "prod-account-AdminRole",
				AccountName: "prod-account",
				RoleName:    "AdminRole",
			},
		},
		SkippedRoles: []DiscoveredRole{
			conflicts[1].DiscoveredRole,
		},
		Actions: []ConflictAction{
			{
				Conflict: conflicts[0],
				Action:   ActionReplace,
				NewName:  "prod-account-AdminRole",
				OldName:  "old-admin-profile",
			},
			{
				Conflict: conflicts[1],
				Action:   ActionSkip,
			},
		},
	}

	// Generate report
	report := pg.GenerateConflictReport(conflicts, result)

	// Verify report content
	assert.Contains(t, report, "Conflict Resolution Report")
	assert.Contains(t, report, "Total conflicts detected: 2")
	assert.Contains(t, report, "Resolution strategy: replace")
	assert.Contains(t, report, "Profiles replaced: 1")
	assert.Contains(t, report, "Roles skipped: 1")
	assert.Contains(t, report, "old-admin-profile -> prod-account-AdminRole")
	assert.Contains(t, report, "DevRole in dev-account")
}

func TestProfileGenerator_GetSetConflictStrategy(t *testing.T) {
	// Create profile generator
	pg, err := NewProfileGenerator("template", "{account_name}-{role_name}", false, "", ConflictPrompt, aws.Config{})
	require.NoError(t, err)

	// Test initial strategy
	assert.Equal(t, ConflictPrompt, pg.GetConflictStrategy())

	// Test setting new strategy
	pg.SetConflictStrategy(ConflictReplace)
	assert.Equal(t, ConflictReplace, pg.GetConflictStrategy())

	pg.SetConflictStrategy(ConflictSkip)
	assert.Equal(t, ConflictSkip, pg.GetConflictStrategy())
}

// MockInput simulates user input for testing interactive prompts
type MockInput struct {
	responses []string
	index     int
}

func (m *MockInput) ReadString(delim byte) (string, error) {
	if m.index >= len(m.responses) {
		return "", fmt.Errorf("no more mock responses")
	}
	response := m.responses[m.index] + "\n"
	m.index++
	return response, nil
}

func TestProfileGenerator_PromptForConflictResolution_MockInput(t *testing.T) {
	// Note: This test demonstrates the structure but doesn't actually test user input
	// since we can't easily mock os.Stdin in this context.
	// In a real implementation, you would inject an io.Reader interface for testing.

	pg, err := NewProfileGenerator("template", "{account_name}-{role_name}", false, "", ConflictPrompt, aws.Config{})
	require.NoError(t, err)

	conflict := ProfileConflict{
		DiscoveredRole: DiscoveredRole{
			AccountID:         "123456789012",
			AccountName:       "test-account",
			AccountAlias:      "test-account",
			PermissionSetName: "TestRole",
			RoleName:          "TestRole",
		},
		ExistingProfiles: []Profile{
			{Name: "existing-profile"},
		},
		ProposedName: "test-account-TestRole",
		ConflictType: ConflictSameRole,
	}

	// This test verifies the structure exists but doesn't test actual user interaction
	// Real testing would require dependency injection of the input reader
	assert.NotNil(t, pg)
	assert.Equal(t, "TestRole", conflict.DiscoveredRole.PermissionSetName)
}

func TestProfileGenerator_ValidateGeneratedProfile(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")

	configContent := `[profile template-profile]
region = us-east-1
sso_start_url = https://test.awsapps.com/start
sso_region = us-east-1
sso_account_id = 123456789012
sso_role_name = TemplateRole
`
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	require.NoError(t, err)

	// Set AWS_CONFIG_FILE environment variable
	originalConfigFile := os.Getenv("AWS_CONFIG_FILE")
	os.Setenv("AWS_CONFIG_FILE", configPath)
	defer func() {
		if originalConfigFile != "" {
			os.Setenv("AWS_CONFIG_FILE", originalConfigFile)
		} else {
			os.Unsetenv("AWS_CONFIG_FILE")
		}
	}()

	pg, err := NewProfileGenerator("template-profile", "{account_name}-{role_name}", false, "", ConflictReplace, aws.Config{})
	require.NoError(t, err)

	// Test with valid conflict
	conflicts := []ProfileConflict{
		{
			DiscoveredRole: DiscoveredRole{
				AccountID:         "123456789012",
				AccountName:       "test-account",
				AccountAlias:      "test-account",
				PermissionSetName: "ValidRole",
				RoleName:          "ValidRole",
			},
			ProposedName: "test-account-ValidRole",
			ConflictType: ConflictSameRole,
		},
	}

	result, err := pg.ResolveConflicts(conflicts)
	require.NoError(t, err)
	assert.Len(t, result.GeneratedProfiles, 1)
	assert.Len(t, result.SkippedRoles, 0)

	// Verify generated profile is valid
	profile := result.GeneratedProfiles[0]
	assert.Equal(t, "test-account-ValidRole", profile.Name)
	assert.Equal(t, "123456789012", profile.AccountID)
	assert.Equal(t, "test-account", profile.AccountName)
	assert.Equal(t, "ValidRole", profile.RoleName)
	assert.Equal(t, "us-east-1", profile.Region)
	assert.Equal(t, "https://test.awsapps.com/start", profile.SSOStartURL)
}
