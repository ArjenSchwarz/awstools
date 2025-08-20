package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/spf13/cobra"
)

// ssoCmd represents the tgw command
var ssoCmd = &cobra.Command{
	Use:   "sso",
	Short: "AWS Single Sign-On commands",
	Long:  `Various AWS SSO commands`,
}

// profileGeneratorCmd represents the profile-generator command with enhanced conflict resolution
var profileGeneratorCmd = &cobra.Command{
	Use:   "profile-generator",
	Short: "Generate AWS CLI profiles for all assumable roles with intelligent conflict resolution",
	Long: `Generate AWS CLI profiles for all assumable roles in AWS IAM Identity Center with
intelligent conflict detection and resolution capabilities.

OVERVIEW:
This command uses an existing SSO profile as a template to discover all accessible accounts
and permission sets through the AWS SSO API, then generates corresponding AWS CLI profiles
using a configurable naming pattern. The enhanced version includes sophisticated conflict
detection and resolution to handle existing profiles gracefully.

CONFLICT RESOLUTION:
When existing profiles are found that correspond to the same AWS roles or use the same
profile names, you can control how conflicts are resolved:

• Default behavior (no flags): Interactive prompts for each conflict
  - Shows detailed conflict information
  - Allows per-conflict decision making
  - Provides cancel option to exit without changes

• --replace-existing: Automatically replace existing profiles
  - Removes old profiles and creates new ones with pattern-based names
  - Preserves custom configuration properties when possible
  - Creates automatic backup before making changes

• --skip-existing: Skip roles that already have profiles
  - Preserves all existing profile configurations
  - Only generates profiles for roles without existing profiles
  - Provides summary of skipped roles

NAMING PATTERNS:
The naming pattern supports the following placeholders:
• {account_name} - AWS account name (from Organizations API)
• {account_id} - 12-digit AWS account ID
• {account_alias} - Account alias if available
• {role_name} - Permission set name (e.g., "AdministratorAccess")
• {region} - AWS region from template profile

SUPPORTED PROFILE FORMATS:
• Legacy SSO format: Direct sso_account_id and sso_role_name properties
• Modern SSO format: sso_session references with separate session configuration
• Mixed environments: Handles both formats transparently

SAFETY FEATURES:
• Automatic backup creation before any modifications
• File locking to prevent concurrent access issues
• Atomic operations with rollback on failures
• Comprehensive error recovery and guidance
• Detailed operation logging and progress reporting

EXAMPLES:

Basic Usage:
  # Generate profiles using 'my-sso-profile' as template
  awstools sso profile-generator --template my-sso-profile

Custom Naming:
  # Use account ID instead of account name in profile names
  awstools sso profile-generator --template my-sso-profile --pattern "{account_id}-{role_name}"
  
  # Use shorter pattern for concise profile names
  awstools sso profile-generator --template my-sso-profile --pattern "{account_name}-{role_name}"

Conflict Resolution:
  # Replace existing profiles with new names based on pattern
  awstools sso profile-generator --template my-sso-profile --replace-existing
  
  # Skip roles that already have profiles (preserve existing)
  awstools sso profile-generator --template my-sso-profile --skip-existing
  
  # Interactive mode with per-conflict decisions (default)
  awstools sso profile-generator --template my-sso-profile

Automation:
  # Auto-approve without confirmation prompts
  awstools sso profile-generator --template my-sso-profile --yes
  
  # Combine auto-approve with conflict resolution
  awstools sso profile-generator --template my-sso-profile --replace-existing --yes

Custom Output:
  # Output to a specific file instead of ~/.aws/config
  awstools sso profile-generator --template my-sso-profile --output-file /path/to/custom-config
  
  # Generate profiles for review without modifying config
  awstools sso profile-generator --template my-sso-profile --output-file ./preview-profiles.txt

Advanced Scenarios:
  # Standardize existing profile names using new pattern
  awstools sso profile-generator --template legacy-sso --pattern "aws-{account_name}-{role_name}" --replace-existing
  
  # Add new roles while preserving existing profiles
  awstools sso profile-generator --template my-sso --skip-existing --yes

BEST PRACTICES:

Profile Name Standardization:
• Use consistent naming patterns across your organization
• Include account identifiers to avoid confusion in multi-account setups
• Consider role-based naming for easier identification
• Example patterns:
  - "{account_name}-{role_name}" (default, human-readable)
  - "{account_id}-{role_name}" (unique, shorter)
  - "aws-{account_name}-{role_name}" (prefixed for organization)

Conflict Resolution Strategy:
• Use --skip-existing for additive operations (only add new profiles)
• Use --replace-existing for standardization (update existing profile names)
• Use interactive mode (default) for mixed scenarios requiring decisions
• Always review changes before applying with --yes flag

Safety and Recovery:
• Automatic backups are created before any modifications
• Backup files are stored with timestamp: ~/.aws/config.backup.YYYYMMDD-HHMMSS
• Test with --output-file first to preview changes
• Keep your original config file backed up separately

Performance Optimization:
• Use specific naming patterns to reduce conflicts
• Run during off-peak hours for large organizations
• Consider batching operations by account or role type

TROUBLESHOOTING:

Common Issues and Solutions:
• "Template profile not found"
  → Run: aws configure list-profiles
  → Verify profile name spelling and existence

• "SSO session expired"
  → Run: aws sso login
  → Check session validity: aws sts get-caller-identity --profile <template>

• "Permission denied on config file"
  → Check permissions: ls -la ~/.aws/config
  → Fix permissions: chmod 600 ~/.aws/config

• "No accessible roles found"
  → Verify SSO permissions in AWS console
  → Check account access through SSO portal
  → Ensure template profile has proper SSO configuration

• "Profile conflicts detected"
  → Use --replace-existing to update existing profiles
  → Use --skip-existing to preserve existing profiles
  → Use interactive mode to decide per conflict

Recovery Procedures:
• Restore from automatic backup: cp ~/.aws/config.backup.* ~/.aws/config
• Verify restored config: aws configure list-profiles
• Test profile functionality: aws sts get-caller-identity --profile <name>

Advanced Usage:
• Custom output locations: --output-file /path/to/custom-config
• Automation scripts: --yes --replace-existing for unattended operation
• Preview mode: --output-file /tmp/preview.txt (review before applying)
• Batch processing: Use shell scripts to process multiple templates

For more information:
• AWS SSO Configuration: https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sso.html
• AWS CLI Profiles: https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html
• SSO Session Management: https://docs.aws.amazon.com/cli/latest/userguide/sso-configure-profile-token.html`,
	Run: profileGenerator,
}

