package cmd

import (
	"github.com/spf13/cobra"
)

// vpcCmd represents the vpc command
var vpcCmd = &cobra.Command{
	Use:   "vpc",
	Short: "VPC commands",
	Long:  `Commands related to a VPC`,
}

func init() {
	rootCmd.AddCommand(vpcCmd)
}
