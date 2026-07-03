package cmd

import (
	"regexp"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/cobra"
)

var alphanumRegex = regexp.MustCompile("[^a-zA-Z0-9]+")

// userlistCmd represents the userlist command
var userlistCmd = &cobra.Command{
	Use:   "userlist",
	Short: "Get an overview of the IAM users in the account",
	Long: `Retrieves a list of all IAM users in the account and the groups they are in.
It also shows the policies they have through either the group or directly. The groups themselves are shown separately, as are policies when using the verbose flag.

The drawio output format links the users to groups and (in verbose mode) both of those to the policies.`,
	RunE: detailUsers,
}

func detailUsers(cmd *cobra.Command, _ []string) error {
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "IAM User overview for account " + getName(helpers.GetAccountID(awsConfig.StsClient()))
	svc := awsConfig.IamClient()
	userlist := helpers.GetUserDetails(svc)
	grouplist := helpers.GetGroupDetails(svc)
	objectlist := []helpers.IAMObject{}
	for _, user := range userlist {
		objectlist = append(objectlist, user)
	}
	for _, group := range grouplist {
		objectlist = append(objectlist, group)
	}
	keys := []string{nameColumn, typeColumn, "Groups", "Users", "PolicyNames", "InheritedPolicyNames", "Console", "API"}
	if settings.IsDrawIO() {
		keys = append(keys, imageColumn)
		keys = append(keys, "DrawioID")
		if settings.IsVerbose() {
			keys = append(keys, "AttachedToGroups")
			keys = append(keys, "AttachedToUsers")
		}
	}
	rows := []output.Record{}
	policylist := make(map[string]helpers.AttachedIAMPolicy)
	for _, object := range objectlist {
		content := make(map[string]any)
		content[nameColumn] = object.GetName()
		content[typeColumn] = object.GetObjectType()
		if user, ok := object.(helpers.IAMUser); ok {
			if user.HasUsedPassword() {
				content["Console"] = user.GetLastPasswordDate().String()
			}
			if user.HasAccessKeys(svc) {
				content["API"] = user.GetLastAccessKeyDate(svc).String()
			}
		}
		content["Groups"] = object.GetGroups()
		content["Users"] = object.GetUsers()
		directPolicyNames := make([]string, 0, len(object.GetDirectPolicies()))
		for policyname := range object.GetDirectPolicies() {
			directPolicyNames = append(directPolicyNames, policyname)
			if settings.IsVerbose() {
				// Get the attached policies
				policy := helpers.AttachedIAMPolicy{Name: policyname}
				if _, ok := policylist[policyname]; ok {
					policy = policylist[policyname]
				}
				policy.AddObject(object)
				policylist[policyname] = policy
			}
		}
		content["PolicyNames"] = directPolicyNames
		inheritedPolicyNames := make([]string, 0, len(object.GetInheritedPolicies()))
		for policyname := range object.GetInheritedPolicies() {
			inheritedPolicyNames = append(inheritedPolicyNames, policyname)
		}
		content["InheritedPolicyNames"] = inheritedPolicyNames

		if settings.IsDrawIO() {
			if object.GetObjectType() == "User" {
				content[imageColumn] = awsShape("General Resources", "User")
			} else {
				content[imageColumn] = awsShape("General Resources", "Users")
			}
			content["DrawioID"] = object.GetID()
		}
		rows = append(rows, content)
	}
	// This will only happen when verbose is set
	for _, policy := range policylist {
		content := make(map[string]any)
		content[nameColumn] = policy.Name
		content[typeColumn] = "Policy"
		if settings.IsDrawIO() {
			content[imageColumn] = awsShape("Security Identity Compliance", "Permissions")
			content["AttachedToUsers"] = policy.Users
			content["AttachedToGroups"] = policy.Groups
			content["DrawioID"] = createID("Policy" + policy.Name)
		} else {
			content["Users"] = policy.Users
			content["Groups"] = policy.Groups

		}
		rows = append(rows, content)
	}

	docs := config.DocumentSet{
		Table: output.New().
			Table(resultTitle, rows, output.WithKeys(keys...)).
			Build(),
	}
	if settings.NeedsGraphFormat() {
		graphRows := make([]map[string]any, len(rows))
		for i, row := range rows {
			graphRows[i] = row
		}
		docs.Graph = output.New().
			Graph(resultTitle, graphEdges(graphRows, nameColumn, "Groups")).
			Build()
	}
	if settings.IsDrawIO() {
		docs.DrawIO = output.New().
			DrawIO(resultTitle, rows, createIamuserlistDrawIOHeader()).
			Build()
	}
	return settings.RenderDocuments(cmd.Context(), docs)
}

// createIamuserlistDrawIOHeader creates and configures the draw.io header settings
func createIamuserlistDrawIOHeader() output.DrawIOHeader {
	header := drawIOBaseHeader("%Name%", "%Image%", "Image,DrawioID")
	header.Identity = "DrawioID"
	header.Layout = output.DrawIOLayoutHorizontalFlow
	connection := drawIOConnection()
	connection.From = "Groups"
	connection.To = nameColumn
	connection.Invert = false
	connection.Label = "Member of"
	header.Connections = append(header.Connections, connection)
	if settings.IsVerbose() {
		connection2 := drawIOConnection()
		connection2.From = "PolicyNames"
		connection2.To = nameColumn
		connection2.Invert = false
		connection2.Label = "Has Policy"
		header.Connections = append(header.Connections, connection2)
	}
	return header
}

func createID(toclean string) string {
	// Make a Regex to say we only want letters and numbers
	return alphanumRegex.ReplaceAllString(toclean, "")
}

func init() {
	iamCmd.AddCommand(userlistCmd)
}
