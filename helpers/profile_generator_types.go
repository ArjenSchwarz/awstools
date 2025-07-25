package helpers

import (
	"fmt"
	"regexp"
	"strings"
)

// TemplateProfile represents a template profile configuration
type TemplateProfile struct {
	Name         string `json:"name" yaml:"name"`
	Region       string `json:"region" yaml:"region"`
	SSOStartURL  string `json:"sso_start_url" yaml:"sso_start_url"`
	SSORegion    string `json:"sso_region" yaml:"sso_region"`
	SSOSession   string `json:"sso_session" yaml:"sso_session"`
	SSOAccountID string `json:"sso_account_id" yaml:"sso_account_id"`
	SSORoleName  string `json:"sso_role_name" yaml:"sso_role_name"`
	IsSSO        bool   `json:"is_sso" yaml:"is_sso"`
	IsValid      bool   `json:"is_valid" yaml:"is_valid"`
}

// Validate checks if the template profile is valid for profile generation
func (tp *TemplateProfile) Validate() error {
	if tp.Name == "" {
		return NewValidationError("template profile name is required", nil)
	}

	if !tp.IsSSO {
		return NewValidationError("template profile must be an SSO profile", nil).
			WithContext("profile_name", tp.Name)
	}

	if tp.SSOStartURL == "" {
		return NewValidationError("SSO start URL is required", nil).
			WithContext("profile_name", tp.Name)
	}

	if tp.SSORegion == "" {
		return NewValidationError("SSO region is required", nil).
			WithContext("profile_name", tp.Name)
	}

	// Validate SSO session format (new format) or account ID + role name (legacy format)
	if tp.SSOSession == "" && (tp.SSOAccountID == "" || tp.SSORoleName == "") {
		return NewValidationError("SSO session or account ID and role name are required", nil).
			WithContext("profile_name", tp.Name)
	}

	tp.IsValid = true
	return nil
}

// IsLegacyFormat returns true if the profile uses the legacy SSO format
func (tp *TemplateProfile) IsLegacyFormat() bool {
	return tp.SSOSession == "" && tp.SSOAccountID != "" && tp.SSORoleName != ""
}

// GeneratedProfile represents a generated profile configuration
type GeneratedProfile struct {
	Name         string `json:"name" yaml:"name"`
	AccountID    string `json:"account_id" yaml:"account_id"`
	AccountName  string `json:"account_name" yaml:"account_name"`
	RoleName     string `json:"role_name" yaml:"role_name"`
	Region       string `json:"region" yaml:"region"`
	SSOStartURL  string `json:"sso_start_url" yaml:"sso_start_url"`
	SSORegion    string `json:"sso_region" yaml:"sso_region"`
	SSOSession   string `json:"sso_session" yaml:"sso_session"`
	SSOAccountID string `json:"sso_account_id" yaml:"sso_account_id"`
	SSORoleName  string `json:"sso_role_name" yaml:"sso_role_name"`
	IsLegacy     bool   `json:"is_legacy" yaml:"is_legacy"`
}

// Validate checks if the generated profile is valid
func (gp *GeneratedProfile) Validate() error {
	if gp.Name == "" {
		return NewValidationError("generated profile name is required", nil)
	}

	if gp.AccountID == "" {
		return NewValidationError("account ID is required", nil).
			WithContext("profile_name", gp.Name)
	}

	if gp.RoleName == "" {
		return NewValidationError("role name is required", nil).
			WithContext("profile_name", gp.Name)
	}

	if gp.SSOStartURL == "" {
		return NewValidationError("SSO start URL is required", nil).
			WithContext("profile_name", gp.Name)
	}

	if gp.SSORegion == "" {
		return NewValidationError("SSO region is required", nil).
			WithContext("profile_name", gp.Name)
	}

	// Validate format consistency
	if gp.IsLegacy && gp.SSOSession != "" {
		return NewValidationError("legacy profile cannot have SSO session", nil).
			WithContext("profile_name", gp.Name)
	}

	if !gp.IsLegacy && gp.SSOSession == "" {
		return NewValidationError("new format profile requires SSO session", nil).
			WithContext("profile_name", gp.Name)
	}

	return nil
}

