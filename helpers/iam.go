package helpers

import (
	"fmt"
	"log"
	"net/url"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
)

var iamSession = iam.New(session.New())

// IAMObject interface for IAM objects
type IAMObject interface {
	GetName() string
	GetUsers() []string
	GetGroups() []string
	GetObjectType() string
	GetDirectPolicies() map[string]string
	GetInheritedPolicies() map[string]string
}

// IAMUser contains information about IAM Users
type IAMUser struct {
	Name                  string
	AttachedPolicies      map[string]string
	InlinePolicies        map[string]string
	Groups                []string
	AttachedGroupPolicies map[string]string
	InlineGroupPolicies   map[string]string
	User                  *iam.User
}

// IAMGroup contains information about IAM Groups
type IAMGroup struct {
	Name             string
	Users            []string
	AttachedPolicies map[string]string
	InlinePolicies   map[string]string
	Group            *iam.Group
}

// IAMSession returns a shared IAMSession
func IAMSession() *iam.IAM {
	return iamSession
}

// GetPoliciesMap retrieves a map of policies with the policy name as the key
// and the actual policy object as the value
func GetPoliciesMap() map[string]*iam.Policy {
	svc := IAMSession()

	result := make(map[string]*iam.Policy)

	params := &iam.ListPoliciesInput{}
	resp, err := svc.ListPolicies(params)
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
func GetUserPoliciesMapForUser(username *string) map[string]string {
	svc := IAMSession()
	result := make(map[string]string)
	params := &iam.ListUserPoliciesInput{
		UserName: username,
	}
	resp, err := svc.ListUserPolicies(params)
	if err != nil {
		panic(err)
	}
	if len(resp.PolicyNames) > 0 {
		for _, policyname := range resp.PolicyNames {
			params := &iam.GetUserPolicyInput{
				PolicyName: policyname,
				UserName:   username,
			}
			resp, err := svc.GetUserPolicy(params)
			if err != nil {
				panic(err)
			}
			policyDocument, err := url.QueryUnescape(*resp.PolicyDocument)
			if err != nil {
				panic(err)
			}
			result[*policyname] = policyDocument
		}
	}
	return result
}

// GetGroupPoliciesMapForGroup retrieves a map of policies for the provided IAM
// groupname where the key is the name of the policy and the value is the actual
// json policy document
func GetGroupPoliciesMapForGroup(groupname *string) map[string]string {
	svc := IAMSession()
	result := make(map[string]string)
	params := &iam.ListGroupPoliciesInput{
		GroupName: groupname,
	}
	resp, err := svc.ListGroupPolicies(params)
	if err != nil {
		panic(err)
	}
	if len(resp.PolicyNames) > 0 {
		for _, policyname := range resp.PolicyNames {
			params := &iam.GetGroupPolicyInput{
				PolicyName: policyname,
				GroupName:  groupname,
			}
			resp, err := svc.GetGroupPolicy(params)
			if err != nil {
				panic(err)
			}
			policyDocument, err := url.QueryUnescape(*resp.PolicyDocument)
			if err != nil {
				panic(err)
			}
			result[*policyname] = policyDocument
		}
	}
	return result
}

// GetGroupPoliciesMapForGroups retrieves all of the policies for the provided
// slice of groups, where the key is the name of the policy and the value is the
// json policy document
func GetGroupPoliciesMapForGroups(groups []string) map[string]string {
	result := make(map[string]string)
	for _, group := range groups {
		groupmap := GetGroupPoliciesMapForGroup(&group)
		for k, v := range groupmap {
			result[k] = v
		}
	}
	return result
}

// GetAttachedPoliciesMapForUser retrieves a map of attached policies for the
// provided IAM username where the key is the name of the policy and the value
// is the actual json policy document
func GetAttachedPoliciesMapForUser(username *string) map[string]string {
	svc := IAMSession()
	result := make(map[string]string)
	params := &iam.ListAttachedUserPoliciesInput{
		UserName: username,
	}
	resp, err := svc.ListAttachedUserPolicies(params)
	if err != nil {
		panic(err)
	}
	if len(resp.AttachedPolicies) > 0 {
		for _, policy := range resp.AttachedPolicies {
			params := &iam.GetPolicyInput{
				PolicyArn: policy.PolicyArn,
			}
			resp, err := svc.GetPolicy(params)
			if err != nil {
				panic(err)
			}
			params2 := &iam.GetPolicyVersionInput{
				PolicyArn: policy.PolicyArn,             // Required
				VersionId: resp.Policy.DefaultVersionId, // Required
			}
			resp2, err := svc.GetPolicyVersion(params2)
			if err != nil {
				panic(err)
			}
			policyDocument, err := url.QueryUnescape(*resp2.PolicyVersion.Document)
			if err != nil {
				panic(err)
			}
			result[*policy.PolicyName] = policyDocument
		}
	}
	return result
}

// GetAttachedPoliciesMapForGroup retrieves a map of attached policies for the
// provided IAM groupname where the key is the name of the policy and the value
// is the actual json policy document
func GetAttachedPoliciesMapForGroup(groupname *string) map[string]string {
	svc := IAMSession()
	result := make(map[string]string)
	params := &iam.ListAttachedGroupPoliciesInput{
		GroupName: groupname,
	}
	resp, err := svc.ListAttachedGroupPolicies(params)
	if err != nil {
		panic(err)
	}
	if len(resp.AttachedPolicies) > 0 {
		for _, policy := range resp.AttachedPolicies {
			params := &iam.GetPolicyInput{
				PolicyArn: policy.PolicyArn,
			}
			resp, err := svc.GetPolicy(params)
			if err != nil {
				panic(err)
			}
			params2 := &iam.GetPolicyVersionInput{
				PolicyArn: policy.PolicyArn,             // Required
				VersionId: resp.Policy.DefaultVersionId, // Required
			}
			resp2, err := svc.GetPolicyVersion(params2)
			if err != nil {
				panic(err)
			}
			policyDocument, err := url.QueryUnescape(*resp2.PolicyVersion.Document)
			if err != nil {
				panic(err)
			}
			result[*policy.PolicyName] = policyDocument
		}
	}
	return result
}

// GetAttachedPoliciesMapForGroups retrieves a map of attached policies for the
// slice of IAM groupnames where the key is the name of the policy and the value
// is the actual json policy document
func GetAttachedPoliciesMapForGroups(groups []string) map[string]string {
	result := make(map[string]string)
	for _, group := range groups {
		groupmap := GetAttachedPoliciesMapForGroup(&group)
		for k, v := range groupmap {
			result[k] = v
		}
	}
	return result
}

// GetGroupNameSliceForUser retrieves a slice of all the groups the provided
// IAM username belongs to
func GetGroupNameSliceForUser(username *string) []string {
	svc := IAMSession()
	params := &iam.ListGroupsForUserInput{
		UserName: username,
	}
	resp, err := svc.ListGroupsForUser(params)

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
func GetAccountSummary() (map[string]*int64, error) {
	svc := IAMSession()
	input := &iam.GetAccountSummaryInput{}

	result, err := svc.GetAccountSummary(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeServiceFailureException:
				fmt.Println(iam.ErrCodeServiceFailureException, aerr.Error())
				panic(aerr)
			default:
				fmt.Println(aerr.Error())
				panic(aerr)
			}
		} else {
			panic(err)
		}
	}
	return result.SummaryMap, nil
}

