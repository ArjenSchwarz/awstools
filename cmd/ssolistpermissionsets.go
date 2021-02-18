package cmd

import (
	"fmt"
	"strings"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/spf13/cobra"
)

// ssoOverviewByAccountCmd represents the sso OverviewByAccount command
var ssoListPermissionSetsCmd = &cobra.Command{
	Use:   "list-permission-sets",
	Short: "A list of the SSO Permission Sets",
	Long: `Provides an overview of all the permission sets and their attached policies and deployed accounts

By default this command gives an output showing the number of managed policies attached and whether it has an inline policy. To expand this and see the details, use the --verbose (-v) flag.
	`,
	Run: ssoListPermissionSets,
}

func init() {
	ssoCmd.AddCommand(ssoListPermissionSetsCmd)
}

func ssoListPermissionSets(cmd *cobra.Command, args []string) {
	awsConfig := config.DefaultAwsConfig()
	resultTitle := "SSO Overview per permission set"
	ssoInstance := helpers.GetSSOAccountInstance(awsConfig.SsoClient())
	keys := []string{"PermissionSet", "AccountIDs", "Arn", "ManagedPolicies", "InlinePolicy"}
	output := helpers.OutputArray{Keys: keys, Title: resultTitle}
	output.SortKey = "PermissionSet"
	stringSeparator := ", "

	for _, permissionset := range ssoInstance.PermissionSets {
		permchildren := []string{}
		content := make(map[string]string)
		content["PermissionSet"] = permissionset.Name
		content["Arn"] = permissionset.Arn
		if *settings.Verbose {
			content["ManagedPolicies"] = strings.Join(permissionset.GetManagedPolicyNames(), stringSeparator)
			content["InlinePolicy"] = permissionset.InlinePolicy
		} else {
			content["ManagedPolicies"] = fmt.Sprint(len(permissionset.GetManagedPolicyNames()))
			inlinePolicy := "False"
			if permissionset.InlinePolicy != "" {
				inlinePolicy = "True"
			}
			content["InlinePolicy"] = inlinePolicy
		}
		for _, account := range permissionset.Accounts {
			permchildren = append(permchildren, getName(account.AccountID))
		}
		content["AccountIDs"] = strings.Join(permchildren, stringSeparator)
		holder := helpers.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	output.Write(*settings)
}
