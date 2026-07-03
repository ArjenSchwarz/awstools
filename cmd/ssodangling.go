package cmd

import (
	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/cobra"
)

// ssoDanglingCmd represents the sso Dangling command
var ssoDanglingCmd = &cobra.Command{
	Use:   "dangling",
	Short: "An overview of unassigned permission sets",
	Long: `Lists all permission sets that aren't assigned to an account

Includes full details on the managed and inline policies.`,
	RunE: ssoDangling,
}

func init() {
	ssoCmd.AddCommand(ssoDanglingCmd)
}

func ssoDangling(cmd *cobra.Command, _ []string) error {
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "Dangling Permission Sets"
	ssoInstance, err := helpers.GetSSOAccountInstance(awsConfig.SsoClient())
	if err != nil {
		return err
	}
	keys := []string{permissionSetColumn, "Arn", "ManagedPolicies", "InlinePolicy"}
	rows := []map[string]any{}
	for _, permissionset := range ssoInstance.PermissionSets {
		if len(permissionset.Accounts) == 0 {
			content := make(map[string]any)
			content[permissionSetColumn] = permissionset.Name
			content["Arn"] = permissionset.Arn
			content["ManagedPolicies"] = permissionset.GetManagedPolicyNames()
			content["InlinePolicy"] = permissionset.InlinePolicy
			rows = append(rows, content)
		}
	}
	doc := output.New().
		Table(resultTitle, rows, output.WithKeys(keys...)).
		Build()
	return settings.RenderDocument(cmd.Context(), doc)
}
