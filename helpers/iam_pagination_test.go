package helpers

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

// mockIAMClient implements the IAMClient interface for testing.
// It simulates paginated responses by splitting data across pages.
type mockIAMClient struct {
	users           []types.User
	groups          []types.Group
	policies        []types.Policy
	userPolicies    map[string][]string               // username -> policy names
	groupPolicies   map[string][]string               // groupname -> policy names
	attachedUserPol map[string][]types.AttachedPolicy // username -> attached policies
	attachedGrpPol  map[string][]types.AttachedPolicy // groupname -> attached policies
	groupsForUser   map[string][]types.Group          // username -> groups
	usersInGroup    map[string][]types.User           // groupname -> users
	pageSize        int                               // items per page for simulating truncation
}

// Ensure mockIAMClient satisfies IAMClient at compile time.
var _ IAMClient = (*mockIAMClient)(nil)

// ListUsers returns paginated user results. The mock uses Marker for pagination.
func (m *mockIAMClient) ListUsers(_ context.Context, input *iam.ListUsersInput, _ ...func(*iam.Options)) (*iam.ListUsersOutput, error) {
	start := 0
	if input.Marker != nil {
		fmt.Sscanf(*input.Marker, "%d", &start)
	}
	pageSize := m.pageSize
	if pageSize == 0 {
		pageSize = 100
	}
	end := start + pageSize
	if end > len(m.users) {
		end = len(m.users)
	}
	output := &iam.ListUsersOutput{
		Users:       m.users[start:end],
		IsTruncated: end < len(m.users),
	}
	if end < len(m.users) {
		marker := fmt.Sprintf("%d", end)
		output.Marker = &marker
	}
	return output, nil
}

// ListGroups returns paginated group results.
func (m *mockIAMClient) ListGroups(_ context.Context, input *iam.ListGroupsInput, _ ...func(*iam.Options)) (*iam.ListGroupsOutput, error) {
	start := 0
	if input.Marker != nil {
		fmt.Sscanf(*input.Marker, "%d", &start)
	}
	pageSize := m.pageSize
	if pageSize == 0 {
		pageSize = 100
	}
	end := start + pageSize
	if end > len(m.groups) {
		end = len(m.groups)
	}
	output := &iam.ListGroupsOutput{
		Groups:      m.groups[start:end],
		IsTruncated: end < len(m.groups),
	}
	if end < len(m.groups) {
		marker := fmt.Sprintf("%d", end)
		output.Marker = &marker
	}
	return output, nil
}

// ListPolicies returns paginated policy results.
func (m *mockIAMClient) ListPolicies(_ context.Context, input *iam.ListPoliciesInput, _ ...func(*iam.Options)) (*iam.ListPoliciesOutput, error) {
	start := 0
	if input.Marker != nil {
		fmt.Sscanf(*input.Marker, "%d", &start)
	}
	pageSize := m.pageSize
	if pageSize == 0 {
		pageSize = 100
	}
	end := start + pageSize
	if end > len(m.policies) {
		end = len(m.policies)
	}
	output := &iam.ListPoliciesOutput{
		Policies:    m.policies[start:end],
		IsTruncated: end < len(m.policies),
	}
	if end < len(m.policies) {
		marker := fmt.Sprintf("%d", end)
		output.Marker = &marker
	}
	return output, nil
}

// ListUserPolicies returns paginated inline policy names for a user.
func (m *mockIAMClient) ListUserPolicies(_ context.Context, input *iam.ListUserPoliciesInput, _ ...func(*iam.Options)) (*iam.ListUserPoliciesOutput, error) {
	allPolicies := m.userPolicies[*input.UserName]
	start := 0
	if input.Marker != nil {
		fmt.Sscanf(*input.Marker, "%d", &start)
	}
	pageSize := m.pageSize
	if pageSize == 0 {
		pageSize = 100
	}
	end := start + pageSize
	if end > len(allPolicies) {
		end = len(allPolicies)
	}
	output := &iam.ListUserPoliciesOutput{
		PolicyNames: allPolicies[start:end],
		IsTruncated: end < len(allPolicies),
	}
	if end < len(allPolicies) {
		marker := fmt.Sprintf("%d", end)
		output.Marker = &marker
	}
	return output, nil
}