func init() {
	rootCmd.AddCommand(ssoCmd)
	ssoCmd.AddCommand(profileGeneratorCmd)

	// Core flags for profile generation
	profileGeneratorCmd.Flags().StringP("template", "t", "", "Template profile name (required) - must be an existing SSO profile")
	profileGeneratorCmd.Flags().StringP("pattern", "p", "{account_name}-{role_name}", "Naming pattern for generated profiles (supports placeholders: {account_name}, {account_id}, {account_alias}, {role_name}, {region})")
	profileGeneratorCmd.Flags().BoolP("yes", "y", false, "Auto-approve appending profiles without confirmation prompts")
	profileGeneratorCmd.Flags().StringP("output-file", "F", "", "Output to file instead of appending to ~/.aws/config (useful for preview or custom locations)")

	// Enhanced conflict resolution flags with detailed descriptions
	profileGeneratorCmd.Flags().Bool("replace-existing", false, "Replace existing profiles with new names based on pattern (creates automatic backup for safety)")
	profileGeneratorCmd.Flags().Bool("skip-existing", false, "Skip generating profiles for roles that already have profiles (preserves existing configurations completely)")

	// Mark template flag as required since it's essential for operation
	_ = profileGeneratorCmd.MarkFlagRequired("template")

	// Ensure conflict resolution flags are mutually exclusive to prevent ambiguous behavior
	profileGeneratorCmd.MarkFlagsMutuallyExclusive("replace-existing", "skip-existing")
}

var ssoresourceid string

