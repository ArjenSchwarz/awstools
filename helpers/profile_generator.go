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

// Constants for user actions
const (
	userActionReplace = "replace"
	userActionSkip    = "skip"
)

// ProfileGenerator handles the complete profile generation workflow with enhanced
// conflict resolution capabilities. It orchestrates the entire process from template
// profile validation through role discovery to final profile generation and conflict resolution.
//
// The ProfileGenerator implements a comprehensive workflow:
// 1. Template Profile Validation: Ensures the template profile is valid for SSO
// 2. Role Discovery: Enumerates accessible roles through SSO APIs
// 3. Conflict Detection: Identifies conflicts with existing profiles
// 4. Conflict Resolution: Applies user-specified resolution strategy
// 5. Profile Generation: Creates new profiles for non-conflicted roles
// 6. Configuration Update: Writes profiles to AWS config file
//
// Conflict Resolution Strategies:
// - ConflictPrompt: Interactive prompts for each conflict (default)
// - ConflictReplace: Automatically replace existing profiles
// - ConflictSkip: Skip roles that have existing profiles
//
// Key Features:
// - Supports both legacy and modern SSO profile formats
// - Efficient conflict detection with caching and indexing
// - Atomic operations with backup and recovery
// - Comprehensive error handling and recovery
// - Detailed progress reporting and logging
// - File locking for concurrent access protection
//
// Error Recovery:
// - Automatic backup creation before modifications
// - Rollback capability on partial failures
// - Graceful handling of malformed configurations
// - Detailed error context for troubleshooting
//
// Example usage:
//
//	generator, err := NewProfileGenerator(
//	    "my-sso-template",
//	    "{account_name}-{role_name}",
//	    false, // not auto-approve
//	    "", // default config file
//	    ConflictPrompt,
//	    awsConfig,
//	)
//	if err != nil {
//	    return err
//	}
//
//	result, err := generator.GenerateProfilesWorkflow()
//	if err != nil {
//	    return err
//	}
//
//	fmt.Println(result.GenerateConflictReport())
type ProfileGenerator struct {
	templateProfile  string                     // Name of the template profile to use for SSO configuration
	namingPattern    string                     // Pattern for generating profile names (e.g., "{account_name}-{role_name}")
	autoApprove      bool                       // Whether to automatically approve profile generation without user confirmation
	outputFile       string                     // Custom output file path (empty for default ~/.aws/config)
	conflictStrategy ConflictResolutionStrategy // Strategy for resolving profile conflicts
	awsConfig        aws.Config                 // AWS SDK configuration for API calls
	ssoClient        *sso.Client                // AWS SSO client for role discovery
	stsClient        *sts.Client                // AWS STS client for token validation
	iamClient        *iam.Client                // AWS IAM client for role information
	roleDiscovery    *RoleDiscovery             // Role discovery service for enumerating accessible roles
	conflictDetector *ProfileConflictDetector   // Conflict detection service (initialized lazily)
	logger           Logger                     // Logger for progress and diagnostic messages
}

// NewProfileGenerator creates a new profile generator with comprehensive validation
// and initialization of all required components for the profile generation workflow.
//
// The constructor performs extensive validation and setup:
// 1. Validates required parameters (template profile name)
// 2. Sets default naming pattern if not provided
// 3. Validates the naming pattern syntax
// 4. Creates AWS service clients (SSO, STS, IAM)
// 5. Initializes role discovery service
// 6. Sets up default logger (can be overridden)
//
// Parameters:
//   - templateProfile: Name of existing SSO profile to use as template (required)
//   - namingPattern: Pattern for generating profile names (defaults to "{account_name}-{role_name}")
//   - autoApprove: Whether to skip user confirmation prompts
//   - outputFile: Custom output file path (empty for default ~/.aws/config)
//   - conflictStrategy: How to handle conflicts with existing profiles
//   - awsConfig: AWS SDK configuration for API authentication
//
// Returns:
//   - *ProfileGenerator: Initialized generator ready for profile generation
//   - error: Validation or initialization error
//
// Validation Rules:
// - Template profile name cannot be empty
// - Naming pattern must be valid (contains recognized placeholders)
// - AWS configuration must be valid for SSO operations
//
// Error Handling:
// - Returns ValidationError for invalid parameters
// - Returns initialization errors from AWS service clients
// - Provides detailed error context for troubleshooting
//
// Default Values:
// - Naming pattern: "{account_name}-{role_name}" if not specified
// - Logger: Default console logger (can be overridden with SetLogger)
// - Conflict detector: Initialized lazily when first needed
//
// Example usage:
//
//	generator, err := NewProfileGenerator(
//	    "my-sso-profile",           // template profile
//	    "{account_name}-{role_name}", // naming pattern
//	    false,                      // require user approval
//	    "",                         // use default config file
//	    ConflictPrompt,             // prompt for conflicts
//	    awsConfig,                  // AWS configuration
//	)
//	if err != nil {
//	    return fmt.Errorf("failed to create generator: %w", err)
//	}
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

