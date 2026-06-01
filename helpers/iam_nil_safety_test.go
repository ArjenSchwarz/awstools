package helpers

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

// partialIAMClient is a mock IAMClient that simulates partial AWS SDK responses
// where optional pointer fields (names, IDs, documents, paths) are nil. These
// partial responses can occur in practice and previously caused the IAM helper
// paths to panic with nil pointer dereferences (T-1240).
type partialIAMClient struct {
	users  []types.User
	groups []types.Group
	roles  []types.Role

	// userPolicyNames maps a username to the inline policy names returned by
	// ListUserPolicies.
	userPolicyNames map[string][]string
	// attachedUserPolicies maps a username to attached policies whose
	// PolicyName may be nil.
	attachedUserPolicies map[string][]types.AttachedPolicy
	// groupsForUser maps a username to groups (with possibly-nil GroupName).
	groupsForUser map[string][]types.Group
	// usersInGroup maps a groupname to users (with possibly-nil UserName).
	usersInGroup map[string][]types.User

	// nilUserPolicyDocument makes GetUserPolicy return a nil PolicyDocument.
	nilUserPolicyDocument bool
	// nilGroupPolicyDocument makes GetGroupPolicy return a nil PolicyDocument.
	nilGroupPolicyDocument bool
	// nilDefaultVersionID makes GetPolicy return a Policy with a nil
	// DefaultVersionId.
	nilDefaultVersionID bool
	// nilPolicyVersionDocument makes GetPolicyVersion return a PolicyVersion
	// with a nil Document.
	nilPolicyVersionDocument bool
}

var _ IAMClient = (*partialIAMClient)(nil)

func (m *partialIAMClient) ListUsers(_ context.Context, _ *iam.ListUsersInput, _ ...func(*iam.Options)) (*iam.ListUsersOutput, error) {
	return &iam.ListUsersOutput{Users: m.users}, nil
}

func (m *partialIAMClient) ListGroups(_ context.Context, _ *iam.ListGroupsInput, _ ...func(*iam.Options)) (*iam.ListGroupsOutput, error) {
	return &iam.ListGroupsOutput{Groups: m.groups}, nil
}

func (m *partialIAMClient) ListPolicies(_ context.Context, _ *iam.ListPoliciesInput, _ ...func(*iam.Options)) (*iam.ListPoliciesOutput, error) {
	// One policy with a nil PolicyName, to exercise GetPoliciesMap.
	return &iam.ListPoliciesOutput{Policies: []types.Policy{{Arn: aws.String("arn:aws:iam::123456789012:policy/p")}}}, nil
}

func (m *partialIAMClient) ListUserPolicies(_ context.Context, input *iam.ListUserPoliciesInput, _ ...func(*iam.Options)) (*iam.ListUserPoliciesOutput, error) {
	var names []string
	if input.UserName != nil {
		names = m.userPolicyNames[*input.UserName]
	}
	return &iam.ListUserPoliciesOutput{PolicyNames: names}, nil
}

func (m *partialIAMClient) ListGroupPolicies(_ context.Context, _ *iam.ListGroupPoliciesInput, _ ...func(*iam.Options)) (*iam.ListGroupPoliciesOutput, error) {
	return &iam.ListGroupPoliciesOutput{PolicyNames: []string{"inline-group-policy"}}, nil
}

func (m *partialIAMClient) ListAttachedUserPolicies(_ context.Context, input *iam.ListAttachedUserPoliciesInput, _ ...func(*iam.Options)) (*iam.ListAttachedUserPoliciesOutput, error) {
	var pols []types.AttachedPolicy
	if input.UserName != nil {
		pols = m.attachedUserPolicies[*input.UserName]
	}
	return &iam.ListAttachedUserPoliciesOutput{AttachedPolicies: pols}, nil
}

func (m *partialIAMClient) ListAttachedGroupPolicies(_ context.Context, _ *iam.ListAttachedGroupPoliciesInput, _ ...func(*iam.Options)) (*iam.ListAttachedGroupPoliciesOutput, error) {
	// Attached policy with a nil PolicyName.
	return &iam.ListAttachedGroupPoliciesOutput{AttachedPolicies: []types.AttachedPolicy{{
		PolicyArn: aws.String("arn:aws:iam::123456789012:policy/attached"),
	}}}, nil
}

func (m *partialIAMClient) ListGroupsForUser(_ context.Context, input *iam.ListGroupsForUserInput, _ ...func(*iam.Options)) (*iam.ListGroupsForUserOutput, error) {
	var groups []types.Group
	if input.UserName != nil {
		groups = m.groupsForUser[*input.UserName]
	}
	return &iam.ListGroupsForUserOutput{Groups: groups}, nil
}

func (m *partialIAMClient) GetGroup(_ context.Context, input *iam.GetGroupInput, _ ...func(*iam.Options)) (*iam.GetGroupOutput, error) {
	var users []types.User
	if input.GroupName != nil {
		users = m.usersInGroup[*input.GroupName]
	}
	return &iam.GetGroupOutput{Users: users, Group: &types.Group{GroupName: input.GroupName}}, nil
}