// GetUserPolicy returns a user policy document.
func (m *mockIAMClient) GetUserPolicy(_ context.Context, input *iam.GetUserPolicyInput, _ ...func(*iam.Options)) (*iam.GetUserPolicyOutput, error) {
	doc := fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:Get*","Resource":"*"}]}`)
	return &iam.GetUserPolicyOutput{
		PolicyName:     input.PolicyName,
		UserName:       input.UserName,
		PolicyDocument: &doc,
	}, nil
}

// ListGroupPolicies returns paginated inline policy names for a group.
func (m *mockIAMClient) ListGroupPolicies(_ context.Context, input *iam.ListGroupPoliciesInput, _ ...func(*iam.Options)) (*iam.ListGroupPoliciesOutput, error) {
	allPolicies := m.groupPolicies[*input.GroupName]
	start := 0
	if input.Marker != nil {
		fmt.Sscanf(*input.Marker, "%d", &start)
	}
	pageSize := m.pageSize
	if pageSize == 0 {
		pageSize = 100
	}
	end := start + pageSize
	if end > len(allPolicies) {
		end = len(allPolicies)
	}
	output := &iam.ListGroupPoliciesOutput{
		PolicyNames: allPolicies[start:end],
		IsTruncated: end < len(allPolicies),
	}
	if end < len(allPolicies) {
		marker := fmt.Sprintf("%d", end)
		output.Marker = &marker
	}
	return output, nil
}

// GetGroupPolicy returns a group policy document.
func (m *mockIAMClient) GetGroupPolicy(_ context.Context, input *iam.GetGroupPolicyInput, _ ...func(*iam.Options)) (*iam.GetGroupPolicyOutput, error) {
	doc := fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:Get*","Resource":"*"}]}`)
	return &iam.GetGroupPolicyOutput{
		PolicyName:     input.PolicyName,
		GroupName:      input.GroupName,
		PolicyDocument: &doc,
	}, nil
}

// ListAttachedUserPolicies returns paginated attached policies for a user.
func (m *mockIAMClient) ListAttachedUserPolicies(_ context.Context, input *iam.ListAttachedUserPoliciesInput, _ ...func(*iam.Options)) (*iam.ListAttachedUserPoliciesOutput, error) {
	allPolicies := m.attachedUserPol[*input.UserName]
	start := 0
	if input.Marker != nil {
		fmt.Sscanf(*input.Marker, "%d", &start)
	}
	pageSize := m.pageSize
	if pageSize == 0 {
		pageSize = 100
	}
	end := start + pageSize
	if end > len(allPolicies) {
		end = len(allPolicies)
	}
	output := &iam.ListAttachedUserPoliciesOutput{
		AttachedPolicies: allPolicies[start:end],
		IsTruncated:      end < len(allPolicies),
	}
	if end < len(allPolicies) {
		marker := fmt.Sprintf("%d", end)
		output.Marker = &marker
	}
	return output, nil
}

// ListAttachedGroupPolicies returns paginated attached policies for a group.
func (m *mockIAMClient) ListAttachedGroupPolicies(_ context.Context, input *iam.ListAttachedGroupPoliciesInput, _ ...func(*iam.Options)) (*iam.ListAttachedGroupPoliciesOutput, error) {
	allPolicies := m.attachedGrpPol[*input.GroupName]
	start := 0
	if input.Marker != nil {
		fmt.Sscanf(*input.Marker, "%d", &start)
	}
	pageSize := m.pageSize
	if pageSize == 0 {
		pageSize = 100
	}
	end := start + pageSize
	if end > len(allPolicies) {
		end = len(allPolicies)
	}
	output := &iam.ListAttachedGroupPoliciesOutput{
		AttachedPolicies: allPolicies[start:end],
		IsTruncated:      end < len(allPolicies),
	}
	if end < len(allPolicies) {
		marker := fmt.Sprintf("%d", end)
		output.Marker = &marker
	}
	return output, nil
}

// ListGroupsForUser returns paginated groups for a user.
func (m *mockIAMClient) ListGroupsForUser(_ context.Context, input *iam.ListGroupsForUserInput, _ ...func(*iam.Options)) (*iam.ListGroupsForUserOutput, error) {
	allGroups := m.groupsForUser[*input.UserName]
	start := 0
	if input.Marker != nil {
		fmt.Sscanf(*input.Marker, "%d", &start)
	}
	pageSize := m.pageSize
	if pageSize == 0 {
		pageSize = 100
	}
	end := start + pageSize
	if end > len(allGroups) {
		end = len(allGroups)
	}
	output := &iam.ListGroupsForUserOutput{
		Groups:      allGroups[start:end],
		IsTruncated: end < len(allGroups),
	}
	if end < len(allGroups) {
		marker := fmt.Sprintf("%d", end)
		output.Marker = &marker
	}
	return output, nil
}