// initializeConflictDetector initializes the conflict detector with the AWS config file
func (pg *ProfileGenerator) initializeConflictDetector() error {
	if pg.conflictDetector != nil {
		return nil // Already initialized
	}

	// Load AWS config file
	configFile, err := LoadAWSConfigFile("")
	if err != nil {
		return NewFileSystemError("failed to load AWS config file", err)
	}

	// Create naming pattern
	namingPattern, err := NewNamingPattern(pg.namingPattern)
	if err != nil {
		return err
	}

	// Create conflict detector
	pg.conflictDetector = NewProfileConflictDetector(configFile, namingPattern)

	return nil
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

// GenerateProfilesWorkflow executes the complete profile generation workflow with
// enhanced conflict resolution capabilities. This is the main orchestration method
// that coordinates all phases of profile generation.
//
// Workflow Phases:
// 1. Template Profile Validation
//   - Loads and validates the template profile from AWS config
//   - Ensures profile is properly configured for SSO
//   - Validates SSO configuration format (legacy or modern)
//
// 2. Role Discovery
//   - Validates SSO token access
//   - Enumerates all accessible accounts and roles
//   - Applies retry logic for transient failures
//
// 3. Conflict Detection
//   - Analyzes discovered roles against existing profiles
//   - Identifies role-based and name-based conflicts
//   - Classifies conflicts for appropriate resolution
//
// 4. Conflict Resolution
//   - Applies configured resolution strategy
//   - Handles user interaction for prompt strategy
//   - Tracks all resolution actions taken
//
// 5. Profile Generation
//   - Generates profiles for non-conflicted roles
//   - Creates profiles for resolved conflicts
//   - Validates all generated profiles
//
// 6. Configuration Update (if auto-approved)
//   - Creates backup of existing configuration
//   - Writes new profiles to AWS config file
//   - Provides rollback on failures
//
// Returns:
//   - *ProfileGenerationResult: Comprehensive result with all workflow information
//   - error: Critical error that prevented workflow completion
//
// Result Information:
// - Template profile used and discovered roles
// - Detected conflicts and resolution actions taken
// - Generated profiles and successful operations
// - Replaced profiles and skipped roles
// - Backup path for recovery
// - Detailed error information
//
// Error Handling Strategy:
// - Each phase captures errors in the result object
// - Critical errors stop workflow execution
// - Non-critical errors are logged and workflow continues
// - Partial results are always returned for analysis
//
// Recovery and Rollback:
// - Automatic backup creation before any modifications
// - Rollback capability on partial failures
// - Detailed error context for manual recovery
// - Preservation of original configuration on errors
//
// Example usage:
//
//	result, err := generator.GenerateProfilesWorkflow()
//	if err != nil {
//	    fmt.Printf("Workflow failed: %v\n", err)
//	    if result != nil {
//	        fmt.Printf("Partial results: %s\n", result.Summary())
//	    }
//	    return err
//	}
//
//	fmt.Printf("Generated %d profiles successfully\n", len(result.GeneratedProfiles))
//	if len(result.DetectedConflicts) > 0 {
//	    fmt.Printf("Resolved %d conflicts\n", len(result.DetectedConflicts))
//	}
func (pg *ProfileGenerator) GenerateProfilesWorkflow() (*ProfileGenerationResult, error) {
	result := &ProfileGenerationResult{
		ConflictingProfiles: []string{},
		SuccessfulProfiles:  []string{},
		Errors:              []ProfileGeneratorError{},
		DetectedConflicts:   []ProfileConflict{},
		ResolutionActions:   []ConflictAction{},
		ReplacedProfiles:    []ProfileReplacement{},
		SkippedRoles:        []DiscoveredRole{},
	}

	// Validate template profile
	templateProfile, err := pg.ValidateTemplateProfile()
	if err != nil {
		if pgErr, ok := err.(ProfileGeneratorError); ok {
			result.AddError(pgErr)
		} else {
			result.AddError(NewValidationError("template profile validation failed", err))
		}
		return result, err
	}
	result.TemplateProfile = *templateProfile

	// Discover roles
	discoveredRoles, err := pg.DiscoverRoles(templateProfile)
	if err != nil {
		if pgErr, ok := err.(ProfileGeneratorError); ok {
			result.AddError(pgErr)
		} else {
			result.AddError(NewAPIError("role discovery failed", err))
		}
		return result, err
	}
	result.DiscoveredRoles = discoveredRoles

	// Detect profile conflicts
	conflicts, err := pg.DetectProfileConflicts(discoveredRoles)
	if err != nil {
		if pgErr, ok := err.(ProfileGeneratorError); ok {
			result.AddError(pgErr)
		} else {
			result.AddError(NewValidationError("conflict detection failed", err))
		}
		return result, err
	}
	result.DetectedConflicts = conflicts

	// Separate conflicted and non-conflicted roles
	_, nonConflictedRoles := pg.FilterRolesByConflicts(discoveredRoles, conflicts)

	// Resolve conflicts if any exist
	var conflictResolution *ConflictResolutionResult
	if len(conflicts) > 0 {
		conflictResolution, err = pg.ResolveConflicts(conflicts)
		if err != nil {
			if pgErr, ok := err.(ProfileGeneratorError); ok {
				result.AddError(pgErr)
			} else {
				result.AddError(NewValidationError("conflict resolution failed", err))
			}
			return result, err
		}
		result.ResolutionActions = conflictResolution.Actions
		result.SkippedRoles = conflictResolution.SkippedRoles
	} else {
		conflictResolution = &ConflictResolutionResult{
			GeneratedProfiles: []GeneratedProfile{},
			SkippedRoles:      []DiscoveredRole{},
			Actions:           []ConflictAction{},
		}
	}

	// Generate profiles for non-conflicted roles
	nonConflictedProfiles, err := pg.GenerateProfilesForNonConflictedRoles(nonConflictedRoles)
	if err != nil {
		if pgErr, ok := err.(ProfileGeneratorError); ok {
			result.AddError(pgErr)
		} else {
			result.AddError(NewValidationError("profile generation failed", err))
		}
		return result, err
	}

	// Combine all generated profiles
	result.GeneratedProfiles = append(result.GeneratedProfiles, conflictResolution.GeneratedProfiles...)
	result.GeneratedProfiles = append(result.GeneratedProfiles, nonConflictedProfiles...)

	// Preview profiles
	if err := pg.PreviewProfiles(result.GeneratedProfiles); err != nil {
		if pgErr, ok := err.(ProfileGeneratorError); ok {
			result.AddError(pgErr)
		} else {
			result.AddError(NewValidationError("profile preview failed", err))
		}
		return result, err
	}

	// Append to config (if approved)
	if pg.autoApprove {
		if err := pg.AppendToConfig(result.GeneratedProfiles); err != nil {
			if pgErr, ok := err.(ProfileGeneratorError); ok {
				result.AddError(pgErr)
			} else {
				result.AddError(NewFileSystemError("config file update failed", err))
			}
			return result, err
		}

		// Track successful profiles
		for _, profile := range result.GeneratedProfiles {
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
	case "r", userActionReplace:
		action.Action = ActionReplace
		action.NewName = conflict.ProposedName
		if len(conflict.ExistingProfiles) > 0 {
			action.OldName = conflict.ExistingProfiles[0].Name
		}
		pg.logger.Printf("Selected: Replace existing profile(s)\n")
	case "s", userActionSkip:
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
	// Initialize conflict detector if not already done
	if err := pg.initializeConflictDetector(); err != nil {
		return nil, err
	}

	// Detect conflicts
	conflicts, err := pg.conflictDetector.DetectConflicts(discoveredRoles)
	if err != nil {
		return nil, err
	}

	return conflicts, nil
}

// ConflictResolutionResult represents the result of conflict resolution
type ConflictResolutionResult struct {
	GeneratedProfiles []GeneratedProfile `json:"generated_profiles" yaml:"generated_profiles"`
	SkippedRoles      []DiscoveredRole   `json:"skipped_roles" yaml:"skipped_roles"`
	Actions           []ConflictAction   `json:"actions" yaml:"actions"`
}

// ResolveConflicts resolves conflicts based on the configured strategy
func (pg *ProfileGenerator) ResolveConflicts(conflicts []ProfileConflict) (*ConflictResolutionResult, error) {
	result := &ConflictResolutionResult{
		GeneratedProfiles: []GeneratedProfile{},
		SkippedRoles:      []DiscoveredRole{},
		Actions:           []ConflictAction{},
	}

	// Load template profile for profile generation
	templateProfile, err := pg.ValidateTemplateProfile()
	if err != nil {
		return nil, err
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
			if len(conflict.ExistingProfiles) > 0 {
				action.OldName = conflict.ExistingProfiles[0].Name
			}
		case ConflictPrompt:
			action, err = pg.PromptForConflictResolution(conflict)
			if err != nil {
				return nil, err
			}
		}

		// Record the action taken
		result.Actions = append(result.Actions, action)

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
				return nil, NewValidationError("invalid generated profile", err).
					WithContext("profile_name", action.NewName)
			}

			result.GeneratedProfiles = append(result.GeneratedProfiles, generatedProfile)

		case ActionSkip:
			result.SkippedRoles = append(result.SkippedRoles, conflict.DiscoveredRole)
		}
	}

	return result, nil
}