func (m *partialIAMClient) GetUserPolicy(_ context.Context, input *iam.GetUserPolicyInput, _ ...func(*iam.Options)) (*iam.GetUserPolicyOutput, error) {
	out := &iam.GetUserPolicyOutput{PolicyName: input.PolicyName, UserName: input.UserName}
	if !m.nilUserPolicyDocument {
		out.PolicyDocument = aws.String("%7B%7D")
	}
	return out, nil
}

func (m *partialIAMClient) GetGroupPolicy(_ context.Context, input *iam.GetGroupPolicyInput, _ ...func(*iam.Options)) (*iam.GetGroupPolicyOutput, error) {
	out := &iam.GetGroupPolicyOutput{PolicyName: input.PolicyName, GroupName: input.GroupName}
	if !m.nilGroupPolicyDocument {
		out.PolicyDocument = aws.String("%7B%7D")
	}
	return out, nil
}

func (m *partialIAMClient) GetPolicy(_ context.Context, input *iam.GetPolicyInput, _ ...func(*iam.Options)) (*iam.GetPolicyOutput, error) {
	policy := &types.Policy{Arn: input.PolicyArn}
	if !m.nilDefaultVersionID {
		policy.DefaultVersionId = aws.String("v1")
	}
	return &iam.GetPolicyOutput{Policy: policy}, nil
}

func (m *partialIAMClient) GetPolicyVersion(_ context.Context, _ *iam.GetPolicyVersionInput, _ ...func(*iam.Options)) (*iam.GetPolicyVersionOutput, error) {
	version := &types.PolicyVersion{VersionId: aws.String("v1")}
	if !m.nilPolicyVersionDocument {
		version.Document = aws.String("%7B%7D")
	}
	return &iam.GetPolicyVersionOutput{PolicyVersion: version}, nil
}

func (m *partialIAMClient) ListRoles(_ context.Context, _ *iam.ListRolesInput, _ ...func(*iam.Options)) (*iam.ListRolesOutput, error) {
	return &iam.ListRolesOutput{Roles: m.roles}, nil
}

func (m *partialIAMClient) ListRolePolicies(_ context.Context, _ *iam.ListRolePoliciesInput, _ ...func(*iam.Options)) (*iam.ListRolePoliciesOutput, error) {
	return &iam.ListRolePoliciesOutput{}, nil
}

func (m *partialIAMClient) ListAttachedRolePolicies(_ context.Context, _ *iam.ListAttachedRolePoliciesInput, _ ...func(*iam.Options)) (*iam.ListAttachedRolePoliciesOutput, error) {
	return &iam.ListAttachedRolePoliciesOutput{}, nil
}

func (m *partialIAMClient) ListAccessKeys(_ context.Context, _ *iam.ListAccessKeysInput, _ ...func(*iam.Options)) (*iam.ListAccessKeysOutput, error) {
	return &iam.ListAccessKeysOutput{}, nil
}

func (m *partialIAMClient) ListAccountAliases(_ context.Context, _ *iam.ListAccountAliasesInput, _ ...func(*iam.Options)) (*iam.ListAccountAliasesOutput, error) {
	return &iam.ListAccountAliasesOutput{}, nil
}

func (m *partialIAMClient) GetRolePolicy(_ context.Context, _ *iam.GetRolePolicyInput, _ ...func(*iam.Options)) (*iam.GetRolePolicyOutput, error) {
	return &iam.GetRolePolicyOutput{}, nil
}

func (m *partialIAMClient) GetAccountSummary(_ context.Context, _ *iam.GetAccountSummaryInput, _ ...func(*iam.Options)) (*iam.GetAccountSummaryOutput, error) {
	return &iam.GetAccountSummaryOutput{SummaryMap: map[string]int32{}}, nil
}

func (m *partialIAMClient) GetAccessKeyLastUsed(_ context.Context, _ *iam.GetAccessKeyLastUsedInput, _ ...func(*iam.Options)) (*iam.GetAccessKeyLastUsedOutput, error) {
	return &iam.GetAccessKeyLastUsedOutput{}, nil
}

// TestGetPoliciesMap_NilPolicyName verifies GetPoliciesMap does not panic when
// a policy is returned without a PolicyName.
func TestGetPoliciesMap_NilPolicyName(t *testing.T) {
	mock := &partialIAMClient{}
	result := GetPoliciesMap(mock) // must not panic on *policy.PolicyName
	if len(result) != 1 {
		t.Errorf("GetPoliciesMap() returned %d entries, want 1", len(result))
	}
}

// TestGetUserPoliciesMapForUser_NilDocument verifies the helper does not panic
// when GetUserPolicy returns a nil PolicyDocument.
func TestGetUserPoliciesMapForUser_NilDocument(t *testing.T) {
	mock := &partialIAMClient{
		userPolicyNames:       map[string][]string{"u": {"p1"}},
		nilUserPolicyDocument: true,
	}
	username := "u"
	_ = GetUserPoliciesMapForUser(&username, mock) // must not panic on *resp.PolicyDocument
}

