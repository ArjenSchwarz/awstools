package helpers

import (
	"net/url"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
)

var iamSession = iam.New(session.New())

// IAMSession returns a shared IAMSession
func IAMSession() *iam.IAM {
	return iamSession
}

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
