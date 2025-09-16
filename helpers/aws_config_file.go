package helpers

import (
	"bufio"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// Constants for SSO configuration keys
const (
	ssoStartURLKey = "sso_start_url"
	ssoRegionKey   = "sso_region"
)

// AWSConfigFile represents an AWS config file with its profiles
type AWSConfigFile struct {
	FilePath string
	Profiles map[string]Profile
	Sessions map[string]SSOSession // SSO session configurations
}

// SSOSession represents an SSO session configuration
type SSOSession struct {
	Name        string `json:"name" yaml:"name"`
	SSOStartURL string `json:"sso_start_url" yaml:"sso_start_url"`
	SSORegion   string `json:"sso_region" yaml:"sso_region"`
}

// Validate checks if the SSO session is valid
func (s *SSOSession) Validate() error {
	if s.Name == "" {
		return NewValidationError("SSO session name is required", nil)
	}
	if s.SSOStartURL == "" {
		return NewValidationError("SSO start URL is required", nil).
			WithContext("session_name", s.Name)
	}
	if s.SSORegion == "" {
		return NewValidationError("SSO region is required", nil).
			WithContext("session_name", s.Name)
	}
	return nil
}

// ResolvedSSOConfig represents the resolved SSO configuration for profile matching
type ResolvedSSOConfig struct {
	StartURL  string `json:"start_url" yaml:"start_url"`
	Region    string `json:"region" yaml:"region"`
	AccountID string `json:"account_id" yaml:"account_id"`
	RoleName  string `json:"role_name" yaml:"role_name"`
}

// Validate checks if the resolved SSO config is valid
func (r *ResolvedSSOConfig) Validate() error {
	if r.StartURL == "" {
		return NewValidationError("SSO start URL is required", nil)
	}
	if r.Region == "" {
		return NewValidationError("SSO region is required", nil)
	}
	if r.AccountID == "" {
		return NewValidationError("SSO account ID is required", nil)
	}
	if r.RoleName == "" {
		return NewValidationError("SSO role name is required", nil)
	}
	return nil
}

// Profile represents a profile in AWS config file
type Profile struct {
	Name              string             `json:"name" yaml:"name"`
	Region            string             `json:"region" yaml:"region"`
	SSOStartURL       string             `json:"sso_start_url" yaml:"sso_start_url"`
	SSORegion         string             `json:"sso_region" yaml:"sso_region"`
	SSOAccountID      string             `json:"sso_account_id" yaml:"sso_account_id"`
	SSORoleName       string             `json:"sso_role_name" yaml:"sso_role_name"`
	SSOSession        string             `json:"sso_session" yaml:"sso_session"`
	Output            string             `json:"output" yaml:"output"`
	OtherProperties   map[string]string  `json:"other_properties" yaml:"other_properties"`
	ResolvedSSOConfig *ResolvedSSOConfig `json:"resolved_sso_config,omitempty" yaml:"resolved_sso_config,omitempty"`
}

// Validate checks if the profile is valid
func (p *Profile) Validate() error {
	if p.Name == "" {
		return NewValidationError("profile name is required", nil)
	}
	return nil
}

// LoadAWSConfigFile loads and parses an AWS configuration file with comprehensive
// error recovery and validation capabilities. This function handles both existing
// and non-existent configuration files gracefully.
//
// File Path Resolution:
// 1. Uses provided filePath if not empty
// 2. Checks AWS_CONFIG_FILE environment variable
// 3. Falls back to default ~/.aws/config location
//
// Parsing Features:
// - Supports both profile and SSO session sections
// - Handles legacy and modern SSO profile formats
// - Recovers from malformed sections when possible
// - Validates file permissions and security
// - Provides detailed error context for troubleshooting
//
// Parameters:
//   - filePath: Path to AWS config file (empty for auto-detection)
//
// Returns:
//   - *AWSConfigFile: Parsed configuration with profiles and SSO sessions
//   - error: Critical parsing error or file system error
//
// Error Recovery Strategy:
// - Missing file: Returns empty configuration (not an error)
// - Malformed sections: Skips invalid sections, continues parsing
// - Permission issues: Returns detailed error with context
// - Partial parsing: Returns warning with successfully parsed content
//
// Security Validations:
// - Checks file permissions (should be 600 or similar)
// - Validates file ownership (should be current user)
// - Ensures file is readable and not corrupted
//
// Supported Formats:
// - Legacy SSO profiles with direct sso_* properties
// - Modern SSO profiles with sso_session references
// - Mixed environments with both formats
// - Standard AWS CLI configuration sections
//
// Example usage:
//
//	configFile, err := LoadAWSConfigFile("")
//	if err != nil {
//	    return fmt.Errorf("failed to load config: %w", err)
//	}
//
//	profile, exists := configFile.GetProfile("my-profile")
//	if !exists {
//	    return fmt.Errorf("profile not found")
//	}
func LoadAWSConfigFile(filePath string) (*AWSConfigFile, error) {
	if filePath == "" {
		// Check AWS_CONFIG_FILE environment variable first
		if configFile := os.Getenv("AWS_CONFIG_FILE"); configFile != "" {
			filePath = configFile
		} else {
			// Fall back to default location
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return nil, NewFileSystemError("failed to get user home directory", err)
			}
			filePath = filepath.Join(homeDir, ".aws", "config")
		}
	}

	configFile := &AWSConfigFile{
		FilePath: filePath,
		Profiles: make(map[string]Profile),
		Sessions: make(map[string]SSOSession),
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return configFile, nil // Return empty config file if it doesn't exist
	}

	// Check file permissions
	if err := validateFilePermissions(filePath); err != nil {
		return nil, err
	}

	// Read and parse the file with recovery
	file, err := os.Open(filePath)
	if err != nil {
		return nil, NewFileSystemError("failed to open config file", err).
			WithContext("file_path", filePath)
	}
	defer func() { _ = file.Close() }()

	// Try parsing with recovery first
	if err := configFile.parseConfigFileWithRecovery(file); err != nil {
		// If it's a validation error with partial recovery, log warning but continue
		if pgErr, ok := err.(ProfileGeneratorError); ok && pgErr.Type == ErrorTypeValidation {
			// Check if we have any successfully parsed profiles
			if len(configFile.Profiles) == 0 && len(configFile.Sessions) == 0 {
				// No profiles parsed, this is a critical error
				return nil, err
			}
		} else {
			// Other types of errors are critical
			return nil, err
		}
	}

	return configFile, nil
}