// ToConfigString returns the profile configuration in AWS config file format
func (gp *GeneratedProfile) ToConfigString() string {
	var config strings.Builder
	config.WriteString(fmt.Sprintf("[profile %s]\n", gp.Name))
	config.WriteString(fmt.Sprintf("region = %s\n", gp.Region))
	config.WriteString(fmt.Sprintf("sso_start_url = %s\n", gp.SSOStartURL))
	config.WriteString(fmt.Sprintf("sso_region = %s\n", gp.SSORegion))

	if gp.IsLegacy {
		config.WriteString(fmt.Sprintf("sso_account_id = %s\n", gp.SSOAccountID))
		config.WriteString(fmt.Sprintf("sso_role_name = %s\n", gp.SSORoleName))
	} else {
		config.WriteString(fmt.Sprintf("sso_session = %s\n", gp.SSOSession))
	}

	return config.String()
}

// DiscoveredRole represents a role discovered through SSO enumeration
type DiscoveredRole struct {
	AccountID         string `json:"account_id" yaml:"account_id"`
	AccountName       string `json:"account_name" yaml:"account_name"`
	AccountAlias      string `json:"account_alias" yaml:"account_alias"`
	PermissionSetName string `json:"permission_set_name" yaml:"permission_set_name"`
	PermissionSetArn  string `json:"permission_set_arn,omitempty" yaml:"permission_set_arn,omitempty"`
	RoleName          string `json:"role_name" yaml:"role_name"`
}

// Validate checks if the discovered role is valid
func (dr *DiscoveredRole) Validate() error {
	if dr.AccountID == "" {
		return NewValidationError("account ID is required", nil)
	}

	if dr.PermissionSetName == "" {
		return NewValidationError("permission set name is required", nil).
			WithContext("account_id", dr.AccountID)
	}

	if dr.RoleName == "" {
		return NewValidationError("role name is required", nil).
			WithContext("account_id", dr.AccountID).
			WithContext("permission_set_name", dr.PermissionSetName)
	}

	// Validate account ID format (12 digits)
	if matched, _ := regexp.MatchString(`^\d{12}$`, dr.AccountID); !matched {
		return NewValidationError("invalid account ID format", nil).
			WithContext("account_id", dr.AccountID)
	}

	return nil
}

// ProfileGenerationResult represents the result of profile generation
type ProfileGenerationResult struct {
	TemplateProfile     TemplateProfile         `json:"template_profile" yaml:"template_profile"`
	DiscoveredRoles     []DiscoveredRole        `json:"discovered_roles" yaml:"discovered_roles"`
	GeneratedProfiles   []GeneratedProfile      `json:"generated_profiles" yaml:"generated_profiles"`
	ConflictingProfiles []string                `json:"conflicting_profiles" yaml:"conflicting_profiles"`
	SuccessfulProfiles  []string                `json:"successful_profiles" yaml:"successful_profiles"`
	Errors              []ProfileGeneratorError `json:"errors" yaml:"errors"`
}

