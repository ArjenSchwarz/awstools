package helpers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// ProfileGenerator handles the complete profile generation workflow
type ProfileGenerator struct {
	templateProfile string
	namingPattern   string
	autoApprove     bool
	outputFile      string
	awsConfig       aws.Config
	ssoClient       *sso.Client
	stsClient       *sts.Client
	roleDiscovery   *RoleDiscovery
	logger          Logger
}

// NewProfileGenerator creates a new profile generator
func NewProfileGenerator(templateProfile, namingPattern string, autoApprove bool, outputFile string, awsConfig aws.Config) (*ProfileGenerator, error) {
	if templateProfile == "" {
		return nil, NewValidationError("template profile name is required", nil)
	}

	if namingPattern == "" {
		namingPattern = "{account_name}-{role_name}" // Default pattern
	}

	// Validate naming pattern
	if _, err := NewNamingPattern(namingPattern); err != nil {
		return nil, NewValidationError("invalid naming pattern", err).
			WithContext("pattern", namingPattern)
	}

	// Create AWS service clients
	ssoClient := sso.NewFromConfig(awsConfig)
	stsClient := sts.NewFromConfig(awsConfig)

	// Create role discovery
	roleDiscovery, err := NewRoleDiscovery(ssoClient, stsClient)
	if err != nil {
		return nil, err
	}

	pg := &ProfileGenerator{
		templateProfile: templateProfile,
		namingPattern:   namingPattern,
		autoApprove:     autoApprove,
		outputFile:      outputFile,
		awsConfig:       awsConfig,
		ssoClient:       ssoClient,
		stsClient:       stsClient,
		roleDiscovery:   roleDiscovery,
		logger:          &defaultLogger{},
	}

	return pg, nil
}

// SetLogger sets a custom logger
func (pg *ProfileGenerator) SetLogger(logger Logger) {
	pg.logger = logger
	pg.roleDiscovery.SetLogger(logger)
}

// ValidateTemplateProfile validates the template profile configuration
func (pg *ProfileGenerator) ValidateTemplateProfile() (*TemplateProfile, error) {
	// Load AWS config file
	configFile, err := LoadAWSConfigFile("")
	if err != nil {
		return nil, NewFileSystemError("failed to load AWS config file", err)
	}

	// Get template profile
	profile, exists := configFile.GetProfile(pg.templateProfile)
	if !exists {
		return nil, NewValidationError("template profile not found", nil).
			WithContext("profile_name", pg.templateProfile).
			WithContext("available_profiles", configFile.GetProfileNames())
	}

	// Convert to TemplateProfile
	templateProfile := &TemplateProfile{
		Name:         profile.Name,
		Region:       profile.Region,
		SSOStartURL:  profile.SSOStartURL,
		SSORegion:    profile.SSORegion,
		SSOAccountID: profile.SSOAccountID,
		SSORoleName:  profile.SSORoleName,
		SSOSession:   profile.SSOSession,
		IsSSO:        profile.IsSSO(),
	}

	// Validate template profile
	if err := templateProfile.Validate(); err != nil {
		return nil, err
	}

	return templateProfile, nil
}

// DiscoverRoles discovers all accessible roles using the template profile
func (pg *ProfileGenerator) DiscoverRoles(templateProfile *TemplateProfile) ([]DiscoveredRole, error) {
	// Validate token access before discovery
	if err := pg.roleDiscovery.ValidateTokenAccess(templateProfile); err != nil {
		return nil, err
	}

	// Discover roles with retry
	roles, err := pg.roleDiscovery.DiscoverRolesWithRetry(templateProfile, 3)
	if err != nil {
		return nil, err
	}

	if len(roles) == 0 {
		return nil, NewAPIError("no accessible roles found", nil)
	}

	return roles, nil
}

