package cmd

import (
	"strings"

	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/spf13/cobra"
)

// ssoOverviewByAccountCmd represents the sso OverviewByAccount command
var ssoOverviewByAccountCmd = &cobra.Command{
	Use:   "by-account",
	Short: "A basic overview of the SSO Config Permission Sets by account",
	Long: `Provides an overview of all the permission sets and assignments attached to an account,
	grouped by account.

	Verbose mode will add the policies for the permissionsets.
	`,
	Run: ssoOverviewByAccount,
}

func init() {
	ssoCmd.AddCommand(ssoOverviewByAccountCmd)
}

func ssoOverviewByAccount(cmd *cobra.Command, args []string) {
	resultTitle := "SSO Overview per account"
	svc := helpers.SSOSession()
	ssoInstance := helpers.GetSSOAccountInstance(svc)
	keys := []string{"AccountID", "PermissionSet", "Principal"}
	if *settings.Verbose {
		keys = append(keys, "ManagedPolicies", "InlinePolicy")
	}
	output := helpers.OutputArray{Keys: keys, Title: resultTitle}
	stringSeparator := ", "
	for _, account := range ssoInstance.Accounts {
		for _, assignment := range account.AccountAssignments {
			content := make(map[string]string)
			content["AccountID"] = getName(account.AccountID)
			content["PermissionSet"] = assignment.PermissionSet.Name
			content["Principal"] = getName(assignment.PrincipalID)
			if *settings.Verbose {
				content["ManagedPolicies"] = strings.Join(assignment.PermissionSet.GetManagedPolicyNames(), stringSeparator)
				content["InlinePolicy"] = assignment.PermissionSet.InlinePolicy
			}
			holder := helpers.OutputHolder{Contents: content}
			output.AddHolder(holder)
		}
	}
	output.Write(*settings)
}