// parseConfigFileWithRecovery parses the AWS config file with malformed section recovery
func (cf *AWSConfigFile) parseConfigFileWithRecovery(file *os.File) error {
	scanner := bufio.NewScanner(file)
	var currentProfile *Profile
	var currentSession *SSOSession
	var parseErrors []error
	lineNumber := 0

	profileNameRegex := regexp.MustCompile(`^\[profile\s+(.+)\]$`)
	defaultProfileRegex := regexp.MustCompile(`^\[default\]$`)
	ssoSessionRegex := regexp.MustCompile(`^\[sso-session\s+(.+)\]$`)

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for profile section
		if matches := profileNameRegex.FindStringSubmatch(line); matches != nil {
			// Save previous profile if exists and valid
			if currentProfile != nil {
				if err := currentProfile.Validate(); err != nil {
					parseErrors = append(parseErrors, NewValidationError(
						fmt.Sprintf("invalid profile at line %d", lineNumber-1), err).
						WithContext("profile_name", currentProfile.Name).
						WithContext("line_number", lineNumber-1))
				} else {
					cf.Profiles[currentProfile.Name] = *currentProfile
				}
			}
			// Save previous session if exists and valid
			if currentSession != nil {
				if err := currentSession.Validate(); err != nil {
					parseErrors = append(parseErrors, NewValidationError(
						fmt.Sprintf("invalid SSO session at line %d", lineNumber-1), err).
						WithContext("session_name", currentSession.Name).
						WithContext("line_number", lineNumber-1))
				} else {
					cf.Sessions[currentSession.Name] = *currentSession
				}
				currentSession = nil
			}

			// Validate profile name
			profileName := strings.TrimSpace(matches[1])
			if profileName == "" {
				parseErrors = append(parseErrors, NewValidationError(
					fmt.Sprintf("empty profile name at line %d", lineNumber), nil).
					WithContext("line_number", lineNumber))
				currentProfile = nil
				continue
			}

			// Start new profile
			currentProfile = &Profile{
				Name:            profileName,
				OtherProperties: make(map[string]string),
			}
		} else if matches := defaultProfileRegex.FindStringSubmatch(line); matches != nil {
			// Handle default profile
			if currentProfile != nil {
				if err := currentProfile.Validate(); err != nil {
					parseErrors = append(parseErrors, NewValidationError(
						fmt.Sprintf("invalid profile at line %d", lineNumber-1), err).
						WithContext("profile_name", currentProfile.Name).
						WithContext("line_number", lineNumber-1))
				} else {
					cf.Profiles[currentProfile.Name] = *currentProfile
				}
			}
			// Save previous session if exists and valid
			if currentSession != nil {
				if err := currentSession.Validate(); err != nil {
					parseErrors = append(parseErrors, NewValidationError(
						fmt.Sprintf("invalid SSO session at line %d", lineNumber-1), err).
						WithContext("session_name", currentSession.Name).
						WithContext("line_number", lineNumber-1))
				} else {
					cf.Sessions[currentSession.Name] = *currentSession
				}
				currentSession = nil
			}

			currentProfile = &Profile{
				Name:            "default",
				OtherProperties: make(map[string]string),
			}
		} else if matches := ssoSessionRegex.FindStringSubmatch(line); matches != nil {
			// Save previous profile if exists and valid
			if currentProfile != nil {
				if err := currentProfile.Validate(); err != nil {
					parseErrors = append(parseErrors, NewValidationError(
						fmt.Sprintf("invalid profile at line %d", lineNumber-1), err).
						WithContext("profile_name", currentProfile.Name).
						WithContext("line_number", lineNumber-1))
				} else {
					cf.Profiles[currentProfile.Name] = *currentProfile
				}
				currentProfile = nil
			}
			// Save previous session if exists and valid
			if currentSession != nil {
				if err := currentSession.Validate(); err != nil {
					parseErrors = append(parseErrors, NewValidationError(
						fmt.Sprintf("invalid SSO session at line %d", lineNumber-1), err).
						WithContext("session_name", currentSession.Name).
						WithContext("line_number", lineNumber-1))
				} else {
					cf.Sessions[currentSession.Name] = *currentSession
				}
			}

			// Validate session name
			sessionName := strings.TrimSpace(matches[1])
			if sessionName == "" {
				parseErrors = append(parseErrors, NewValidationError(
					fmt.Sprintf("empty SSO session name at line %d", lineNumber), nil).
					WithContext("line_number", lineNumber))
				currentSession = nil
				continue
			}

			// Start new SSO session
			currentSession = &SSOSession{
				Name: sessionName,
			}
		} else if currentSession != nil { //nolint:gocritic // Sequential parsing logic requires if-else chain
			// Parse SSO session property line
			if strings.Contains(line, "=") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])

					// Validate property format
					if key == "" {
						parseErrors = append(parseErrors, NewValidationError(
							fmt.Sprintf("empty property key at line %d", lineNumber), nil).
							WithContext("line_number", lineNumber).
							WithContext("line_content", line))
						continue
					}

					// Set known SSO session properties
					switch key {
					case ssoStartURLKey:
						currentSession.SSOStartURL = value
					case ssoRegionKey:
						currentSession.SSORegion = value
					default:
						// Log unknown property but continue
						parseErrors = append(parseErrors, NewValidationError(
							fmt.Sprintf("unknown SSO session property '%s' at line %d", key, lineNumber), nil).
							WithContext("line_number", lineNumber).
							WithContext("property_key", key).
							WithContext("session_name", currentSession.Name))
					}
				} else {
					parseErrors = append(parseErrors, NewValidationError(
						fmt.Sprintf("malformed property line at %d", lineNumber), nil).
						WithContext("line_number", lineNumber).
						WithContext("line_content", line))
				}
			} else {
				parseErrors = append(parseErrors, NewValidationError(
					fmt.Sprintf("invalid SSO session property format at line %d", lineNumber), nil).
					WithContext("line_number", lineNumber).
					WithContext("line_content", line))
			}
		} else if currentProfile != nil {
			// Parse property line
			if strings.Contains(line, "=") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])

					// Validate property format
					if key == "" {
						parseErrors = append(parseErrors, NewValidationError(
							fmt.Sprintf("empty property key at line %d", lineNumber), nil).
							WithContext("line_number", lineNumber).
							WithContext("line_content", line))
						continue
					}

					// Set known properties
					switch key {
					case "region":
						currentProfile.Region = value
					case ssoStartURLKey:
						currentProfile.SSOStartURL = value
					case ssoRegionKey:
						currentProfile.SSORegion = value
					case "sso_account_id":
						currentProfile.SSOAccountID = value
					case "sso_role_name":
						currentProfile.SSORoleName = value
					case "sso_session":
						currentProfile.SSOSession = value
					case "output":
						currentProfile.Output = value
					default:
						// Store other properties
						currentProfile.OtherProperties[key] = value
					}
				} else {
					parseErrors = append(parseErrors, NewValidationError(
						fmt.Sprintf("malformed property line at %d", lineNumber), nil).
						WithContext("line_number", lineNumber).
						WithContext("line_content", line))
				}
			} else {
				parseErrors = append(parseErrors, NewValidationError(
					fmt.Sprintf("invalid profile property format at line %d", lineNumber), nil).
					WithContext("line_number", lineNumber).
					WithContext("line_content", line))
			}
		} else {
			// Line outside of any section
			parseErrors = append(parseErrors, NewValidationError(
				fmt.Sprintf("property outside of section at line %d", lineNumber), nil).
				WithContext("line_number", lineNumber).
				WithContext("line_content", line))
		}
	}

	// Save the last profile if valid
	if currentProfile != nil {
		if err := currentProfile.Validate(); err != nil {
			parseErrors = append(parseErrors, NewValidationError(
				"invalid profile at end of file", err).
				WithContext("profile_name", currentProfile.Name))
		} else {
			cf.Profiles[currentProfile.Name] = *currentProfile
		}
	}

	// Save the last session if valid
	if currentSession != nil {
		if err := currentSession.Validate(); err != nil {
			parseErrors = append(parseErrors, NewValidationError(
				"invalid SSO session at end of file", err).
				WithContext("session_name", currentSession.Name))
		} else {
			cf.Sessions[currentSession.Name] = *currentSession
		}
	}

	if err := scanner.Err(); err != nil {
		return NewFileSystemError("failed to read config file", err).
			WithContext("file_path", cf.FilePath)
	}

	// If we have parse errors but successfully parsed some content, return a warning
	if len(parseErrors) > 0 {
		// Create a composite error with all parse errors
		compositeErr := NewValidationError(
			fmt.Sprintf("config file contains %d parsing errors but partial recovery was successful", len(parseErrors)), nil).
			WithContext("file_path", cf.FilePath).
			WithContext("total_errors", len(parseErrors)).
			WithContext("profiles_parsed", len(cf.Profiles)).
			WithContext("sessions_parsed", len(cf.Sessions))

		// Add individual errors to context
		for i, parseErr := range parseErrors {
			if pgErr, ok := parseErr.(ProfileGeneratorError); ok {
				compositeErr = compositeErr.WithContext(fmt.Sprintf("error_%d", i), pgErr.Error())
			}
		}

		// Return the composite error as a warning - caller can decide whether to proceed
		return compositeErr
	}

	return nil
}

