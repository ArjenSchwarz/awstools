package helpers

import (
	"fmt"
	"strings"
)

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ProfileConflictDetector handles the logic for detecting profile conflicts
type ProfileConflictDetector struct {
	configFile         *AWSConfigFile
	namingPattern      *NamingPattern
	logger             Logger
	profileIndex       *ProfileLookupIndex
	resolvedSSOConfigs map[string]*ResolvedSSOConfig // Profile name -> resolved SSO config
}

// NewProfileConflictDetector creates a new profile conflict detector
func NewProfileConflictDetector(configFile *AWSConfigFile, namingPattern *NamingPattern) *ProfileConflictDetector {
	detector := &ProfileConflictDetector{
		configFile:         configFile,
		namingPattern:      namingPattern,
		logger:             &defaultLogger{},
		resolvedSSOConfigs: make(map[string]*ResolvedSSOConfig),
	}

	// Build profile lookup index for efficient O(1) lookups
	if index, err := configFile.BuildProfileLookupIndex(); err == nil {
		detector.profileIndex = index
	}

	// Pre-resolve and cache SSO configurations for all profiles
	detector.preResolveSSO()

	return detector
}

// SetLogger sets a custom logger
func (pcd *ProfileConflictDetector) SetLogger(logger Logger) {
	pcd.logger = logger
}

// preResolveSSO pre-resolves and caches SSO configurations for all profiles
func (pcd *ProfileConflictDetector) preResolveSSO() {
	for profileName, profile := range pcd.configFile.Profiles {
		if !profile.IsSSO() {
			continue
		}

		if resolvedConfig, err := pcd.configFile.ResolveProfileSSOConfig(profile); err == nil {
			pcd.resolvedSSOConfigs[profileName] = resolvedConfig
		}
	}
}

// DetectConflicts analyzes all discovered roles for conflicts
func (pcd *ProfileConflictDetector) DetectConflicts(discoveredRoles []DiscoveredRole) ([]ProfileConflict, error) {
	if len(discoveredRoles) == 0 {
		return []ProfileConflict{}, nil
	}

	// Pre-allocate slice capacity based on discovered roles count
	// Estimate that 20-30% of roles might have conflicts
	estimatedConflicts := max(len(discoveredRoles)/4, 10)
	conflicts := make([]ProfileConflict, 0, estimatedConflicts)

	for _, role := range discoveredRoles {
		conflict, err := pcd.AnalyzeRole(role)
		if err != nil {
			pcd.logger.Printf("Warning: failed to analyze role %s in account %s: %v",
				role.RoleName, role.AccountID, err)
			continue
		}

		if conflict != nil {
			conflicts = append(conflicts, *conflict)
		}
	}

	return conflicts, nil
}

// AnalyzeRole checks a single role for conflicts
func (pcd *ProfileConflictDetector) AnalyzeRole(role DiscoveredRole) (*ProfileConflict, error) {
	// Validate the discovered role
	if err := role.Validate(); err != nil {
		return nil, NewValidationError("invalid discovered role", err).
			WithContext("account_id", role.AccountID).
			WithContext("role_name", role.RoleName)
	}

	// Generate the proposed profile name using the naming pattern
	proposedName, err := pcd.namingPattern.GenerateProfileName(
		role.AccountID,
		role.AccountName,
		role.AccountAlias,
		role.RoleName,
		"", // Region will be filled from template profile during actual generation
	)
	if err != nil {
		return nil, NewValidationError("failed to generate profile name", err).
			WithContext("account_id", role.AccountID).
			WithContext("role_name", role.RoleName)
	}

	// Find existing profiles that match this role
	existingProfiles, err := pcd.findMatchingProfiles(role)
	if err != nil {
		return nil, err
	}

	// Check for name conflicts (same name but potentially different role)
	nameConflictProfiles := pcd.findNameConflictProfiles(proposedName, role)

	// Combine all conflicting profiles
	allConflictingProfiles := pcd.combineConflictingProfiles(existingProfiles, nameConflictProfiles)

	// If no conflicts found, return nil
	if len(allConflictingProfiles) == 0 {
		return nil, nil
	}

	// Classify the type of conflict
	conflictType := pcd.ClassifyConflict(allConflictingProfiles, proposedName, role)

	// Create and return the conflict
	conflict := &ProfileConflict{
		DiscoveredRole:   role,
		ExistingProfiles: allConflictingProfiles,
		ProposedName:     proposedName,
		ConflictType:     conflictType,
	}

	// Validate the conflict before returning
	if err := conflict.Validate(); err != nil {
		return nil, NewValidationError("invalid profile conflict", err).
			WithContext("proposed_name", proposedName)
	}

	return conflict, nil
}

// ClassifyConflict determines the type of conflict detected
func (pcd *ProfileConflictDetector) ClassifyConflict(existingProfiles []Profile, proposedName string, role DiscoveredRole) ConflictType {
	// Check if any existing profile matches the same role (SSO configuration)
	for _, profile := range existingProfiles {
		matches, err := pcd.configFile.MatchesRole(profile, role.AccountID, role.RoleName, "")
		if err != nil {
			pcd.logger.Printf("Warning: failed to match profile %s against role: %v", profile.Name, err)
			continue
		}

		if matches {
			return ConflictSameRole
		}
	}

	// Check if any existing profile has the same name as proposed
	for _, profile := range existingProfiles {
		if profile.Name == proposedName {
			return ConflictSameName
		}
	}

	// Default to same role conflict if we can't determine
	return ConflictSameRole
}

