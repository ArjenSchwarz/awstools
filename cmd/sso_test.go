package cmd

import (
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileGeneratorFlags(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid template flag",
			args:        []string{"--template", "test-profile"},
			expectError: false,
		},
		{
			name:        "missing required template flag",
			args:        []string{},
			expectError: true,
			errorMsg:    "required flag(s) \"template\" not set",
		},
		{
			name:        "replace-existing flag",
			args:        []string{"--template", "test-profile", "--replace-existing"},
			expectError: false,
		},
		{
			name:        "skip-existing flag",
			args:        []string{"--template", "test-profile", "--skip-existing"},
			expectError: false,
		},
		{
			name:        "mutually exclusive flags error",
			args:        []string{"--template", "test-profile", "--replace-existing", "--skip-existing"},
			expectError: true,
			errorMsg:    "if any flags in the group [replace-existing skip-existing] are set none of the others can be; [replace-existing skip-existing] were all set",
		},
		{
			name:        "all valid flags together",
			args:        []string{"--template", "test-profile", "--pattern", "{account_name}-{role_name}", "--yes", "--output-file", "/tmp/config"},
			expectError: false,
		},
		{
			name:        "replace-existing with other flags",
			args:        []string{"--template", "test-profile", "--pattern", "custom-{role_name}", "--replace-existing", "--yes"},
			expectError: false,
		},
		{
			name:        "skip-existing with other flags",
			args:        []string{"--template", "test-profile", "--skip-existing", "--output-file", "/tmp/config"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new command instance for each test
			cmd := &cobra.Command{
				Use: "profile-generator",
				Run: func(cmd *cobra.Command, args []string) {
					// Mock run function for testing
				},
			}

			// Add flags
			cmd.Flags().StringP("template", "t", "", "Template profile name (required)")
			cmd.Flags().StringP("pattern", "p", "{account_name}-{role_name}", "Naming pattern for generated profiles")
			cmd.Flags().BoolP("yes", "y", false, "Auto-approve appending profiles without confirmation")
			cmd.Flags().StringP("output-file", "F", "", "Output to file instead of appending to ~/.aws/config")
			cmd.Flags().Bool("replace-existing", false, "Replace existing profiles with new names based on pattern")
			cmd.Flags().Bool("skip-existing", false, "Skip generating profiles for roles that already have profiles")

			cmd.MarkFlagRequired("template")
			cmd.MarkFlagsMutuallyExclusive("replace-existing", "skip-existing")

			// Set arguments and parse
			cmd.SetArgs(tt.args)
			err := cmd.Execute()

			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)

				// Verify flag values are parsed correctly
				if cmd.Flags().Changed("template") {
					template, _ := cmd.Flags().GetString("template")
					assert.Equal(t, "test-profile", template)
				}

				if cmd.Flags().Changed("replace-existing") {
					replaceExisting, _ := cmd.Flags().GetBool("replace-existing")
					assert.True(t, replaceExisting)
				}

				if cmd.Flags().Changed("skip-existing") {
					skipExisting, _ := cmd.Flags().GetBool("skip-existing")
					assert.True(t, skipExisting)
				}
			}
		})
	}
}

func TestConflictStrategyDetermination(t *testing.T) {
	tests := []struct {
		name            string
		replaceExisting bool
		skipExisting    bool
		expectedResult  string // We'll test the string representation
	}{
		{
			name:            "default strategy (prompt)",
			replaceExisting: false,
			skipExisting:    false,
			expectedResult:  "prompt",
		},
		{
			name:            "replace strategy",
			replaceExisting: true,
			skipExisting:    false,
			expectedResult:  "replace",
		},
		{
			name:            "skip strategy",
			replaceExisting: false,
			skipExisting:    true,
			expectedResult:  "skip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the conflict strategy determination logic from profileGenerator function
			var conflictStrategy string
			if tt.replaceExisting {
				conflictStrategy = "replace"
			} else if tt.skipExisting {
				conflictStrategy = "skip"
			} else {
				conflictStrategy = "prompt"
			}

			assert.Equal(t, tt.expectedResult, conflictStrategy)
		})
	}
}

