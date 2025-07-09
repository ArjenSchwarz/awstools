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

Examples:
  # Generate profiles using 'my-sso-profile' as template
  awstools sso profile-generator --template my-sso-profile

  # Use custom naming pattern
  awstools sso profile-generator --template my-sso-profile --pattern "{account_id}-{role_name}"

  # Auto-approve without confirmation
  awstools sso profile-generator --template my-sso-profile --yes

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

	// Mark template flag as required
	profileGeneratorCmd.MarkFlagRequired("template")
}

var ssoresourceid string

// profileGenerator implements the profile-generator command
func profileGenerator(cmd *cobra.Command, args []string) {
	// Parse command line flags
	templateProfile, _ := cmd.Flags().GetString("template")
	namingPattern, _ := cmd.Flags().GetString("pattern")
	autoApprove, _ := cmd.Flags().GetBool("yes")
	outputFile, _ := cmd.Flags().GetString("output-file")

	// Get AWS config
	awsConfig := config.DefaultAwsConfig(*settings)

	// Create profile generator
	generator, err := helpers.NewProfileGenerator(templateProfile, namingPattern, autoApprove, outputFile, awsConfig.Config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating profile generator: %v\n", err)
		os.Exit(1)
	}

	// Execute the complete workflow
	result, err := generator.GenerateProfilesWorkflow()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating profiles: %v\n", err)
		os.Exit(1)
	}

	// If not auto-approved, ask for user confirmation
	if !autoApprove && len(result.GeneratedProfiles) > 0 {
		if !confirmProfileAddition(result.GeneratedProfiles) {
			fmt.Println("Profile generation cancelled by user.")
			return
		}

		// Append profiles to config
		if err := generator.AppendToConfig(result.GeneratedProfiles); err != nil {
			fmt.Fprintf(os.Stderr, "Error appending profiles to config: %v\n", err)
			os.Exit(1)
		}

		// Update result to mark profiles as successful
		for _, profile := range result.GeneratedProfiles {
			result.SuccessfulProfiles = append(result.SuccessfulProfiles, profile.Name)
		}
	}

	// Display results using the output format
	displayProfileGenerationResults(result)

	// Display summary
	summary := generator.GetProfileGenerationSummary(result)
	fmt.Println("\n" + summary)
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
	keys := []string{"ProfileName", "Account", "Role", "Region", "Format"}
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = "Generated AWS CLI Profiles"
	output.Settings.SortKey = "ProfileName"

	for _, profile := range result.GeneratedProfiles {
		content := make(map[string]interface{})
		content["ProfileName"] = profile.Name
		content["Account"] = fmt.Sprintf("%s (%s)", profile.AccountName, profile.AccountID)
		content["Role"] = profile.RoleName
		content["Region"] = profile.Region

		formatType := "New (sso_session)"
		if profile.IsLegacy {
			formatType = "Legacy (sso_account_id + sso_role_name)"
		}
		content["Format"] = formatType

		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}

	output.Write()
}
