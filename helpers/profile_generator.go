package helpers

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// ProfileGenerator handles the complete profile generation workflow
type ProfileGenerator struct {
	templateProfile  string
	namingPattern    string
	autoApprove      bool
	outputFile       string
	conflictStrategy ConflictResolutionStrategy
	awsConfig        aws.Config
	ssoClient        *sso.Client
	stsClient        *sts.Client
	iamClient        *iam.Client
	roleDiscovery    *RoleDiscovery
	logger           Logger
}

// NewProfileGenerator creates a new profile generator
func NewProfileGenerator(templateProfile, namingPattern string, autoApprove bool, outputFile string, conflictStrategy ConflictResolutionStrategy, awsConfig aws.Config) (*ProfileGenerator, error) {
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
	iamClient := iam.NewFromConfig(awsConfig)

	// Create role discovery
	roleDiscovery, err := NewRoleDiscovery(ssoClient, stsClient, iamClient)
	if err != nil {
		return nil, err
	}

	pg := &ProfileGenerator{
		templateProfile:  templateProfile,
		namingPattern:    namingPattern,
		autoApprove:      autoApprove,
		outputFile:       outputFile,
		conflictStrategy: conflictStrategy,
		awsConfig:        awsConfig,
		ssoClient:        ssoClient,
		stsClient:        stsClient,
		iamClient:        iamClient,
		roleDiscovery:    roleDiscovery,
		logger:           &defaultLogger{},
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
	generatedProfiles := make([]GeneratedProfile, 0, len(discoveredRoles))

	for _, role := range discoveredRoles {
		// Generate profile name
		desiredName, err := namingPattern.GenerateProfileName(
			role.AccountID,
			role.AccountName,
			role.AccountAlias,
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
	// Estimate size: base summary (~300 chars) + errors + conflicts
	estimatedSize := 300 + len(result.Errors)*50 + len(result.ConflictingProfiles)*30
	summary.Grow(estimatedSize)

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

// PromptForConflictResolution prompts the user for conflict resolution action
func (pg *ProfileGenerator) PromptForConflictResolution(conflict ProfileConflict) (ConflictAction, error) {
	// Display conflict information
	pg.logger.Printf("\n=== Profile Conflict Detected ===\n")
	pg.logger.Printf("Role: %s in account %s (%s)\n",
		conflict.DiscoveredRole.PermissionSetName,
		conflict.DiscoveredRole.AccountName,
		conflict.DiscoveredRole.AccountID)
	pg.logger.Printf("Proposed profile name: %s\n", conflict.ProposedName)

	if len(conflict.ExistingProfiles) > 0 {
		pg.logger.Printf("Existing profiles for this role:\n")
		for i, profile := range conflict.ExistingProfiles {
			pg.logger.Printf("  %d. %s\n", i+1, profile.Name)
		}
	}

	// Display conflict type
	switch conflict.ConflictType {
	case ConflictSameRole:
		pg.logger.Printf("Conflict type: Same role already has existing profile(s)\n")
	case ConflictSameName:
		pg.logger.Printf("Conflict type: Proposed profile name already exists\n")
	}

	// Prompt for action
	pg.logger.Printf("\nChoose an action:\n")
	pg.logger.Printf("  r) Replace existing profile(s) with new name\n")
	pg.logger.Printf("  s) Skip this role (keep existing profile)\n")
	pg.logger.Printf("  c) Cancel operation (exit without changes)\n")
	pg.logger.Printf("Enter choice (r/s/c): ")

	// Read user input
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return ConflictAction{}, NewValidationError("failed to read user input", err)
	}

	// Parse input
	choice := strings.ToLower(strings.TrimSpace(input))

	action := ConflictAction{
		Conflict: conflict,
	}

	switch choice {
	case "r", "replace":
		action.Action = ActionReplace
		action.NewName = conflict.ProposedName
		if len(conflict.ExistingProfiles) > 0 {
			action.OldName = conflict.ExistingProfiles[0].Name
		}
		pg.logger.Printf("Selected: Replace existing profile(s)\n")
	case "s", "skip":
		action.Action = ActionSkip
		pg.logger.Printf("Selected: Skip this role\n")
	case "c", "cancel":
		return ConflictAction{}, NewValidationError("operation cancelled by user", nil)
	default:
		return ConflictAction{}, NewValidationError("invalid choice", nil).
			WithContext("input", choice).
			WithContext("valid_choices", "r, s, c")
	}

	return action, nil
}

// DetectProfileConflicts detects conflicts between discovered roles and existing profiles
func (pg *ProfileGenerator) DetectProfileConflicts(discoveredRoles []DiscoveredRole) ([]ProfileConflict, error) {
	// Load existing AWS config
	configFile, err := LoadAWSConfigFile("")
	if err != nil {
		return nil, NewFileSystemError("failed to load AWS config file", err)
	}

	// Create naming pattern
	namingPattern, err := NewNamingPattern(pg.namingPattern)
	if err != nil {
		return nil, err
	}

	// Create conflict detector
	conflictDetector := NewProfileConflictDetector(configFile, namingPattern)

	// Detect conflicts
	conflicts, err := conflictDetector.DetectConflicts(discoveredRoles)
	if err != nil {
		return nil, err
	}

	return conflicts, nil
}

// ResolveConflicts resolves conflicts based on the configured strategy
func (pg *ProfileGenerator) ResolveConflicts(conflicts []ProfileConflict) ([]GeneratedProfile, []DiscoveredRole, error) {
	var generatedProfiles []GeneratedProfile
	var skippedRoles []DiscoveredRole
	var actions []ConflictAction

	// Load template profile for profile generation
	templateProfile, err := pg.ValidateTemplateProfile()
	if err != nil {
		return nil, nil, err
	}

	for _, conflict := range conflicts {
		var action ConflictAction
		var err error

		// Determine action based on strategy
		switch pg.conflictStrategy {
		case ConflictReplace:
			action = ConflictAction{
				Conflict: conflict,
				Action:   ActionReplace,
				NewName:  conflict.ProposedName,
			}
			if len(conflict.ExistingProfiles) > 0 {
				action.OldName = conflict.ExistingProfiles[0].Name
			}
		case ConflictSkip:
			action = ConflictAction{
				Conflict: conflict,
				Action:   ActionSkip,
			}
		case ConflictPrompt:
			action, err = pg.PromptForConflictResolution(conflict)
			if err != nil {
				return nil, nil, err
			}
		}

		actions = append(actions, action)

		// Process action
		switch action.Action {
		case ActionReplace:
			// Create generated profile
			generatedProfile := GeneratedProfile{
				Name:         action.NewName,
				AccountID:    conflict.DiscoveredRole.AccountID,
				AccountName:  conflict.DiscoveredRole.AccountName,
				RoleName:     conflict.DiscoveredRole.PermissionSetName,
				Region:       templateProfile.Region,
				SSOStartURL:  templateProfile.SSOStartURL,
				SSORegion:    templateProfile.SSORegion,
				SSOSession:   templateProfile.SSOSession,
				SSOAccountID: conflict.DiscoveredRole.AccountID,
				SSORoleName:  conflict.DiscoveredRole.PermissionSetName,
				IsLegacy:     templateProfile.IsLegacyFormat(),
			}

			if err := generatedProfile.Validate(); err != nil {
				return nil, nil, NewValidationError("invalid generated profile", err).
					WithContext("profile_name", action.NewName)
			}

			generatedProfiles = append(generatedProfiles, generatedProfile)

		case ActionSkip:
			skippedRoles = append(skippedRoles, conflict.DiscoveredRole)
		}
	}

	return generatedProfiles, skippedRoles, nil
}

// GenerateConflictReport creates a detailed report of conflict resolution actions
func (pg *ProfileGenerator) GenerateConflictReport(conflicts []ProfileConflict, actions []ConflictAction) string {
	var report strings.Builder

	report.WriteString("Conflict Resolution Report\n")
	report.WriteString("==========================\n")
	report.WriteString(fmt.Sprintf("Total conflicts detected: %d\n", len(conflicts)))
	report.WriteString(fmt.Sprintf("Resolution strategy: %s\n\n", pg.conflictStrategy.String()))

	if len(actions) == 0 {
		report.WriteString("No actions taken.\n")
		return report.String()
	}

	// Group actions by type
	replaceCount := 0
	skipCount := 0
	createCount := 0

	for _, action := range actions {
		switch action.Action {
		case ActionReplace:
			replaceCount++
		case ActionSkip:
			skipCount++
		case ActionCreate:
			createCount++
		}
	}

	report.WriteString("Action Summary:\n")
	report.WriteString(fmt.Sprintf("  Profiles replaced: %d\n", replaceCount))
	report.WriteString(fmt.Sprintf("  Roles skipped: %d\n", skipCount))
	report.WriteString(fmt.Sprintf("  New profiles created: %d\n", createCount))
	report.WriteString("\n")

	// Detailed actions
	if replaceCount > 0 {
		report.WriteString("Replaced Profiles:\n")
		for _, action := range actions {
			if action.Action == ActionReplace {
				report.WriteString(fmt.Sprintf("  %s -> %s (Role: %s)\n",
					action.OldName, action.NewName,
					action.Conflict.DiscoveredRole.PermissionSetName))
			}
		}
		report.WriteString("\n")
	}

	if skipCount > 0 {
		report.WriteString("Skipped Roles:\n")
		for _, action := range actions {
			if action.Action == ActionSkip {
				report.WriteString(fmt.Sprintf("  %s in %s (existing profiles preserved)\n",
					action.Conflict.DiscoveredRole.PermissionSetName,
					action.Conflict.DiscoveredRole.AccountName))
			}
		}
		report.WriteString("\n")
	}

	return report.String()
}

// GetConflictStrategy returns the current conflict resolution strategy
func (pg *ProfileGenerator) GetConflictStrategy() ConflictResolutionStrategy {
	return pg.conflictStrategy
}

// SetConflictStrategy sets the conflict resolution strategy
func (pg *ProfileGenerator) SetConflictStrategy(strategy ConflictResolutionStrategy) {
	pg.conflictStrategy = strategy
}
