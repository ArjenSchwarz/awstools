package helpers

import (
	"fmt"
	"regexp"
	"strings"
)

// Pre-compiled regular expressions for better performance
var (
	invalidCharsRegex        = regexp.MustCompile(`[<>:"/\\|?*\s]`)
	placeholderRegex         = regexp.MustCompile(`\{[^}]+\}`)
	multipleUnderscoresRegex = regexp.MustCompile(`_+`)
)

// NamingPattern represents a naming pattern for profile generation
type NamingPattern struct {
	Pattern   string
	Variables map[string]string
}

// Supported placeholder variables
const (
	AccountIDPlaceholder    = "{account_id}"
	AccountNamePlaceholder  = "{account_name}"
	AccountAliasPlaceholder = "{account_alias}"
	RoleNamePlaceholder     = "{role_name}"
	RegionPlaceholder       = "{region}"
)

// GetSupportedPlaceholders returns all supported placeholder variables
func GetSupportedPlaceholders() []string {
	return []string{
		AccountIDPlaceholder,
		AccountNamePlaceholder,
		AccountAliasPlaceholder,
		RoleNamePlaceholder,
		RegionPlaceholder,
	}
}

// NewNamingPattern creates a new naming pattern with validation
func NewNamingPattern(pattern string) (*NamingPattern, error) {
	np := &NamingPattern{
		Pattern:   pattern,
		Variables: make(map[string]string),
	}

	if err := np.Validate(); err != nil {
		return nil, err
	}

	return np, nil
}

// Validate checks if the naming pattern is valid
func (np *NamingPattern) Validate() error {
	if np.Pattern == "" {
		return NewValidationError("naming pattern cannot be empty", nil)
	}

	// Check for invalid characters that would cause AWS config file issues
	if invalidCharsRegex.MatchString(np.Pattern) {
		return NewValidationError("naming pattern contains invalid characters", nil).
			WithContext("pattern", np.Pattern).
			WithContext("invalid_chars", "<>:\"/\\|?* and spaces")
	}

	// Extract placeholders from pattern
	placeholders := placeholderRegex.FindAllString(np.Pattern, -1)

	// Validate each placeholder
	supportedPlaceholders := GetSupportedPlaceholders()
	for _, placeholder := range placeholders {
		supported := false
		for _, supportedPlaceholder := range supportedPlaceholders {
			if placeholder == supportedPlaceholder {
				supported = true
				break
			}
		}
		if !supported {
			return NewValidationError("unsupported placeholder in naming pattern", nil).
				WithContext("placeholder", placeholder).
				WithContext("supported_placeholders", supportedPlaceholders)
		}
	}

	// Ensure pattern has at least one placeholder
	if len(placeholders) == 0 {
		return NewValidationError("naming pattern must contain at least one placeholder", nil).
			WithContext("pattern", np.Pattern).
			WithContext("supported_placeholders", supportedPlaceholders)
	}

	return nil
}

// GenerateProfileName generates a profile name using the pattern and provided values
func (np *NamingPattern) GenerateProfileName(accountID, accountName, accountAlias, roleName, region string) (string, error) {
	result := np.Pattern

	// Replace placeholders with actual values
	replacements := map[string]string{
		AccountIDPlaceholder:    accountID,
		AccountNamePlaceholder:  accountName,
		AccountAliasPlaceholder: accountAlias,
		RoleNamePlaceholder:     roleName,
		RegionPlaceholder:       region,
	}

	for placeholder, value := range replacements {
		if strings.Contains(result, placeholder) {
			if value == "" {
				return "", NewValidationError("missing value for placeholder", nil).
					WithContext("placeholder", placeholder).
					WithContext("pattern", np.Pattern)
			}
			// Sanitize the value
			sanitizedValue := SanitizeProfileName(value)
			result = strings.ReplaceAll(result, placeholder, sanitizedValue)
		}
	}

	// Final sanitization
	result = SanitizeProfileName(result)

	if result == "" {
		return "", NewValidationError("generated profile name is empty after sanitization", nil).
			WithContext("pattern", np.Pattern)
	}

	return result, nil
}

