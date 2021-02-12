package cmd

import (
	"github.com/spf13/cobra"
)

// genCmd represents the cfn command
var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate various useful things for awstools",
	Long:  `Generate documentation, CLI completions, and IAM policies`,
}

func init() {
	rootCmd.AddCommand(genCmd)
}
