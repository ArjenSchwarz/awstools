package cmd

import (
	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/ArjenSchwarz/go-output/drawio"
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
	Run: ssoOverviewByAccount,
}

func init() {
	ssoCmd.AddCommand(ssoOverviewByAccountCmd)
	ssoOverviewByAccountCmd.Flags().StringVarP(&ssoresourceid, "resource-id", "r", "", "The account id (or account alias) you want to limit to")

}

func ssoOverviewByAccount(_ *cobra.Command, _ []string) {
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "SSO Overview per account"
	ssoInstance := helpers.GetSSOAccountInstance(awsConfig.SsoClient())
	keys := []string{"AccountID", permissionSetColumn, "Principal"}
	if settings.IsVerbose() {
		keys = append(keys, "ManagedPolicies", "InlinePolicy")
	}
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = resultTitle
	output.Settings.SortKey = "AccountID"
	switch {
	case settings.IsDrawIO():
		output.Settings.DrawIOHeader = createSSOAccountsDrawIOHeader()
		createSSOAccountDrawIOContents(ssoInstance, &output)
	case output.Settings.NeedsFromToColumns():
		output.Settings.AddFromToColumns("DrawIOID", "Children")
		createSSOAccountDrawIOContents(ssoInstance, &output)
	default:
		for _, account := range ssoInstance.Accounts {
			if filteredSSOAccount(account) {
				for _, assignment := range account.AccountAssignments {
					content := make(map[string]interface{})
					content["AccountID"] = getName(account.AccountID)
					content[permissionSetColumn] = assignment.PermissionSet.Name
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

func filteredSSOAccount(account helpers.SSOAccount) bool {
	if ssoresourceid == "" ||
		ssoresourceid == account.AccountID ||
		ssoresourceid == getName(account.AccountID) {
		return true
	}
	return false
}

func createSSOAccountsDrawIOHeader() drawio.Header {
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

func createSSOAccountDrawIOContents(instance helpers.SSOInstance, output *format.OutputArray) {
	output.Keys = []string{"Name", "DrawIOID", "Type", "Children", "Image"}

	content := make(map[string]interface{})
	content["Name"] = getName(instance.Arn)
	content["DrawIOID"] = getName(instance.Arn)
	content["Type"] = "SSO"
	content["Image"] = drawio.AWSShape("Security Identity Compliance", "Single Sign-On")
	content["Children"] = instance.GetAccountList()
	holder := format.OutputHolder{Contents: content}
	output.AddHolder(holder)
	uniquefilter := []string{}
	for _, account := range instance.Accounts {
		if !filteredSSOAccount(account) {
			continue
		}
		accountchildren := []string{}
		content := make(map[string]interface{})
		content["Name"] = getName(account.AccountID)
		content["DrawIOID"] = account.AccountID
		content["Type"] = "Account"
		content["Image"] = drawio.AWSShape("Security Identity Compliance", "Organizations Account")
		for _, assignment := range account.AccountAssignments {
			accountchildren = append(accountchildren, assignment.PermissionSet.Name+account.AccountID)
		}
		content["Children"] = unique(accountchildren)
		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
		for _, assignment := range account.AccountAssignments {
			if !contains(uniquefilter, assignment.PermissionSet.Name+account.AccountID) {
				uniquefilter = append(uniquefilter, assignment.PermissionSet.Name+account.AccountID)
				content := make(map[string]interface{})
				content["Name"] = getName(assignment.PermissionSet.Name)
				content["DrawIOID"] = getName(assignment.PermissionSet.Name + account.AccountID)
				content["Type"] = permissionSetColumn
				content["Image"] = drawio.AWSShape("Security Identity Compliance", "Permissions")
				content["Children"] = assignment.PermissionSet.GetAssignmentIDsByAccount(account.AccountID)
				holder := format.OutputHolder{Contents: content}
				output.AddHolder(holder)
			}
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
