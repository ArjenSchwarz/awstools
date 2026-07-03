package cmd

import (
	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/cobra"
)

// ssoOverviewByAccountCmd represents the sso OverviewByAccount command
var ssoOverviewByPermissionSetCmd = &cobra.Command{
	Use:   "by-permission-set",
	Short: "A basic overview of the SSO Config Permission Sets grouped by permission set",
	Long: `Provides an overview of all the permission sets and assignments attached to an account,
	grouped by permission set.

You can filter the output to a single permission set by supplying the --resource-id (-r) flag with the permission set name or arn.

Verbose mode will add the policies for the permissionsets in the textual output formats drawio output will generate a graph that goes SSO Instance -> Permission Sets -> Accounts -> User/Group. You may notice the same accounts shown multiple times, this is to improve readability not a bug. dot output is currently limited as it shows internal names only
	`,
	RunE: ssoOverviewByPermissionSet,
}

func init() {
	ssoCmd.AddCommand(ssoOverviewByPermissionSetCmd)
	ssoOverviewByPermissionSetCmd.Flags().StringVarP(&ssoresourceid, "resource-id", "r", "", "The permission set name or arn you want to limit to")
}

func ssoOverviewByPermissionSet(cmd *cobra.Command, _ []string) error {
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "SSO Overview per permission set"
	ssoInstance, err := helpers.GetSSOAccountInstance(awsConfig.SsoClient())
	if err != nil {
		return err
	}
	keys := []string{permissionSetColumn, accountIDColumn, "Principal"}
	if settings.IsVerbose() {
		keys = append(keys, "ManagedPolicies", "InlinePolicy")
	}
	rows := []map[string]any{}
	for _, permissionset := range ssoInstance.PermissionSets {
		if !filteredSSOPermissionSet(permissionset) {
			continue
		}
		for _, account := range permissionset.Accounts {
			for _, assignment := range account.AccountAssignments {
				content := make(map[string]any)
				content[permissionSetColumn] = assignment.PermissionSet.Name
				content[accountIDColumn] = getName(account.AccountID)
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
			Table(resultTitle, rows, output.WithKeys(keys...), config.SortOption(permissionSetColumn)).
			Build(),
	}
	if settings.NeedsGraphFormat() || settings.IsDrawIO() {
		records := createSSOPermissionsetsDrawIOContents(ssoInstance)
		if settings.NeedsGraphFormat() {
			graphRows := make([]map[string]any, len(records))
			for index, record := range records {
				graphRows[index] = record
			}
			docs.Graph = output.New().
				Graph(resultTitle, graphEdges(graphRows, drawIOIDColumn, childrenColumn)).
				Build()
		}
		if settings.IsDrawIO() {
			docs.DrawIO = output.New().
				DrawIO(resultTitle, records, createSSOPermissionsetsDrawIOHeader()).
				Build()
		}
	}
	return settings.RenderDocuments(cmd.Context(), docs)
}

func filteredSSOPermissionSet(permissionset helpers.SSOPermissionSet) bool {
	if ssoresourceid == "" ||
		ssoresourceid == permissionset.Arn ||
		ssoresourceid == permissionset.Name {
		return true
	}
	return false
}

func createSSOPermissionsetsDrawIOHeader() output.DrawIOHeader {
	drawioheader := output.DefaultDrawIOHeader()
	drawioheader.Height = "78"
	drawioheader.Width = "78"
	drawioheader.Layout = output.DrawIOLayoutHorizontalTree
	connection := drawIOConnection()
	connection.Invert = false
	connection.From = childrenColumn
	connection.To = drawIOIDColumn
	drawioheader.Connections = append(drawioheader.Connections, connection)
	return drawioheader
}

func createSSOPermissionsetsDrawIOContents(instance helpers.SSOInstance) []output.Record {
	records := []output.Record{}
	content := make(map[string]any)
	content[nameColumn] = getName(instance.Arn)
	content[drawIOIDColumn] = getName(instance.Arn)
	content[typeColumn] = "SSO"
	content[imageColumn] = awsShape("Security Identity Compliance", "Single Sign-On")
	content[childrenColumn] = instance.GetPermissionSetList()
	records = append(records, content)
	uniquefilter := []string{}
	for _, permissionset := range instance.PermissionSets {
		if !filteredSSOPermissionSet(permissionset) {
			continue
		}
		permchildren := []string{}
		content := make(map[string]any)
		content[nameColumn] = getName(permissionset.Name)
		content[drawIOIDColumn] = getName(permissionset.Name)
		content[typeColumn] = permissionSetColumn
		content[imageColumn] = awsShape("Security Identity Compliance", "Permissions")
		for _, account := range permissionset.Accounts {
			permchildren = append(permchildren, account.AccountID+permissionset.Name)
		}
		content[childrenColumn] = permchildren
		records = append(records, content)
		for _, account := range permissionset.Accounts {
			content := make(map[string]any)
			content[nameColumn] = getName(account.AccountID)
			content[drawIOIDColumn] = account.AccountID + permissionset.Name
			content[typeColumn] = accountColumn
			content[imageColumn] = awsShape("Security Identity Compliance", "Organizations Account")
			content[childrenColumn] = account.GetPrincipalIDsForPermissionSet(permissionset)
			records = append(records, content)
			for _, assignment := range account.AccountAssignments {
				if assignment.PermissionSet.Name == permissionset.Name {
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
		}
	}
	return records
}
