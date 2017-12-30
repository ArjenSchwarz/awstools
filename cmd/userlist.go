package cmd

import (
	"strings"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/spf13/cobra"
)

// userlistCmd represents the userlist command
var userlistCmd = &cobra.Command{
	Use:   "userlist",
	Short: "Get an overview of the IAM users in the account",
	Long: `Retrieves a list of all IAM users in the account, complete with the groups they are in and the policies they have through either the group or directly.

The verbose option will add the details of the policies to the output.`,
	Run: detailUsers,
}

func detailUsers(cmd *cobra.Command, args []string) {
	userlist := helpers.GetUserDetails(config.DefaultAwsConfig())

	keys := []string{"User", "Groups", "Policy Names"}
	if *settings.Verbose {
		keys = append(keys, "Policies")
	}
	output := helpers.OutputArray{Keys: keys}
	for _, user := range userlist {
		content := make(map[string]string)
		content["User"] = user.Username
		content["Groups"] = strings.Join(user.Groups, ", ")
		policyNames := make([]string, 0, len(user.GetAllPolicies()))
		policyDetails := make([]string, 0, len(user.GetAllPolicies()))
		for policyname, policydetail := range user.GetAllPolicies() {
			policyNames = append(policyNames, policyname)
			policyDetails = append(policyDetails, policydetail)
		}
		content["Policy Names"] = strings.Join(policyNames, ", ")
		if *settings.Verbose {
			content["Policies"] = strings.Join(policyDetails, ",\n")
		}
		holder := helpers.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	output.Write(*settings)
}

func init() {
	iamCmd.AddCommand(userlistCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// userlistCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// userlistCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}