// profileGenerator implements the profile-generator command
func profileGenerator(cmd *cobra.Command, _ []string) {
	// Parse command line flags
	templateProfile, _ := cmd.Flags().GetString("template")
	namingPattern, _ := cmd.Flags().GetString("pattern")
	autoApprove, _ := cmd.Flags().GetBool("yes")
	outputFile, _ := cmd.Flags().GetString("output-file")
	replaceExisting, _ := cmd.Flags().GetBool("replace-existing")
	skipExisting, _ := cmd.Flags().GetBool("skip-existing")

	// Determine conflict resolution strategy
	var conflictStrategy helpers.ConflictResolutionStrategy
	switch {
	case replaceExisting:
		conflictStrategy = helpers.ConflictReplace
	case skipExisting:
		conflictStrategy = helpers.ConflictSkip
	default:
		conflictStrategy = helpers.ConflictPrompt
	}

	// Get AWS config
	awsConfig := config.DefaultAwsConfig(*settings)

	// Create profile generator
	generator, err := helpers.NewProfileGenerator(templateProfile, namingPattern, autoApprove, outputFile, conflictStrategy, awsConfig.Config)
	if err != nil {
		displayErrorWithRecovery("Error creating profile generator", err)
		os.Exit(1)
	}

	// Display initialization information
	displayInitializationInfo(templateProfile, namingPattern, conflictStrategy)

	// Execute the complete workflow with enhanced progress reporting
	result, err := executeProfileGenerationWorkflow(generator)
	if err != nil {
		displayErrorWithRecovery("Error during profile generation workflow", err)
		os.Exit(1)
	}

	// Display comprehensive results
	displayComprehensiveResults(result, generator, autoApprove)
}

