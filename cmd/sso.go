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

// profileGeneratorCmd represents the profile-generator command
var profileGeneratorCmd = &cobra.Command{
	Use:   "profile-generator",
	Short: "Generate AWS CLI profiles for all assumable roles",
	Long: `Generate AWS CLI profiles for all assumable roles in AWS IAM Identity Center.

This command uses an existing SSO profile as a template to discover all accessible accounts
and permission sets, then generates corresponding AWS CLI profiles using a configurable
naming pattern.

When existing profiles are found for the same roles, you can control how conflicts are resolved:
- Default behavior: prompt for each conflict
- --replace-existing: replace existing profiles with new names based on the pattern
- --skip-existing: skip generating profiles for roles that already have profiles

Examples:
  # Generate profiles using 'my-sso-profile' as template
  awstools sso profile-generator --template my-sso-profile

  # Use custom naming pattern
  awstools sso profile-generator --template my-sso-profile --pattern "{account_id}-{role_name}"

  # Auto-approve without confirmation
  awstools sso profile-generator --template my-sso-profile --yes

  # Replace existing profiles with new names based on pattern
  awstools sso profile-generator --template my-sso-profile --replace-existing

  # Skip roles that already have profiles
  awstools sso profile-generator --template my-sso-profile --skip-existing

  # Output to a specific file instead of ~/.aws/config
  awstools sso profile-generator --template my-sso-profile --output-file /path/to/config
`,
	Run: profileGenerator,
}

func init() {
	rootCmd.AddCommand(ssoCmd)
	ssoCmd.AddCommand(profileGeneratorCmd)

	// Add flags for profile-generator command
	profileGeneratorCmd.Flags().StringP("template", "t", "", "Template profile name (required)")
	profileGeneratorCmd.Flags().StringP("pattern", "p", "{account_name}-{role_name}", "Naming pattern for generated profiles")
	profileGeneratorCmd.Flags().BoolP("yes", "y", false, "Auto-approve appending profiles without confirmation")
	profileGeneratorCmd.Flags().StringP("output-file", "F", "", "Output to file instead of appending to ~/.aws/config")

	// Conflict resolution flags
	profileGeneratorCmd.Flags().Bool("replace-existing", false, "Replace existing profiles with new names based on pattern")
	profileGeneratorCmd.Flags().Bool("skip-existing", false, "Skip generating profiles for roles that already have profiles")

	// Mark template flag as required
	profileGeneratorCmd.MarkFlagRequired("template")

	// Add mutual exclusion validation for conflict resolution flags
	profileGeneratorCmd.MarkFlagsMutuallyExclusive("replace-existing", "skip-existing")
}

var ssoresourceid string

// profileGenerator implements the profile-generator command
func profileGenerator(cmd *cobra.Command, args []string) {
	// Parse command line flags
	templateProfile, _ := cmd.Flags().GetString("template")
	namingPattern, _ := cmd.Flags().GetString("pattern")
	autoApprove, _ := cmd.Flags().GetBool("yes")
	outputFile, _ := cmd.Flags().GetString("output-file")
	replaceExisting, _ := cmd.Flags().GetBool("replace-existing")
	skipExisting, _ := cmd.Flags().GetBool("skip-existing")

	// Determine conflict resolution strategy
	var conflictStrategy helpers.ConflictResolutionStrategy
	if replaceExisting {
		conflictStrategy = helpers.ConflictReplace
	} else if skipExisting {
		conflictStrategy = helpers.ConflictSkip
	} else {
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

// displayProfileGenerationResults displays the generation results using go-output
func displayProfileGenerationResults(result *helpers.ProfileGenerationResult) {
	if len(result.GeneratedProfiles) == 0 {
		fmt.Println("No profiles were generated.")
		return
	}

	// Create output for generated profiles
	keys := []string{"ProfileName", "Account", "Role", "Region", "Format", "Status"}
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = "Generated AWS CLI Profiles"
	output.Settings.SortKey = "ProfileName"

	for _, profile := range result.GeneratedProfiles {
		content := make(map[string]any)
		content["ProfileName"] = profile.Name
		content["Account"] = fmt.Sprintf("%s (%s)", profile.AccountName, profile.AccountID)
		content["Role"] = profile.RoleName
		content["Region"] = profile.Region

		formatType := "New (sso_session)"
		if profile.IsLegacy {
			formatType = "Legacy (sso_account_id + sso_role_name)"
		}
		content["Format"] = formatType

		// Determine status based on whether this profile was part of conflict resolution
		status := "New"
		for _, action := range result.ResolutionActions {
			if action.Action == helpers.ActionReplace && action.NewName == profile.Name {
				status = "Replaced"
				break
			}
		}
		content["Status"] = status

		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}

	output.Write()
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

// displayErrorWithRecovery shows an error with contextual recovery information
func displayErrorWithRecovery(message string, err error) {
	fmt.Fprintf(os.Stderr, "❌ %s: %v\n", message, err)

	// Provide specific recovery guidance based on error type
	if pgErr, ok := err.(helpers.ProfileGeneratorError); ok {
		switch pgErr.Type {
		case helpers.ErrorTypeValidation:
			fmt.Fprintln(os.Stderr, "💡 Recovery: Check the input parameters and configuration")
			if pgErr.Context != nil {
				fmt.Fprintln(os.Stderr, "📋 Error Context:")
				for key, value := range pgErr.Context {
					fmt.Fprintf(os.Stderr, "   %s: %v\n", key, value)
				}
			}
		case helpers.ErrorTypeFileSystem:
			fmt.Fprintln(os.Stderr, "💡 Recovery: Check file permissions and ensure the directory exists")
			fmt.Fprintln(os.Stderr, "   - Verify ~/.aws directory exists and is writable")
			fmt.Fprintln(os.Stderr, "   - Check disk space availability")
		case helpers.ErrorTypeAPI:
			fmt.Fprintln(os.Stderr, "💡 Recovery: Check AWS credentials and connectivity")
			fmt.Fprintln(os.Stderr, "   - Run: aws sso login")
			fmt.Fprintln(os.Stderr, "   - Verify network connectivity to AWS")
			fmt.Fprintln(os.Stderr, "   - Check if the template profile is valid")
		default:
			fmt.Fprintln(os.Stderr, "💡 Recovery: Review the error message and check your configuration")
			fmt.Fprintln(os.Stderr, "   - Verify all required parameters are provided")
			fmt.Fprintln(os.Stderr, "   - Check AWS SSO session status")
		}
	} else {
		fmt.Fprintln(os.Stderr, "💡 Recovery: Review the error message and check your configuration")
		fmt.Fprintln(os.Stderr, "   - Verify all required parameters are provided")
		fmt.Fprintln(os.Stderr, "   - Check AWS SSO session status")
	}
}