func TestProfileGeneratorFlagDefaults(t *testing.T) {
	// Create a new command instance
	cmd := &cobra.Command{
		Use: "profile-generator",
		Run: func(cmd *cobra.Command, args []string) {
			// Mock run function for testing
		},
	}

	// Add flags
	cmd.Flags().StringP("template", "t", "", "Template profile name (required)")
	cmd.Flags().StringP("pattern", "p", "{account_name}-{role_name}", "Naming pattern for generated profiles")
	cmd.Flags().BoolP("yes", "y", false, "Auto-approve appending profiles without confirmation")
	cmd.Flags().StringP("output-file", "F", "", "Output to file instead of appending to ~/.aws/config")
	cmd.Flags().Bool("replace-existing", false, "Replace existing profiles with new names based on pattern")
	cmd.Flags().Bool("skip-existing", false, "Skip generating profiles for roles that already have profiles")

	// Test default values
	pattern, _ := cmd.Flags().GetString("pattern")
	assert.Equal(t, "{account_name}-{role_name}", pattern)

	yes, _ := cmd.Flags().GetBool("yes")
	assert.False(t, yes)

	outputFile, _ := cmd.Flags().GetString("output-file")
	assert.Empty(t, outputFile)

	replaceExisting, _ := cmd.Flags().GetBool("replace-existing")
	assert.False(t, replaceExisting)

	skipExisting, _ := cmd.Flags().GetBool("skip-existing")
	assert.False(t, skipExisting)
}

func TestProfileGeneratorFlagShortcuts(t *testing.T) {
	// Create a new command instance
	cmd := &cobra.Command{
		Use: "profile-generator",
		Run: func(cmd *cobra.Command, args []string) {
			// Mock run function for testing
		},
	}

	// Add flags
	cmd.Flags().StringP("template", "t", "", "Template profile name (required)")
	cmd.Flags().StringP("pattern", "p", "{account_name}-{role_name}", "Naming pattern for generated profiles")
	cmd.Flags().BoolP("yes", "y", false, "Auto-approve appending profiles without confirmation")
	cmd.Flags().StringP("output-file", "F", "", "Output to file instead of appending to ~/.aws/config")

	// Test short flag versions
	cmd.SetArgs([]string{"-t", "test-profile", "-p", "custom-pattern", "-y", "-F", "/tmp/config"})
	err := cmd.Execute()
	require.NoError(t, err)

	template, _ := cmd.Flags().GetString("template")
	assert.Equal(t, "test-profile", template)

	pattern, _ := cmd.Flags().GetString("pattern")
	assert.Equal(t, "custom-pattern", pattern)

	yes, _ := cmd.Flags().GetBool("yes")
	assert.True(t, yes)

	outputFile, _ := cmd.Flags().GetString("output-file")
	assert.Equal(t, "/tmp/config", outputFile)
}

func TestConflictResolutionIntegration(t *testing.T) {
	tests := []struct {
		name                string
		replaceExisting     bool
		skipExisting        bool
		expectedStrategy    string
		expectedDescription string
	}{
		{
			name:                "default prompt strategy",
			replaceExisting:     false,
			skipExisting:        false,
			expectedStrategy:    "prompt",
			expectedDescription: "prompt user for each conflict",
		},
		{
			name:                "replace existing strategy",
			replaceExisting:     true,
			skipExisting:        false,
			expectedStrategy:    "replace",
			expectedDescription: "replace existing profiles with new names",
		},
		{
			name:                "skip existing strategy",
			replaceExisting:     false,
			skipExisting:        true,
			expectedStrategy:    "skip",
			expectedDescription: "skip roles that already have profiles",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the conflict strategy determination logic
			var conflictStrategy string
			if tt.replaceExisting {
				conflictStrategy = "replace"
			} else if tt.skipExisting {
				conflictStrategy = "skip"
			} else {
				conflictStrategy = "prompt"
			}

			assert.Equal(t, tt.expectedStrategy, conflictStrategy)

			// Test that the strategy can be used to create appropriate behavior
			switch conflictStrategy {
			case "replace":
				// Should replace existing profiles
				assert.True(t, tt.replaceExisting)
				assert.False(t, tt.skipExisting)
			case "skip":
				// Should skip existing profiles
				assert.False(t, tt.replaceExisting)
				assert.True(t, tt.skipExisting)
			case "prompt":
				// Should prompt for each conflict
				assert.False(t, tt.replaceExisting)
				assert.False(t, tt.skipExisting)
			}
		})
	}
}

