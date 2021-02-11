package helpers

import (
	"context"
	"errors"
	"log"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

var cachedUsers []types.User

// IAMSession returns a shared IAMSession
func IAMSession(config aws.Config) *iam.Client {
	return iam.NewFromConfig(config)
}

// GetPoliciesMap retrieves a map of policies with the policy name as the key
// and the actual policy object as the value
func GetPoliciesMap(svc *iam.Client) map[string]types.Policy {
	result := make(map[string]types.Policy)

	params := &iam.ListPoliciesInput{}
	resp, err := svc.ListPolicies(context.TODO(), params)
	if err != nil {
		panic(err)
	}
	for _, policy := range resp.Policies {
		result[*policy.PolicyName] = policy
	}
	return result
}

// GetUserPoliciesMapForUser retrieves a map of policies for the provided IAM
// username where the key is the name of the policy and the value is the actual
// json policy document
func GetUserPoliciesMapForUser(username *string, svc *iam.Client) map[string]string {
	result := make(map[string]string)
	params := &iam.ListUserPoliciesInput{
		UserName: username,
	}
	resp, err := svc.ListUserPolicies(context.TODO(), params)
	if err != nil {
		panic(err)
	}
	if len(resp.PolicyNames) > 0 {
		for _, policyname := range resp.PolicyNames {
			params := &iam.GetUserPolicyInput{
				PolicyName: aws.String(policyname),
				UserName:   username,
			}
			resp, err := svc.GetUserPolicy(context.TODO(), params)
			if err != nil {
				panic(err)
			}
			policyDocument, err := url.QueryUnescape(*resp.PolicyDocument)
			if err != nil {
				panic(err)
			}
			result[policyname] = policyDocument
		}
	}
	return result
}

// GetGroupPoliciesMapForGroup retrieves a map of policies for the provided IAM
// groupname where the key is the name of the policy and the value is the actual
// json policy document
func GetGroupPoliciesMapForGroup(groupname *string, svc *iam.Client) map[string]string {
	result := make(map[string]string)
	params := &iam.ListGroupPoliciesInput{
		GroupName: groupname,
	}
	resp, err := svc.ListGroupPolicies(context.TODO(), params)
	if err != nil {
		panic(err)
	}
	if len(resp.PolicyNames) > 0 {
		for _, policyname := range resp.PolicyNames {
			params := &iam.GetGroupPolicyInput{
				PolicyName: aws.String(policyname),
				GroupName:  groupname,
			}
			resp, err := svc.GetGroupPolicy(context.TODO(), params)
			if err != nil {
				panic(err)
			}
			policyDocument, err := url.QueryUnescape(*resp.PolicyDocument)
			if err != nil {
				panic(err)
			}
			result[policyname] = policyDocument
		}
	}
	return result
}

// GetGroupPoliciesMapForGroups retrieves all of the policies for the provided
// slice of groups, where the key is the name of the policy and the value is the
// json policy document
func GetGroupPoliciesMapForGroups(groups []string, svc *iam.Client) map[string]string {
	result := make(map[string]string)
	for _, group := range groups {
		groupmap := GetGroupPoliciesMapForGroup(&group, svc)
		for k, v := range groupmap {
			result[k] = v
		}
	}
	return result
}

// GetAttachedPoliciesMapForUser retrieves a map of attached policies for the
// provided IAM username where the key is the name of the policy and the value
// is the actual json policy document
func GetAttachedPoliciesMapForUser(username *string, svc *iam.Client) map[string]string {
	result := make(map[string]string)
	params := &iam.ListAttachedUserPoliciesInput{
		UserName: username,
	}
	resp, err := svc.ListAttachedUserPolicies(context.TODO(), params)
	if err != nil {
		panic(err)
	}
	if len(resp.AttachedPolicies) > 0 {
		for _, policy := range resp.AttachedPolicies {
			result[*policy.PolicyName] = getAttachedPolicy(policy.PolicyArn, svc)
		}
	}
	return result
}

// GetAttachedPoliciesMapForGroup retrieves a map of attached policies for the
// provided IAM groupname where the key is the name of the policy and the value
// is the actual json policy document
func GetAttachedPoliciesMapForGroup(groupname *string, svc *iam.Client) map[string]string {
	result := make(map[string]string)
	params := &iam.ListAttachedGroupPoliciesInput{
		GroupName: groupname,
	}
	resp, err := svc.ListAttachedGroupPolicies(context.TODO(), params)
	if err != nil {
		panic(err)
	}
	if len(resp.AttachedPolicies) > 0 {
		for _, policy := range resp.AttachedPolicies {
			result[*policy.PolicyName] = getAttachedPolicy(policy.PolicyArn, svc)
		}
	}
	return result
}

// GetAttachedPoliciesMapForGroups retrieves a map of attached policies for the
// slice of IAM groupnames where the key is the name of the policy and the value
// is the actual json policy document
func GetAttachedPoliciesMapForGroups(groups []string, svc *iam.Client) map[string]string {
	result := make(map[string]string)
	for _, group := range groups {
		groupmap := GetAttachedPoliciesMapForGroup(&group, svc)
		for k, v := range groupmap {
			result[k] = v
		}
	}
	return result
}

// GetGroupNameSliceForUser retrieves a slice of all the groups the provided
// IAM username belongs to
func GetGroupNameSliceForUser(username *string, svc *iam.Client) []string {
	params := &iam.ListGroupsForUserInput{
		UserName: username,
	}
	resp, err := svc.ListGroupsForUser(context.TODO(), params)

	if err != nil {
		panic(err)
	}
	groups := make([]string, len(resp.Groups))
	if len(resp.Groups) > 0 {
		for counter, group := range resp.Groups {
			groups[counter] = *group.GroupName
		}
	}
	return groups
}

// GetAccountSummary retrieves the account summary map which contains high level
// information about the root account
func GetAccountSummary(svc *iam.Client) (map[string]int32, error) {
	input := &iam.GetAccountSummaryInput{}

	result, err := svc.GetAccountSummary(context.TODO(), input)
	if err != nil {
		var authexc *types.InvalidAuthenticationCodeException
		if errors.As(err, &authexc) {
			log.Println("error:", authexc)
		}
		panic(err)
	}
	return result.SummaryMap, nil
}

func getUserList(svc *iam.Client) []types.User {
	if cachedUsers == nil {
		resp, err := svc.ListUsers(context.TODO(), &iam.ListUsersInput{})
		if err != nil {
			panic(err)
		}
		cachedUsers = resp.Users
	}
	return cachedUsers
}

// GetUserDetails collects detailed information about a user, consisting mostly
// of the groups and policies it follows.
func GetUserDetails(svc *iam.Client) []IAMUser {
	users := getUserList(svc)
	c := make(chan IAMUser)
	userlist := make([]IAMUser, len(users))
	for _, user := range users {
		go func(user types.User) {
			userStruct := IAMUser{
				Name: *user.UserName,
				User: &user,
			}
			userStruct.Groups = GetGroupNameSliceForUser(user.UserName, svc)
			userStruct.InlinePolicies = GetUserPoliciesMapForUser(user.UserName, svc)
			userStruct.AttachedPolicies = GetAttachedPoliciesMapForUser(user.UserName, svc)
			userStruct.InlineGroupPolicies = GetGroupPoliciesMapForGroups(userStruct.Groups, svc)
			userStruct.AttachedGroupPolicies = GetAttachedPoliciesMapForGroups(userStruct.Groups, svc)
			c <- userStruct
		}(user)
	}
	for i := 0; i < len(users); i++ {
		userlist[i] = <-c
	}
	return userlist
}

func getAllUsersInGroup(groupname string, svc *iam.Client) []string {
	input := iam.GetGroupInput{
		GroupName: &groupname,
	}
	resp, err := svc.GetGroup(context.TODO(), &input)
	if err != nil {
		panic(err)
	}
	result := []string{}
	for _, user := range resp.Users {
		result = append(result, *user.UserName)
	}
	return result
}

// GetGroupDetails collects detailed information about a group, consisting mostly
// of the users and policies it follows.
func GetGroupDetails(svc *iam.Client) []IAMGroup {
	resp, err := svc.ListGroups(context.TODO(), &iam.ListGroupsInput{})
	if err != nil {
		panic(err)
	}
	c := make(chan IAMGroup)
	grouplist := make([]IAMGroup, len(resp.Groups))
	for _, group := range resp.Groups {
		go func(group types.Group) {
			groupStruct := IAMGroup{
				Name:  *group.GroupName,
				Group: &group,
			}
			groupStruct.InlinePolicies = GetGroupPoliciesMapForGroups([]string{*group.GroupName}, svc)
			groupStruct.AttachedPolicies = GetAttachedPoliciesMapForGroups([]string{*group.GroupName}, svc)
			groupStruct.Users = getAllUsersInGroup(*group.GroupName, svc)
			c <- groupStruct
		}(group)
	}
	for i := 0; i < len(resp.Groups); i++ {
		grouplist[i] = <-c
	}
	return grouplist
}

// GetAllPolicies retrieves a map of all the users policies
func (user IAMUser) GetAllPolicies() map[string]string {
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

// GetAccountAlias returns the account alias in a map of [accountid]accountalias
// If no alias is present, it will return the account ID instead
func GetAccountAlias(svc *iam.Client) map[string]string {
	alias := make(map[string]string)
	alias[GetAccountID()] = GetAccountID()

	input := &iam.ListAccountAliasesInput{}
	result, err := svc.ListAccountAliases(context.TODO(), input)
	if err != nil {
		log.Fatal(err.Error())
	}
	if len(result.AccountAliases) > 0 {
		alias[GetAccountID()] = result.AccountAliases[0]
	}
	return alias
}

func getAttachedPolicy(policyArn *string, svc *iam.Client) string {
	params := &iam.GetPolicyInput{
		PolicyArn: policyArn,
	}
	resp, err := svc.GetPolicy(context.TODO(), params)
	if err != nil {
		panic(err)
	}
	params2 := &iam.GetPolicyVersionInput{
		PolicyArn: policyArn,                    // Required
		VersionId: resp.Policy.DefaultVersionId, // Required
	}
	resp2, err := svc.GetPolicyVersion(context.TODO(), params2)
	if err != nil {
		panic(err)
	}
	policyDocument, err := url.QueryUnescape(*resp2.PolicyVersion.Document)
	if err != nil {
		panic(err)
	}
	return policyDocument
}