// GetUserDetails collects detailed information about a user, consisting mostly
// of the groups and policies it follows.
func GetUserDetails() []IAMUser {
	svc := IAMSession()
	resp, err := svc.ListUsers(&iam.ListUsersInput{})
	if err != nil {
		panic(err)
	}
	c := make(chan IAMUser)
	userlist := make([]IAMUser, len(resp.Users))
	for _, user := range resp.Users {
		go func(user *iam.User) {
			userStruct := IAMUser{
				Name: *user.UserName,
				User: user,
			}
			userStruct.Groups = GetGroupNameSliceForUser(user.UserName)
			userStruct.InlinePolicies = GetUserPoliciesMapForUser(user.UserName)
			userStruct.AttachedPolicies = GetAttachedPoliciesMapForUser(user.UserName)
			userStruct.InlineGroupPolicies = GetGroupPoliciesMapForGroups(userStruct.Groups)
			userStruct.AttachedGroupPolicies = GetAttachedPoliciesMapForGroups(userStruct.Groups)
			c <- userStruct
		}(user)
	}
	for i := 0; i < len(resp.Users); i++ {
		userlist[i] = <-c
	}
	return userlist
}

// GetGroupDetails collects detailed information about a group, consisting mostly
// of the users and policies it follows.
func GetGroupDetails() []IAMGroup {
	svc := IAMSession()
	resp, err := svc.ListGroups(&iam.ListGroupsInput{})
	if err != nil {
		panic(err)
	}
	c := make(chan IAMGroup)
	grouplist := make([]IAMGroup, len(resp.Groups))
	for _, group := range resp.Groups {
		go func(group *iam.Group) {
			groupStruct := IAMGroup{
				Name:  *group.GroupName,
				Group: group,
			}
			groupStruct.InlinePolicies = GetGroupPoliciesMapForGroups([]string{*group.GroupName})
			groupStruct.AttachedPolicies = GetAttachedPoliciesMapForGroups([]string{*group.GroupName})
			c <- groupStruct
		}(group)
	}
	for i := 0; i < len(resp.Groups); i++ {
		grouplist[i] = <-c
	}
	return grouplist
}

