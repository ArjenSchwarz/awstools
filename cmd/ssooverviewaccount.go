package cmd

import (
	"strings"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/drawio"
	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/spf13/cobra"
)

// ssoOverviewByAccountCmd represents the sso OverviewByAccount command
var ssoOverviewByAccountCmd = &cobra.Command{
	Use:   "by-account",
	Short: "A basic overview of the SSO Config Permission Sets by account",
	Long: `Provides an overview of all the permission sets and assignments attached to an account,
	grouped by account.

	Verbose mode will add the policies for the permissionsets in the textual output formats
	drawio output will generate a graph that goes SSO Instance -> Accounts -> Permission Sets -> User/Group
	dot output is currently limited as it shows internal names only
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
	switch settings.GetOutputFormat() {
	case "drawio":
		output.DrawIOHeader = createSSOAccountsDrawIOHeader()
		createSSOAccountDrawIOContents(ssoInstance, &output)
	case "dot":
		dotcolumns := config.DotColumns{
			From: "DrawIOID",
			To:   "Children",
		}
		settings.DotColumns = &dotcolumns
		createSSOAccountDrawIOContents(ssoInstance, &output)
	default:
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
	}
	output.Write(*settings)
}

func createSSOAccountsDrawIOHeader() drawio.Header {
	drawioheader := drawio.DefaultHeader()
	drawioheader.SetHeightAndWidth("78", "78")
	drawioheader.SetLayout(drawio.LayoutVerticalTree)
	connection := drawio.NewConnection()
	connection.Invert = false
	connection.From = "Children"
	connection.To = "DrawIOID"
	drawioheader.AddConnection(connection)
	return drawioheader
}

func createSSOAccountDrawIOContents(instance helpers.SSOInstance, output *helpers.OutputArray) {
	output.Keys = []string{"Name", "DrawIOID", "Type", "Children", "Image"}

	content := make(map[string]string)
	content["Name"] = getName(instance.Arn)
	content["DrawIOID"] = getName(instance.Arn)
	content["Type"] = "SSO"
	content["Image"] = drawio.ShapeAWSSingleSignOn
	content["Children"] = strings.Join(instance.GetAccountList(), ",")
	holder := helpers.OutputHolder{Contents: content}
	output.AddHolder(holder)
	uniquefilter := []string{}
	for _, account := range instance.Accounts {
		accountchildren := []string{}
		content := make(map[string]string)
		content["Name"] = getName(account.AccountID)
		content["DrawIOID"] = account.AccountID
		content["Type"] = "Account"
		content["Image"] = drawio.ShapeAWSOrganizationsAccount
		for _, assignment := range account.AccountAssignments {
			accountchildren = append(accountchildren, assignment.PermissionSet.Name+account.AccountID)
		}
		content["Children"] = strings.Join(unique(accountchildren), ",")
		holder := helpers.OutputHolder{Contents: content}
		output.AddHolder(holder)
		for _, assignment := range account.AccountAssignments {
			if !contains(uniquefilter, assignment.PermissionSet.Name+account.AccountID) {
				uniquefilter = append(uniquefilter, assignment.PermissionSet.Name+account.AccountID)
				content := make(map[string]string)
				content["Name"] = getName(assignment.PermissionSet.Name)
				content["DrawIOID"] = getName(assignment.PermissionSet.Name + account.AccountID)
				content["Type"] = "PermissionSet"
				content["Image"] = drawio.ShapeAWSIdentityandAccessManagementIAMPermissions
				content["Children"] = strings.Join(assignment.PermissionSet.GetAssignmentIdsByAccount(account.AccountID), ",")
				holder := helpers.OutputHolder{Contents: content}
				output.AddHolder(holder)
			}
			if !contains(uniquefilter, assignment.PrincipalID) {
				uniquefilter = append(uniquefilter, assignment.PrincipalID)
				content := make(map[string]string)
				content["Name"] = getName(assignment.PrincipalID)
				content["DrawIOID"] = assignment.PrincipalID
				content["Type"] = assignment.PrincipalType
				switch assignment.PrincipalType {
				case "USER":
					content["Image"] = drawio.ShapeAWSUser
				case "GROUP":
					content["Image"] = drawio.ShapeAWSUsers
				}
				holder := helpers.OutputHolder{Contents: content}
				output.AddHolder(holder)
			}
		}
	}
}