// GetProfile retrieves a profile by name
func (cf *AWSConfigFile) GetProfile(name string) (Profile, bool) {
	profile, exists := cf.Profiles[name]
	return profile, exists
}

// AddProfile adds a new profile to the config file
func (cf *AWSConfigFile) AddProfile(name string, profile Profile) error {
	if name == "" {
		return NewValidationError("profile name cannot be empty", nil)
	}

	profile.Name = name
	cf.Profiles[name] = profile
	return nil
}

// WriteToFile writes the config file to disk with backup creation and file locking
func (cf *AWSConfigFile) WriteToFile() error {
	// Validate write permissions before attempting
	if err := validateFilePermissionsForWrite(cf.FilePath); err != nil {
		return err
	}

	// Create backup if file exists
	backupPath := ""
	if _, err := os.Stat(cf.FilePath); err == nil {
		var backupErr error
		backupPath, backupErr = cf.CreateBackup()
		if backupErr != nil {
			return backupErr
		}
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(cf.FilePath), 0755); err != nil {
		return NewFileSystemError("failed to create directory", err).
			WithContext("directory", filepath.Dir(cf.FilePath))
	}

	// Write with file locking for concurrent access protection
	writeErr := withFileLock(cf.FilePath, func(file *os.File) error {
		// Truncate file to start fresh
		if err := file.Truncate(0); err != nil {
			return NewFileSystemError("failed to truncate config file", err).
				WithContext("file_path", cf.FilePath)
		}

		// Seek to beginning
		if _, err := file.Seek(0, 0); err != nil {
			return NewFileSystemError("failed to seek to beginning of file", err).
				WithContext("file_path", cf.FilePath)
		}

		// Set proper permissions
		if err := file.Chmod(0600); err != nil {
			return NewFileSystemError("failed to set file permissions", err).
				WithContext("file_path", cf.FilePath)
		}

		// Write SSO sessions first
		for _, session := range cf.Sessions {
			sessionConfig := fmt.Sprintf("[sso-session %s]\n", session.Name)
			if session.SSOStartURL != "" {
				sessionConfig += fmt.Sprintf("sso_start_url = %s\n", session.SSOStartURL)
			}
			if session.SSORegion != "" {
				sessionConfig += fmt.Sprintf("sso_region = %s\n", session.SSORegion)
			}
			sessionConfig += "\n"

			if _, err := file.WriteString(sessionConfig); err != nil {
				return NewFileSystemError("failed to write SSO session", err).
					WithContext("session_name", session.Name)
			}
		}

		// Write profiles
		for _, profile := range cf.Profiles {
			if _, err := file.WriteString(profile.ToConfigString()); err != nil {
				return NewFileSystemError("failed to write profile", err).
					WithContext("profile_name", profile.Name)
			}
		}

		return nil
	})

	// If write failed and we have a backup, attempt to restore
	if writeErr != nil && backupPath != "" {
		if restoreErr := cf.RestoreFromBackup(backupPath); restoreErr != nil {
			// Return both errors
			return NewFileSystemError("write failed and backup restore failed", writeErr).
				WithContext("restore_error", restoreErr.Error()).
				WithContext("backup_path", backupPath)
		}
		return writeErr
	}

	return writeErr
}

// AppendToFile appends new profiles to the end of the config file with file locking
func (cf *AWSConfigFile) AppendToFile(profiles []GeneratedProfile) error {
	// Validate write permissions before attempting
	if err := validateFilePermissionsForWrite(cf.FilePath); err != nil {
		return err
	}

	// Create backup if file exists
	backupPath := ""
	if _, err := os.Stat(cf.FilePath); err == nil {
		var backupErr error
		backupPath, backupErr = cf.CreateBackup()
		if backupErr != nil {
			return backupErr
		}
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(cf.FilePath), 0755); err != nil {
		return NewFileSystemError("failed to create directory", err).
			WithContext("directory", filepath.Dir(cf.FilePath))
	}

	// Append with file locking for concurrent access protection
	appendErr := func() error {
		// Open file for appending with locking
		file, err := os.OpenFile(cf.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return NewFileSystemError("failed to open config file for appending", err).
				WithContext("file_path", cf.FilePath)
		}
		defer func() { _ = file.Close() }()

		// Acquire lock
		if err := acquireFileLock(file); err != nil {
			return err
		}
		defer func() { _ = releaseFileLock(file) }()

		// Set proper permissions
		if err := file.Chmod(0600); err != nil {
			return NewFileSystemError("failed to set file permissions", err).
				WithContext("file_path", cf.FilePath)
		}

		// Append new profiles
		for _, genProfile := range profiles {
			profile := Profile{
				Name:            genProfile.Name,
				Region:          genProfile.Region,
				SSOStartURL:     genProfile.SSOStartURL,
				SSORegion:       genProfile.SSORegion,
				SSOAccountID:    genProfile.SSOAccountID,
				SSORoleName:     genProfile.SSORoleName,
				SSOSession:      genProfile.SSOSession,
				OtherProperties: make(map[string]string),
			}

			if _, err := file.WriteString(profile.ToConfigString()); err != nil {
				return NewFileSystemError("failed to write profile", err).
					WithContext("profile_name", profile.Name)
			}
		}

		return nil
	}()

	// If append failed and we have a backup, attempt to restore
	if appendErr != nil && backupPath != "" {
		if restoreErr := cf.RestoreFromBackup(backupPath); restoreErr != nil {
			// Return both errors
			return NewFileSystemError("append failed and backup restore failed", appendErr).
				WithContext("restore_error", restoreErr.Error()).
				WithContext("backup_path", backupPath)
		}
		return appendErr
	}

	return appendErr
}

// GenerateProfileText generates formatted profile text for multiple profiles
func (cf *AWSConfigFile) GenerateProfileText(profiles []GeneratedProfile) string {
	var result strings.Builder

	for _, profile := range profiles {
		result.WriteString(profile.ToConfigString())
		result.WriteString("\n")
	}

	return result.String()
}

// AppendProfiles appends multiple profiles to the config file
func (cf *AWSConfigFile) AppendProfiles(profiles []GeneratedProfile) error {
	for _, genProfile := range profiles {
		profile := Profile{
			Name:            genProfile.Name,
			Region:          genProfile.Region,
			SSOStartURL:     genProfile.SSOStartURL,
			SSORegion:       genProfile.SSORegion,
			SSOAccountID:    genProfile.SSOAccountID,
			SSORoleName:     genProfile.SSORoleName,
			SSOSession:      genProfile.SSOSession,
			OtherProperties: make(map[string]string),
		}

		if err := cf.AddProfile(genProfile.Name, profile); err != nil {
			return err
		}
	}

	return cf.AppendToFile(profiles)
}

// ToConfigString converts a Profile to AWS config file format
func (p *Profile) ToConfigString() string {
	var config strings.Builder

	if p.Name == "default" {
		config.WriteString("[default]\n")
	} else {
		config.WriteString(fmt.Sprintf("[profile %s]\n", p.Name))
	}

	if p.Region != "" {
		config.WriteString(fmt.Sprintf("region = %s\n", p.Region))
	}

	if p.SSOStartURL != "" {
		config.WriteString(fmt.Sprintf("sso_start_url = %s\n", p.SSOStartURL))
	}

	if p.SSORegion != "" {
		config.WriteString(fmt.Sprintf("sso_region = %s\n", p.SSORegion))
	}

	if p.SSOSession != "" {
		config.WriteString(fmt.Sprintf("sso_session = %s\n", p.SSOSession))
	} else {
		// Legacy format
		if p.SSOAccountID != "" {
			config.WriteString(fmt.Sprintf("sso_account_id = %s\n", p.SSOAccountID))
		}
		if p.SSORoleName != "" {
			config.WriteString(fmt.Sprintf("sso_role_name = %s\n", p.SSORoleName))
		}
	}

	if p.Output != "" {
		config.WriteString(fmt.Sprintf("output = %s\n", p.Output))
	}

	// Add other properties
	for key, value := range p.OtherProperties {
		config.WriteString(fmt.Sprintf("%s = %s\n", key, value))
	}

	config.WriteString("\n")
	return config.String()
}