// Validate checks if the profile generation result is valid
func (pgr *ProfileGenerationResult) Validate() error {
	if err := pgr.TemplateProfile.Validate(); err != nil {
		return err
	}

	for _, role := range pgr.DiscoveredRoles {
		if err := role.Validate(); err != nil {
			return err
		}
	}

	for _, profile := range pgr.GeneratedProfiles {
		if err := profile.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// HasErrors returns true if there are any errors in the result
func (pgr *ProfileGenerationResult) HasErrors() bool {
	return len(pgr.Errors) > 0
}

// AddError adds an error to the result
func (pgr *ProfileGenerationResult) AddError(err ProfileGeneratorError) {
	pgr.Errors = append(pgr.Errors, err)
}

// Summary returns a summary of the profile generation result
func (pgr *ProfileGenerationResult) Summary() string {
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Template Profile: %s\n", pgr.TemplateProfile.Name))
	summary.WriteString(fmt.Sprintf("Discovered Roles: %d\n", len(pgr.DiscoveredRoles)))
	summary.WriteString(fmt.Sprintf("Generated Profiles: %d\n", len(pgr.GeneratedProfiles)))
	summary.WriteString(fmt.Sprintf("Successful Profiles: %d\n", len(pgr.SuccessfulProfiles)))
	summary.WriteString(fmt.Sprintf("Conflicting Profiles: %d\n", len(pgr.ConflictingProfiles)))
	summary.WriteString(fmt.Sprintf("Errors: %d\n", len(pgr.Errors)))
	return summary.String()
}

// ConflictResolutionStrategy defines how to handle existing profile conflicts
type ConflictResolutionStrategy int

const (
	ConflictPrompt  ConflictResolutionStrategy = iota // Default: prompt user for each conflict
	ConflictReplace                                   // Replace existing profiles
	ConflictSkip                                      // Skip roles with existing profiles
)

// String returns the string representation of the conflict resolution strategy
func (crs ConflictResolutionStrategy) String() string {
	switch crs {
	case ConflictPrompt:
		return "prompt"
	case ConflictReplace:
		return "replace"
	case ConflictSkip:
		return "skip"
	default:
		return "unknown"
	}
}

// Validate checks if the conflict resolution strategy is valid
func (crs ConflictResolutionStrategy) Validate() error {
	switch crs {
	case ConflictPrompt, ConflictReplace, ConflictSkip:
		return nil
	default:
		return NewValidationError("invalid conflict resolution strategy", nil).
			WithContext("strategy", crs.String())
	}
}

// ConflictType represents the type of profile conflict detected
type ConflictType int

const (
	ConflictSameRole ConflictType = iota // Same SSO account ID and role name
	ConflictSameName                     // Same profile name but different role
)

// String returns the string representation of the conflict type
func (ct ConflictType) String() string {
	switch ct {
	case ConflictSameRole:
		return "same_role"
	case ConflictSameName:
		return "same_name"
	default:
		return "unknown"
	}
}

// ProfileConflict represents a detected conflict between discovered role and existing profile
type ProfileConflict struct {
	DiscoveredRole   DiscoveredRole `json:"discovered_role" yaml:"discovered_role"`
	ExistingProfiles []Profile      `json:"existing_profiles" yaml:"existing_profiles"`
	ProposedName     string         `json:"proposed_name" yaml:"proposed_name"`
	ConflictType     ConflictType   `json:"conflict_type" yaml:"conflict_type"`
}

// Validate checks if the profile conflict is valid
func (pc *ProfileConflict) Validate() error {
	if err := pc.DiscoveredRole.Validate(); err != nil {
		return err
	}

	if len(pc.ExistingProfiles) == 0 {
		return NewValidationError("profile conflict must have at least one existing profile", nil).
			WithContext("proposed_name", pc.ProposedName)
	}

	if pc.ProposedName == "" {
		return NewValidationError("proposed profile name is required", nil)
	}

	for _, profile := range pc.ExistingProfiles {
		if err := profile.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// ActionType represents the action taken for a specific conflict
type ActionType int

const (
	ActionReplace ActionType = iota // Replace existing profile
	ActionSkip                      // Skip generating profile for this role
	ActionCreate                    // Create new profile (no conflict)
)

// String returns the string representation of the action type
func (at ActionType) String() string {
	switch at {
	case ActionReplace:
		return "replace"
	case ActionSkip:
		return "skip"
	case ActionCreate:
		return "create"
	default:
		return "unknown"
	}
}

// ConflictAction represents the action taken for a specific conflict
type ConflictAction struct {
	Conflict ProfileConflict `json:"conflict" yaml:"conflict"`
	Action   ActionType      `json:"action" yaml:"action"`
	NewName  string          `json:"new_name" yaml:"new_name"`
	OldName  string          `json:"old_name" yaml:"old_name"`
}

// Validate checks if the conflict action is valid
func (ca *ConflictAction) Validate() error {
	if err := ca.Conflict.Validate(); err != nil {
		return err
	}

	switch ca.Action {
	case ActionReplace:
		if ca.NewName == "" {
			return NewValidationError("new profile name is required for replace action", nil).
				WithContext("old_name", ca.OldName)
		}
		if ca.OldName == "" {
			return NewValidationError("old profile name is required for replace action", nil).
				WithContext("new_name", ca.NewName)
		}
	case ActionSkip:
		if ca.OldName == "" {
			return NewValidationError("old profile name is required for skip action", nil)
		}
	case ActionCreate:
		if ca.NewName == "" {
			return NewValidationError("new profile name is required for create action", nil)
		}
	default:
		return NewValidationError("invalid action type", nil).
			WithContext("action", ca.Action.String())
	}

	return nil
}

// ProfileReplacement represents a profile that was replaced during conflict resolution
type ProfileReplacement struct {
	OldProfile Profile          `json:"old_profile" yaml:"old_profile"`
	NewProfile GeneratedProfile `json:"new_profile" yaml:"new_profile"`
	OldName    string           `json:"old_name" yaml:"old_name"`
	NewName    string           `json:"new_name" yaml:"new_name"`
}

// Validate checks if the profile replacement is valid
func (pr *ProfileReplacement) Validate() error {
	if err := pr.OldProfile.Validate(); err != nil {
		return err
	}

	if err := pr.NewProfile.Validate(); err != nil {
		return err
	}

	if pr.OldName == "" {
		return NewValidationError("old profile name is required", nil)
	}

	if pr.NewName == "" {
		return NewValidationError("new profile name is required", nil)
	}

	return nil
}
