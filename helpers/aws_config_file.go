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
}

// Profile represents a profile in AWS config file
type Profile struct {
	Name            string
	Region          string
	SSOStartURL     string
	SSORegion       string
	SSOAccountID    string
	SSORoleName     string
	SSOSession      string
	Output          string
	OtherProperties map[string]string
}

// LoadAWSConfigFile loads an AWS config file and parses its profiles
func LoadAWSConfigFile(filePath string) (*AWSConfigFile, error) {
	if filePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, NewFileSystemError("failed to get user home directory", err)
		}
		filePath = filepath.Join(homeDir, ".aws", "config")
	}

	configFile := &AWSConfigFile{
		FilePath: filePath,
		Profiles: make(map[string]Profile),
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
	profileNameRegex := regexp.MustCompile(`^\[profile\s+(.+)\]$`)
	defaultProfileRegex := regexp.MustCompile(`^\[default\]$`)

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

	return cf.WriteToFile()
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