// GenerateConflictSummary creates a human-readable summary of all conflicts
func (pcd *ProfileConflictDetector) GenerateConflictSummary(conflicts []ProfileConflict) string {
	if len(conflicts) == 0 {
		return "No profile conflicts detected."
	}

	// Estimate buffer size based on conflict count
	// Each conflict typically generates ~200-300 characters
	estimatedSize := len(conflicts)*250 + 100 // Extra for header
	var summary strings.Builder
	summary.Grow(estimatedSize)

	summary.WriteString(fmt.Sprintf("Profile Conflicts Detected: %d\n", len(conflicts)))
	summary.WriteString("=====================================\n\n")

	for i, conflict := range conflicts {
		summary.WriteString(fmt.Sprintf("Conflict %d:\n", i+1))
		summary.WriteString(fmt.Sprintf("  Proposed Profile: %s\n", conflict.ProposedName))
		summary.WriteString(fmt.Sprintf("  Account: %s (%s)\n", conflict.DiscoveredRole.AccountName, conflict.DiscoveredRole.AccountID))
		summary.WriteString(fmt.Sprintf("  Role: %s\n", conflict.DiscoveredRole.RoleName))
		summary.WriteString(fmt.Sprintf("  Conflict Type: %s\n", conflict.ConflictType.String()))
		summary.WriteString("  Existing Profiles:\n")

		for _, existingProfile := range conflict.ExistingProfiles {
			summary.WriteString(fmt.Sprintf("    - %s", existingProfile.Name))
			if existingProfile.SSOAccountID != "" && existingProfile.SSORoleName != "" {
				summary.WriteString(fmt.Sprintf(" (Account: %s, Role: %s)", existingProfile.SSOAccountID, existingProfile.SSORoleName))
			}
			summary.WriteString("\n")
		}
		summary.WriteString("\n")
	}

	return summary.String()
}

// findMatchingProfiles finds existing profiles that match the discovered role's SSO configuration
func (pcd *ProfileConflictDetector) findMatchingProfiles(role DiscoveredRole) ([]Profile, error) {
	// Use profile index for efficient lookup if available
	if pcd.profileIndex != nil {
		// First try to find by account ID, then filter by role name
		accountProfiles := pcd.profileIndex.FindByAccount(role.AccountID)
		matchingProfiles := make([]Profile, 0, len(accountProfiles))

		for _, profile := range accountProfiles {
			// Use cached resolved SSO config
			if resolvedConfig, exists := pcd.resolvedSSOConfigs[profile.Name]; exists {
				if resolvedConfig.AccountID == role.AccountID && resolvedConfig.RoleName == role.RoleName {
					matchingProfiles = append(matchingProfiles, profile)
				}
			}
		}

		return matchingProfiles, nil
	}

	// Fallback to linear search if index is not available
	matchingProfiles := make([]Profile, 0, 2) // Most roles have 0-2 matching profiles

	for profileName, profile := range pcd.configFile.Profiles {
		if !profile.IsSSO() {
			continue
		}

		// Use cached resolved SSO config
		if resolvedConfig, exists := pcd.resolvedSSOConfigs[profileName]; exists {
			if resolvedConfig.AccountID == role.AccountID && resolvedConfig.RoleName == role.RoleName {
				matchingProfiles = append(matchingProfiles, profile)
			}
		}
	}

	return matchingProfiles, nil
}

// findNameConflictProfiles finds existing profiles that have the same name as the proposed profile
func (pcd *ProfileConflictDetector) findNameConflictProfiles(proposedName string, role DiscoveredRole) []Profile {
	// Use profile index for efficient lookup if available
	if pcd.profileIndex != nil && pcd.profileIndex.HasName(proposedName) {
		profile := pcd.profileIndex.ByName[proposedName]
		// Only add as name conflict if it's not already a role match
		if resolvedConfig, exists := pcd.resolvedSSOConfigs[profile.Name]; exists {
			if resolvedConfig.AccountID != role.AccountID || resolvedConfig.RoleName != role.RoleName {
				return []Profile{profile}
			}
		} else if profile.SSOAccountID != role.AccountID || profile.SSORoleName != role.RoleName {
			return []Profile{profile}
		}
	} else if profile, exists := pcd.configFile.Profiles[proposedName]; exists {
		// Fallback to direct lookup
		// Only add as name conflict if it's not already a role match
		if profile.SSOAccountID != role.AccountID || profile.SSORoleName != role.RoleName {
			return []Profile{profile}
		}
	}

	return []Profile{}
}

// combineConflictingProfiles combines role-matching and name-conflicting profiles, removing duplicates
func (pcd *ProfileConflictDetector) combineConflictingProfiles(roleMatches, nameConflicts []Profile) []Profile {
	// Use more efficient deduplication with seen map and direct slice append
	totalEstimate := len(roleMatches) + len(nameConflicts)
	if totalEstimate == 0 {
		return []Profile{}
	}

	combined := make([]Profile, 0, totalEstimate)
	seen := make(map[string]bool, totalEstimate)

	// Add role matches
	for _, profile := range roleMatches {
		if !seen[profile.Name] {
			seen[profile.Name] = true
			combined = append(combined, profile)
		}
	}

	// Add name conflicts
	for _, profile := range nameConflicts {
		if !seen[profile.Name] {
			seen[profile.Name] = true
			combined = append(combined, profile)
		}
	}

	return combined
}
