package cmd

import "github.com/spf13/cobra"

// cfnCmd represents the cfn command
var cfnCmd = &cobra.Command{
	Use:   "cfn",
	Short: "CloudFormation commands",
	Long:  `This lets you run various CloudFormation related commands, please look at the options available.`,
}

var stackname *string

func init() {
	rootCmd.AddCommand(cfnCmd)
	stackname = cfnCmd.PersistentFlags().StringP("stack", "s", "", "The name of the stack")
	// Every cfn subcommand operates on a specific stack, so require the flag.
	// Cobra rejects the command with a usage error before the Run function (and
	// thus any AWS config loading or API call) executes. See T-1382.
	if err := cfnCmd.MarkPersistentFlagRequired("stack"); err != nil {
		panic(err)
	}
}
