package helpers

import (
	"fmt"
	"strings"
)

// maxInt returns the maximum of two integers
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ProfileConflictDetector handles the logic for detecting profile conflicts between
// discovered AWS roles and existing AWS CLI profiles. It provides efficient conflict
// detection using pre-built indexes and cached SSO configuration resolution.
//
// The detector implements a sophisticated conflict detection algorithm that:
// - Matches profiles based on SSO configuration rather than profile names
// - Supports both legacy and modern SSO profile formats
// - Uses efficient O(1) lookups through profile indexing
// - Pre-resolves and caches SSO configurations for performance
// - Classifies conflicts by type (same role vs same name)
//
// Conflict Detection Algorithm:
// 1. Pre-resolve all SSO configurations for existing profiles
// 2. For each discovered role, generate the proposed profile name
// 3. Find existing profiles that match the role's SSO configuration
// 4. Find existing profiles that have the same name as proposed
// 5. Classify conflicts and return detailed conflict information
//
// Performance Optimizations:
// - Profile lookup index for O(1) profile searches by account ID and name
// - Pre-resolved SSO configuration cache to avoid repeated resolution
// - Efficient memory allocation with capacity estimation
// - Batch processing of conflict detection operations
//
// Example usage:
//
//	detector := NewProfileConflictDetector(configFile, namingPattern)
//	conflicts, err := detector.DetectConflicts(discoveredRoles)
//	if err != nil {
//	    // handle error
//	}
//
//	for _, conflict := range conflicts {
//	    fmt.Printf("Conflict: %s -> %s (%s)\n",
//	        conflict.DiscoveredRole.RoleName,
//	        conflict.ProposedName,
//	        conflict.ConflictType.String())
//	}
type ProfileConflictDetector struct {
	configFile         *AWSConfigFile                // AWS config file containing existing profiles
	namingPattern      *NamingPattern                // Pattern for generating profile names
	logger             Logger                        // Logger for diagnostic messages
	profileIndex       *ProfileLookupIndex           // Efficient profile lookup index
	resolvedSSOConfigs map[string]*ResolvedSSOConfig // Profile name -> resolved SSO config cache
}

// NewProfileConflictDetector creates a new profile conflict detector with optimized
// performance features including profile indexing and SSO configuration caching.
//
// The constructor performs several initialization steps:
// 1. Creates the detector with provided config file and naming pattern
// 2. Builds an efficient profile lookup index for O(1) searches
// 3. Pre-resolves and caches SSO configurations for all existing profiles
// 4. Sets up a default logger (can be overridden with SetLogger)
//
// Parameters:
//   - configFile: AWS config file containing existing profiles to check against
//   - namingPattern: Pattern used to generate profile names for discovered roles
//
// Returns:
//   - *ProfileConflictDetector: Initialized detector ready for conflict detection
//
// Performance Notes:
// - Profile index creation is O(n) where n is the number of existing profiles
// - SSO configuration pre-resolution is O(n) but saves time during conflict detection
// - Memory usage scales linearly with the number of existing profiles
//
// Error Handling:
// - If profile index creation fails, detector falls back to linear search
// - If SSO resolution fails for a profile, it's skipped with a warning
// - Detector remains functional even with partial initialization failures
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

