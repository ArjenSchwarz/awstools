package cmd

import (
	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/ArjenSchwarz/go-output/drawio"
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
	Run: ssoOverviewByPermissionSet,
}

func init() {
	ssoCmd.AddCommand(ssoOverviewByPermissionSetCmd)
	ssoOverviewByPermissionSetCmd.Flags().StringVarP(&ssoresourceid, "resource-id", "r", "", "The permission set name or arn you want to limit to")
}

func ssoOverviewByPermissionSet(_ *cobra.Command, _ []string) {
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "SSO Overview per permission set"
	ssoInstance := helpers.GetSSOAccountInstance(awsConfig.SsoClient())
	keys := []string{"PermissionSet", "AccountID", "Principal"}
	if settings.IsVerbose() {
		keys = append(keys, "ManagedPolicies", "InlinePolicy")
	}
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = resultTitle
	output.Settings.SortKey = "PermissionSet"
	switch {
	case settings.IsDrawIO():
		output.Settings.DrawIOHeader = createSSOPermissionsetsDrawIOHeader()
		createSSOPermissionsetsDrawIOContents(ssoInstance, &output)
	case output.Settings.NeedsFromToColumns():
		output.Settings.AddFromToColumns("DrawIOID", "Children")
		createSSOPermissionsetsDrawIOContents(ssoInstance, &output)
	default:
		for _, permissionset := range ssoInstance.PermissionSets {
			if !filteredSSOPermissionSet(permissionset) {
				continue
			}
			for _, account := range permissionset.Accounts {
				for _, assignment := range account.AccountAssignments {
					content := make(map[string]interface{})
					content["PermissionSet"] = assignment.PermissionSet.Name
					content["AccountID"] = getName(account.AccountID)
					content["Principal"] = getName(assignment.PrincipalID)
					if settings.IsVerbose() {
						content["ManagedPolicies"] = assignment.PermissionSet.GetManagedPolicyNames()
						content["InlinePolicy"] = assignment.PermissionSet.InlinePolicy
					}
					holder := format.OutputHolder{Contents: content}
					output.AddHolder(holder)
				}
			}
		}
	}
	output.Write()
}

func filteredSSOPermissionSet(permissionset helpers.SSOPermissionSet) bool {
	if ssoresourceid == "" ||
		ssoresourceid == permissionset.Arn ||
		ssoresourceid == permissionset.Name {
		return true
	}
	return false
}

func createSSOPermissionsetsDrawIOHeader() drawio.Header {
	drawioheader := drawio.DefaultHeader()
	drawioheader.SetHeightAndWidth("78", "78")
	drawioheader.SetLayout(drawio.LayoutHorizontalTree)
	connection := drawio.NewConnection()
	connection.Invert = false
	connection.From = "Children"
	connection.To = "DrawIOID"
	drawioheader.AddConnection(connection)
	return drawioheader
}

func createSSOPermissionsetsDrawIOContents(instance helpers.SSOInstance, output *format.OutputArray) {
	output.Keys = []string{"Name", "DrawIOID", "Type", "Children", "Image"}

	content := make(map[string]interface{})
	content["Name"] = getName(instance.Arn)
	content["DrawIOID"] = getName(instance.Arn)
	content["Type"] = "SSO"
	content["Image"] = drawio.AWSShape("Security Identity Compliance", "Single Sign-On")
	content["Children"] = instance.GetPermissionSetList()
	holder := format.OutputHolder{Contents: content}
	output.AddHolder(holder)
	uniquefilter := []string{}
	for _, permissionset := range instance.PermissionSets {
		if !filteredSSOPermissionSet(permissionset) {
			continue
		}
		permchildren := []string{}
		content := make(map[string]interface{})
		content["Name"] = getName(permissionset.Name)
		content["DrawIOID"] = getName(permissionset.Name)
		content["Type"] = "PermissionSet"
		content["Image"] = drawio.AWSShape("Security Identity Compliance", "Permissions")
		for _, account := range permissionset.Accounts {
			permchildren = append(permchildren, account.AccountID+permissionset.Name)
		}
		content["Children"] = permchildren
		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
		for _, account := range permissionset.Accounts {
			content := make(map[string]interface{})
			content["Name"] = getName(account.AccountID)
			content["DrawIOID"] = account.AccountID + permissionset.Name
			content["Type"] = "Account"
			content["Image"] = drawio.AWSShape("Security Identity Compliance", "Organizations Account")
			content["Children"] = account.GetPrincipalIDsForPermissionSet(permissionset)
			holder := format.OutputHolder{Contents: content}
			output.AddHolder(holder)
			for _, assignment := range account.AccountAssignments {
				if assignment.PermissionSet.Name == permissionset.Name {
					if !contains(uniquefilter, assignment.PrincipalID) {
						uniquefilter = append(uniquefilter, assignment.PrincipalID)
						content := make(map[string]interface{})
						content["Name"] = getName(assignment.PrincipalID)
						content["DrawIOID"] = assignment.PrincipalID
						content["Type"] = assignment.PrincipalType
						switch assignment.PrincipalType {
						case "USER":
							content["Image"] = drawio.AWSShape("General Resources", "User")
						case "GROUP":
							content["Image"] = drawio.AWSShape("General Resources", "Users")
						}
						holder := format.OutputHolder{Contents: content}
						output.AddHolder(holder)
					}
				}
			}
		}
	}
}