// GenerateProfiles generates profiles from discovered roles
func (pg *ProfileGenerator) GenerateProfiles(templateProfile *TemplateProfile, discoveredRoles []DiscoveredRole) ([]GeneratedProfile, error) {
	if len(discoveredRoles) == 0 {
		return nil, NewValidationError("no roles to generate profiles for", nil)
	}

	namingPattern, err := NewNamingPattern(pg.namingPattern)
	if err != nil {
		return nil, err
	}

	// Load existing profiles to detect conflicts
	configFile, err := LoadAWSConfigFile("")
	if err != nil {
		return nil, NewFileSystemError("failed to load AWS config file", err)
	}

	conflictResolver := NewProfileNameConflictResolver(configFile.GetProfileNames())
	var generatedProfiles []GeneratedProfile

	for _, role := range discoveredRoles {
		// Generate profile name
		desiredName, err := namingPattern.GenerateProfileName(
			role.AccountID,
			role.AccountName,
			role.PermissionSetName,
			templateProfile.Region,
		)
		if err != nil {
			return nil, NewValidationError("failed to generate profile name", err).
				WithContext("account_id", role.AccountID).
				WithContext("role_name", role.PermissionSetName)
		}

		// Resolve naming conflicts
		actualName := conflictResolver.ResolveConflict(desiredName)

		// Create generated profile
		generatedProfile := GeneratedProfile{
			Name:         actualName,
			AccountID:    role.AccountID,
			AccountName:  role.AccountName,
			RoleName:     role.PermissionSetName,
			Region:       templateProfile.Region,
			SSOStartURL:  templateProfile.SSOStartURL,
			SSORegion:    templateProfile.SSORegion,
			SSOSession:   templateProfile.SSOSession,
			SSOAccountID: role.AccountID,
			SSORoleName:  role.PermissionSetName,
			IsLegacy:     templateProfile.IsLegacyFormat(),
		}

		// Validate generated profile
		if err := generatedProfile.Validate(); err != nil {
			return nil, NewValidationError("invalid generated profile", err).
				WithContext("profile_name", actualName)
		}

		generatedProfiles = append(generatedProfiles, generatedProfile)
	}

	return generatedProfiles, nil
}

// PreviewProfiles displays profiles for user review
func (pg *ProfileGenerator) PreviewProfiles(profiles []GeneratedProfile) error {
	if len(profiles) == 0 {
		pg.logger.Printf("No profiles to preview\n")
		return nil
	}

	pg.logger.Printf("Generated Profiles Preview:\n")
	pg.logger.Printf("===========================\n")

	for _, profile := range profiles {
		pg.logger.Printf("[profile %s]\n", profile.Name)
		pg.logger.Printf("region = %s\n", profile.Region)
		pg.logger.Printf("sso_start_url = %s\n", profile.SSOStartURL)
		pg.logger.Printf("sso_region = %s\n", profile.SSORegion)

		if profile.IsLegacy {
			pg.logger.Printf("sso_account_id = %s\n", profile.SSOAccountID)
			pg.logger.Printf("sso_role_name = %s\n", profile.SSORoleName)
		} else {
			pg.logger.Printf("sso_session = %s\n", profile.SSOSession)
		}

		pg.logger.Printf("\n") // Empty line between profiles
	}

	return nil
}

// AppendToConfig appends profiles to the AWS config file
func (pg *ProfileGenerator) AppendToConfig(profiles []GeneratedProfile) error {
	if len(profiles) == 0 {
		return NewValidationError("no profiles to append", nil)
	}

	var configFile *AWSConfigFile
	var err error

	// Determine output file path
	outputPath := pg.outputFile
	if outputPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return NewFileSystemError("failed to get user home directory", err)
		}
		outputPath = filepath.Join(homeDir, ".aws", "config")
	}

	// Load existing config or create new one
	configFile, err = LoadAWSConfigFile(outputPath)
	if err != nil {
		return err
	}

	// Detect conflicts
	conflicts := configFile.DetectProfileConflicts(profiles)
	if len(conflicts) > 0 && !pg.autoApprove {
		return NewValidationError("profile conflicts detected", nil).
			WithContext("conflicts", conflicts).
			WithContext("suggestion", "Use --yes flag to auto-approve or rename conflicting profiles")
	}

	// Append profiles
	if err := configFile.AppendProfiles(profiles); err != nil {
		return err
	}

	if len(conflicts) > 0 {
		pg.logger.Printf("Warning: %d existing profiles were overwritten: %v", len(conflicts), conflicts)
	}

	return nil
}

