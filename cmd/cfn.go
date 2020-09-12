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
}