// GetName returns the name of the user
func (user IAMUser) GetName() string {
	return user.Name
}

// GetUsers returns an empty string slice
func (user IAMUser) GetUsers() []string {
	return []string{}
}

// GetGroups returns the list of groups the user has
func (user IAMUser) GetGroups() []string {
	return user.Groups
}

// GetObjectType returns the type of IAM object
func (user IAMUser) GetObjectType() string {
	return "User"
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

// GetDirectPolicies retrieves all directly attached policies for the user
func (user IAMUser) GetDirectPolicies() map[string]string {
	result := make(map[string]string)
	for k, v := range user.InlinePolicies {
		result[k] = v
	}
	for k, v := range user.AttachedPolicies {
		result[k] = v
	}
	return result
}

// GetInheritedPolicies retrieves all inherited policies for the user
func (user IAMUser) GetInheritedPolicies() map[string]string {
	result := make(map[string]string)
	for k, v := range user.InlineGroupPolicies {
		result[k] = v
	}
	for k, v := range user.AttachedGroupPolicies {
		result[k] = v
	}
	return result
}

// GetUsers returns the users attached to the Group
func (group IAMGroup) GetUsers() []string {
	return []string{}
}

// GetGroups returns an empty string slice
func (group IAMGroup) GetGroups() []string {
	return []string{}
}

// GetName returns the name of the group
func (group IAMGroup) GetName() string {
	return group.Name
}

// GetObjectType returns the type of IAM object
func (group IAMGroup) GetObjectType() string {
	return "Group"
}

// GetDirectPolicies retrieves all directly attached policies for the group
func (group IAMGroup) GetDirectPolicies() map[string]string {
	result := make(map[string]string)
	for k, v := range group.InlinePolicies {
		result[k] = v
	}
	for k, v := range group.AttachedPolicies {
		result[k] = v
	}
	return result
}

// GetInheritedPolicies retrieves all inherited policies for the group (none)
func (group IAMGroup) GetInheritedPolicies() map[string]string {
	return make(map[string]string)
}

// GetAccountAlias returns the account alias in a map of [accountid]accountalias
// If no alias is present, it will return the account ID instead
func GetAccountAlias() map[string]string {
	svc := IAMSession()
	alias := make(map[string]string)
	alias[GetAccountID()] = GetAccountID()

	input := &iam.ListAccountAliasesInput{}
	result, err := svc.ListAccountAliases(input)
	if err != nil {
		log.Fatal(err.Error())
	}
	if len(result.AccountAliases) > 0 {
		alias[GetAccountID()] = *result.AccountAliases[0]
	}
	return alias
}