// GetGroup returns paginated users in a group.
func (m *mockIAMClient) GetGroup(_ context.Context, input *iam.GetGroupInput, _ ...func(*iam.Options)) (*iam.GetGroupOutput, error) {
	allUsers := m.usersInGroup[*input.GroupName]
	start := 0
	if input.Marker != nil {
		fmt.Sscanf(*input.Marker, "%d", &start)
	}
	pageSize := m.pageSize
	if pageSize == 0 {
		pageSize = 100
	}
	end := start + pageSize
	if end > len(allUsers) {
		end = len(allUsers)
	}
	output := &iam.GetGroupOutput{
		Users:       allUsers[start:end],
		IsTruncated: end < len(allUsers),
		Group: &types.Group{
			GroupName: input.GroupName,
			GroupId:   aws.String("AGPA" + *input.GroupName),
			Arn:       aws.String("arn:aws:iam::123456789012:group/" + *input.GroupName),
			Path:      aws.String("/"),
		},
	}
	if end < len(allUsers) {
		marker := fmt.Sprintf("%d", end)
		output.Marker = &marker
	}
	return output, nil
}

// GetPolicy returns a policy with a default version.
func (m *mockIAMClient) GetPolicy(_ context.Context, input *iam.GetPolicyInput, _ ...func(*iam.Options)) (*iam.GetPolicyOutput, error) {
	return &iam.GetPolicyOutput{
		Policy: &types.Policy{
			PolicyName:       aws.String("mock-policy"),
			Arn:              input.PolicyArn,
			DefaultVersionId: aws.String("v1"),
		},
	}, nil
}

