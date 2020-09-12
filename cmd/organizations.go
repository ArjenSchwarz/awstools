package cmd

import (
	"github.com/spf13/cobra"
)

// organizationsCmd represents the organizations command
var organizationsCmd = &cobra.Command{
	Use:   "organizations",
	Short: "AWS Organizations commands",
	Long:  `Functionalities related to AWS Organizations`,
}

func init() {
	rootCmd.AddCommand(organizationsCmd)
}
