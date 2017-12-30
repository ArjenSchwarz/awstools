package helpers

import (
	"fmt"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

// IAMUser contains information about IAM Users
type IAMUser struct {
	Username              string
	AttachedPolicies      map[string]string
	InlinePolicies        map[string]string
	Groups                []string
	AttachedGroupPolicies map[string]string
	InlineGroupPolicies   map[string]string
	User                  *iam.User
}

// GetPoliciesMap retrieves a map of policies with the policy name as the key
// and the actual policy object as the value
func GetPoliciesMap(config aws.Config) map[string]iam.Policy {
	svc := iam.New(config)

	result := make(map[string]iam.Policy)

	params := &iam.ListPoliciesInput{}
	req := svc.ListPoliciesRequest(params)
	resp, err := req.Send()
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
func GetUserPoliciesMapForUser(username *string, config aws.Config) map[string]string {
	svc := iam.New(config)
	result := make(map[string]string)
	params := &iam.ListUserPoliciesInput{
		UserName: username,
	}
	req := svc.ListUserPoliciesRequest(params)
	resp, err := req.Send()
	if err != nil {
		panic(err)
	}
	if len(resp.PolicyNames) > 0 {
		for _, policyname := range resp.PolicyNames {
			params := &iam.GetUserPolicyInput{
				PolicyName: &policyname,
				UserName:   username,
			}
			req := svc.GetUserPolicyRequest(params)
			resp, err := req.Send()
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
func GetGroupPoliciesMapForGroup(groupname *string, config aws.Config) map[string]string {
	svc := iam.New(config)
	result := make(map[string]string)
	params := &iam.ListGroupPoliciesInput{
		GroupName: groupname,
	}
	req := svc.ListGroupPoliciesRequest(params)
	resp, err := req.Send()
	if err != nil {
		panic(err)
	}
	if len(resp.PolicyNames) > 0 {
		for _, policyname := range resp.PolicyNames {
			params := &iam.GetGroupPolicyInput{
				PolicyName: &policyname,
				GroupName:  groupname,
			}
			req := svc.GetGroupPolicyRequest(params)
			resp, err := req.Send()
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
func GetGroupPoliciesMapForGroups(groups []string, config aws.Config) map[string]string {
	result := make(map[string]string)
	for _, group := range groups {
		groupmap := GetGroupPoliciesMapForGroup(&group, config)
		for k, v := range groupmap {
			result[k] = v
		}
	}
	return result
}

// GetAttachedPoliciesMapForUser retrieves a map of attached policies for the
// provided IAM username where the key is the name of the policy and the value
// is the actual json policy document
func GetAttachedPoliciesMapForUser(username *string, config aws.Config) map[string]string {
	svc := iam.New(config)
	result := make(map[string]string)
	params := &iam.ListAttachedUserPoliciesInput{
		UserName: username,
	}
	req := svc.ListAttachedUserPoliciesRequest(params)
	resp, err := req.Send()
	if err != nil {
		panic(err)
	}
	if len(resp.AttachedPolicies) > 0 {
		for _, policy := range resp.AttachedPolicies {
			params := &iam.GetPolicyInput{
				PolicyArn: policy.PolicyArn,
			}
			req := svc.GetPolicyRequest(params)
			resp, err := req.Send()
			if err != nil {
				panic(err)
			}
			params2 := &iam.GetPolicyVersionInput{
				PolicyArn: policy.PolicyArn,             // Required
				VersionId: resp.Policy.DefaultVersionId, // Required
			}
			req2 := svc.GetPolicyVersionRequest(params2)
			resp2, err := req2.Send()
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
func GetAttachedPoliciesMapForGroup(groupname *string, config aws.Config) map[string]string {
	svc := iam.New(config)
	result := make(map[string]string)
	params := &iam.ListAttachedGroupPoliciesInput{
		GroupName: groupname,
	}
	req := svc.ListAttachedGroupPoliciesRequest(params)
	resp, err := req.Send()
	if err != nil {
		panic(err)
	}
	if len(resp.AttachedPolicies) > 0 {
		for _, policy := range resp.AttachedPolicies {
			params := &iam.GetPolicyInput{
				PolicyArn: policy.PolicyArn,
			}
			req := svc.GetPolicyRequest(params)
			resp, err := req.Send()
			if err != nil {
				panic(err)
			}
			params2 := &iam.GetPolicyVersionInput{
				PolicyArn: policy.PolicyArn,             // Required
				VersionId: resp.Policy.DefaultVersionId, // Required
			}
			req2 := svc.GetPolicyVersionRequest(params2)
			resp2, err := req2.Send()
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
func GetAttachedPoliciesMapForGroups(groups []string, config aws.Config) map[string]string {
	result := make(map[string]string)
	for _, group := range groups {
		groupmap := GetAttachedPoliciesMapForGroup(&group, config)
		for k, v := range groupmap {
			result[k] = v
		}
	}
	return result
}

// GetGroupNameSliceForUser retrieves a slice of all the groups the provided
// IAM username belongs to
func GetGroupNameSliceForUser(username *string, config aws.Config) []string {
	svc := iam.New(config)
	params := &iam.ListGroupsForUserInput{
		UserName: username,
	}
	req := svc.ListGroupsForUserRequest(params)
	resp, err := req.Send()

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
func GetAccountSummary(config aws.Config) (map[string]int64, error) {
	svc := iam.New(config)
	input := &iam.GetAccountSummaryInput{}

	req := svc.GetAccountSummaryRequest(input)
	resp, err := req.Send()
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
	return resp.SummaryMap, nil
}

// GetUserDetails collects detailed information about a user, consisting mostly
// of the groups and policies it follows.
func GetUserDetails(config aws.Config) []IAMUser {
	svc := iam.New(config)
	req := svc.ListUsersRequest(&iam.ListUsersInput{})
	resp, err := req.Send()
	if err != nil {
		panic(err)
	}
	c := make(chan IAMUser)
	userlist := make([]IAMUser, len(resp.Users))
	for _, user := range resp.Users {
		go func(user iam.User) {
			userStruct := IAMUser{
				Username: *user.UserName,
				User:     &user,
			}
			userStruct.Groups = GetGroupNameSliceForUser(user.UserName, config)
			userStruct.InlinePolicies = GetUserPoliciesMapForUser(user.UserName, config)
			userStruct.AttachedPolicies = GetAttachedPoliciesMapForUser(user.UserName, config)
			userStruct.InlineGroupPolicies = GetGroupPoliciesMapForGroups(userStruct.Groups, config)
			userStruct.AttachedGroupPolicies = GetAttachedPoliciesMapForGroups(userStruct.Groups, config)
			c <- userStruct
		}(user)
	}
	for i := 0; i < len(resp.Users); i++ {
		userlist[i] = <-c
	}
	return userlist

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
