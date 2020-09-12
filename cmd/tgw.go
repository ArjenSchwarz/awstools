package cmd

import (
	"github.com/spf13/cobra"
)

// tgwCmd represents the tgw command
var tgwCmd = &cobra.Command{
	Use:   "tgw",
	Short: "Transit Gateway commands",
	Long:  `Various Transit Gateway commands`,
}

func init() {
	rootCmd.AddCommand(tgwCmd)
}