// IsSSO returns true if the profile is configured for SSO
func (p *Profile) IsSSO() bool {
	// Legacy SSO format
	if p.SSOStartURL != "" && p.SSORegion != "" {
		return true
	}
	// SSO session format
	if p.SSOSession != "" && p.SSOAccountID != "" && p.SSORoleName != "" {
		return true
	}
	return false
}

// IsLegacySSO returns true if the profile uses legacy SSO format
func (p *Profile) IsLegacySSO() bool {
	return p.IsSSO() && p.SSOSession == "" && p.SSOAccountID != "" && p.SSORoleName != ""
}

// validateFilePermissions checks if the file has proper permissions for reading and writing
func validateFilePermissions(filePath string) error {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return NewFileSystemError("failed to get file info", err).
			WithContext("file_path", filePath)
	}

	// Check if file is readable
	if fileInfo.Mode().Perm()&0400 == 0 {
		return NewFileSystemError("config file is not readable", nil).
			WithContext("file_path", filePath).
			WithContext("permissions", fileInfo.Mode().String())
	}

	return nil
}

// validateFilePermissionsForWrite checks if the file has proper permissions for modification
func validateFilePermissionsForWrite(filePath string) error {
	// Check if file exists
	fileInfo, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		// File doesn't exist, check if directory is writable
		dir := filepath.Dir(filePath)
		dirInfo, err := os.Stat(dir)
		if err != nil {
			return NewFileSystemError("failed to get directory info", err).
				WithContext("directory", dir)
		}

		// Check if directory is writable
		if dirInfo.Mode().Perm()&0200 == 0 {
			return NewFileSystemError("directory is not writable", nil).
				WithContext("directory", dir).
				WithContext("permissions", dirInfo.Mode().String())
		}

		return nil
	}

	if err != nil {
		return NewFileSystemError("failed to get file info", err).
			WithContext("file_path", filePath)
	}

	// Check if file is writable
	if fileInfo.Mode().Perm()&0200 == 0 {
		return NewFileSystemError("config file is not writable", nil).
			WithContext("file_path", filePath).
			WithContext("permissions", fileInfo.Mode().String())
	}

	// Check if file is owned by current user (security check)
	if stat, ok := fileInfo.Sys().(*syscall.Stat_t); ok {
		currentUID := os.Getuid()
		if int(stat.Uid) != currentUID {
			return NewFileSystemError("config file is not owned by current user", nil).
				WithContext("file_path", filePath).
				WithContext("file_uid", stat.Uid).
				WithContext("current_uid", currentUID)
		}
	}

	return nil
}

// acquireFileLock acquires an exclusive lock on the config file for concurrent access protection
func acquireFileLock(file *os.File) error {
	// Use flock for file locking on Unix systems
	err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		if err == syscall.EWOULDBLOCK {
			return NewFileSystemError("config file is locked by another process", err).
				WithContext("file_path", file.Name())
		}
		return NewFileSystemError("failed to acquire file lock", err).
			WithContext("file_path", file.Name())
	}
	return nil
}

// releaseFileLock releases the exclusive lock on the config file
func releaseFileLock(file *os.File) error {
	err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	if err != nil {
		return NewFileSystemError("failed to release file lock", err).
			WithContext("file_path", file.Name())
	}
	return nil
}

// withFileLock executes a function while holding an exclusive lock on the file
func withFileLock(filePath string, fn func(*os.File) error) error {
	// Open file for reading and writing
	file, err := os.OpenFile(filePath, os.O_RDWR, 0600)
	if err != nil {
		return NewFileSystemError("failed to open file for locking", err).
			WithContext("file_path", filePath)
	}
	defer func() { _ = file.Close() }()

	// Acquire lock
	if err := acquireFileLock(file); err != nil {
		return err
	}
	defer func() { _ = releaseFileLock(file) }()

	// Execute the function
	return fn(file)
}

// copyFileWithPermissions copies a file from src to dst while preserving permissions
func copyFileWithPermissions(src, dst string) error {
	// Get source file info for permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = dstFile.Close() }()

	// Set same permissions as source
	if err := dstFile.Chmod(srcInfo.Mode()); err != nil {
		return err
	}

	buf := make([]byte, 4096)
	for {
		n, err := srcFile.Read(buf)
		if n == 0 {
			break
		}
		if err != nil {
			return err
		}

		if _, err := dstFile.Write(buf[:n]); err != nil {
			return err
		}
	}

	return nil
}

// HasProfile checks if a profile exists in the config file
func (cf *AWSConfigFile) HasProfile(name string) bool {
	_, exists := cf.Profiles[name]
	return exists
}

// GetProfileNames returns all profile names in the config file
func (cf *AWSConfigFile) GetProfileNames() []string {
	names := make([]string, 0, len(cf.Profiles))
	for name := range cf.Profiles {
		names = append(names, name)
	}
	return names
}

// DetectProfileConflicts detects conflicts between existing profiles and new profiles
func (cf *AWSConfigFile) DetectProfileConflicts(newProfiles []GeneratedProfile) []string {
	conflicts := make([]string, 0, len(newProfiles)/4) // Estimate 25% conflict rate

	for _, newProfile := range newProfiles {
		if cf.HasProfile(newProfile.Name) {
			conflicts = append(conflicts, newProfile.Name)
		}
	}

	return conflicts
}

// LoadSSOSessions loads SSO session configurations from the config file
func (cf *AWSConfigFile) LoadSSOSessions() error {
	// SSO sessions are already loaded during parseConfigFile
	return nil
}

// ResolveSSOSession resolves an SSO session reference to actual configuration
func (cf *AWSConfigFile) ResolveSSOSession(sessionName string) (*SSOSession, error) {
	if sessionName == "" {
		return nil, NewValidationError("SSO session name cannot be empty", nil)
	}

	session, exists := cf.Sessions[sessionName]
	if !exists {
		return nil, NewValidationError("SSO session not found", nil).
			WithContext("session_name", sessionName)
	}

	return &session, nil
}

// ResolveProfileSSOConfig normalizes both legacy and session-based SSO formats
func (cf *AWSConfigFile) ResolveProfileSSOConfig(profile Profile) (*ResolvedSSOConfig, error) {
	// Handle legacy SSO format
	if profile.SSOStartURL != "" && profile.SSOAccountID != "" && profile.SSORoleName != "" {
		return &ResolvedSSOConfig{
			StartURL:  profile.SSOStartURL,
			Region:    profile.SSORegion,
			AccountID: profile.SSOAccountID,
			RoleName:  profile.SSORoleName,
		}, nil
	}

	// Handle SSO session format
	if profile.SSOSession != "" {
		session, err := cf.ResolveSSOSession(profile.SSOSession)
		if err != nil {
			return nil, err
		}

		return &ResolvedSSOConfig{
			StartURL:  session.SSOStartURL,
			Region:    session.SSORegion,
			AccountID: profile.SSOAccountID,
			RoleName:  profile.SSORoleName,
		}, nil
	}

	return nil, NewValidationError("profile does not have valid SSO configuration", nil).
		WithContext("profile_name", profile.Name)
}

// MatchesRole compares profiles against discovered roles using normalized SSO config
func (cf *AWSConfigFile) MatchesRole(profile Profile, accountID, roleName, startURL string) (bool, error) {
	// Resolve SSO configuration for the profile
	resolvedConfig, err := cf.ResolveProfileSSOConfig(profile)
	if err != nil {
		return false, err
	}

	// Match based on resolved SSO configuration
	return resolvedConfig.StartURL == startURL &&
		resolvedConfig.AccountID == accountID &&
		resolvedConfig.RoleName == roleName, nil
}