// SanitizeProfileName sanitizes a profile name by removing invalid characters
func SanitizeProfileName(name string) string {
	// Replace invalid characters with underscores
	sanitized := invalidCharsRegex.ReplaceAllString(name, "_")

	// Remove multiple consecutive underscores
	sanitized = multipleUnderscoresRegex.ReplaceAllString(sanitized, "_")

	// Remove leading/trailing underscores
	sanitized = strings.Trim(sanitized, "_")

	return sanitized
}

// ProfileNameConflictResolver handles profile name conflicts
type ProfileNameConflictResolver struct {
	existingNames map[string]bool
	conflicts     map[string]int
}

// NewProfileNameConflictResolver creates a new conflict resolver
func NewProfileNameConflictResolver(existingNames []string) *ProfileNameConflictResolver {
	resolver := &ProfileNameConflictResolver{
		existingNames: make(map[string]bool),
		conflicts:     make(map[string]int),
	}

	for _, name := range existingNames {
		resolver.existingNames[name] = true
	}

	return resolver
}

// ResolveConflict resolves naming conflicts by appending unique identifiers
func (resolver *ProfileNameConflictResolver) ResolveConflict(desiredName string) string {
	if !resolver.existingNames[desiredName] {
		resolver.existingNames[desiredName] = true
		return desiredName
	}

	// Generate unique name by appending counter
	counter := 1
	if existingCounter, exists := resolver.conflicts[desiredName]; exists {
		counter = existingCounter + 1
	}

	var uniqueName string
	for {
		uniqueName = fmt.Sprintf("%s_%d", desiredName, counter)
		if !resolver.existingNames[uniqueName] {
			break
		}
		counter++
	}

	resolver.conflicts[desiredName] = counter
	resolver.existingNames[uniqueName] = true
	return uniqueName
}

// GetConflictCount returns the number of conflicts for a given name
func (resolver *ProfileNameConflictResolver) GetConflictCount(name string) int {
	return resolver.conflicts[name]
}

// GetAllConflicts returns all conflicts as a map
func (resolver *ProfileNameConflictResolver) GetAllConflicts() map[string]int {
	return resolver.conflicts
}

// TestPattern tests a naming pattern with sample data
func TestPattern(pattern string) (*NamingPattern, []string, error) {
	np, err := NewNamingPattern(pattern)
	if err != nil {
		return nil, nil, err
	}

	// Test with sample data
	samples := []struct {
		accountID    string
		accountName  string
		accountAlias string
		roleName     string
		region       string
	}{
		{"123456789012", "production", "prod", "PowerUserAccess", "us-east-1"},
		{"210987654321", "development", "dev", "ReadOnlyAccess", "us-west-2"},
		{"555666777888", "staging", "stage", "Administrator", "eu-west-1"},
	}

	var results []string
	for _, sample := range samples {
		result, err := np.GenerateProfileName(sample.accountID, sample.accountName, sample.accountAlias, sample.roleName, sample.region)
		if err != nil {
			return nil, nil, err
		}
		results = append(results, result)
	}

	return np, results, nil
}

// ValidatePatternExamples validates common naming pattern examples
func ValidatePatternExamples() map[string]error {
	examples := map[string]string{
		"account_name_and_role":   "{account_name}-{role_name}",
		"account_id_and_role":     "{account_id}-{role_name}",
		"account_alias_and_role":  "{account_alias}-{role_name}",
		"sso_prefix":              "sso-{account_name}-{role_name}",
		"sso_alias_prefix":        "sso-{account_alias}-{role_name}",
		"with_region":             "{account_name}-{region}-{role_name}",
		"all_variables":           "{account_id}-{account_name}-{account_alias}-{role_name}-{region}",
		"invalid_chars":           "{account_name} {role_name}",
		"no_placeholders":         "static-name",
		"unsupported_placeholder": "{account_name}-{invalid_placeholder}",
	}

	results := make(map[string]error)
	for name, pattern := range examples {
		_, err := NewNamingPattern(pattern)
		results[name] = err
	}

	return results
}
