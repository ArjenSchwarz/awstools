package cmd

import (
	"strings"

	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/aws/aws-sdk-go/service/iam"
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
	svc := helpers.IAMSession()
	resp, err := svc.ListUsers(&iam.ListUsersInput{})
	if err != nil {
		panic(err)
	}
	c := make(chan iamUser)
	userlist := make([]iamUser, len(resp.Users))
	for _, user := range resp.Users {
		go func(user *iam.User) {
			userStruct := iamUser{
				Username: *user.UserName,
			}
			userStruct.Groups = helpers.GetGroupNameSliceForUser(user.UserName)
			userStruct.InlinePolicies = helpers.GetUserPoliciesMapForUser(user.UserName)
			userStruct.AttachedPolicies = helpers.GetAttachedPoliciesMapForUser(user.UserName)
			userStruct.InlineGroupPolicies = helpers.GetGroupPoliciesMapForGroups(userStruct.Groups)
			userStruct.AttachedGroupPolicies = helpers.GetAttachedPoliciesMapForGroups(userStruct.Groups)
			c <- userStruct
		}(user)
	}
	for i := 0; i < len(resp.Users); i++ {
		userlist[i] = <-c
	}
	keys := []string{"User", "Groups", "Policy Names"}
	if *settings.Verbose {
		keys = append(keys, "Policies")
	}
	output := helpers.OutputArray{Keys: keys}
	for _, user := range userlist {
		content := make(map[string]string)
		content["User"] = user.Username
		content["Groups"] = strings.Join(user.Groups, ", ")
		policyNames := make([]string, 0, len(user.GetAllPolicies()))
		policyDetails := make([]string, 0, len(user.GetAllPolicies()))
		for policyname, policydetail := range user.GetAllPolicies() {
			policyNames = append(policyNames, policyname)
			policyDetails = append(policyDetails, policydetail)
		}
		content["Policy Names"] = strings.Join(policyNames, ", ")
		if *settings.Verbose {
			content["Policies"] = strings.Join(policyDetails, ",\n")
		}
		holder := helpers.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	output.Write(*settings)
}

func (user iamUser) GetAllPolicies() map[string]string {
	result := make(map[string]string)
	for k, v := range user.InlinePolicies {
		result[k] = v
	}
	for k, v := range user.AttachedPolicies {
		result[k] = v
	}
	for k, v := range user.InlineGroupPolicies {
		result[k] = v
	}
	for k, v := range user.AttachedGroupPolicies {
		result[k] = v
	}
	return result
}

func init() {
	iamCmd.AddCommand(userlistCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// userlistCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// userlistCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}
