package cmd

import (
	"log"
	"regexp"
	"strings"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/drawio"
	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/ArjenSchwarz/awstools/lib/format"
	"github.com/spf13/cobra"
)

// userlistCmd represents the userlist command
var userlistCmd = &cobra.Command{
	Use:   "userlist",
	Short: "Get an overview of the IAM users in the account",
	Long: `Retrieves a list of all IAM users in the account and the groups they are in.
It also shows the policies they have through either the group or directly. The groups themselves are shown separately, as are policies when using the verbose flag.

The drawio output format links the users to groups and (in verbose mode) both of those to the policies.`,
	Run: detailUsers,
}

func detailUsers(cmd *cobra.Command, args []string) {
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
	keys := []string{"Name", "Type", "Groups", "Users", "PolicyNames", "InheritedPolicyNames", "Console", "API"}
	stringSeparator := ", "
	if settings.IsDrawIO() {
		keys = append(keys, "Image")
		keys = append(keys, "DrawioID")
		stringSeparator = ","
		if *settings.Verbose {
			keys = append(keys, "AttachedToGroups")
			keys = append(keys, "AttachedToUsers")
		}
	}
	output := format.OutputArray{Keys: keys, Settings: format.NewOutputSettings(*settings)}
	output.Settings.Title = resultTitle
	switch settings.GetOutputFormat() {
	case "drawio":
		output.Settings.DrawIOHeader = createIamuserlistDrawIOHeader()
	case "dot":
		output.Settings.AddDotFromToColumns("Name", "Groups")
	}
	policylist := make(map[string]helpers.AttachedIAMPolicy)
	for _, object := range objectlist {
		content := make(map[string]interface{})
		content["Name"] = object.GetName()
		content["Type"] = object.GetObjectType()
		if user, ok := object.(helpers.IAMUser); ok {
			if user.HasUsedPassword() {
				content["Console"] = user.GetLastPasswordDate().String()
			}
			if user.HasAccessKeys(svc) {
				content["API"] = user.GetLastAccessKeyDate(svc).String()
			}
		}
		content["Groups"] = strings.Join(object.GetGroups(), ",")
		content["Users"] = strings.Join(object.GetUsers(), ",")
		directPolicyNames := make([]string, 0, len(object.GetDirectPolicies()))
		directPolicyDetails := make([]string, 0, len(object.GetDirectPolicies()))
		for policyname, policydetail := range object.GetDirectPolicies() {
			directPolicyNames = append(directPolicyNames, policyname)
			directPolicyDetails = append(directPolicyDetails, policydetail)
			if *settings.Verbose {
				// Get the attached policies
				policy := helpers.AttachedIAMPolicy{Name: policyname}
				if _, ok := policylist[policyname]; ok {
					policy = policylist[policyname]
				}
				policy.AddObject(object)
				policylist[policyname] = policy
			}
		}
		content["PolicyNames"] = strings.Join(directPolicyNames, stringSeparator)
		inheritedPolicyNames := make([]string, 0, len(object.GetInheritedPolicies()))
		inheritedPolicyDetails := make([]string, 0, len(object.GetInheritedPolicies()))
		for policyname, policydetail := range object.GetInheritedPolicies() {
			inheritedPolicyNames = append(inheritedPolicyNames, policyname)
			inheritedPolicyDetails = append(inheritedPolicyDetails, policydetail)
		}
		content["InheritedPolicyNames"] = strings.Join(inheritedPolicyNames, stringSeparator)

		if settings.IsDrawIO() {
			if object.GetObjectType() == "User" {
				content["Image"] = drawio.AWSShape("General Resources", "User")
			} else {
				content["Image"] = drawio.AWSShape("General Resources", "Users")
			}
			content["DrawioID"] = object.GetID()
		}
		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	// This will only happen when verbose is set
	for _, policy := range policylist {
		content := make(map[string]interface{})
		content["Name"] = policy.Name
		content["Type"] = "Policy"
		if settings.IsDrawIO() {
			content["Image"] = drawio.AWSShape("Security Identity Compliance", "Permissions")
			content["AttachedToUsers"] = strings.Join(policy.Users, stringSeparator)
			content["AttachedToGroups"] = strings.Join(policy.Groups, stringSeparator)
			content["DrawioID"] = createID("Policy" + policy.Name)
		} else {
			content["Users"] = strings.Join(policy.Users, stringSeparator)
			content["Groups"] = strings.Join(policy.Groups, stringSeparator)

		}
		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	output.Write(*settings)
}

// createIamuserlistDrawIOHeader creates and configures the draw.io header settings
func createIamuserlistDrawIOHeader() drawio.Header {
	drawioheader := drawio.NewHeader("%Name%", "%Image%", "Image,DrawioID")
	drawioheader.SetHeightAndWidth("78", "78")
	drawioheader.SetIdentity("DrawioID")
	drawioheader.SetLayout(drawio.LayoutHorizontalFlow)
	connection := drawio.NewConnection()
	connection.From = "Groups"
	connection.To = "Name"
	connection.Invert = false
	connection.Label = "Member of"
	drawioheader.AddConnection(connection)
	if *settings.Verbose {
		connection2 := drawio.NewConnection()
		connection2.From = "PolicyNames"
		connection2.To = "Name"
		connection2.Invert = false
		connection2.Label = "Has Policy"
		drawioheader.AddConnection(connection2)
	}
	return drawioheader
}

func createID(toclean string) string {
	// Make a Regex to say we only want letters and numbers
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		log.Fatal(err)
	}
	return reg.ReplaceAllString(toclean, "")
}

func init() {
	iamCmd.AddCommand(userlistCmd)
}
