package cmd

import (
	"strings"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/spf13/cobra"
)

// ssoOverviewByAccountCmd represents the sso OverviewByAccount command
var ssoListPermissionSetsCmd = &cobra.Command{
	Use:   "list-permission-sets",
	Short: "A list of the SSO Permission Sets",
	Long: `Provides an overview of all the permission sets and their attached policies

	You can filter the output to a single permission set by supplying the --resource-id (-r) flag with the
	permission set name or arn.
	`,
	Run: ssoListPermissionSets,
}

func init() {
	ssoCmd.AddCommand(ssoListPermissionSetsCmd)
}

func ssoListPermissionSets(cmd *cobra.Command, args []string) {
	resultTitle := "SSO Overview per permission set"
	svc := helpers.SSOSession(config.DefaultAwsConfig())
	ssoInstance := helpers.GetSSOAccountInstance(svc)
	keys := []string{"PermissionSet", "AccountIDs", "ManagedPolicies", "InlinePolicy"}
	if *settings.Verbose {
		keys = append(keys, "ManagedPolicies", "InlinePolicy")
	}
	output := helpers.OutputArray{Keys: keys, Title: resultTitle}
	output.SortKey = "PermissionSet"
	stringSeparator := ", "

	for _, permissionset := range ssoInstance.PermissionSets {
		permchildren := []string{}
		content := make(map[string]string)
		content["PermissionSet"] = permissionset.Name
		content["ManagedPolicies"] = strings.Join(permissionset.GetManagedPolicyNames(), stringSeparator)
		content["InlinePolicy"] = permissionset.InlinePolicy
		for _, account := range permissionset.Accounts {
			permchildren = append(permchildren, getName(account.AccountID))
		}
		content["AccountIDs"] = strings.Join(permchildren, stringSeparator)
		holder := helpers.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	output.Write(*settings)
}