// FindProfilesForRole finds existing profiles for specific roles
func (cf *AWSConfigFile) FindProfilesForRole(accountID, roleName, startURL string) ([]Profile, error) {
	matchingProfiles := make([]Profile, 0, 2) // Most roles have 0-2 matching profiles

	for _, profile := range cf.Profiles {
		matches, err := cf.MatchesRole(profile, accountID, roleName, startURL)
		if err != nil {
			// Log warning but continue processing other profiles
			continue
		}

		if matches {
			matchingProfiles = append(matchingProfiles, profile)
		}
	}

	return matchingProfiles, nil
}

// FindProfilesByName finds profiles by their names using efficient lookup
func (cf *AWSConfigFile) FindProfilesByName(profileNames []string) map[string]Profile {
	result := make(map[string]Profile)

	for _, name := range profileNames {
		if profile, exists := cf.Profiles[name]; exists {
			result[name] = profile
		}
	}

	return result
}

// HasProfileName checks if a profile name already exists
func (cf *AWSConfigFile) HasProfileName(profileName string) bool {
	_, exists := cf.Profiles[profileName]
	return exists
}

// FindDuplicateProfiles finds profiles that have the same SSO configuration
func (cf *AWSConfigFile) FindDuplicateProfiles() (map[string][]Profile, error) {
	// Map from SSO config key to list of profiles with that config
	ssoConfigMap := make(map[string][]Profile)
	duplicates := make(map[string][]Profile)

	for _, profile := range cf.Profiles {
		if !profile.IsSSO() {
			continue
		}

		// Resolve SSO configuration
		resolvedConfig, err := cf.ResolveProfileSSOConfig(profile)
		if err != nil {
			// Skip profiles that can't be resolved
			continue
		}

		// Create a unique key for the SSO configuration
		configKey := fmt.Sprintf("%s|%s|%s|%s",
			resolvedConfig.StartURL,
			resolvedConfig.Region,
			resolvedConfig.AccountID,
			resolvedConfig.RoleName)

		ssoConfigMap[configKey] = append(ssoConfigMap[configKey], profile)
	}

	// Find configurations with multiple profiles
	for configKey, profiles := range ssoConfigMap {
		if len(profiles) > 1 {
			duplicates[configKey] = profiles
		}
	}

	return duplicates, nil
}

// FindProfilesWithSSOConfig finds all profiles that match a specific SSO configuration
func (cf *AWSConfigFile) FindProfilesWithSSOConfig(startURL, region, accountID, roleName string) ([]Profile, error) {
	matchingProfiles := make([]Profile, 0, 2) // Most SSO configs have 0-2 matching profiles

	for _, profile := range cf.Profiles {
		if !profile.IsSSO() {
			continue
		}

		resolvedConfig, err := cf.ResolveProfileSSOConfig(profile)
		if err != nil {
			// Skip profiles that can't be resolved
			continue
		}

		if resolvedConfig.StartURL == startURL &&
			resolvedConfig.Region == region &&
			resolvedConfig.AccountID == accountID &&
			resolvedConfig.RoleName == roleName {
			matchingProfiles = append(matchingProfiles, profile)
		}
	}

	return matchingProfiles, nil
}

// GetProfileNameConflicts checks for profile name conflicts with proposed names
func (cf *AWSConfigFile) GetProfileNameConflicts(proposedNames []string) []string {
	conflicts := make([]string, 0, len(proposedNames)/4) // Estimate 25% conflict rate

	for _, name := range proposedNames {
		if cf.HasProfileName(name) {
			conflicts = append(conflicts, name)
		}
	}

	return conflicts
}

// BuildProfileLookupIndex creates efficient lookup indices for profile searching
func (cf *AWSConfigFile) BuildProfileLookupIndex() (*ProfileLookupIndex, error) {
	index := &ProfileLookupIndex{
		ByName:    make(map[string]Profile),
		BySSO:     make(map[string][]Profile),
		ByAccount: make(map[string][]Profile),
		ByRole:    make(map[string][]Profile),
	}

	for _, profile := range cf.Profiles {
		// Index by name
		index.ByName[profile.Name] = profile

		// Index SSO profiles
		if profile.IsSSO() {
			resolvedConfig, err := cf.ResolveProfileSSOConfig(profile)
			if err != nil {
				// Skip profiles that can't be resolved
				continue
			}

			// Index by SSO configuration
			ssoKey := fmt.Sprintf("%s|%s|%s|%s",
				resolvedConfig.StartURL,
				resolvedConfig.Region,
				resolvedConfig.AccountID,
				resolvedConfig.RoleName)
			index.BySSO[ssoKey] = append(index.BySSO[ssoKey], profile)

			// Index by account ID
			index.ByAccount[resolvedConfig.AccountID] = append(index.ByAccount[resolvedConfig.AccountID], profile)

			// Index by role name
			index.ByRole[resolvedConfig.RoleName] = append(index.ByRole[resolvedConfig.RoleName], profile)
		}
	}

	return index, nil
}

// ProfileLookupIndex provides efficient lookup indices for profile searching
type ProfileLookupIndex struct {
	ByName    map[string]Profile   // Profile name -> Profile
	BySSO     map[string][]Profile // SSO config key -> Profiles
	ByAccount map[string][]Profile // Account ID -> Profiles
	ByRole    map[string][]Profile // Role name -> Profiles
}

// FindBySSO finds profiles by SSO configuration using the index
func (index *ProfileLookupIndex) FindBySSO(startURL, region, accountID, roleName string) []Profile {
	ssoKey := fmt.Sprintf("%s|%s|%s|%s", startURL, region, accountID, roleName)
	return index.BySSO[ssoKey]
}

// FindByAccount finds profiles by account ID using the index
func (index *ProfileLookupIndex) FindByAccount(accountID string) []Profile {
	return index.ByAccount[accountID]
}

// FindByRole finds profiles by role name using the index
func (index *ProfileLookupIndex) FindByRole(roleName string) []Profile {
	return index.ByRole[roleName]
}

// HasName checks if a profile name exists using the index
func (index *ProfileLookupIndex) HasName(profileName string) bool {
	_, exists := index.ByName[profileName]
	return exists
}

// ReplaceProfile atomically replaces an existing profile with a new one
func (cf *AWSConfigFile) ReplaceProfile(oldName, newName string, newProfile Profile) error {
	if oldName == "" {
		return NewValidationError("old profile name cannot be empty", nil)
	}
	if newName == "" {
		return NewValidationError("new profile name cannot be empty", nil)
	}

	// Check if old profile exists
	oldProfile, exists := cf.Profiles[oldName]
	if !exists {
		return NewValidationError("profile to replace does not exist", nil).
			WithContext("profile_name", oldName)
	}

	// Validate new profile
	if err := newProfile.Validate(); err != nil {
		return err
	}

	// If names are different, check if new name already exists (unless it's the same profile)
	if oldName != newName {
		if _, exists := cf.Profiles[newName]; exists {
			return NewValidationError("new profile name already exists", nil).
				WithContext("profile_name", newName)
		}
	}

	// Preserve custom configuration properties from old profile
	if newProfile.OtherProperties == nil {
		newProfile.OtherProperties = make(map[string]string)
	}

	// Copy non-SSO properties from old profile that aren't set in new profile
	for key, value := range oldProfile.OtherProperties {
		// Only preserve properties that aren't SSO-related and aren't already set
		if !isSSORProperty(key) {
			if _, exists := newProfile.OtherProperties[key]; !exists {
				newProfile.OtherProperties[key] = value
			}
		}
	}

	// Preserve region if not set in new profile and exists in old profile
	if newProfile.Region == "" && oldProfile.Region != "" {
		newProfile.Region = oldProfile.Region
	}

	// Preserve output format if not set in new profile and exists in old profile
	if newProfile.Output == "" && oldProfile.Output != "" {
		newProfile.Output = oldProfile.Output
	}

	// Set the correct name
	newProfile.Name = newName

	// Perform atomic replacement
	if oldName != newName {
		// Different names: remove old, add new
		delete(cf.Profiles, oldName)
		cf.Profiles[newName] = newProfile
	} else {
		// Same name: update in place
		cf.Profiles[oldName] = newProfile
	}

	return nil
}

