package cmd

import (
	"strings"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/drawio"
	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/spf13/cobra"
)

// userlistCmd represents the userlist command
var userlistCmd = &cobra.Command{
	Use:   "userlist",
	Short: "Get an overview of the IAM users in the account",
	Long: `Retrieves a list of all IAM users in the account, complete with the groups they are in and the policies they have through either the group or directly.

The verbose option will add the details of the policies to the output.`,
	Run: detailUsers,
}

func detailUsers(cmd *cobra.Command, args []string) {
	resultTitle := "IAM User overview for account " + getName(helpers.GetAccountID())
	setIamuserlistConfig()
	userlist := helpers.GetUserDetails()
	grouplist := helpers.GetGroupDetails()
	objectlist := []helpers.IAMObject{helpers.IAMGroup{}, helpers.IAMUser{}}
	for _, user := range userlist {
		objectlist = append(objectlist, user)
	}
	for _, group := range grouplist {
		objectlist = append(objectlist, group)
	}
	keys := []string{"Name", "Type", "Groups", "Users", "PolicyNames", "InheritedPolicyNames"}
	if settings.IsDrawIO() {
		keys = append(keys, "Image")
	}
	if *settings.Verbose {
		keys = append(keys, "Policies")
	}
	output := helpers.OutputArray{Keys: keys, Title: resultTitle}
	for _, user := range objectlist {
		content := make(map[string]string)
		content["Name"] = user.GetName()
		content["Type"] = user.GetObjectType()
		content["Groups"] = strings.Join(user.GetGroups(), ",")
		content["Users"] = strings.Join(user.GetUsers(), ",")
		userpolicyNames := make([]string, 0, len(user.GetDirectPolicies()))
		userpolicyDetails := make([]string, 0, len(user.GetDirectPolicies()))
		for policyname, policydetail := range user.GetDirectPolicies() {
			userpolicyNames = append(userpolicyNames, policyname)
			userpolicyDetails = append(userpolicyDetails, policydetail)
		}
		content["PolicyNames"] = strings.Join(userpolicyNames, ", ")
		grouppolicyNames := make([]string, 0, len(user.GetInheritedPolicies()))
		grouppolicyDetails := make([]string, 0, len(user.GetInheritedPolicies()))
		for policyname, policydetail := range user.GetInheritedPolicies() {
			grouppolicyNames = append(grouppolicyNames, policyname)
			grouppolicyDetails = append(grouppolicyDetails, policydetail)
		}
		content["InheritedPolicyNames"] = strings.Join(grouppolicyNames, ", ")
		// if *settings.Verbose {
		// 	content["Policies"] = strings.Join(policyDetails, ",\n")
		// }
		if settings.IsDrawIO() {
			if user.GetObjectType() == "User" {
				content["Image"] = drawio.ShapeAWSUser
			} else {
				content["Image"] = drawio.ShapeAWSUsers
			}
		}
		holder := helpers.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	output.Write(*settings)
}

func setIamuserlistConfig() {
	switch settings.GetOutputFormat() {
	case "drawio":
		drawioheader := drawio.NewHeader("%Name%", "%Image%", "Image")
		connection := drawio.NewConnection()
		connection.From = "Groups"
		connection.To = "Name"
		connection.Invert = false
		drawioheader.AddConnection(connection)
		header := drawioheader.String()
		settings.OutputHeaders = &header
	case "dot":
		dotcolumns := config.DotColumns{
			From: "Name",
			To:   "Groups",
		}
		settings.DotColumns = &dotcolumns
	}
}

func init() {
	iamCmd.AddCommand(userlistCmd)
}
