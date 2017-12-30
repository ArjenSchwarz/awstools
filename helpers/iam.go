package helpers

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
)

var iamSession = iam.New(session.New())

type iamUser struct {
	Username              string
	AttachedPolicies      map[string]string
	InlinePolicies        map[string]string
	Groups                []string
	AttachedGroupPolicies map[string]string
	InlineGroupPolicies   map[string]string
	User                  *iam.User
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

func GetAccountPasswordPolicy() (*iam.PasswordPolicy, error) {
	svc := IAMSession()
	input := &iam.GetAccountPasswordPolicyInput{}

	result, err := svc.GetAccountPasswordPolicy(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				fmt.Println(iam.ErrCodeNoSuchEntityException, aerr.Error())
				return nil, aerr
			case iam.ErrCodeServiceFailureException:
				fmt.Println(iam.ErrCodeServiceFailureException, aerr.Error())
				panic(aerr)
			default:
				fmt.Println(aerr.Error())
				panic(aerr)
			}
		} else {
			fmt.Println(err.Error())
			panic(err)
		}
	}
	return result.PasswordPolicy, nil
}

func GetUserDetails() []iamUser {
	svc := IAMSession()
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
				User:     user,
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

func GetAllMFAs() []*iam.VirtualMFADevice {
	svc := IAMSession()
	input := &iam.ListVirtualMFADevicesInput{}

	result, err := svc.ListVirtualMFADevices(input)
	if err != nil {
		panic(err)
	}
	return result.VirtualMFADevices
}

func (user iamUser) HasMFA() bool {
	mfas := GetAllMFAs()
	for _, mfa := range mfas {
		if aws.StringValue(mfa.User.UserName) == user.Username &&
			!strings.Contains(aws.StringValue(mfa.SerialNumber), "root-account-mfa-device") {
			return true
		}
	}
	return false
}

func (user iamUser) HasPassword() bool {
	if user.User.PasswordLastUsed != nil {
		return true
	}
	return false
}

func (user iamUser) GetAccessKeys() []*iam.AccessKeyMetadata {
	svc := IAMSession()
	input := &iam.ListAccessKeysInput{
		UserName: aws.String(user.Username),
	}

	result, err := svc.ListAccessKeys(input)
	if err != nil {
		panic(err)
	}

	return result.AccessKeyMetadata
}

func (user iamUser) HasOldAccessKeys() bool {
	for _, key := range user.GetAccessKeys() {
		sinceTime := time.Since(aws.TimeValue(key.CreateDate))
		if sinceTime.Hours() > 8760 {
			return true
		}
	}
	return false
}

func (user iamUser) HasAdminAccess() bool {
	policies := user.GetAllPolicies()
	for policyname := range policies {
		if policyname == "AdministratorAccess" {
			return true
		}
	}
	return false
}

func (user iamUser) NeverActive() bool {
	svc := IAMSession()
	if user.User.PasswordLastUsed != nil {
		return false
	}
	for _, key := range user.GetAccessKeys() {
		keyusage, err := svc.GetAccessKeyLastUsed(&iam.GetAccessKeyLastUsedInput{
			AccessKeyId: key.AccessKeyId,
		})
		if err != nil {
			panic(err)
		}
		if keyusage.AccessKeyLastUsed != nil && keyusage.AccessKeyLastUsed.LastUsedDate != nil {
			return false
		}
	}
	return true
}

func (user iamUser) IsInactive() bool {
	// time since last login = 6 months = 180 * 24 hours
	comparisonHours := float64(4320)
	svc := IAMSession()
	if user.User.PasswordLastUsed != nil {
		sinceTime := time.Since(aws.TimeValue(user.User.PasswordLastUsed))
		if sinceTime.Hours() < comparisonHours {
			return false
		}
	}

	for _, key := range user.GetAccessKeys() {
		keyusage, err := svc.GetAccessKeyLastUsed(&iam.GetAccessKeyLastUsedInput{
			AccessKeyId: key.AccessKeyId,
		})
		if err != nil {
			panic(err)
		}
		if keyusage.AccessKeyLastUsed != nil && keyusage.AccessKeyLastUsed.LastUsedDate != nil {
			sinceTime := time.Since(aws.TimeValue(keyusage.AccessKeyLastUsed.LastUsedDate))
			if sinceTime.Hours() < comparisonHours {
				return false
			}
		}
	}
	return true
}
