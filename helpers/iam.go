package helpers

import (
	"context"
	"errors"
	"log"
	"maps"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// IAMClient defines the subset of IAM client methods used by the helpers
// package. The concrete *iam.Client satisfies this interface.
type IAMClient interface {
	ListUsers(ctx context.Context, params *iam.ListUsersInput, optFns ...func(*iam.Options)) (*iam.ListUsersOutput, error)
	ListGroups(ctx context.Context, params *iam.ListGroupsInput, optFns ...func(*iam.Options)) (*iam.ListGroupsOutput, error)
	ListPolicies(ctx context.Context, params *iam.ListPoliciesInput, optFns ...func(*iam.Options)) (*iam.ListPoliciesOutput, error)
	ListUserPolicies(ctx context.Context, params *iam.ListUserPoliciesInput, optFns ...func(*iam.Options)) (*iam.ListUserPoliciesOutput, error)
	ListGroupPolicies(ctx context.Context, params *iam.ListGroupPoliciesInput, optFns ...func(*iam.Options)) (*iam.ListGroupPoliciesOutput, error)
	ListAttachedUserPolicies(ctx context.Context, params *iam.ListAttachedUserPoliciesInput, optFns ...func(*iam.Options)) (*iam.ListAttachedUserPoliciesOutput, error)
	ListAttachedGroupPolicies(ctx context.Context, params *iam.ListAttachedGroupPoliciesInput, optFns ...func(*iam.Options)) (*iam.ListAttachedGroupPoliciesOutput, error)
	ListGroupsForUser(ctx context.Context, params *iam.ListGroupsForUserInput, optFns ...func(*iam.Options)) (*iam.ListGroupsForUserOutput, error)
	ListRoles(ctx context.Context, params *iam.ListRolesInput, optFns ...func(*iam.Options)) (*iam.ListRolesOutput, error)
	ListRolePolicies(ctx context.Context, params *iam.ListRolePoliciesInput, optFns ...func(*iam.Options)) (*iam.ListRolePoliciesOutput, error)
	ListAttachedRolePolicies(ctx context.Context, params *iam.ListAttachedRolePoliciesInput, optFns ...func(*iam.Options)) (*iam.ListAttachedRolePoliciesOutput, error)
	ListAccessKeys(ctx context.Context, params *iam.ListAccessKeysInput, optFns ...func(*iam.Options)) (*iam.ListAccessKeysOutput, error)
	ListAccountAliases(ctx context.Context, params *iam.ListAccountAliasesInput, optFns ...func(*iam.Options)) (*iam.ListAccountAliasesOutput, error)
	GetUserPolicy(ctx context.Context, params *iam.GetUserPolicyInput, optFns ...func(*iam.Options)) (*iam.GetUserPolicyOutput, error)
	GetGroupPolicy(ctx context.Context, params *iam.GetGroupPolicyInput, optFns ...func(*iam.Options)) (*iam.GetGroupPolicyOutput, error)
	GetGroup(ctx context.Context, params *iam.GetGroupInput, optFns ...func(*iam.Options)) (*iam.GetGroupOutput, error)
	GetPolicy(ctx context.Context, params *iam.GetPolicyInput, optFns ...func(*iam.Options)) (*iam.GetPolicyOutput, error)
	GetPolicyVersion(ctx context.Context, params *iam.GetPolicyVersionInput, optFns ...func(*iam.Options)) (*iam.GetPolicyVersionOutput, error)
	GetRolePolicy(ctx context.Context, params *iam.GetRolePolicyInput, optFns ...func(*iam.Options)) (*iam.GetRolePolicyOutput, error)
	GetAccountSummary(ctx context.Context, params *iam.GetAccountSummaryInput, optFns ...func(*iam.Options)) (*iam.GetAccountSummaryOutput, error)
	GetAccessKeyLastUsed(ctx context.Context, params *iam.GetAccessKeyLastUsedInput, optFns ...func(*iam.Options)) (*iam.GetAccessKeyLastUsedOutput, error)
}

var cachedUsers []types.User

// GetPoliciesMap retrieves a map of policies with the policy name as the key
// and the actual policy object as the value
func GetPoliciesMap(svc IAMClient) map[string]types.Policy {
	result := make(map[string]types.Policy)

	paginator := iam.NewListPoliciesPaginator(svc, &iam.ListPoliciesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			panic(err)
		}
		for _, policy := range page.Policies {
			result[*policy.PolicyName] = policy
		}
	}
	return result
}