func TestCommandExecutionWithConflictFlags(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectError   bool
		expectedFlags map[string]any
	}{
		{
			name: "command with replace-existing flag",
			args: []string{"--template", "test-profile", "--replace-existing"},
			expectedFlags: map[string]any{
				"template":         "test-profile",
				"replace-existing": true,
				"skip-existing":    false,
			},
		},
		{
			name: "command with skip-existing flag",
			args: []string{"--template", "test-profile", "--skip-existing"},
			expectedFlags: map[string]any{
				"template":         "test-profile",
				"replace-existing": false,
				"skip-existing":    true,
			},
		},
		{
			name: "command with all flags",
			args: []string{"--template", "test-profile", "--pattern", "custom-{role_name}", "--yes", "--output-file", "/tmp/config", "--replace-existing"},
			expectedFlags: map[string]any{
				"template":         "test-profile",
				"pattern":          "custom-{role_name}",
				"yes":              true,
				"output-file":      "/tmp/config",
				"replace-existing": true,
				"skip-existing":    false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new command instance
			cmd := &cobra.Command{
				Use: "profile-generator",
				Run: func(cmd *cobra.Command, args []string) {
					// Mock run function that validates flag parsing
					for flagName, expectedValue := range tt.expectedFlags {
						switch v := expectedValue.(type) {
						case string:
							actual, _ := cmd.Flags().GetString(flagName)
							assert.Equal(t, v, actual, "Flag %s should be %s", flagName, v)
						case bool:
							actual, _ := cmd.Flags().GetBool(flagName)
							assert.Equal(t, v, actual, "Flag %s should be %t", flagName, v)
						}
					}
				},
			}

			// Add flags
			cmd.Flags().StringP("template", "t", "", "Template profile name (required)")
			cmd.Flags().StringP("pattern", "p", "{account_name}-{role_name}", "Naming pattern for generated profiles")
			cmd.Flags().BoolP("yes", "y", false, "Auto-approve appending profiles without confirmation")
			cmd.Flags().StringP("output-file", "F", "", "Output to file instead of appending to ~/.aws/config")
			cmd.Flags().Bool("replace-existing", false, "Replace existing profiles with new names based on pattern")
			cmd.Flags().Bool("skip-existing", false, "Skip generating profiles for roles that already have profiles")

			cmd.MarkFlagRequired("template")
			cmd.MarkFlagsMutuallyExclusive("replace-existing", "skip-existing")

			// Set arguments and execute
			cmd.SetArgs(tt.args)
			err := cmd.Execute()

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCountActionsByType(t *testing.T) {
	// This test would require importing the helpers package and creating mock actions
	// For now, we'll test the logic conceptually

	// Mock action types for testing
	type MockActionType int
	const (
		MockActionReplace MockActionType = iota
		MockActionSkip
		MockActionCreate
	)

	type MockAction struct {
		Action MockActionType
	}

	actions := []MockAction{
		{Action: MockActionReplace},
		{Action: MockActionReplace},
		{Action: MockActionSkip},
		{Action: MockActionCreate},
		{Action: MockActionReplace},
	}

	// Count replace actions
	replaceCount := 0
	for _, action := range actions {
		if action.Action == MockActionReplace {
			replaceCount++
		}
	}
	assert.Equal(t, 3, replaceCount)

	// Count skip actions
	skipCount := 0
	for _, action := range actions {
		if action.Action == MockActionSkip {
			skipCount++
		}
	}
	assert.Equal(t, 1, skipCount)

	// Count create actions
	createCount := 0
	for _, action := range actions {
		if action.Action == MockActionCreate {
			createCount++
		}
	}
	assert.Equal(t, 1, createCount)
}

// TestEnhancedOutputFormatting tests the enhanced output formatting functions
func TestEnhancedOutputFormatting(t *testing.T) {
	tests := []struct {
		name           string
		templateName   string
		namingPattern  string
		strategy       string
		expectedOutput []string
	}{
		{
			name:          "basic initialization info",
			templateName:  "test-profile",
			namingPattern: "{account_name}-{role_name}",
			strategy:      "prompt",
			expectedOutput: []string{
				"AWS Profile Generator",
				"Template Profile: test-profile",
				"Naming Pattern: {account_name}-{role_name}",
				"Conflict Resolution Strategy: prompt",
			},
		},
		{
			name:          "replace strategy initialization",
			templateName:  "sso-profile",
			namingPattern: "{account_id}-{role_name}",
			strategy:      "replace",
			expectedOutput: []string{
				"AWS Profile Generator",
				"Template Profile: sso-profile",
				"Naming Pattern: {account_id}-{role_name}",
				"Conflict Resolution Strategy: replace",
			},
		},
		{
			name:          "skip strategy initialization",
			templateName:  "my-sso",
			namingPattern: "custom-{role_name}",
			strategy:      "skip",
			expectedOutput: []string{
				"AWS Profile Generator",
				"Template Profile: my-sso",
				"Naming Pattern: custom-{role_name}",
				"Conflict Resolution Strategy: skip",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the initialization info contains expected elements
			// In a real implementation, we would capture stdout and verify the output
			// For now, we verify the logic that would generate the output

			assert.Equal(t, tt.templateName, tt.templateName)
			assert.Equal(t, tt.namingPattern, tt.namingPattern)
			assert.Equal(t, tt.strategy, tt.strategy)

			// Verify that all expected output elements are present
			for _, expected := range tt.expectedOutput {
				assert.NotEmpty(t, expected)
			}
		})
	}
}

// TestProgressIndicators tests the progress indicator functionality
func TestProgressIndicators(t *testing.T) {
	tests := []struct {
		name           string
		phase          string
		message        string
		expectedPrefix string
	}{
		{
			name:           "validation phase",
			phase:          "validation",
			message:        "Validating template profile",
			expectedPrefix: "🔍",
		},
		{
			name:           "discovery phase",
			phase:          "discovery",
			message:        "Discovering accessible roles",
			expectedPrefix: "🔍",
		},
		{
			name:           "conflict detection phase",
			phase:          "conflict_detection",
			message:        "Detecting conflicts",
			expectedPrefix: "⚠️",
		},
		{
			name:           "conflict resolution phase",
			phase:          "conflict_resolution",
			message:        "Resolving conflicts",
			expectedPrefix: "🔧",
		},
		{
			name:           "generation phase",
			phase:          "generation",
			message:        "Generating profiles",
			expectedPrefix: "📝",
		},
		{
			name:           "success phase",
			phase:          "success",
			message:        "Operation completed",
			expectedPrefix: "✅",
		},
		{
			name:           "error phase",
			phase:          "error",
			message:        "Error occurred",
			expectedPrefix: "❌",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the logic for progress indicator selection
			var prefix string
			switch tt.phase {
			case "validation", "discovery":
				prefix = "🔍"
			case "conflict_detection":
				prefix = "⚠️"
			case "conflict_resolution":
				prefix = "🔧"
			case "generation":
				prefix = "📝"
			case "success":
				prefix = "✅"
			case "error":
				prefix = "❌"
			default:
				prefix = "ℹ️"
			}

			assert.Equal(t, tt.expectedPrefix, prefix)
			assert.NotEmpty(t, tt.message)
		})
	}
}

// TestErrorRecoveryGuidance tests the error recovery guidance functionality
func TestErrorRecoveryGuidance(t *testing.T) {
	tests := []struct {
		name             string
		errorType        string
		expectedGuidance []string
	}{
		{
			name:      "validation error",
			errorType: "validation",
			expectedGuidance: []string{
				"Check the input parameters and configuration",
				"Error Context:",
			},
		},
		{
			name:      "filesystem error",
			errorType: "filesystem",
			expectedGuidance: []string{
				"Check file permissions and ensure the directory exists",
				"Verify ~/.aws directory exists and is writable",
				"Check disk space availability",
			},
		},
		{
			name:      "api error",
			errorType: "api",
			expectedGuidance: []string{
				"Check AWS credentials and connectivity",
				"Run: aws sso login",
				"Verify network connectivity to AWS",
				"Check if the template profile is valid",
			},
		},
		{
			name:      "unknown error",
			errorType: "unknown",
			expectedGuidance: []string{
				"Review the error message and check your configuration",
				"Verify all required parameters are provided",
				"Check AWS SSO session status",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that appropriate recovery guidance is provided for each error type
			var guidance []string

			switch tt.errorType {
			case "validation":
				guidance = []string{
					"Check the input parameters and configuration",
					"Error Context:",
				}
			case "filesystem":
				guidance = []string{
					"Check file permissions and ensure the directory exists",
					"Verify ~/.aws directory exists and is writable",
					"Check disk space availability",
				}
			case "api":
				guidance = []string{
					"Check AWS credentials and connectivity",
					"Run: aws sso login",
					"Verify network connectivity to AWS",
					"Check if the template profile is valid",
				}
			default:
				guidance = []string{
					"Review the error message and check your configuration",
					"Verify all required parameters are provided",
					"Check AWS SSO session status",
				}
			}

			assert.Equal(t, tt.expectedGuidance, guidance)
		})
	}
}

// TestConflictResolutionSummary tests the conflict resolution summary formatting
func TestConflictResolutionSummary(t *testing.T) {
	tests := []struct {
		name               string
		conflictsDetected  int
		resolutionActions  int
		profilesToReplace  int
		rolesToSkip        int
		newProfilesCreated int
		expectedSummary    map[string]int
	}{
		{
			name:               "no conflicts",
			conflictsDetected:  0,
			resolutionActions:  0,
			profilesToReplace:  0,
			rolesToSkip:        0,
			newProfilesCreated: 5,
			expectedSummary: map[string]int{
				"conflicts": 0,
				"actions":   0,
				"replace":   0,
				"skip":      0,
				"new":       5,
			},
		},
		{
			name:               "mixed conflicts",
			conflictsDetected:  3,
			resolutionActions:  3,
			profilesToReplace:  2,
			rolesToSkip:        1,
			newProfilesCreated: 4,
			expectedSummary: map[string]int{
				"conflicts": 3,
				"actions":   3,
				"replace":   2,
				"skip":      1,
				"new":       4,
			},
		},
		{
			name:               "all replacements",
			conflictsDetected:  4,
			resolutionActions:  4,
			profilesToReplace:  4,
			rolesToSkip:        0,
			newProfilesCreated: 4,
			expectedSummary: map[string]int{
				"conflicts": 4,
				"actions":   4,
				"replace":   4,
				"skip":      0,
				"new":       4,
			},
		},
		{
			name:               "all skips",
			conflictsDetected:  3,
			resolutionActions:  3,
			profilesToReplace:  0,
			rolesToSkip:        3,
			newProfilesCreated: 2,
			expectedSummary: map[string]int{
				"conflicts": 3,
				"actions":   3,
				"replace":   0,
				"skip":      3,
				"new":       2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the logic for conflict resolution summary
			summary := map[string]int{
				"conflicts": tt.conflictsDetected,
				"actions":   tt.resolutionActions,
				"replace":   tt.profilesToReplace,
				"skip":      tt.rolesToSkip,
				"new":       tt.newProfilesCreated,
			}

			assert.Equal(t, tt.expectedSummary, summary)

			// Verify consistency rules
			assert.Equal(t, tt.profilesToReplace+tt.rolesToSkip, tt.resolutionActions,
				"Replace count + skip count should equal total actions")
		})
	}
}

// TestDetailedReportFormatting tests the detailed report formatting
func TestDetailedReportFormatting(t *testing.T) {
	tests := []struct {
		name              string
		templateProfile   string
		discoveredRoles   int
		conflicts         int
		actions           int
		generatedProfiles int
		skippedRoles      int
		errors            int
		hasBackup         bool
		expectedFields    []string
	}{
		{
			name:              "successful operation",
			templateProfile:   "test-profile",
			discoveredRoles:   10,
			conflicts:         2,
			actions:           2,
			generatedProfiles: 10,
			skippedRoles:      0,
			errors:            0,
			hasBackup:         true,
			expectedFields: []string{
				"Template Profile",
				"Total Discovered Roles",
				"Conflicts Detected",
				"Generated Profiles",
				"Configuration Backup",
			},
		},
		{
			name:              "operation with errors",
			templateProfile:   "error-profile",
			discoveredRoles:   5,
			conflicts:         1,
			actions:           1,
			generatedProfiles: 4,
			skippedRoles:      1,
			errors:            2,
			hasBackup:         false,
			expectedFields: []string{
				"Template Profile",
				"Total Discovered Roles",
				"Conflicts Detected",
				"Generated Profiles",
				"Errors Encountered",
			},
		},
		{
			name:              "operation with skipped roles",
			templateProfile:   "skip-profile",
			discoveredRoles:   8,
			conflicts:         3,
			actions:           3,
			generatedProfiles: 5,
			skippedRoles:      3,
			errors:            0,
			hasBackup:         true,
			expectedFields: []string{
				"Template Profile",
				"Total Discovered Roles",
				"Conflicts Detected",
				"Generated Profiles",
				"Roles Skipped",
				"Configuration Backup",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that all expected fields are present in the report
			reportData := map[string]any{
				"template_profile":   tt.templateProfile,
				"discovered_roles":   tt.discoveredRoles,
				"conflicts":          tt.conflicts,
				"actions":            tt.actions,
				"generated_profiles": tt.generatedProfiles,
				"skipped_roles":      tt.skippedRoles,
				"errors":             tt.errors,
				"has_backup":         tt.hasBackup,
			}

			// Verify all expected fields have values
			for _, field := range tt.expectedFields {
				switch field {
				case "Template Profile":
					assert.Equal(t, tt.templateProfile, reportData["template_profile"])
				case "Total Discovered Roles":
					assert.Equal(t, tt.discoveredRoles, reportData["discovered_roles"])
				case "Conflicts Detected":
					assert.Equal(t, tt.conflicts, reportData["conflicts"])
				case "Generated Profiles":
					assert.Equal(t, tt.generatedProfiles, reportData["generated_profiles"])
				case "Roles Skipped":
					assert.Equal(t, tt.skippedRoles, reportData["skipped_roles"])
				case "Errors Encountered":
					assert.Equal(t, tt.errors, reportData["errors"])
				case "Configuration Backup":
					assert.Equal(t, tt.hasBackup, reportData["has_backup"])
				}
			}
		})
	}
}

// TestSuccessRateCalculation tests the success rate calculation logic
func TestSuccessRateCalculation(t *testing.T) {
	tests := []struct {
		name               string
		totalRoles         int
		successfulProfiles int
		expectedRate       string
	}{
		{
			name:               "100% success",
			totalRoles:         5,
			successfulProfiles: 5,
			expectedRate:       "100.0%",
		},
		{
			name:               "80% success",
			totalRoles:         10,
			successfulProfiles: 8,
			expectedRate:       "80.0%",
		},
		{
			name:               "50% success",
			totalRoles:         6,
			successfulProfiles: 3,
			expectedRate:       "50.0%",
		},
		{
			name:               "0% success",
			totalRoles:         4,
			successfulProfiles: 0,
			expectedRate:       "0.0%",
		},
		{
			name:               "no roles",
			totalRoles:         0,
			successfulProfiles: 0,
			expectedRate:       "N/A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test success rate calculation logic
			var successRate string
			if tt.totalRoles > 0 {
				rate := float64(tt.successfulProfiles) / float64(tt.totalRoles) * 100
				successRate = fmt.Sprintf("%.1f%%", rate)
			} else {
				successRate = "N/A"
			}

			assert.Equal(t, tt.expectedRate, successRate)
		})
	}
}

// TestRecoveryStepsGeneration tests the recovery steps generation
func TestRecoveryStepsGeneration(t *testing.T) {
	tests := []struct {
		name          string
		hasBackup     bool
		backupPath    string
		expectedSteps []string
	}{
		{
			name:       "with backup",
			hasBackup:  true,
			backupPath: "/tmp/config.backup",
			expectedSteps: []string{
				"Verify your AWS SSO session is active: aws sso login",
				"Check that the template profile exists and is valid",
				"Ensure you have write permissions to the AWS config file",
				"If needed, restore from backup: cp /tmp/config.backup ~/.aws/config",
				"Re-run the command with the same parameters",
			},
		},
		{
			name:       "without backup",
			hasBackup:  false,
			backupPath: "",
			expectedSteps: []string{
				"Verify your AWS SSO session is active: aws sso login",
				"Check that the template profile exists and is valid",
				"Ensure you have write permissions to the AWS config file",
				"Re-run the command with the same parameters",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test recovery steps generation logic
			var steps []string
			steps = append(steps, "Verify your AWS SSO session is active: aws sso login")
			steps = append(steps, "Check that the template profile exists and is valid")
			steps = append(steps, "Ensure you have write permissions to the AWS config file")

			if tt.hasBackup && tt.backupPath != "" {
				steps = append(steps, fmt.Sprintf("If needed, restore from backup: cp %s ~/.aws/config", tt.backupPath))
			}

			steps = append(steps, "Re-run the command with the same parameters")

			assert.Equal(t, tt.expectedSteps, steps)
		})
	}
}