// GenerateProfilesWorkflow executes the complete profile generation workflow
func (pg *ProfileGenerator) GenerateProfilesWorkflow() (*ProfileGenerationResult, error) {
	result := &ProfileGenerationResult{
		ConflictingProfiles: []string{},
		SuccessfulProfiles:  []string{},
		Errors:              []ProfileGeneratorError{},
	}

	// Validate template profile
	templateProfile, err := pg.ValidateTemplateProfile()
	if err != nil {
		result.AddError(err.(ProfileGeneratorError))
		return result, err
	}
	result.TemplateProfile = *templateProfile

	// Discover roles
	discoveredRoles, err := pg.DiscoverRoles(templateProfile)
	if err != nil {
		result.AddError(err.(ProfileGeneratorError))
		return result, err
	}
	result.DiscoveredRoles = discoveredRoles

	// Generate profiles
	generatedProfiles, err := pg.GenerateProfiles(templateProfile, discoveredRoles)
	if err != nil {
		result.AddError(err.(ProfileGeneratorError))
		return result, err
	}
	result.GeneratedProfiles = generatedProfiles

	// Preview profiles
	if err := pg.PreviewProfiles(generatedProfiles); err != nil {
		result.AddError(err.(ProfileGeneratorError))
		return result, err
	}

	// Append to config (if approved)
	if pg.autoApprove {
		if err := pg.AppendToConfig(generatedProfiles); err != nil {
			result.AddError(err.(ProfileGeneratorError))
			return result, err
		}

		// Track successful profiles
		for _, profile := range generatedProfiles {
			result.SuccessfulProfiles = append(result.SuccessfulProfiles, profile.Name)
		}
	}

	return result, nil
}

// GetProfileGenerationSummary returns a summary of the profile generation
func (pg *ProfileGenerator) GetProfileGenerationSummary(result *ProfileGenerationResult) string {
	var summary strings.Builder

	summary.WriteString("Profile Generation Summary\n")
	summary.WriteString("=========================\n")
	summary.WriteString(fmt.Sprintf("Template Profile: %s\n", result.TemplateProfile.Name))
	summary.WriteString(fmt.Sprintf("Naming Pattern: %s\n", pg.namingPattern))
	summary.WriteString(fmt.Sprintf("Discovered Roles: %d\n", len(result.DiscoveredRoles)))
	summary.WriteString(fmt.Sprintf("Generated Profiles: %d\n", len(result.GeneratedProfiles)))
	summary.WriteString(fmt.Sprintf("Successful Profiles: %d\n", len(result.SuccessfulProfiles)))
	summary.WriteString(fmt.Sprintf("Conflicting Profiles: %d\n", len(result.ConflictingProfiles)))
	summary.WriteString(fmt.Sprintf("Errors: %d\n", len(result.Errors)))

	if len(result.Errors) > 0 {
		summary.WriteString("\nErrors:\n")
		for i, err := range result.Errors {
			summary.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
		}
	}

	if len(result.ConflictingProfiles) > 0 {
		summary.WriteString("\nConflicting Profiles:\n")
		for i, profile := range result.ConflictingProfiles {
			summary.WriteString(fmt.Sprintf("  %d. %s\n", i+1, profile))
		}
	}

	return summary.String()
}

// ValidateConfiguration validates the generator configuration
func (pg *ProfileGenerator) ValidateConfiguration() error {
	// Validate template profile name
	if pg.templateProfile == "" {
		return NewValidationError("template profile name is required", nil)
	}

	// Validate naming pattern
	if _, err := NewNamingPattern(pg.namingPattern); err != nil {
		return NewValidationError("invalid naming pattern", err).
			WithContext("pattern", pg.namingPattern)
	}

	// Validate output file path (if specified)
	if pg.outputFile != "" {
		dir := filepath.Dir(pg.outputFile)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return NewFileSystemError("output directory does not exist", err).
				WithContext("directory", dir)
		}
	}

	return nil
}

// GetTemplateProfile returns the template profile name
func (pg *ProfileGenerator) GetTemplateProfile() string {
	return pg.templateProfile
}

// GetNamingPattern returns the naming pattern
func (pg *ProfileGenerator) GetNamingPattern() string {
	return pg.namingPattern
}

// IsAutoApprove returns whether auto-approval is enabled
func (pg *ProfileGenerator) IsAutoApprove() bool {
	return pg.autoApprove
}

// GetOutputFile returns the output file path
func (pg *ProfileGenerator) GetOutputFile() string {
	return pg.outputFile
}