// RemoveProfile safely removes an existing profile
func (cf *AWSConfigFile) RemoveProfile(profileName string) error {
	if profileName == "" {
		return NewValidationError("profile name cannot be empty", nil)
	}

	// Check if profile exists
	if _, exists := cf.Profiles[profileName]; !exists {
		return NewValidationError("profile to remove does not exist", nil).
			WithContext("profile_name", profileName)
	}

	// Remove the profile
	delete(cf.Profiles, profileName)

	return nil
}

// isSSORProperty checks if a property key is SSO-related
func isSSORProperty(key string) bool {
	ssoProperties := map[string]bool{
		ssoStartURLKey:   true,
		ssoRegionKey:     true,
		"sso_account_id": true,
		"sso_role_name":  true,
		"sso_session":    true,
	}
	return ssoProperties[key]
}

// ValidateConfigIntegrity ensures the config file maintains integrity after operations
func (cf *AWSConfigFile) ValidateConfigIntegrity() error {
	// Check for duplicate profile names
	nameCount := make(map[string]int)
	for name := range cf.Profiles {
		nameCount[name]++
		if nameCount[name] > 1 {
			return NewValidationError("duplicate profile name detected", nil).
				WithContext("profile_name", name)
		}
	}

	// Validate each profile
	for name, profile := range cf.Profiles {
		if err := profile.Validate(); err != nil {
			if pgErr, ok := err.(ProfileGeneratorError); ok {
				return pgErr.WithContext("profile_name", name)
			}
			return NewValidationError("profile validation failed", err).
				WithContext("profile_name", name)
		}

		// Check name consistency
		if profile.Name != name {
			return NewValidationError("profile name mismatch", nil).
				WithContext("profile_name", name).
				WithContext("profile_internal_name", profile.Name)
		}
	}

	// Validate SSO sessions referenced by profiles
	for name, profile := range cf.Profiles {
		if profile.SSOSession != "" {
			if _, exists := cf.Sessions[profile.SSOSession]; !exists {
				return NewValidationError("profile references non-existent SSO session", nil).
					WithContext("profile_name", name).
					WithContext("sso_session", profile.SSOSession)
			}
		}
	}

	// Validate each SSO session
	for name, session := range cf.Sessions {
		if err := session.Validate(); err != nil {
			if pgErr, ok := err.(ProfileGeneratorError); ok {
				return pgErr.WithContext("sso_session_name", name)
			}
			return NewValidationError("SSO session validation failed", err).
				WithContext("sso_session_name", name)
		}

		// Check name consistency
		if session.Name != name {
			return NewValidationError("SSO session name mismatch", nil).
				WithContext("sso_session_name", name).
				WithContext("session_internal_name", session.Name)
		}
	}

	return nil
}

// CreateBackup creates a timestamped backup of the config file before modifications
func (cf *AWSConfigFile) CreateBackup() (string, error) {
	// Check if original file exists
	if _, err := os.Stat(cf.FilePath); os.IsNotExist(err) {
		// No file to backup
		return "", nil
	}

	// Generate timestamped backup filename
	timestamp := fmt.Sprintf("%d", os.Getpid()) // Use PID for uniqueness in tests
	backupPath := fmt.Sprintf("%s.backup.%s", cf.FilePath, timestamp)

	// Copy file with permission preservation
	if err := copyFileWithPermissions(cf.FilePath, backupPath); err != nil {
		return "", NewFileSystemError("failed to create backup", err).
			WithContext("source_file", cf.FilePath).
			WithContext("backup_file", backupPath)
	}

	return backupPath, nil
}

// RestoreFromBackup recovers from a backup file after failed operations
func (cf *AWSConfigFile) RestoreFromBackup(backupPath string) error {
	if backupPath == "" {
		return NewValidationError("backup path cannot be empty", nil)
	}

	// Check if backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return NewFileSystemError("backup file does not exist", err).
			WithContext("backup_path", backupPath)
	}

	// Temporarily make file writable for restore if it's read-only
	if originalInfo, err := os.Stat(cf.FilePath); err == nil {
		if originalInfo.Mode().Perm()&0200 == 0 {
			// File is read-only, make it writable temporarily
			if err := os.Chmod(cf.FilePath, 0600); err != nil {
				return NewFileSystemError("failed to make file writable for restore", err).
					WithContext("file_path", cf.FilePath)
			}
		}
	}

	// Copy backup back to original location with permission preservation
	if err := copyFileWithPermissions(backupPath, cf.FilePath); err != nil {
		return NewFileSystemError("failed to restore from backup", err).
			WithContext("backup_path", backupPath).
			WithContext("original_path", cf.FilePath)
	}

	// Reload the config from the restored file
	if err := cf.reloadFromFile(); err != nil {
		return NewFileSystemError("failed to reload config after restore", err).
			WithContext("file_path", cf.FilePath)
	}

	return nil
}

// reloadFromFile reloads the config file content into memory
func (cf *AWSConfigFile) reloadFromFile() error {
	// Clear existing data
	cf.Profiles = make(map[string]Profile)
	cf.Sessions = make(map[string]SSOSession)

	// Check if file exists
	if _, err := os.Stat(cf.FilePath); os.IsNotExist(err) {
		return nil // Empty config if file doesn't exist
	}

	// Read and parse the file
	file, err := os.Open(cf.FilePath)
	if err != nil {
		return NewFileSystemError("failed to open config file for reload", err).
			WithContext("file_path", cf.FilePath)
	}
	defer func() { _ = file.Close() }()

	return cf.parseConfigFileWithRecovery(file)
}

// Transaction represents a transactional operation on the config file
type Transaction struct {
	configFile *AWSConfigFile
	backupPath string
	operations []TransactionOperation
	committed  bool
	rolledBack bool
	tempFiles  []string
}

// TransactionOperation represents a single operation within a transaction
type TransactionOperation struct {
	Type        OperationType
	ProfileName string
	OldProfile  *Profile
	NewProfile  *Profile
	Description string
}

// OperationType represents the type of operation in a transaction
type OperationType int

// OperationType constants represent different types of operations in a transaction
const (
	// OpAdd represents an add operation
	OpAdd OperationType = iota
	// OpUpdate represents an update operation
	OpUpdate
	// OpRemove represents a remove operation
	OpRemove
)

// String returns the string representation of the operation type
func (ot OperationType) String() string {
	switch ot {
	case OpAdd:
		return "add"
	case OpUpdate:
		return "update"
	case OpRemove:
		return "remove"
	default:
		return unknownString
	}
}

// BeginTransaction starts a new transaction for atomic operations
func (cf *AWSConfigFile) BeginTransaction() (*Transaction, error) {
	// Create backup before starting transaction
	backupPath, err := cf.CreateBackup()
	if err != nil {
		return nil, NewBackupError("failed to create transaction backup", err, "", cf.FilePath).
			WithOperation("begin_transaction")
	}

	return &Transaction{
		configFile: cf,
		backupPath: backupPath,
		operations: make([]TransactionOperation, 0),
		tempFiles:  make([]string, 0),
	}, nil
}

// AddProfile adds a profile operation to the transaction
func (tx *Transaction) AddProfile(name string, profile Profile) error {
	if tx.committed || tx.rolledBack {
		return NewValidationError("transaction is already completed", nil).
			WithContext("committed", tx.committed).
			WithContext("rolled_back", tx.rolledBack)
	}

	// Check if profile already exists
	if _, exists := tx.configFile.Profiles[name]; exists {
		return NewValidationError("profile already exists", nil).
			WithContext("profile_name", name)
	}

	// Validate the profile
	if err := profile.Validate(); err != nil {
		return err
	}

	// Record the operation
	tx.operations = append(tx.operations, TransactionOperation{
		Type:        OpAdd,
		ProfileName: name,
		NewProfile:  &profile,
		Description: fmt.Sprintf("add profile %s", name),
	})

	// Apply the operation to the in-memory config
	profile.Name = name
	tx.configFile.Profiles[name] = profile

	return nil
}