// TestGetGroupPoliciesMapForGroup_NilDocument verifies the helper does not
// panic when GetGroupPolicy returns a nil PolicyDocument.
func TestGetGroupPoliciesMapForGroup_NilDocument(t *testing.T) {
	mock := &partialIAMClient{nilGroupPolicyDocument: true}
	groupname := "g"
	_ = GetGroupPoliciesMapForGroup(&groupname, mock) // must not panic on *resp.PolicyDocument
}

// TestGetAttachedPoliciesMapForGroup_NilPolicyName verifies the helper does not
// panic when an attached policy has a nil PolicyName.
func TestGetAttachedPoliciesMapForGroup_NilPolicyName(t *testing.T) {
	mock := &partialIAMClient{}
	groupname := "g"
	_ = GetAttachedPoliciesMapForGroup(&groupname, mock) // must not panic on *policy.PolicyName
}

// TestGetAttachedPolicy_NilFields verifies getAttachedPolicy (via the user path)
// does not panic when DefaultVersionId or the version Document is nil.
func TestGetAttachedPolicy_NilFields(t *testing.T) {
	attached := []types.AttachedPolicy{{
		PolicyName: aws.String("attached"),
		PolicyArn:  aws.String("arn:aws:iam::123456789012:policy/attached"),
	}}
	mock := &partialIAMClient{
		attachedUserPolicies:     map[string][]types.AttachedPolicy{"u": attached},
		nilDefaultVersionID:      true,
		nilPolicyVersionDocument: true,
	}
	username := "u"
	_ = GetAttachedPoliciesMapForUser(&username, mock) // must not panic on resp.Policy.DefaultVersionId / *resp2.PolicyVersion.Document
}

// TestGetGroupNameSliceForUser_NilGroupName verifies the helper does not panic
// when a group is returned with a nil GroupName.
func TestGetGroupNameSliceForUser_NilGroupName(t *testing.T) {
	mock := &partialIAMClient{
		groupsForUser: map[string][]types.Group{"u": {{GroupId: aws.String("id")}}},
	}
	username := "u"
	_ = GetGroupNameSliceForUser(&username, mock) // must not panic on *group.GroupName
}

// TestGetUserDetails_NilUserName verifies GetUserDetails does not panic when a
// user is returned with a nil UserName.
func TestGetUserDetails_NilUserName(t *testing.T) {
	cachedUsers = nil
	defer func() { cachedUsers = nil }()
	mock := &partialIAMClient{
		users: []types.User{{UserId: aws.String("AIDAEXAMPLE")}}, // nil UserName
	}
	result := GetUserDetails(mock) // must not panic on *user.UserName
	if len(result) != 1 {
		t.Errorf("GetUserDetails() returned %d users, want 1", len(result))
	}
}

// TestGetGroupDetails_NilGroupName verifies GetGroupDetails does not panic when
// a group is returned with a nil GroupName.
func TestGetGroupDetails_NilGroupName(t *testing.T) {
	mock := &partialIAMClient{
		groups:       []types.Group{{GroupId: aws.String("AGPAEXAMPLE")}}, // nil GroupName
		usersInGroup: map[string][]types.User{},
	}
	result := GetGroupDetails(mock) // must not panic on *group.GroupName
	if len(result) != 1 {
		t.Errorf("GetGroupDetails() returned %d groups, want 1", len(result))
	}
}

// TestGetAllUsersInGroup_NilUserName verifies getAllUsersInGroup does not panic
// when a member user has a nil UserName.
func TestGetAllUsersInGroup_NilUserName(t *testing.T) {
	mock := &partialIAMClient{
		usersInGroup: map[string][]types.User{"g": {{UserId: aws.String("AIDAEXAMPLE")}}},
	}
	_ = getAllUsersInGroup("g", mock) // must not panic on *user.UserName
}

// TestGetRoleType_NilPath verifies getRoleType does not panic when Path is nil.
func TestGetRoleType_NilPath(t *testing.T) {
	role := types.Role{} // nil Path
	got := getRoleType(role)
	if got != IAMRoleTypeUserDefined {
		t.Errorf("getRoleType(nil path) = %s, want %s", got, IAMRoleTypeUserDefined)
	}
}

// TestGetRoleDetails_PartialMetadata verifies GetRoleDetails does not panic when
// a role is returned with nil AssumeRolePolicyDocument, RoleName, RoleId, and
// Path.
func TestGetRoleDetails_PartialMetadata(t *testing.T) {
	mock := &partialIAMClient{
		roles: []types.Role{{Arn: aws.String("arn:aws:iam::123456789012:role/r")}},
	}
	result := GetRoleDetails(false, mock) // must not panic on nil role pointer fields
	if len(result) != 1 {
		t.Errorf("GetRoleDetails() returned %d roles, want 1", len(result))
	}
}
