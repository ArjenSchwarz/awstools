package cmd

import (
	"github.com/spf13/cobra"
)

// sgCmd represents the sg command
var sgCmd = &cobra.Command{
	Use:   "sg",
	Short: "Security Group commands",
	Long:  `Various security group related tasks`,
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
