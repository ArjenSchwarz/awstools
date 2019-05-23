package cmd

import (
	"github.com/spf13/cobra"
)

// organizationsCmd represents the organizations command
var organizationsCmd = &cobra.Command{
	Use:   "organizations",
	Short: "Organizations related functions",
	Long:  `Functionalities related to AWS Organizations`,
}

func init() {
	RootCmd.AddCommand(organizationsCmd)
}
