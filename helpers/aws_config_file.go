package helpers

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
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
			// Save previous session if exists
			if currentSession != nil {
				cf.Sessions[currentSession.Name] = *currentSession
				currentSession = nil
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
			// Save previous session if exists
			if currentSession != nil {
				cf.Sessions[currentSession.Name] = *currentSession
				currentSession = nil
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
		"sso_start_url":  true,
		"sso_region":     true,
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

	// Copy backup back to original location with permission preservation
	if err := copyFileWithPermissions(backupPath, cf.FilePath); err != nil {
		return NewFileSystemError("failed to restore from backup", err).
			WithContext("backup_path", backupPath).
			WithContext("target_path", cf.FilePath)
	}

	// Reload the config from the restored file
	restoredConfig, err := LoadAWSConfigFile(cf.FilePath)
	if err != nil {
		return NewFileSystemError("failed to reload config after restore", err).
			WithContext("file_path", cf.FilePath)
	}

	// Update current instance with restored data
	cf.Profiles = restoredConfig.Profiles
	cf.Sessions = restoredConfig.Sessions

	return nil
}

// copyFileWithPermissions copies a file while preserving permissions
func copyFileWithPermissions(src, dst string) error {
	// Get source file info for permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Copy file content
	if err := copyFile(src, dst); err != nil {
		return err
	}

	// Set same permissions as source
	if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
		return err
	}

	return nil
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
		os.Remove(tempPath)

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
		os.Remove(tempPath)

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
		os.Remove(backupPath)
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
	defer file.Close()

	// Set proper permissions
	if err := file.Chmod(0600); err != nil {
		return NewFileSystemError("failed to set file permissions", err).
			WithContext("file_path", filePath)
	}

	// Acquire exclusive lock with timeout
	if err := cf.acquireFileLock(file); err != nil {
		return err
	}
	defer cf.releaseFileLock(file)

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
	for k, v := range cf.Profiles {
		originalProfiles[k] = v
	}

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
	for k, v := range cf.Profiles {
		originalProfiles[k] = v
	}

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