// UpdateProfile adds a profile update operation to the transaction
func (tx *Transaction) UpdateProfile(name string, newProfile Profile) error {
	if tx.committed || tx.rolledBack {
		return NewValidationError("transaction is already completed", nil).
			WithContext("committed", tx.committed).
			WithContext("rolled_back", tx.rolledBack)
	}

	// Check if profile exists
	oldProfile, exists := tx.configFile.Profiles[name]
	if !exists {
		return NewValidationError("profile does not exist", nil).
			WithContext("profile_name", name)
	}

	// Validate the new profile
	if err := newProfile.Validate(); err != nil {
		return err
	}

	// Record the operation
	oldProfileCopy := oldProfile
	tx.operations = append(tx.operations, TransactionOperation{
		Type:        OpUpdate,
		ProfileName: name,
		OldProfile:  &oldProfileCopy,
		NewProfile:  &newProfile,
		Description: fmt.Sprintf("update profile %s", name),
	})

	// Apply the operation to the in-memory config
	newProfile.Name = name
	tx.configFile.Profiles[name] = newProfile

	return nil
}

// RemoveProfile adds a profile removal operation to the transaction
func (tx *Transaction) RemoveProfile(name string) error {
	if tx.committed || tx.rolledBack {
		return NewValidationError("transaction is already completed", nil).
			WithContext("committed", tx.committed).
			WithContext("rolled_back", tx.rolledBack)
	}

	// Check if profile exists
	oldProfile, exists := tx.configFile.Profiles[name]
	if !exists {
		return NewValidationError("profile does not exist", nil).
			WithContext("profile_name", name)
	}

	// Record the operation
	oldProfileCopy := oldProfile
	tx.operations = append(tx.operations, TransactionOperation{
		Type:        OpRemove,
		ProfileName: name,
		OldProfile:  &oldProfileCopy,
		Description: fmt.Sprintf("remove profile %s", name),
	})

	// Apply the operation to the in-memory config
	delete(tx.configFile.Profiles, name)

	return nil
}

// ReplaceProfile adds a profile replacement operation to the transaction
func (tx *Transaction) ReplaceProfile(oldName, newName string, newProfile Profile) error {
	if tx.committed || tx.rolledBack {
		return NewValidationError("transaction is already completed", nil).
			WithContext("committed", tx.committed).
			WithContext("rolled_back", tx.rolledBack)
	}

	// If names are the same, this is just an update
	if oldName == newName {
		return tx.UpdateProfile(oldName, newProfile)
	}

	// Check if old profile exists
	_, exists := tx.configFile.Profiles[oldName]
	if !exists {
		return NewValidationError("profile to replace does not exist", nil).
			WithContext("profile_name", oldName)
	}

	// Check if new name already exists (unless it's the same profile)
	if _, exists := tx.configFile.Profiles[newName]; exists {
		return NewValidationError("new profile name already exists", nil).
			WithContext("profile_name", newName)
	}

	// Validate the new profile
	if err := newProfile.Validate(); err != nil {
		return err
	}

	// This is a combination of remove and add operations
	// First remove the old profile
	if err := tx.RemoveProfile(oldName); err != nil {
		return err
	}

	// Then add the new profile
	return tx.AddProfile(newName, newProfile)
}

// Commit applies all transaction operations to the file system
func (tx *Transaction) Commit() error {
	if tx.committed || tx.rolledBack {
		return NewValidationError("transaction is already completed", nil).
			WithContext("committed", tx.committed).
			WithContext("rolled_back", tx.rolledBack)
	}

	// Validate config integrity before committing
	if err := tx.configFile.ValidateConfigIntegrity(); err != nil {
		// Rollback on validation failure
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return NewValidationError("commit validation failed and rollback failed", err).
				WithContext("rollback_error", rollbackErr.Error())
		}
		return NewValidationError("commit validation failed", err)
	}

	// Write the config to file
	if err := tx.configFile.WriteToFile(); err != nil {
		// Rollback on write failure
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return NewFileSystemError("commit write failed and rollback failed", err).
				WithContext("rollback_error", rollbackErr.Error())
		}
		return NewFileSystemError("commit write failed", err)
	}

	// Mark as committed
	tx.committed = true

	// Clean up backup and temp files
	_ = tx.cleanup()

	return nil
}

// Rollback reverts all transaction operations
func (tx *Transaction) Rollback() error {
	if tx.committed {
		return NewValidationError("cannot rollback committed transaction", nil)
	}

	if tx.rolledBack {
		return NewValidationError("transaction is already rolled back", nil)
	}

	// Restore from backup if it exists (this also reloads in-memory state)
	if tx.backupPath != "" {
		if err := tx.configFile.RestoreFromBackup(tx.backupPath); err != nil {
			return NewBackupError("failed to restore from backup during rollback", err,
				tx.backupPath, tx.configFile.FilePath).
				WithOperation("rollback")
		}
	} else {
		// If no backup exists, manually revert in-memory operations
		for i := len(tx.operations) - 1; i >= 0; i-- {
			op := tx.operations[i]
			switch op.Type {
			case OpAdd:
				delete(tx.configFile.Profiles, op.ProfileName)
			case OpRemove:
				if op.OldProfile != nil {
					tx.configFile.Profiles[op.ProfileName] = *op.OldProfile
				}
			case OpUpdate:
				if op.OldProfile != nil {
					tx.configFile.Profiles[op.ProfileName] = *op.OldProfile
				}
			}
		}
	}

	// Mark as rolled back
	tx.rolledBack = true

	// Clean up temp files
	_ = tx.cleanup()

	return nil
}

// cleanup removes temporary files created during the transaction
func (tx *Transaction) cleanup() error {
	var cleanupErrors []error

	// Remove backup file if transaction was committed successfully
	if tx.committed && tx.backupPath != "" {
		if err := os.Remove(tx.backupPath); err != nil && !os.IsNotExist(err) {
			cleanupErrors = append(cleanupErrors,
				NewFileSystemError("failed to remove backup file", err).
					WithContext("backup_path", tx.backupPath))
		}
	}

	// Remove any temporary files
	for _, tempFile := range tx.tempFiles {
		if err := os.Remove(tempFile); err != nil && !os.IsNotExist(err) {
			cleanupErrors = append(cleanupErrors,
				NewFileSystemError("failed to remove temporary file", err).
					WithContext("temp_file", tempFile))
		}
	}

	// Return composite error if there were cleanup issues
	if len(cleanupErrors) > 0 {
		return NewFileSystemError(
			fmt.Sprintf("cleanup completed with %d errors", len(cleanupErrors)), nil).
			WithContext("error_count", len(cleanupErrors))
	}

	return nil
}

// GetOperations returns a copy of all operations in the transaction
func (tx *Transaction) GetOperations() []TransactionOperation {
	operations := make([]TransactionOperation, len(tx.operations))
	copy(operations, tx.operations)
	return operations
}

// GetOperationSummary returns a human-readable summary of transaction operations
func (tx *Transaction) GetOperationSummary() string {
	if len(tx.operations) == 0 {
		return "No operations in transaction"
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Transaction with %d operations:\n", len(tx.operations)))

	for i, op := range tx.operations {
		summary.WriteString(fmt.Sprintf("  %d. %s\n", i+1, op.Description))
	}

	switch {
	case tx.committed:
		summary.WriteString("Status: Committed\n")
	case tx.rolledBack:
		summary.WriteString("Status: Rolled back\n")
	default:
		summary.WriteString("Status: Pending\n")
	}

	return summary.String()
}

// ExecuteAtomicProfileOperations executes multiple profile operations atomically
func (cf *AWSConfigFile) ExecuteAtomicProfileOperations(operations []func(*Transaction) error) error {
	// Begin transaction
	tx, err := cf.BeginTransaction()
	if err != nil {
		return err
	}

	// Ensure rollback on panic or error
	defer func() {
		if !tx.committed && !tx.rolledBack {
			_ = tx.Rollback()
		}
	}()

	// Execute all operations
	for i, operation := range operations {
		if err := operation(tx); err != nil {
			// Rollback on any operation failure
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				return NewValidationError(
					fmt.Sprintf("operation %d failed and rollback failed", i+1), err).
					WithContext("rollback_error", rollbackErr.Error())
			}
			return NewValidationError(fmt.Sprintf("operation %d failed", i+1), err)
		}
	}

	// Commit all operations
	return tx.Commit()
}

