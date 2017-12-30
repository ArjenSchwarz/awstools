package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// sgCmd represents the sg command
var sgCmd = &cobra.Command{
	Use:   "sg",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Work your own magic here
		fmt.Println("sg called")
	},
}

var groupname *string
var tag *string
var vpc *string

func init() {
	RootCmd.AddCommand(sgCmd)
	groupname = sgCmd.PersistentFlags().StringP("groupname", "g", "", "The name of the securitygroup")
	tag = sgCmd.PersistentFlags().StringP("tag", "t", "", "key:value pair of tag value")
	vpc = sgCmd.PersistentFlags().String("vpc", "", "VPC Id")
}
