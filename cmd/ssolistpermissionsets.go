package cmd

import (
	"fmt"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/cobra"
)

// ssoOverviewByAccountCmd represents the sso OverviewByAccount command
var ssoListPermissionSetsCmd = &cobra.Command{
	Use:   "list-permission-sets",
	Short: "A list of the SSO Permission Sets",
	Long: `Provides an overview of all the permission sets and their attached policies and deployed accounts

By default this command gives an output showing the number of managed policies attached and whether it has an inline policy. To expand this and see the details, use the --verbose (-v) flag.
	`,
	RunE: ssoListPermissionSets,
}

func init() {
	ssoCmd.AddCommand(ssoListPermissionSetsCmd)
}

func ssoListPermissionSets(cmd *cobra.Command, _ []string) error {
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "SSO Overview per permission set"
	ssoInstance, err := helpers.GetSSOAccountInstance(awsConfig.SsoClient())
	if err != nil {
		return err
	}
	keys := []string{permissionSetColumn, "AccountIDs", "Arn", "ManagedPolicies", "InlinePolicy"}
	rows := []map[string]any{}

	for _, permissionset := range ssoInstance.PermissionSets {
		permchildren := []string{}
		content := make(map[string]any)
		content[permissionSetColumn] = permissionset.Name
		content["Arn"] = permissionset.Arn
		if settings.IsVerbose() {
			content["ManagedPolicies"] = permissionset.GetManagedPolicyNames()
			content["InlinePolicy"] = permissionset.InlinePolicy
		} else {
			content["ManagedPolicies"] = fmt.Sprint(len(permissionset.GetManagedPolicyNames()))
			inlinePolicy := false
			if permissionset.InlinePolicy != "" {
				inlinePolicy = true
			}
			content["InlinePolicy"] = inlinePolicy
		}
		for _, account := range permissionset.Accounts {
			permchildren = append(permchildren, getName(account.AccountID))
		}
		content["AccountIDs"] = permchildren
		rows = append(rows, content)
	}
	doc := output.New().
		Table(resultTitle, rows, output.WithKeys(keys...), config.SortOption(permissionSetColumn)).
		Build()
	return settings.RenderDocument(cmd.Context(), doc)
}