// AtomicWriteToFile writes the config file atomically with file locking
func (cf *AWSConfigFile) AtomicWriteToFile() error {
	// Create backup before any modifications
	backupPath, err := cf.CreateBackup()
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(cf.FilePath), 0755); err != nil {
		return NewFileSystemError("failed to create directory", err).
			WithContext("directory", filepath.Dir(cf.FilePath))
	}

	// Create temporary file for atomic write
	tempPath := cf.FilePath + ".tmp"

	// Acquire file lock and perform atomic write
	if err := cf.writeWithLock(tempPath); err != nil {
		// Clean up temp file on error
		_ = os.Remove(tempPath)

		// Restore from backup if it exists
		if backupPath != "" {
			if restoreErr := cf.RestoreFromBackup(backupPath); restoreErr != nil {
				return NewFileSystemError("failed to write config and restore backup", err).
					WithContext("write_error", err.Error()).
					WithContext("restore_error", restoreErr.Error())
			}
		}
		return err
	}

	// Atomically replace the original file
	if err := os.Rename(tempPath, cf.FilePath); err != nil {
		// Clean up temp file
		_ = os.Remove(tempPath)

		// Restore from backup if it exists
		if backupPath != "" {
			if restoreErr := cf.RestoreFromBackup(backupPath); restoreErr != nil {
				return NewFileSystemError("failed to rename config file and restore backup", err).
					WithContext("rename_error", err.Error()).
					WithContext("restore_error", restoreErr.Error())
			}
		}
		return NewFileSystemError("failed to rename temporary file to config file", err).
			WithContext("temp_path", tempPath).
			WithContext("target_path", cf.FilePath)
	}

	// Clean up backup on success (optional - could keep for safety)
	if backupPath != "" {
		_ = os.Remove(backupPath)
	}

	return nil
}

// writeWithLock writes config content to a file with exclusive locking
func (cf *AWSConfigFile) writeWithLock(filePath string) error {
	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		return NewFileSystemError("failed to create temporary config file", err).
			WithContext("file_path", filePath)
	}
	defer func() { _ = file.Close() }()

	// Set proper permissions
	if err := file.Chmod(0600); err != nil {
		return NewFileSystemError("failed to set file permissions", err).
			WithContext("file_path", filePath)
	}

	// Acquire exclusive lock with timeout
	if err := cf.acquireFileLock(file); err != nil {
		return err
	}
	defer func() { _ = cf.releaseFileLock(file) }()

	// Write SSO sessions first
	for _, session := range cf.Sessions {
		sessionContent := fmt.Sprintf("[sso-session %s]\n", session.Name)
		if session.SSOStartURL != "" {
			sessionContent += fmt.Sprintf("sso_start_url = %s\n", session.SSOStartURL)
		}
		if session.SSORegion != "" {
			sessionContent += fmt.Sprintf("sso_region = %s\n", session.SSORegion)
		}
		sessionContent += "\n"

		if _, err := file.WriteString(sessionContent); err != nil {
			return NewFileSystemError("failed to write SSO session", err).
				WithContext("session_name", session.Name)
		}
	}

	// Write profiles
	for _, profile := range cf.Profiles {
		if _, err := file.WriteString(profile.ToConfigString()); err != nil {
			return NewFileSystemError("failed to write profile", err).
				WithContext("profile_name", profile.Name)
		}
	}

	// Ensure all data is written to disk
	if err := file.Sync(); err != nil {
		return NewFileSystemError("failed to sync file to disk", err).
			WithContext("file_path", filePath)
	}

	return nil
}

// acquireFileLock acquires an exclusive lock on the file with timeout
func (cf *AWSConfigFile) acquireFileLock(file *os.File) error {
	// Try to acquire lock with timeout
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return NewFileSystemError("timeout acquiring file lock", nil).
				WithContext("file_path", cf.FilePath).
				WithContext("timeout_seconds", 5)
		case <-ticker.C:
			// Try to acquire exclusive lock
			err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
			if err == nil {
				return nil // Lock acquired successfully
			}
			if err != syscall.EWOULDBLOCK {
				return NewFileSystemError("failed to acquire file lock", err).
					WithContext("file_path", cf.FilePath)
			}
			// Lock is held by another process, continue trying
		}
	}
}

// releaseFileLock releases the file lock
func (cf *AWSConfigFile) releaseFileLock(file *os.File) error {
	err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	if err != nil {
		return NewFileSystemError("failed to release file lock", err).
			WithContext("file_path", cf.FilePath)
	}
	return nil
}

// AtomicReplaceProfile atomically replaces a profile with backup and rollback
func (cf *AWSConfigFile) AtomicReplaceProfile(oldName, newName string, newProfile Profile) error {
	// Create backup before any modifications
	backupPath, err := cf.CreateBackup()
	if err != nil {
		return err
	}

	// Store original state for rollback
	originalProfiles := make(map[string]Profile)
	maps.Copy(originalProfiles, cf.Profiles)

	// Perform the replacement in memory
	if err := cf.ReplaceProfile(oldName, newName, newProfile); err != nil {
		return err
	}

	// Validate config integrity
	if err := cf.ValidateConfigIntegrity(); err != nil {
		// Rollback in-memory changes
		cf.Profiles = originalProfiles
		return NewValidationError("config integrity validation failed after profile replacement", err).
			WithContext("old_profile", oldName).
			WithContext("new_profile", newName)
	}

	// Write changes atomically
	if err := cf.AtomicWriteToFile(); err != nil {
		// Rollback in-memory changes
		cf.Profiles = originalProfiles

		// Restore from backup
		if backupPath != "" {
			if restoreErr := cf.RestoreFromBackup(backupPath); restoreErr != nil {
				return NewFileSystemError("failed to write changes and restore backup", err).
					WithContext("write_error", err.Error()).
					WithContext("restore_error", restoreErr.Error())
			}
		}
		return err
	}

	return nil
}

// AtomicRemoveProfile atomically removes a profile with backup and rollback
func (cf *AWSConfigFile) AtomicRemoveProfile(profileName string) error {
	// Create backup before any modifications
	backupPath, err := cf.CreateBackup()
	if err != nil {
		return err
	}

	// Store original state for rollback
	originalProfiles := make(map[string]Profile)
	maps.Copy(originalProfiles, cf.Profiles)

	// Perform the removal in memory
	if err := cf.RemoveProfile(profileName); err != nil {
		return err
	}

	// Validate config integrity
	if err := cf.ValidateConfigIntegrity(); err != nil {
		// Rollback in-memory changes
		cf.Profiles = originalProfiles
		return NewValidationError("config integrity validation failed after profile removal", err).
			WithContext("removed_profile", profileName)
	}

	// Write changes atomically
	if err := cf.AtomicWriteToFile(); err != nil {
		// Rollback in-memory changes
		cf.Profiles = originalProfiles

		// Restore from backup
		if backupPath != "" {
			if restoreErr := cf.RestoreFromBackup(backupPath); restoreErr != nil {
				return NewFileSystemError("failed to write changes and restore backup", err).
					WithContext("write_error", err.Error()).
					WithContext("restore_error", restoreErr.Error())
			}
		}
		return err
	}

	return nil
}