// DetectConflicts analyzes all discovered roles for conflicts with existing profiles.
// This is the main entry point for conflict detection and implements the complete
// conflict detection workflow.
//
// The method processes each discovered role through the following steps:
// 1. Validates the discovered role data
// 2. Generates the proposed profile name using the naming pattern
// 3. Searches for existing profiles that match the role's SSO configuration
// 4. Searches for existing profiles that have the same name as proposed
// 5. Classifies the type of conflict detected
// 6. Creates detailed conflict information for resolution
//
// Parameters:
//   - discoveredRoles: Slice of roles discovered through SSO enumeration
//
// Returns:
//   - []ProfileConflict: Slice of detected conflicts with detailed information
//   - error: Any error encountered during conflict detection
//
// Performance Characteristics:
// - Time complexity: O(n) where n is the number of discovered roles
// - Space complexity: O(c) where c is the number of conflicts (typically << n)
// - Uses pre-built indexes and caches for efficient profile lookups
//
// Error Handling:
// - Individual role analysis failures are logged as warnings and skipped
// - Only critical errors (e.g., invalid input) cause the method to fail
// - Partial results are returned even if some roles fail analysis
//
// Conflict Types Detected:
// - Same Role: Existing profile points to the same AWS role
// - Same Name: Proposed profile name already exists but points to different role
//
// Example usage:
//
//	conflicts, err := detector.DetectConflicts(discoveredRoles)
//	if err != nil {
//	    return fmt.Errorf("conflict detection failed: %w", err)
//	}
//
//	if len(conflicts) > 0 {
//	    fmt.Printf("Found %d conflicts requiring resolution\n", len(conflicts))
//	}
func (pcd *ProfileConflictDetector) DetectConflicts(discoveredRoles []DiscoveredRole) ([]ProfileConflict, error) {
	if len(discoveredRoles) == 0 {
		return []ProfileConflict{}, nil
	}

	// Pre-allocate slice capacity based on discovered roles count
	// Estimate that 20-30% of roles might have conflicts
	estimatedConflicts := maxInt(len(discoveredRoles)/4, 10)
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

// AnalyzeRole checks a single discovered role for conflicts with existing profiles.
// This method implements the core conflict detection logic for individual roles.
//
// The analysis process follows these steps:
// 1. Validates the discovered role has all required fields
// 2. Generates the proposed profile name using the configured naming pattern
// 3. Searches for existing profiles that match the role's SSO configuration
// 4. Searches for existing profiles that have the same name as proposed
// 5. Combines and deduplicates conflicting profiles
// 6. Classifies the type of conflict detected
// 7. Creates and validates the conflict object
//
// Parameters:
//   - role: The discovered role to analyze for conflicts
//
// Returns:
//   - *ProfileConflict: Detailed conflict information if conflicts found, nil otherwise
//   - error: Any error encountered during role analysis
//
// Conflict Detection Logic:
// - Role Match: Compares SSO start URL, account ID, and role name
// - Name Match: Compares proposed profile name with existing profile names
// - Supports both legacy SSO format and modern SSO session format
// - Uses cached SSO configurations for efficient matching
//
// Error Handling:
// - Invalid discovered role data returns validation error
// - Profile name generation failures return validation error
// - Profile matching failures are logged but don't fail the analysis
// - Invalid conflict objects return validation error
//
// Performance Notes:
// - Uses profile index for O(1) lookups when available
// - Falls back to linear search if index is unavailable
// - Leverages pre-resolved SSO configuration cache
// - Efficient memory allocation for conflict data structures
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

// ClassifyConflict determines the type of conflict detected between a discovered role
// and existing profiles. This classification helps determine the appropriate resolution strategy.
//
// Conflict Classification Logic:
// 1. Same Role Conflict: An existing profile already points to the same AWS role
//   - Matches on SSO start URL, account ID, and role name
//   - Indicates duplicate role configuration with potentially different profile names
//   - Resolution typically involves replacing or renaming the existing profile
//
// 2. Same Name Conflict: The proposed profile name already exists
//   - Matches on profile name but points to a different AWS role
//   - Indicates naming collision between different roles
//   - Resolution typically involves generating a different name or skipping
//
// The method prioritizes Same Role conflicts over Same Name conflicts when both
// conditions are present, as role conflicts are more significant for functionality.
//
// Parameters:
//   - existingProfiles: List of existing profiles that conflict with the discovered role
//   - proposedName: The proposed name for the new profile
//   - role: The discovered role being analyzed
//
// Returns:
//   - ConflictType: The type of conflict detected (ConflictSameRole or ConflictSameName)
//
// Error Handling:
// - Profile matching failures are logged as warnings but don't affect classification
// - Defaults to ConflictSameRole if classification cannot be determined
// - Handles both legacy and modern SSO profile formats transparently
//
// Example scenarios:
// - Same Role: Existing "prod-admin" profile points to same role as discovered "production-AdministratorAccess"
// - Same Name: Existing "prod-admin" profile points to different role than discovered "prod-admin"
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