// GetPolicyVersion returns a policy version document.
func (m *mockIAMClient) GetPolicyVersion(_ context.Context, _ *iam.GetPolicyVersionInput, _ ...func(*iam.Options)) (*iam.GetPolicyVersionOutput, error) {
	doc := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"*","Resource":"*"}]}`
	return &iam.GetPolicyVersionOutput{
		PolicyVersion: &types.PolicyVersion{
			Document:  &doc,
			VersionId: aws.String("v1"),
		},
	}, nil
}

// ListRoles stub for interface compliance.
func (m *mockIAMClient) ListRoles(_ context.Context, input *iam.ListRolesInput, _ ...func(*iam.Options)) (*iam.ListRolesOutput, error) {
	return &iam.ListRolesOutput{}, nil
}

// ListRolePolicies stub for interface compliance.
func (m *mockIAMClient) ListRolePolicies(_ context.Context, _ *iam.ListRolePoliciesInput, _ ...func(*iam.Options)) (*iam.ListRolePoliciesOutput, error) {
	return &iam.ListRolePoliciesOutput{}, nil
}

// ListAttachedRolePolicies stub for interface compliance.
func (m *mockIAMClient) ListAttachedRolePolicies(_ context.Context, _ *iam.ListAttachedRolePoliciesInput, _ ...func(*iam.Options)) (*iam.ListAttachedRolePoliciesOutput, error) {
	return &iam.ListAttachedRolePoliciesOutput{}, nil
}

// ListAccessKeys stub for interface compliance.
func (m *mockIAMClient) ListAccessKeys(_ context.Context, _ *iam.ListAccessKeysInput, _ ...func(*iam.Options)) (*iam.ListAccessKeysOutput, error) {
	return &iam.ListAccessKeysOutput{}, nil
}

// ListAccountAliases stub for interface compliance.
func (m *mockIAMClient) ListAccountAliases(_ context.Context, _ *iam.ListAccountAliasesInput, _ ...func(*iam.Options)) (*iam.ListAccountAliasesOutput, error) {
	return &iam.ListAccountAliasesOutput{}, nil
}

// GetRolePolicy stub for interface compliance.
func (m *mockIAMClient) GetRolePolicy(_ context.Context, _ *iam.GetRolePolicyInput, _ ...func(*iam.Options)) (*iam.GetRolePolicyOutput, error) {
	return &iam.GetRolePolicyOutput{}, nil
}

// GetAccountSummary stub for interface compliance.
func (m *mockIAMClient) GetAccountSummary(_ context.Context, _ *iam.GetAccountSummaryInput, _ ...func(*iam.Options)) (*iam.GetAccountSummaryOutput, error) {
	return &iam.GetAccountSummaryOutput{SummaryMap: map[string]int32{}}, nil
}

// GetAccessKeyLastUsed stub for interface compliance.
func (m *mockIAMClient) GetAccessKeyLastUsed(_ context.Context, _ *iam.GetAccessKeyLastUsedInput, _ ...func(*iam.Options)) (*iam.GetAccessKeyLastUsedOutput, error) {
	return &iam.GetAccessKeyLastUsedOutput{}, nil
}

// makeUsers creates n test users.
func makeUsers(n int) []types.User {
	users := make([]types.User, n)
	for i := range n {
		name := fmt.Sprintf("user-%03d", i)
		id := fmt.Sprintf("AIDA%012d", i)
		users[i] = types.User{
			UserName: &name,
			UserId:   &id,
			Arn:      aws.String(fmt.Sprintf("arn:aws:iam::123456789012:user/%s", name)),
			Path:     aws.String("/"),
		}
	}
	return users
}

// makeGroups creates n test groups.
func makeGroups(n int) []types.Group {
	groups := make([]types.Group, n)
	for i := range n {
		name := fmt.Sprintf("group-%03d", i)
		id := fmt.Sprintf("AGPA%012d", i)
		groups[i] = types.Group{
			GroupName: &name,
			GroupId:   &id,
			Arn:       aws.String(fmt.Sprintf("arn:aws:iam::123456789012:group/%s", name)),
			Path:      aws.String("/"),
		}
	}
	return groups
}

// makePolicyNames creates n policy names.
func makePolicyNames(n int, prefix string) []string {
	names := make([]string, n)
	for i := range n {
		names[i] = fmt.Sprintf("%s-policy-%03d", prefix, i)
	}
	return names
}

// makeAttachedPolicies creates n attached policies.
func makeAttachedPolicies(n int, prefix string) []types.AttachedPolicy {
	policies := make([]types.AttachedPolicy, n)
	for i := range n {
		name := fmt.Sprintf("%s-attached-%03d", prefix, i)
		arn := fmt.Sprintf("arn:aws:iam::123456789012:policy/%s", name)
		policies[i] = types.AttachedPolicy{
			PolicyName: &name,
			PolicyArn:  &arn,
		}
	}
	return policies
}

// TestGetUserList_Pagination verifies that getUserList retrieves all users
// across multiple pages. Before the fix, only the first page was returned.
func TestGetUserList_Pagination(t *testing.T) {
	// Reset the cached users so we get a fresh call
	cachedUsers = nil

	totalUsers := 5
	mock := &mockIAMClient{
		users:    makeUsers(totalUsers),
		pageSize: 2, // force 3 pages: [0,1], [2,3], [4]
	}

	result := getUserList(mock)

	if len(result) != totalUsers {
		t.Errorf("getUserList() returned %d users, want %d (pagination bug: only first page returned)", len(result), totalUsers)
	}

	// Verify all users are present
	for i, user := range result {
		expected := fmt.Sprintf("user-%03d", i)
		if *user.UserName != expected {
			t.Errorf("getUserList()[%d].UserName = %s, want %s", i, *user.UserName, expected)
		}
	}

	// Clean up cache
	cachedUsers = nil
}

// TestGetGroupNameSliceForUser_Pagination verifies that all groups for a user
// are returned across multiple pages.
func TestGetGroupNameSliceForUser_Pagination(t *testing.T) {
	totalGroups := 5
	groups := makeGroups(totalGroups)

	mock := &mockIAMClient{
		groupsForUser: map[string][]types.Group{
			"test-user": groups,
		},
		pageSize: 2,
	}

	username := "test-user"
	result := GetGroupNameSliceForUser(&username, mock)

	if len(result) != totalGroups {
		t.Errorf("GetGroupNameSliceForUser() returned %d groups, want %d (pagination bug)", len(result), totalGroups)
	}

	for i, name := range result {
		expected := fmt.Sprintf("group-%03d", i)
		if name != expected {
			t.Errorf("GetGroupNameSliceForUser()[%d] = %s, want %s", i, name, expected)
		}
	}
}

// TestGetAllUsersInGroup_Pagination verifies that all users in a group
// are returned across multiple pages.
func TestGetAllUsersInGroup_Pagination(t *testing.T) {
	totalUsers := 5
	users := makeUsers(totalUsers)

	mock := &mockIAMClient{
		usersInGroup: map[string][]types.User{
			"test-group": users,
		},
		pageSize: 2,
	}

	result := getAllUsersInGroup("test-group", mock)

	if len(result) != totalUsers {
		t.Errorf("getAllUsersInGroup() returned %d users, want %d (pagination bug)", len(result), totalUsers)
	}
}

// TestGetUserPoliciesMapForUser_Pagination verifies that all inline policies
// for a user are returned across multiple pages.
func TestGetUserPoliciesMapForUser_Pagination(t *testing.T) {
	totalPolicies := 5
	policyNames := makePolicyNames(totalPolicies, "user-inline")

	mock := &mockIAMClient{
		userPolicies: map[string][]string{
			"test-user": policyNames,
		},
		pageSize: 2,
	}

	username := "test-user"
	result := GetUserPoliciesMapForUser(&username, mock)

	if len(result) != totalPolicies {
		t.Errorf("GetUserPoliciesMapForUser() returned %d policies, want %d (pagination bug)", len(result), totalPolicies)
	}
}

// TestGetGroupPoliciesMapForGroup_Pagination verifies that all inline policies
// for a group are returned across multiple pages.
func TestGetGroupPoliciesMapForGroup_Pagination(t *testing.T) {
	totalPolicies := 5
	policyNames := makePolicyNames(totalPolicies, "group-inline")

	mock := &mockIAMClient{
		groupPolicies: map[string][]string{
			"test-group": policyNames,
		},
		pageSize: 2,
	}

	groupname := "test-group"
	result := GetGroupPoliciesMapForGroup(&groupname, mock)

	if len(result) != totalPolicies {
		t.Errorf("GetGroupPoliciesMapForGroup() returned %d policies, want %d (pagination bug)", len(result), totalPolicies)
	}
}

// TestGetAttachedPoliciesMapForUser_Pagination verifies that all attached
// policies for a user are returned across multiple pages.
func TestGetAttachedPoliciesMapForUser_Pagination(t *testing.T) {
	totalPolicies := 5
	policies := makeAttachedPolicies(totalPolicies, "user")

	mock := &mockIAMClient{
		attachedUserPol: map[string][]types.AttachedPolicy{
			"test-user": policies,
		},
		pageSize: 2,
	}

	username := "test-user"
	result := GetAttachedPoliciesMapForUser(&username, mock)

	if len(result) != totalPolicies {
		t.Errorf("GetAttachedPoliciesMapForUser() returned %d policies, want %d (pagination bug)", len(result), totalPolicies)
	}
}

// TestGetAttachedPoliciesMapForGroup_Pagination verifies that all attached
// policies for a group are returned across multiple pages.
func TestGetAttachedPoliciesMapForGroup_Pagination(t *testing.T) {
	totalPolicies := 5
	policies := makeAttachedPolicies(totalPolicies, "group")

	mock := &mockIAMClient{
		attachedGrpPol: map[string][]types.AttachedPolicy{
			"test-group": policies,
		},
		pageSize: 2,
	}

	groupname := "test-group"
	result := GetAttachedPoliciesMapForGroup(&groupname, mock)

	if len(result) != totalPolicies {
		t.Errorf("GetAttachedPoliciesMapForGroup() returned %d policies, want %d (pagination bug)", len(result), totalPolicies)
	}
}

// TestGetPoliciesMap_Pagination verifies that all policies are returned
// across multiple pages.
func TestGetPoliciesMap_Pagination(t *testing.T) {
	totalPolicies := 5
	policies := make([]types.Policy, totalPolicies)
	for i := range totalPolicies {
		name := fmt.Sprintf("policy-%03d", i)
		arn := fmt.Sprintf("arn:aws:iam::123456789012:policy/%s", name)
		policies[i] = types.Policy{
			PolicyName: &name,
			Arn:        &arn,
		}
	}

	mock := &mockIAMClient{
		policies: policies,
		pageSize: 2,
	}

	result := GetPoliciesMap(mock)

	if len(result) != totalPolicies {
		t.Errorf("GetPoliciesMap() returned %d policies, want %d (pagination bug)", len(result), totalPolicies)
	}
}