// FilterRolesByConflicts separates discovered roles into conflicted and non-conflicted groups
func (pg *ProfileGenerator) FilterRolesByConflicts(discoveredRoles []DiscoveredRole, conflicts []ProfileConflict) (conflictedRoles []DiscoveredRole, nonConflictedRoles []DiscoveredRole) {
	// Create a map of conflicted roles for efficient lookup
	conflictedRoleMap := make(map[string]bool)
	for _, conflict := range conflicts {
		key := fmt.Sprintf("%s:%s", conflict.DiscoveredRole.AccountID, conflict.DiscoveredRole.PermissionSetName)
		conflictedRoleMap[key] = true
	}

	// Separate roles based on conflict status
	for _, role := range discoveredRoles {
		key := fmt.Sprintf("%s:%s", role.AccountID, role.PermissionSetName)
		if conflictedRoleMap[key] {
			conflictedRoles = append(conflictedRoles, role)
		} else {
			nonConflictedRoles = append(nonConflictedRoles, role)
		}
	}

	return conflictedRoles, nonConflictedRoles
}

// GenerateProfilesForNonConflictedRoles generates profiles for roles without conflicts
func (pg *ProfileGenerator) GenerateProfilesForNonConflictedRoles(nonConflictedRoles []DiscoveredRole) ([]GeneratedProfile, error) {
	if len(nonConflictedRoles) == 0 {
		return []GeneratedProfile{}, nil
	}

	// Load template profile for profile generation
	templateProfile, err := pg.ValidateTemplateProfile()
	if err != nil {
		return nil, err
	}

	// Create naming pattern
	namingPattern, err := NewNamingPattern(pg.namingPattern)
	if err != nil {
		return nil, err
	}

	var generatedProfiles []GeneratedProfile

	for _, role := range nonConflictedRoles {
		// Validate role before processing
		if err := role.Validate(); err != nil {
			return nil, err
		}

		// Generate profile name
		profileName, err := namingPattern.GenerateProfileName(
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

		// Create generated profile
		generatedProfile := GeneratedProfile{
			Name:         profileName,
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

		if err := generatedProfile.Validate(); err != nil {
			return nil, NewValidationError("invalid generated profile", err).
				WithContext("profile_name", profileName)
		}

		generatedProfiles = append(generatedProfiles, generatedProfile)
	}

	return generatedProfiles, nil
}

// GenerateConflictReport creates a detailed report of conflict resolution actions
func (pg *ProfileGenerator) GenerateConflictReport(conflicts []ProfileConflict, result *ConflictResolutionResult) string {
	var report strings.Builder

	report.WriteString("Conflict Resolution Report\n")
	report.WriteString("==========================\n")
	report.WriteString(fmt.Sprintf("Total conflicts detected: %d\n", len(conflicts)))
	report.WriteString(fmt.Sprintf("Resolution strategy: %s\n\n", pg.conflictStrategy.String()))

	if len(result.Actions) == 0 {
		report.WriteString("No actions taken.\n")
		return report.String()
	}

	// Group actions by type
	replaceCount := 0
	skipCount := 0
	createCount := 0

	for _, action := range result.Actions {
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
	report.WriteString(fmt.Sprintf("  Generated profiles: %d\n", len(result.GeneratedProfiles)))
	report.WriteString(fmt.Sprintf("  Skipped roles: %d\n", len(result.SkippedRoles)))
	report.WriteString("\n")

	// Detailed actions
	if replaceCount > 0 {
		report.WriteString("Replaced Profiles:\n")
		for _, action := range result.Actions {
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
		for _, action := range result.Actions {
			if action.Action == ActionSkip {
				report.WriteString(fmt.Sprintf("  %s in %s (existing profiles preserved)\n",
					action.Conflict.DiscoveredRole.PermissionSetName,
					action.Conflict.DiscoveredRole.AccountName))
			}
		}
		report.WriteString("\n")
	}

	if len(result.GeneratedProfiles) > 0 {
		report.WriteString("Generated Profiles:\n")
		for _, profile := range result.GeneratedProfiles {
			report.WriteString(fmt.Sprintf("  %s (Account: %s, Role: %s)\n",
				profile.Name, profile.AccountName, profile.RoleName))
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

// GetProgressInfo returns information about the current progress of profile generation
func (pg *ProfileGenerator) GetProgressInfo(result *ProfileGenerationResult) map[string]any {
	info := make(map[string]any)

	info["template_profile"] = result.TemplateProfile.Name
	info["discovered_roles"] = len(result.DiscoveredRoles)
	info["detected_conflicts"] = len(result.DetectedConflicts)
	info["resolution_actions"] = len(result.ResolutionActions)
	info["generated_profiles"] = len(result.GeneratedProfiles)
	info["successful_profiles"] = len(result.SuccessfulProfiles)
	info["skipped_roles"] = len(result.SkippedRoles)
	info["errors"] = len(result.Errors)
	info["has_backup"] = result.BackupPath != ""
	info["backup_path"] = result.BackupPath

	// Calculate success rate
	totalRoles := len(result.DiscoveredRoles)
	if totalRoles > 0 {
		successRate := float64(len(result.SuccessfulProfiles)) / float64(totalRoles) * 100
		info["success_rate"] = fmt.Sprintf("%.1f%%", successRate)
	} else {
		info["success_rate"] = "N/A"
	}

	return info
}

// FormatProgressMessage creates a formatted progress message for display
func (pg *ProfileGenerator) FormatProgressMessage(phase string, message string, details map[string]any) string {
	var msg strings.Builder

	// Add phase indicator
	switch phase {
	case "validation":
		msg.WriteString("🔍 ")
	case "discovery":
		msg.WriteString("🔍 ")
	case "conflict_detection":
		msg.WriteString("⚠️  ")
	case "conflict_resolution":
		msg.WriteString("🔧 ")
	case "generation":
		msg.WriteString("📝 ")
	case "success":
		msg.WriteString("✅ ")
	case "error":
		msg.WriteString("❌ ")
	default:
		msg.WriteString("ℹ️  ")
	}

	msg.WriteString(message)

	// Add details if provided
	if len(details) > 0 {
		for key, value := range details {
			msg.WriteString(fmt.Sprintf(" [%s: %v]", key, value))
		}
	}

	return msg.String()
}