// confirmProfileAddition asks user to confirm adding profiles
func confirmProfileAddition(profiles []helpers.GeneratedProfile) bool {
	fmt.Printf("\nReady to add %d profiles to AWS CLI configuration.\n", len(profiles))
	fmt.Print("Do you want to continue? (y/N): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// countActionsByType counts the number of actions of a specific type
func countActionsByType(actions []helpers.ConflictAction, actionType helpers.ActionType) int {
	count := 0
	for _, action := range actions {
		if action.Action == actionType {
			count++
		}
	}
	return count
}

// displayInitializationInfo shows initialization details with progress indicators
func displayInitializationInfo(templateProfile, namingPattern string, strategy helpers.ConflictResolutionStrategy) {
	fmt.Println("AWS Profile Generator")
	fmt.Println("====================")
	fmt.Printf("Template Profile: %s\n", templateProfile)
	fmt.Printf("Naming Pattern: %s\n", namingPattern)
	fmt.Printf("Conflict Resolution Strategy: %s\n", strategy.String())
	fmt.Println()
}

// executeProfileGenerationWorkflow runs the workflow with progress indicators
func executeProfileGenerationWorkflow(generator *helpers.ProfileGenerator) (*helpers.ProfileGenerationResult, error) {
	fmt.Println("🔍 Phase 1: Validating template profile...")

	// Execute the complete workflow with progress tracking
	result, err := generator.GenerateProfilesWorkflow()
	if err != nil {
		return result, err
	}

	// Display phase completion
	fmt.Printf("✅ Template profile validated: %s\n", result.TemplateProfile.Name)
	fmt.Printf("🔍 Phase 2: Discovering accessible roles... Found %d roles\n", len(result.DiscoveredRoles))

	if len(result.DetectedConflicts) > 0 {
		fmt.Printf("⚠️  Phase 3: Conflict detection... Found %d conflicts\n", len(result.DetectedConflicts))
		fmt.Printf("🔧 Phase 4: Resolving conflicts... %d actions taken\n", len(result.ResolutionActions))
	} else {
		fmt.Println("✅ Phase 3: No conflicts detected")
	}

	fmt.Printf("📝 Phase 5: Profile generation complete... %d profiles ready\n", len(result.GeneratedProfiles))
	fmt.Println()

	return result, nil
}

// displayComprehensiveResults shows detailed results with enhanced formatting
func displayComprehensiveResults(result *helpers.ProfileGenerationResult, generator *helpers.ProfileGenerator, autoApprove bool) {
	// Display conflict resolution summary if conflicts were detected
	if len(result.DetectedConflicts) > 0 {
		displayConflictResolutionSummary(result)
	}

	// Display detailed conflict report
	if len(result.ResolutionActions) > 0 {
		displayDetailedConflictReport(result, generator)
	}

	// Handle profile confirmation and application
	if !autoApprove && len(result.GeneratedProfiles) > 0 {
		if !confirmProfileAddition(result.GeneratedProfiles) {
			fmt.Println("❌ Profile generation cancelled by user.")
			return
		}

		fmt.Println("💾 Applying profiles to AWS configuration...")
		if err := generator.AppendToConfig(result.GeneratedProfiles); err != nil {
			displayErrorWithRecovery("Error appending profiles to config", err)
			os.Exit(1)
		}

		// Update result to mark profiles as successful
		for _, profile := range result.GeneratedProfiles {
			result.SuccessfulProfiles = append(result.SuccessfulProfiles, profile.Name)
		}
		fmt.Printf("✅ Successfully applied %d profiles to configuration\n", len(result.SuccessfulProfiles))
	}

	// Display results using the enhanced output format
	displayEnhancedProfileResults(result)

	// Display final comprehensive summary
	displayFinalOperationSummary(result, generator.GetConflictStrategy())

	// Display recovery information if needed
	displayRecoveryInformation(result)
}

// displayConflictResolutionSummary shows a summary of conflict detection and resolution
func displayConflictResolutionSummary(result *helpers.ProfileGenerationResult) {
	fmt.Println("Conflict Resolution Summary")
	fmt.Println("===========================")
	fmt.Printf("Conflicts Detected: %d\n", len(result.DetectedConflicts))
	fmt.Printf("Resolution Actions: %d\n", len(result.ResolutionActions))

	replaceCount := countActionsByType(result.ResolutionActions, helpers.ActionReplace)
	skipCount := countActionsByType(result.ResolutionActions, helpers.ActionSkip)

	fmt.Printf("  • Profiles to Replace: %d\n", replaceCount)
	fmt.Printf("  • Roles to Skip: %d\n", skipCount)
	fmt.Printf("  • New Profiles Created: %d\n", len(result.GeneratedProfiles))
	fmt.Println()
}

// displayDetailedConflictReport shows detailed information about conflict resolution
func displayDetailedConflictReport(result *helpers.ProfileGenerationResult, generator *helpers.ProfileGenerator) {
	conflictReport := generator.GenerateConflictReport(result.DetectedConflicts, &helpers.ConflictResolutionResult{
		GeneratedProfiles: result.GeneratedProfiles,
		SkippedRoles:      result.SkippedRoles,
		Actions:           result.ResolutionActions,
	})
	fmt.Println("Detailed Conflict Report")
	fmt.Println("========================")
	fmt.Println(conflictReport)
}

// displayEnhancedProfileResults displays the generation results with enhanced formatting
func displayEnhancedProfileResults(result *helpers.ProfileGenerationResult) {
	if len(result.GeneratedProfiles) == 0 {
		fmt.Println("ℹ️  No profiles were generated.")
		return
	}

	// Create enhanced output for generated profiles
	keys := []string{"ProfileName", "Account", "Role", "Region", "Format", "Status", "Action"}
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = "Generated AWS CLI Profiles"
	output.Settings.SortKey = "ProfileName"

	for _, profile := range result.GeneratedProfiles {
		content := make(map[string]any)
		content["ProfileName"] = profile.Name
		content["Account"] = fmt.Sprintf("%s (%s)", profile.AccountName, profile.AccountID)
		content["Role"] = profile.RoleName
		content["Region"] = profile.Region

		formatType := "SSO Session"
		if profile.IsLegacy {
			formatType = "Legacy SSO"
		}
		content["Format"] = formatType

		// Determine status and action based on conflict resolution
		status := "New"
		action := "Created"

		for _, resolutionAction := range result.ResolutionActions {
			if resolutionAction.Action == helpers.ActionReplace && resolutionAction.NewName == profile.Name {
				status = "Replaced"
				action = fmt.Sprintf("Replaced %s", resolutionAction.OldName)
				break
			}
		}

		content["Status"] = status
		content["Action"] = action

		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}

	output.Write()

	// Display skipped roles if any
	if len(result.SkippedRoles) > 0 {
		fmt.Printf("\nSkipped Roles (%d)\n", len(result.SkippedRoles))
		fmt.Println("================")
		for _, role := range result.SkippedRoles {
			fmt.Printf("  • %s in %s (%s) - existing profile preserved\n",
				role.PermissionSetName, role.AccountName, role.AccountID)
		}
		fmt.Println()
	}

	// Display replaced profiles if any
	if len(result.ReplacedProfiles) > 0 {
		fmt.Printf("\nReplaced Profiles (%d)\n", len(result.ReplacedProfiles))
		fmt.Println("====================")
		for _, replacement := range result.ReplacedProfiles {
			fmt.Printf("  • %s → %s\n", replacement.OldName, replacement.NewName)
		}
		fmt.Println()
	}
}

// displayFinalOperationSummary displays a comprehensive final summary
func displayFinalOperationSummary(result *helpers.ProfileGenerationResult, strategy helpers.ConflictResolutionStrategy) {
	fmt.Println("Final Operation Summary")
	fmt.Println("=======================")
	fmt.Printf("Template Profile: %s\n", result.TemplateProfile.Name)
	fmt.Printf("Conflict Resolution Strategy: %s\n", strategy.String())
	fmt.Printf("Total Discovered Roles: %d\n", len(result.DiscoveredRoles))
	fmt.Printf("Conflicts Detected: %d\n", len(result.DetectedConflicts))
	fmt.Printf("Profiles Generated: %d\n", len(result.GeneratedProfiles))
	fmt.Printf("Profiles Successfully Applied: %d\n", len(result.SuccessfulProfiles))
	fmt.Printf("Roles Skipped: %d\n", len(result.SkippedRoles))

	if len(result.Errors) > 0 {
		fmt.Printf("❌ Errors Encountered: %d\n", len(result.Errors))
	} else {
		fmt.Println("✅ Operation completed successfully")
	}

	if result.BackupPath != "" {
		fmt.Printf("💾 Configuration Backup: %s\n", result.BackupPath)
	}

	// Display specific conflict resolution results
	if len(result.ResolutionActions) > 0 {
		replaceCount := countActionsByType(result.ResolutionActions, helpers.ActionReplace)
		skipCount := countActionsByType(result.ResolutionActions, helpers.ActionSkip)

		if replaceCount > 0 {
			fmt.Printf("🔄 Profiles Replaced: %d\n", replaceCount)
		}
		if skipCount > 0 {
			fmt.Printf("⏭️  Roles Skipped Due to Conflicts: %d\n", skipCount)
		}
	}
	fmt.Println()
}

// displayRecoveryInformation shows recovery guidance when errors occur
func displayRecoveryInformation(result *helpers.ProfileGenerationResult) {
	if len(result.Errors) == 0 {
		return
	}

	fmt.Println("Error Details and Recovery Guidance")
	fmt.Println("===================================")

	for i, err := range result.Errors {
		fmt.Printf("%d. %s\n", i+1, err.Error())

		// Provide specific recovery guidance based on error type
		switch err.Type {
		case helpers.ErrorTypeValidation:
			fmt.Println("   💡 Recovery: Check the input parameters and try again")
			if err.Context != nil {
				fmt.Println("   📋 Context:")
				for key, value := range err.Context {
					fmt.Printf("      %s: %v\n", key, value)
				}
			}
		case helpers.ErrorTypeFileSystem:
			fmt.Println("   💡 Recovery: Check file permissions and disk space")
			if result.BackupPath != "" {
				fmt.Printf("   🔄 Restore from backup: %s\n", result.BackupPath)
			}
		case helpers.ErrorTypeAPI:
			fmt.Println("   💡 Recovery: Check AWS credentials and network connectivity")
			fmt.Println("   🔄 Try running: aws sso login")
		default:
			fmt.Println("   💡 Recovery: Review the error message and check your configuration")
		}
		fmt.Println()
	}

	// General recovery guidance
	fmt.Println("General Recovery Steps:")
	fmt.Println("1. Verify your AWS SSO session is active: aws sso login")
	fmt.Println("2. Check that the template profile exists and is valid")
	fmt.Println("3. Ensure you have write permissions to the AWS config file")
	if result.BackupPath != "" {
		fmt.Printf("4. If needed, restore from backup: cp %s ~/.aws/config\n", result.BackupPath)
	}
	fmt.Println("5. Re-run the command with the same parameters")
	fmt.Println()
}

// displayErrorWithRecovery shows an error with comprehensive contextual recovery information
func displayErrorWithRecovery(message string, err error) {
	fmt.Fprintf(os.Stderr, "❌ %s: %v\n", message, err)

	// Provide specific recovery guidance based on error type
	if pgErr, ok := err.(helpers.ProfileGeneratorError); ok {
		switch pgErr.Type {
		case helpers.ErrorTypeValidation:
			fmt.Fprintln(os.Stderr, "💡 Recovery: Check the input parameters and configuration")
			fmt.Fprintln(os.Stderr, "   Common validation issues:")
			fmt.Fprintln(os.Stderr, "   - Template profile name is empty or doesn't exist")
			fmt.Fprintln(os.Stderr, "   - Template profile is not configured for SSO")
			fmt.Fprintln(os.Stderr, "   - Naming pattern contains invalid placeholders")
			fmt.Fprintln(os.Stderr, "   - Conflicting flags provided (--replace-existing and --skip-existing)")
			if pgErr.Context != nil {
				fmt.Fprintln(os.Stderr, "📋 Error Context:")
				for key, value := range pgErr.Context {
					fmt.Fprintf(os.Stderr, "   %s: %v\n", key, value)
				}
			}
		case helpers.ErrorTypeFileSystem:
			fmt.Fprintln(os.Stderr, "💡 Recovery: Check file permissions and ensure the directory exists")
			fmt.Fprintln(os.Stderr, "   File system troubleshooting:")
			fmt.Fprintln(os.Stderr, "   - Verify ~/.aws directory exists: mkdir -p ~/.aws")
			fmt.Fprintln(os.Stderr, "   - Check file permissions: ls -la ~/.aws/config")
			fmt.Fprintln(os.Stderr, "   - Ensure config file is writable: chmod 600 ~/.aws/config")
			fmt.Fprintln(os.Stderr, "   - Check disk space availability: df -h ~")
			fmt.Fprintln(os.Stderr, "   - Verify file ownership: ls -la ~/.aws/")
		case helpers.ErrorTypeAPI:
			fmt.Fprintln(os.Stderr, "💡 Recovery: Check AWS credentials and connectivity")
			fmt.Fprintln(os.Stderr, "   AWS API troubleshooting:")
			fmt.Fprintln(os.Stderr, "   - Refresh SSO session: aws sso login")
			fmt.Fprintln(os.Stderr, "   - Verify network connectivity to AWS")
			fmt.Fprintln(os.Stderr, "   - Check template profile configuration: aws configure list --profile <template-name>")
			fmt.Fprintln(os.Stderr, "   - Test SSO access: aws sts get-caller-identity --profile <template-name>")
			fmt.Fprintln(os.Stderr, "   - Verify SSO session is not expired")
		default:
			fmt.Fprintln(os.Stderr, "💡 Recovery: Review the error message and check your configuration")
			fmt.Fprintln(os.Stderr, "   General troubleshooting:")
			fmt.Fprintln(os.Stderr, "   - Verify all required parameters are provided")
			fmt.Fprintln(os.Stderr, "   - Check AWS SSO session status")
			fmt.Fprintln(os.Stderr, "   - Review command syntax and flag usage")
		}
	} else {
		fmt.Fprintln(os.Stderr, "💡 Recovery: Review the error message and check your configuration")
		fmt.Fprintln(os.Stderr, "   Basic troubleshooting steps:")
		fmt.Fprintln(os.Stderr, "   - Verify all required parameters are provided")
		fmt.Fprintln(os.Stderr, "   - Check AWS SSO session status: aws sso login")
		fmt.Fprintln(os.Stderr, "   - Ensure template profile exists and is valid")
		fmt.Fprintln(os.Stderr, "   - Review command documentation: awstools sso profile-generator --help")
	}

	// Add common troubleshooting section
	fmt.Fprintln(os.Stderr, "\n🔧 Common Solutions:")
	fmt.Fprintln(os.Stderr, "   1. Refresh your SSO session: aws sso login")
	fmt.Fprintln(os.Stderr, "   2. Verify template profile: aws configure list --profile <template-name>")
	fmt.Fprintln(os.Stderr, "   3. Check file permissions: ls -la ~/.aws/config")
	fmt.Fprintln(os.Stderr, "   4. Review command syntax: awstools sso profile-generator --help")
	fmt.Fprintln(os.Stderr, "   5. Test with minimal flags first: awstools sso profile-generator --template <name>")
}
