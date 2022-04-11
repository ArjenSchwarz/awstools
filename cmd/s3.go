package cmd

import "github.com/spf13/cobra"

// cfnCmd represents the cfn command
var s3Cmd = &cobra.Command{
	Use:   "s3",
	Short: "S3 commands",
	Long:  `This lets you run various S3 related commands, please look at the options available.`,
}

func init() {
	rootCmd.AddCommand(s3Cmd)
}
