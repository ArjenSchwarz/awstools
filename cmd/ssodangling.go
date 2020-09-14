package cmd

import (
	"strings"

	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/spf13/cobra"
)

// ssoDanglingCmd represents the sso Dangling command
var ssoDanglingCmd = &cobra.Command{
	Use:   "dangling",
	Short: "An overview of unassigned permission sets",
	Long: `Lists all permission sets that aren't assigned to an account
	`,
	Run: ssoDangling,
}

func init() {
	ssoCmd.AddCommand(ssoDanglingCmd)
}

func ssoDangling(cmd *cobra.Command, args []string) {
	resultTitle := "Dangling Permission Sets"
	svc := helpers.SSOSession()
	ssoInstance := helpers.GetSSOAccountInstance(svc)
	keys := []string{"PermissionSet", "ManagedPolicies", "InlinePolicy"}
	output := helpers.OutputArray{Keys: keys, Title: resultTitle}
	stringSeparator := ", "
	for _, permissionset := range ssoInstance.PermissionSets {
		if len(permissionset.Accounts) == 0 {
			content := make(map[string]string)
			content["PermissionSet"] = permissionset.Name
			content["ManagedPolicies"] = strings.Join(permissionset.GetManagedPolicyNames(), stringSeparator)
			content["InlinePolicy"] = permissionset.InlinePolicy
			holder := helpers.OutputHolder{Contents: content}
			output.AddHolder(holder)
		}
	}
	output.Write(*settings)
}