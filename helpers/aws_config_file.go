package helpers

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

// LoadAWSConfigFile loads an AWS config file and parses its profiles
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

	// Read and parse the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, NewFileSystemError("failed to open config file", err).
			WithContext("file_path", filePath)
	}
	defer file.Close()

	if err := configFile.parseConfigFile(file); err != nil {
		return nil, err
	}

	return configFile, nil
}

// parseConfigFile parses the AWS config file content
func (cf *AWSConfigFile) parseConfigFile(file *os.File) error {
	scanner := bufio.NewScanner(file)
	var currentProfile *Profile
	var currentSession *SSOSession
	profileNameRegex := regexp.MustCompile(`^\[profile\s+(.+)\]$`)
	defaultProfileRegex := regexp.MustCompile(`^\[default\]$`)
	ssoSessionRegex := regexp.MustCompile(`^\[sso-session\s+(.+)\]$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for profile section
		if matches := profileNameRegex.FindStringSubmatch(line); matches != nil {
			// Save previous profile if exists
			if currentProfile != nil {
				cf.Profiles[currentProfile.Name] = *currentProfile
			}

			// Start new profile
			currentProfile = &Profile{
				Name:            matches[1],
				OtherProperties: make(map[string]string),
			}
		} else if matches := defaultProfileRegex.FindStringSubmatch(line); matches != nil {
			// Handle default profile
			if currentProfile != nil {
				cf.Profiles[currentProfile.Name] = *currentProfile
			}

			currentProfile = &Profile{
				Name:            "default",
				OtherProperties: make(map[string]string),
			}
		} else if matches := ssoSessionRegex.FindStringSubmatch(line); matches != nil {
			// Save previous profile if exists
			if currentProfile != nil {
				cf.Profiles[currentProfile.Name] = *currentProfile
				currentProfile = nil
			}
			// Save previous session if exists
			if currentSession != nil {
				cf.Sessions[currentSession.Name] = *currentSession
			}

			// Start new SSO session
			currentSession = &SSOSession{
				Name: matches[1],
			}
		} else if currentSession != nil {
			// Parse SSO session property line
			if strings.Contains(line, "=") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])

					// Set known SSO session properties
					switch key {
					case "sso_start_url":
						currentSession.SSOStartURL = value
					case "sso_region":
						currentSession.SSORegion = value
					}
				}
			}
		} else if currentProfile != nil {
			// Parse property line
			if strings.Contains(line, "=") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])

					// Set known properties
					switch key {
					case "region":
						currentProfile.Region = value
					case "sso_start_url":
						currentProfile.SSOStartURL = value
					case "sso_region":
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
				}
			}
		}
	}

	// Save the last profile
	if currentProfile != nil {
		cf.Profiles[currentProfile.Name] = *currentProfile
	}

	// Save the last session
	if currentSession != nil {
		cf.Sessions[currentSession.Name] = *currentSession
	}

	if err := scanner.Err(); err != nil {
		return NewFileSystemError("failed to read config file", err).
			WithContext("file_path", cf.FilePath)
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

// WriteToFile writes the config file to disk with backup creation
func (cf *AWSConfigFile) WriteToFile() error {
	// Create backup if file exists
	if _, err := os.Stat(cf.FilePath); err == nil {
		backupPath := cf.FilePath + ".backup"
		if err := copyFile(cf.FilePath, backupPath); err != nil {
			return NewFileSystemError("failed to create backup", err).
				WithContext("file_path", cf.FilePath).
				WithContext("backup_path", backupPath)
		}
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(cf.FilePath), 0755); err != nil {
		return NewFileSystemError("failed to create directory", err).
			WithContext("directory", filepath.Dir(cf.FilePath))
	}

	// Write config file
	file, err := os.Create(cf.FilePath)
	if err != nil {
		return NewFileSystemError("failed to create config file", err).
			WithContext("file_path", cf.FilePath)
	}
	defer file.Close()

	// Set proper permissions
	if err := file.Chmod(0600); err != nil {
		return NewFileSystemError("failed to set file permissions", err).
			WithContext("file_path", cf.FilePath)
	}

	// Write profiles
	for _, profile := range cf.Profiles {
		if _, err := file.WriteString(profile.ToConfigString()); err != nil {
			return NewFileSystemError("failed to write profile", err).
				WithContext("profile_name", profile.Name)
		}
	}

	return nil
}

// AppendToFile appends new profiles to the end of the config file
func (cf *AWSConfigFile) AppendToFile(profiles []GeneratedProfile) error {
	// Create backup if file exists
	if _, err := os.Stat(cf.FilePath); err == nil {
		backupPath := cf.FilePath + ".backup"
		if err := copyFile(cf.FilePath, backupPath); err != nil {
			return NewFileSystemError("failed to create backup", err).
				WithContext("file_path", cf.FilePath).
				WithContext("backup_path", backupPath)
		}
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(cf.FilePath), 0755); err != nil {
		return NewFileSystemError("failed to create directory", err).
			WithContext("directory", filepath.Dir(cf.FilePath))
	}

	// Open file for appending
	file, err := os.OpenFile(cf.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return NewFileSystemError("failed to open config file for appending", err).
			WithContext("file_path", cf.FilePath)
	}
	defer file.Close()

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
	return p.SSOStartURL != "" && p.SSORegion != ""
}

// IsLegacySSO returns true if the profile uses legacy SSO format
func (p *Profile) IsLegacySSO() bool {
	return p.IsSSO() && p.SSOSession == "" && p.SSOAccountID != "" && p.SSORoleName != ""
}

// validateFilePermissions checks if the file has proper permissions
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

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if err := dstFile.Chmod(0600); err != nil {
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
	var names []string
	for name := range cf.Profiles {
		names = append(names, name)
	}
	return names
}

// DetectProfileConflicts detects conflicts between existing profiles and new profiles
func (cf *AWSConfigFile) DetectProfileConflicts(newProfiles []GeneratedProfile) []string {
	var conflicts []string

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
	var matchingProfiles []Profile

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
