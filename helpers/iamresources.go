package helpers

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

// IAM Role type
const (
	IAMRoleTypeSSOManaged  = "SSO Managed Role"
	IAMRoleTypeServiceRole = "Service Role"
	IAMRoleTypeUserDefined = "User defined Role"
)

// IAM Policy Type
const (
	IAMPolicyTypeAttached   = "Attached Policy"
	IAMPolicyTypeInline     = "Inline Policy"
	IAMPolicyTypeAssumeRole = "Assume Role Policy"
)

// IAM Principal Type
const (
	IAMPrincipalTypeService = "Service"
	IAMPrincipalTypeAWS     = "AWS"
)

// IAM Object Type
const (
	IAMObjectTypeGroup = "Group"
	IAMObjectTypeUser  = "User"
)

// IAMObject interface for IAM objects
type IAMObject interface {
	GetName() string
	GetID() string
	GetUsers() []string
	GetGroups() []string
	GetObjectType() string
	GetDirectPolicies() map[string]string
	GetInheritedPolicies() map[string]string
}

// IAMRole is an abstracted version of an IAM Role
type IAMRole struct {
	Name             string
	ID               string
	Path             string
	AssumeRolePolicy IAMPolicyDocument
	InlinePolicies   map[string]*IAMPolicyDocument
	AttachedPolicies map[string]*IAMPolicyDocument
	Role             *types.Role
	Type             string
	Verbose          bool
}

// CanBeAssumedFrom returns information about the assumerole policy
func (role *IAMRole) CanBeAssumedFrom() []string {
	allowances := []string{}
	for _, statement := range role.AssumeRolePolicy.Statement {
		if statement.Action == "sts:AssumeRole" || statement.Action == "sts:AssumeRoleWithSAML" {
			// Create a slice of keys for consistent ordering
			keys := make([]string, 0, len(statement.Principal))
			for key := range statement.Principal {
				keys = append(keys, key)
			}
			// Sort keys alphabetically for consistent ordering
			sort.Strings(keys)
			for _, key := range keys {
				allowance := fmt.Sprintf("%s: %s", key, statement.Principal[key])
				allowances = append(allowances, allowance)
			}
		}
	}
	return allowances
}

// GetPolicyNames returns the names of the policies attached to the role
func (role IAMRole) GetPolicyNames() []string {
	policynames := []string{}
	for policyname := range role.InlinePolicies {
		policy := fmt.Sprintf("%s (inline for %s)", policyname, role.Name)
		policynames = append(policynames, policy)
	}
	for policyname := range role.AttachedPolicies {
		policynames = append(policynames, policyname)
	}
	return policynames
}

// IAMPolicyDocument is an abstracted version of an IAM Policy Document
type IAMPolicyDocument struct {
	Name      string
	Version   string
	Type      string
	Statement []IAMPolicyDocumentStatement
	Roles     []*IAMRole
	Groups    []*IAMGroup
	Users     []*IAMUser
}

// AddRole adds the role to the policy document
func (policy *IAMPolicyDocument) AddRole(role *IAMRole) {
	policy.Roles = append(policy.Roles, role)
}

// GetRoleNames returns the names of the roles the policy is attached to
func (policy *IAMPolicyDocument) GetRoleNames() []string {
	result := []string{}
	for _, role := range policy.Roles {
		result = append(result, role.Name)
	}
	return result
}

// IAMPolicyDocumentStatement is an abstracted version of a Statement for a policy document
type IAMPolicyDocumentStatement struct {
	Effect    string
	Principal map[string]string
	Action    interface{}
	Condition interface{}
	Resource  interface{}
}

// IAMUser contains information about IAM Users
type IAMUser struct {
	Name                  string
	ID                    string
	AttachedPolicies      map[string]string
	InlinePolicies        map[string]string
	Groups                []string
	AttachedGroupPolicies map[string]string
	InlineGroupPolicies   map[string]string
	User                  *types.User
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
	return IAMObjectTypeUser
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

// HasAccessKeys checks if a user has access keys
func (user IAMUser) HasAccessKeys(svc *iam.Client) bool {
	input := &iam.ListAccessKeysInput{
		UserName: aws.String(user.Name),
	}

	result, err := svc.ListAccessKeys(context.TODO(), input)
	if err != nil {
		panic(err)
	}
	return len(result.AccessKeyMetadata) > 0
}

// GetLastAccessKeyDate returns the last date an access key was used
func (user IAMUser) GetLastAccessKeyDate(svc *iam.Client) time.Time {
	input := &iam.ListAccessKeysInput{
		UserName: aws.String(user.Name),
	}

	result, err := svc.ListAccessKeys(context.TODO(), input)
	if err != nil {
		panic(err)
	}
	var lastaccess time.Time
	for _, key := range result.AccessKeyMetadata {
		keyusage, err := svc.GetAccessKeyLastUsed(context.TODO(), &iam.GetAccessKeyLastUsedInput{
			AccessKeyId: key.AccessKeyId,
		})
		if err != nil {
			panic(err)
		}
		if keyusage.AccessKeyLastUsed != nil && keyusage.AccessKeyLastUsed.LastUsedDate != nil {
			if lastaccess.IsZero() || lastaccess.Before(*keyusage.AccessKeyLastUsed.LastUsedDate) {
				lastaccess = *keyusage.AccessKeyLastUsed.LastUsedDate
			}
		}
	}

	return lastaccess
}

// HasUsedPassword checks if the user has used their password
func (user IAMUser) HasUsedPassword() bool {
	return user.User.PasswordLastUsed != nil
}

// GetLastPasswordDate returns the last date the user's password was used
func (user IAMUser) GetLastPasswordDate() time.Time {
	return *user.User.PasswordLastUsed
}

// GetID returns the ID of the object
func (user IAMUser) GetID() string {
	return user.ID
}

// IAMGroup contains information about IAM Groups
type IAMGroup struct {
	Name             string
	ID               string
	Users            []string
	AttachedPolicies map[string]string
	InlinePolicies   map[string]string
	Group            *types.Group
}

// GetUsers returns the users attached to the Group
func (group IAMGroup) GetUsers() []string {
	return group.Users
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
	return IAMObjectTypeGroup
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

// GetID returns the ID of the object
func (group IAMGroup) GetID() string {
	return group.ID
}

// AttachedIAMPolicy is used to connect usernames, groups, and policy names
type AttachedIAMPolicy struct {
	Name   string
	Users  []string
	Groups []string
}

// AddObject adds an IAMObject (user or group) to the AttachedIAMPolicy
func (policy *AttachedIAMPolicy) AddObject(object IAMObject) {
	switch object.GetObjectType() {
	case IAMObjectTypeGroup:
		policy.Groups = append(policy.Groups, object.GetName())
	case IAMObjectTypeUser:
		policy.Users = append(policy.Users, object.GetName())
	}
}
