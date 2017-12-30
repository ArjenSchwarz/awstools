package cmd

import "github.com/spf13/cobra"

// iamCmd represents the iam command
var iamCmd = &cobra.Command{
	Use:   "iam",
	Short: "IAM commands",
	Long:  `Various commands that deal with IAM users`,
}

func init() {
	RootCmd.AddCommand(iamCmd)
}
