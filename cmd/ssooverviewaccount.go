package cmd

import (
	"maps"
	"slices"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/cobra"
)

// ssoOverviewByAccountCmd represents the sso OverviewByAccount command
var ssoOverviewByAccountCmd = &cobra.Command{
	Use:   "by-account",
	Short: "A basic overview of the SSO Config Permission Sets by account",
	Long: `Provides an overview of all the permission sets and assignments attached to an account,
	grouped by account.

You can filter the output to a single account by supplying the --resource-id (-r) flag with the account ID or, if you use a name file, the account alias from the name file.

Verbose mode will add the policies for the permissionsets in the textual output formats drawio output will generate a graph that goes SSO Instance -> Accounts -> Permission Sets -> User/Group You may notice the same permission sets shown multiple times, this is to improve readability not a bug. dot output is currently limited as it shows internal names only
	`,
	RunE: ssoOverviewByAccount,
}

func init() {
	ssoCmd.AddCommand(ssoOverviewByAccountCmd)
	ssoOverviewByAccountCmd.Flags().StringVarP(&ssoresourceid, "resource-id", "r", "", "The account id (or account alias) you want to limit to")

}

func ssoOverviewByAccount(cmd *cobra.Command, _ []string) error {
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "SSO Overview per account"
	ssoInstance, err := helpers.GetSSOAccountInstance(awsConfig.SsoClient())
	if err != nil {
		return err
	}
	keys := []string{accountIDColumn, permissionSetColumn, "Principal"}
	if settings.IsVerbose() {
		keys = append(keys, "ManagedPolicies", "InlinePolicy")
	}
	rows := []map[string]any{}
	for _, accountID := range slices.Sorted(maps.Keys(ssoInstance.Accounts)) {
		account := ssoInstance.Accounts[accountID]
		if filteredSSOAccount(account) {
			for _, assignment := range account.AccountAssignments {
				content := make(map[string]any)
				content[accountIDColumn] = getName(account.AccountID)
				content[permissionSetColumn] = assignment.PermissionSet.Name
				content["Principal"] = getName(assignment.PrincipalID)
				if settings.IsVerbose() {
					content["ManagedPolicies"] = assignment.PermissionSet.GetManagedPolicyNames()
					content["InlinePolicy"] = assignment.PermissionSet.InlinePolicy
				}
				rows = append(rows, content)
			}
		}
	}
	docs := config.DocumentSet{
		Table: output.New().
			Table(resultTitle, rows, output.WithKeys(keys...), config.SortOption(accountIDColumn)).
			Build(),
	}
	if settings.NeedsGraphFormat() || settings.IsDrawIO() {
		records := createSSOAccountDrawIOContents(ssoInstance)
		if settings.NeedsGraphFormat() {
			docs.Graph = output.New().
				Graph(resultTitle, graphEdges(records, drawIOIDColumn, childrenColumn)).
				Build()
		}
		if settings.IsDrawIO() {
			docs.DrawIO = output.New().
				DrawIO(resultTitle, drawIORecords(records), createSSOOverviewDrawIOHeader()).
				Build()
		}
	}
	return settings.RenderDocuments(cmd.Context(), docs)
}

func filteredSSOAccount(account helpers.SSOAccount) bool {
	if ssoresourceid == "" ||
		ssoresourceid == account.AccountID ||
		ssoresourceid == getName(account.AccountID) {
		return true
	}
	return false
}

// createSSOOverviewDrawIOHeader is shared by the two sso overview commands,
// whose drawio output has the same shape: a horizontal tree connected from
// the Children column to the DrawioID column.
func createSSOOverviewDrawIOHeader() output.DrawIOHeader {
	drawioheader := drawIOBaseHeader("%Name%", "%Image%", imageColumn)
	drawioheader.Layout = output.DrawIOLayoutHorizontalTree
	connection := drawIOConnection()
	connection.Invert = false
	connection.From = childrenColumn
	connection.To = drawIOIDColumn
	drawioheader.Connections = append(drawioheader.Connections, connection)
	return drawioheader
}

func createSSOAccountDrawIOContents(instance helpers.SSOInstance) []output.Record {
	records := []output.Record{}
	content := make(map[string]any)
	content[nameColumn] = getName(instance.Arn)
	content[drawIOIDColumn] = getName(instance.Arn)
	content[typeColumn] = "SSO"
	content[imageColumn] = awsShape("Security Identity Compliance", "Single Sign-On")
	content[childrenColumn] = instance.GetAccountList()
	records = append(records, content)
	uniquefilter := []string{}
	for _, accountID := range slices.Sorted(maps.Keys(instance.Accounts)) {
		account := instance.Accounts[accountID]
		if !filteredSSOAccount(account) {
			continue
		}
		accountchildren := []string{}
		content := make(map[string]any)
		content[nameColumn] = getName(account.AccountID)
		content[drawIOIDColumn] = account.AccountID
		content[typeColumn] = accountColumn
		content[imageColumn] = awsShape("Security Identity Compliance", "Organizations Account")
		for _, assignment := range account.AccountAssignments {
			accountchildren = append(accountchildren, assignment.PermissionSet.Name+account.AccountID)
		}
		content[childrenColumn] = unique(accountchildren)
		records = append(records, content)
		for _, assignment := range account.AccountAssignments {
			if !contains(uniquefilter, assignment.PermissionSet.Name+account.AccountID) {
				uniquefilter = append(uniquefilter, assignment.PermissionSet.Name+account.AccountID)
				content := make(map[string]any)
				content[nameColumn] = getName(assignment.PermissionSet.Name)
				content[drawIOIDColumn] = getName(assignment.PermissionSet.Name + account.AccountID)
				content[typeColumn] = permissionSetColumn
				content[imageColumn] = awsShape("Security Identity Compliance", "Permissions")
				content[childrenColumn] = assignment.PermissionSet.GetAssignmentIDsByAccount(account.AccountID)
				records = append(records, content)
			}
			if !contains(uniquefilter, assignment.PrincipalID) {
				uniquefilter = append(uniquefilter, assignment.PrincipalID)
				content := make(map[string]any)
				content[nameColumn] = getName(assignment.PrincipalID)
				content[drawIOIDColumn] = assignment.PrincipalID
				content[typeColumn] = assignment.PrincipalType
				switch assignment.PrincipalType {
				case "USER":
					content[imageColumn] = awsShape("General Resources", "User")
				case "GROUP":
					content[imageColumn] = awsShape("General Resources", "Users")
				}
				records = append(records, content)
			}
		}
	}
	return records
}