// GetUserPoliciesMapForUser retrieves a map of policies for the provided IAM
// username where the key is the name of the policy and the value is the actual
// json policy document
func GetUserPoliciesMapForUser(username *string, svc IAMClient) map[string]string {
	result := make(map[string]string)

	paginator := iam.NewListUserPoliciesPaginator(svc, &iam.ListUserPoliciesInput{
		UserName: username,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			panic(err)
		}
		for _, policyname := range page.PolicyNames {
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
func GetGroupPoliciesMapForGroup(groupname *string, svc IAMClient) map[string]string {
	result := make(map[string]string)

	paginator := iam.NewListGroupPoliciesPaginator(svc, &iam.ListGroupPoliciesInput{
		GroupName: groupname,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			panic(err)
		}
		for _, policyname := range page.PolicyNames {
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
func GetGroupPoliciesMapForGroups(groups []string, svc IAMClient) map[string]string {
	result := make(map[string]string)
	for _, group := range groups {
		groupmap := GetGroupPoliciesMapForGroup(&group, svc)
		maps.Copy(result, groupmap)
	}
	return result
}

// GetAttachedPoliciesMapForUser retrieves a map of attached policies for the
// provided IAM username where the key is the name of the policy and the value
// is the actual json policy document
func GetAttachedPoliciesMapForUser(username *string, svc IAMClient) map[string]string {
	result := make(map[string]string)

	paginator := iam.NewListAttachedUserPoliciesPaginator(svc, &iam.ListAttachedUserPoliciesInput{
		UserName: username,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			panic(err)
		}
		for _, policy := range page.AttachedPolicies {
			result[*policy.PolicyName] = getAttachedPolicy(policy.PolicyArn, svc)
		}
	}
	return result
}

// GetAttachedPoliciesMapForGroup retrieves a map of attached policies for the
// provided IAM groupname where the key is the name of the policy and the value
// is the actual json policy document
func GetAttachedPoliciesMapForGroup(groupname *string, svc IAMClient) map[string]string {
	result := make(map[string]string)

	paginator := iam.NewListAttachedGroupPoliciesPaginator(svc, &iam.ListAttachedGroupPoliciesInput{
		GroupName: groupname,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			panic(err)
		}
		for _, policy := range page.AttachedPolicies {
			result[*policy.PolicyName] = getAttachedPolicy(policy.PolicyArn, svc)
		}
	}
	return result
}

// GetAttachedPoliciesMapForGroups retrieves a map of attached policies for the
// slice of IAM groupnames where the key is the name of the policy and the value
// is the actual json policy document
func GetAttachedPoliciesMapForGroups(groups []string, svc IAMClient) map[string]string {
	result := make(map[string]string)
	for _, group := range groups {
		groupmap := GetAttachedPoliciesMapForGroup(&group, svc)
		maps.Copy(result, groupmap)
	}
	return result
}

// GetGroupNameSliceForUser retrieves a slice of all the groups the provided
// IAM username belongs to
func GetGroupNameSliceForUser(username *string, svc IAMClient) []string {
	var groups []string

	paginator := iam.NewListGroupsForUserPaginator(svc, &iam.ListGroupsForUserInput{
		UserName: username,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			panic(err)
		}
		for _, group := range page.Groups {
			groups = append(groups, *group.GroupName)
		}
	}
	return groups
}

// GetAccountSummary retrieves the account summary map which contains high level
// information about the root account
func GetAccountSummary(svc IAMClient) (map[string]int32, error) {
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

func getUserList(svc IAMClient) []types.User {
	if cachedUsers == nil {
		paginator := iam.NewListUsersPaginator(svc, &iam.ListUsersInput{})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(context.TODO())
			if err != nil {
				panic(err)
			}
			cachedUsers = append(cachedUsers, page.Users...)
		}
	}
	return cachedUsers
}

// GetUserDetails collects detailed information about a user, consisting mostly
// of the groups and policies it follows.
func GetUserDetails(svc IAMClient) []IAMUser {
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
	for i := range users {
		userlist[i] = <-c
	}
	return userlist
}

func getAllUsersInGroup(groupname string, svc IAMClient) []string {
	var result []string

	paginator := iam.NewGetGroupPaginator(svc, &iam.GetGroupInput{
		GroupName: &groupname,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			panic(err)
		}
		for _, user := range page.Users {
			result = append(result, *user.UserName)
		}
	}
	return result
}

// GetGroupDetails collects detailed information about a group, consisting mostly
// of the users and policies it follows.
func GetGroupDetails(svc IAMClient) []IAMGroup {
	var allGroups []types.Group

	paginator := iam.NewListGroupsPaginator(svc, &iam.ListGroupsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			panic(err)
		}
		allGroups = append(allGroups, page.Groups...)
	}

	c := make(chan IAMGroup)
	grouplist := make([]IAMGroup, len(allGroups))
	for _, group := range allGroups {
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
	for i := 0; i < len(allGroups); i++ {
		grouplist[i] = <-c
	}
	return grouplist
}

// GetAllPolicies retrieves a map of all the users policies
func (user IAMUser) GetAllPolicies() map[string]string {
	result := make(map[string]string)
	maps.Copy(result, user.InlinePolicies)
	maps.Copy(result, user.AttachedPolicies)
	maps.Copy(result, user.InlineGroupPolicies)
	maps.Copy(result, user.AttachedGroupPolicies)
	return result
}

// GetAccountAlias returns the account alias in a map of [accountid]accountalias
// If no alias is present, it will return the account ID instead
func GetAccountAlias(svc *iam.Client, stsSvc *sts.Client) map[string]string {
	alias := make(map[string]string)
	alias[GetAccountID(stsSvc)] = GetAccountID(stsSvc)

	input := &iam.ListAccountAliasesInput{}
	result, err := svc.ListAccountAliases(context.TODO(), input)
	if err != nil {
		log.Fatal(err.Error())
	}
	if len(result.AccountAliases) > 0 {
		alias[GetAccountID(stsSvc)] = result.AccountAliases[0]
	}
	return alias
}

func getAttachedPolicy(policyArn *string, svc IAMClient) string {
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
