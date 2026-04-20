package helpers

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
	ssotypes "github.com/aws/aws-sdk-go-v2/service/ssoadmin/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSSOAdminClient implements SSOAdminAPI for testing.
type mockSSOAdminClient struct {
	ListInstancesFunc                           func(ctx context.Context, params *ssoadmin.ListInstancesInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListInstancesOutput, error)
	ListPermissionSetsFunc                      func(ctx context.Context, params *ssoadmin.ListPermissionSetsInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListPermissionSetsOutput, error)
	DescribePermissionSetFunc                   func(ctx context.Context, params *ssoadmin.DescribePermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.DescribePermissionSetOutput, error)
	ListAccountsForProvisionedPermissionSetFunc func(ctx context.Context, params *ssoadmin.ListAccountsForProvisionedPermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListAccountsForProvisionedPermissionSetOutput, error)
	ListAccountAssignmentsFunc                  func(ctx context.Context, params *ssoadmin.ListAccountAssignmentsInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListAccountAssignmentsOutput, error)
	ListManagedPoliciesInPermissionSetFunc      func(ctx context.Context, params *ssoadmin.ListManagedPoliciesInPermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListManagedPoliciesInPermissionSetOutput, error)
	GetInlinePolicyForPermissionSetFunc         func(ctx context.Context, params *ssoadmin.GetInlinePolicyForPermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.GetInlinePolicyForPermissionSetOutput, error)
}

func (m *mockSSOAdminClient) ListInstances(ctx context.Context, params *ssoadmin.ListInstancesInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListInstancesOutput, error) {
	return m.ListInstancesFunc(ctx, params, optFns...)
}

func (m *mockSSOAdminClient) ListPermissionSets(ctx context.Context, params *ssoadmin.ListPermissionSetsInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListPermissionSetsOutput, error) {
	return m.ListPermissionSetsFunc(ctx, params, optFns...)
}

func (m *mockSSOAdminClient) DescribePermissionSet(ctx context.Context, params *ssoadmin.DescribePermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.DescribePermissionSetOutput, error) {
	return m.DescribePermissionSetFunc(ctx, params, optFns...)
}

func (m *mockSSOAdminClient) ListAccountsForProvisionedPermissionSet(ctx context.Context, params *ssoadmin.ListAccountsForProvisionedPermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListAccountsForProvisionedPermissionSetOutput, error) {
	return m.ListAccountsForProvisionedPermissionSetFunc(ctx, params, optFns...)
}

func (m *mockSSOAdminClient) ListAccountAssignments(ctx context.Context, params *ssoadmin.ListAccountAssignmentsInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListAccountAssignmentsOutput, error) {
	return m.ListAccountAssignmentsFunc(ctx, params, optFns...)
}

func (m *mockSSOAdminClient) ListManagedPoliciesInPermissionSet(ctx context.Context, params *ssoadmin.ListManagedPoliciesInPermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListManagedPoliciesInPermissionSetOutput, error) {
	return m.ListManagedPoliciesInPermissionSetFunc(ctx, params, optFns...)
}

func (m *mockSSOAdminClient) GetInlinePolicyForPermissionSet(ctx context.Context, params *ssoadmin.GetInlinePolicyForPermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.GetInlinePolicyForPermissionSetOutput, error) {
	return m.GetInlinePolicyForPermissionSetFunc(ctx, params, optFns...)
}

func TestSSOInstance_Struct(t *testing.T) {
	instance := SSOInstance{
		IdentityStoreID: "d-1234567890",
		Arn:             "arn:aws:sso:::instance/ssoins-1234567890abcdef",
		PermissionSets:  []SSOPermissionSet{},
		Accounts:        make(map[string]SSOAccount),
	}

	if instance.IdentityStoreID != "d-1234567890" {
		t.Errorf("Expected IdentityStoreID to be 'd-1234567890', got %s", instance.IdentityStoreID)
	}

	if instance.Arn != "arn:aws:sso:::instance/ssoins-1234567890abcdef" {
		t.Errorf("Expected Arn to be 'arn:aws:sso:::instance/ssoins-1234567890abcdef', got %s", instance.Arn)
	}

	if len(instance.PermissionSets) != 0 {
		t.Errorf("Expected PermissionSets to be empty, got %d", len(instance.PermissionSets))
	}

	if len(instance.Accounts) != 0 {
		t.Errorf("Expected Accounts to be empty, got %d", len(instance.Accounts))
	}
}

func TestSSOPermissionSet_Struct(t *testing.T) {
	createdTime := time.Now()
	permissionSet := SSOPermissionSet{
		Arn:             "arn:aws:sso:::permissionSet/ssoins-1234567890abcdef/ps-1234567890abcdef",
		Name:            "ReadOnlyAccess",
		Description:     "Read-only access to AWS resources",
		CreatedAt:       createdTime,
		SessionDuration: "PT8H",
		Accounts:        []SSOAccount{},
		ManagedPolicies: []SSOPolicy{},
		InlinePolicy:    "",
		Instance:        nil,
	}

	if permissionSet.Name != "ReadOnlyAccess" {
		t.Errorf("Expected Name to be 'ReadOnlyAccess', got %s", permissionSet.Name)
	}

	if permissionSet.Description != "Read-only access to AWS resources" {
		t.Errorf("Expected Description to be 'Read-only access to AWS resources', got %s", permissionSet.Description)
	}

	if permissionSet.SessionDuration != "PT8H" {
		t.Errorf("Expected SessionDuration to be 'PT8H', got %s", permissionSet.SessionDuration)
	}

	if !permissionSet.CreatedAt.Equal(createdTime) {
		t.Errorf("Expected CreatedAt to be %v, got %v", createdTime, permissionSet.CreatedAt)
	}

	if len(permissionSet.Accounts) != 0 {
		t.Errorf("Expected Accounts to be empty, got %d", len(permissionSet.Accounts))
	}

	if len(permissionSet.ManagedPolicies) != 0 {
		t.Errorf("Expected ManagedPolicies to be empty, got %d", len(permissionSet.ManagedPolicies))
	}
}

func TestSSOAccount_Struct(t *testing.T) {
	account := SSOAccount{
		AccountID: "123456789012",
	}

	if account.AccountID != "123456789012" {
		t.Errorf("Expected AccountID to be '123456789012', got %s", account.AccountID)
	}
}

func TestSSOPolicy_Struct(t *testing.T) {
	policy := SSOPolicy{
		Arn:  "arn:aws:iam::aws:policy/ReadOnlyAccess",
		Name: "ReadOnlyAccess",
	}

	if policy.Arn != "arn:aws:iam::aws:policy/ReadOnlyAccess" {
		t.Errorf("Expected Arn to be 'arn:aws:iam::aws:policy/ReadOnlyAccess', got %s", policy.Arn)
	}

	if policy.Name != "ReadOnlyAccess" {
		t.Errorf("Expected Name to be 'ReadOnlyAccess', got %s", policy.Name)
	}
}

func TestSSOPermissionSet_WithManagedPolicies(t *testing.T) {
	policy1 := SSOPolicy{
		Arn:  "arn:aws:iam::aws:policy/ReadOnlyAccess",
		Name: "ReadOnlyAccess",
	}

	policy2 := SSOPolicy{
		Arn:  "arn:aws:iam::aws:policy/IAMReadOnlyAccess",
		Name: "IAMReadOnlyAccess",
	}

	permissionSet := SSOPermissionSet{
		Name:            "MultiPolicySet",
		ManagedPolicies: []SSOPolicy{policy1, policy2},
	}

	if len(permissionSet.ManagedPolicies) != 2 {
		t.Errorf("Expected 2 managed policies, got %d", len(permissionSet.ManagedPolicies))
	}

	if permissionSet.ManagedPolicies[0].Name != "ReadOnlyAccess" {
		t.Errorf("Expected first policy name to be 'ReadOnlyAccess', got %s", permissionSet.ManagedPolicies[0].Name)
	}

	if permissionSet.ManagedPolicies[1].Name != "IAMReadOnlyAccess" {
		t.Errorf("Expected second policy name to be 'IAMReadOnlyAccess', got %s", permissionSet.ManagedPolicies[1].Name)
	}
}

// Regression test for T-891: ListInstances must paginate across all pages.
// AWS caps ListInstances at a small page size, so discovery must follow
// NextToken to avoid silently dropping the caller's SSO instance when it
// happens to appear on a later page.
func TestListInstances_Pagination(t *testing.T) {
	mock := newBasicMock()
	listInstancesCalls := 0
	mock.ListInstancesFunc = func(ctx context.Context, params *ssoadmin.ListInstancesInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListInstancesOutput, error) {
		listInstancesCalls++
		switch listInstancesCalls {
		case 1:
			assert.Nil(t, params.NextToken, "first call should not have NextToken")
			return &ssoadmin.ListInstancesOutput{
				Instances: []ssotypes.InstanceMetadata{},
				NextToken: aws.String("page2"),
			}, nil
		case 2:
			assert.Equal(t, "page2", *params.NextToken)
			return &ssoadmin.ListInstancesOutput{
				Instances: []ssotypes.InstanceMetadata{
					{
						IdentityStoreId: aws.String("d-laterpage"),
						InstanceArn:     aws.String("arn:aws:sso:::instance/ssoins-laterpage"),
					},
				},
				NextToken: nil,
			}, nil
		default:
			t.Fatal("ListInstances called too many times")
			return nil, nil
		}
	}

	instance, err := GetSSOAccountInstance(mock)
	require.NoError(t, err)
	assert.Equal(t, 2, listInstancesCalls, "ListInstances should be called twice to follow NextToken")
	assert.Equal(t, "d-laterpage", instance.IdentityStoreID,
		"should find the SSO instance on the second page")
	assert.Equal(t, "arn:aws:sso:::instance/ssoins-laterpage", instance.Arn)
}

// Regression tests for T-479: pagination must collect all pages

// newBasicMock returns a mock with common defaults set for a single permission set.
// Callers override specific functions to test pagination scenarios.
func newBasicMock() *mockSSOAdminClient {
	createdDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	return &mockSSOAdminClient{
		ListInstancesFunc: func(ctx context.Context, params *ssoadmin.ListInstancesInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListInstancesOutput, error) {
			return &ssoadmin.ListInstancesOutput{
				Instances: []ssotypes.InstanceMetadata{
					{
						IdentityStoreId: aws.String("d-1234567890"),
						InstanceArn:     aws.String("arn:aws:sso:::instance/ssoins-abc"),
					},
				},
			}, nil
		},
		ListPermissionSetsFunc: func(ctx context.Context, params *ssoadmin.ListPermissionSetsInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListPermissionSetsOutput, error) {
			return &ssoadmin.ListPermissionSetsOutput{
				PermissionSets: []string{"arn:aws:sso:::permissionSet/ps-001"},
			}, nil
		},
		DescribePermissionSetFunc: func(ctx context.Context, params *ssoadmin.DescribePermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.DescribePermissionSetOutput, error) {
			return &ssoadmin.DescribePermissionSetOutput{
				PermissionSet: &ssotypes.PermissionSet{
					Name:            aws.String("TestPS"),
					CreatedDate:     &createdDate,
					SessionDuration: aws.String("PT1H"),
				},
			}, nil
		},
		ListAccountsForProvisionedPermissionSetFunc: func(ctx context.Context, params *ssoadmin.ListAccountsForProvisionedPermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListAccountsForProvisionedPermissionSetOutput, error) {
			return &ssoadmin.ListAccountsForProvisionedPermissionSetOutput{
				AccountIds: []string{},
			}, nil
		},
		ListAccountAssignmentsFunc: func(ctx context.Context, params *ssoadmin.ListAccountAssignmentsInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListAccountAssignmentsOutput, error) {
			return &ssoadmin.ListAccountAssignmentsOutput{
				AccountAssignments: []ssotypes.AccountAssignment{},
			}, nil
		},
		ListManagedPoliciesInPermissionSetFunc: func(ctx context.Context, params *ssoadmin.ListManagedPoliciesInPermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListManagedPoliciesInPermissionSetOutput, error) {
			return &ssoadmin.ListManagedPoliciesInPermissionSetOutput{
				AttachedManagedPolicies: []ssotypes.AttachedManagedPolicy{},
			}, nil
		},
		GetInlinePolicyForPermissionSetFunc: func(ctx context.Context, params *ssoadmin.GetInlinePolicyForPermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.GetInlinePolicyForPermissionSetOutput, error) {
			return &ssoadmin.GetInlinePolicyForPermissionSetOutput{
				InlinePolicy: aws.String(""),
			}, nil
		},
	}
}

func TestListPermissionSets_Pagination(t *testing.T) {
	mock := newBasicMock()
	callCount := 0
	mock.ListPermissionSetsFunc = func(ctx context.Context, params *ssoadmin.ListPermissionSetsInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListPermissionSetsOutput, error) {
		callCount++
		switch callCount {
		case 1:
			assert.Nil(t, params.NextToken, "first call should not have NextToken")
			return &ssoadmin.ListPermissionSetsOutput{
				PermissionSets: []string{"arn:ps-001", "arn:ps-002"},
				NextToken:      aws.String("page2"),
			}, nil
		case 2:
			assert.Equal(t, "page2", *params.NextToken)
			return &ssoadmin.ListPermissionSetsOutput{
				PermissionSets: []string{"arn:ps-003"},
				NextToken:      nil,
			}, nil
		default:
			t.Fatal("ListPermissionSets called too many times")
			return nil, nil
		}
	}

	// DescribePermissionSet needs to handle three different ARNs
	createdDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	mock.DescribePermissionSetFunc = func(ctx context.Context, params *ssoadmin.DescribePermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.DescribePermissionSetOutput, error) {
		return &ssoadmin.DescribePermissionSetOutput{
			PermissionSet: &ssotypes.PermissionSet{
				Name:            aws.String("PS-" + *params.PermissionSetArn),
				CreatedDate:     &createdDate,
				SessionDuration: aws.String("PT1H"),
			},
		}, nil
	}

	instance, err := GetSSOAccountInstance(mock)
	require.NoError(t, err)
	assert.Equal(t, 3, len(instance.PermissionSets), "should collect permission sets across two pages")
	assert.Equal(t, 2, callCount, "ListPermissionSets should be called twice")
}

func TestListAccountsForProvisionedPermissionSet_Pagination(t *testing.T) {
	mock := newBasicMock()
	accountCallCount := 0
	mock.ListAccountsForProvisionedPermissionSetFunc = func(ctx context.Context, params *ssoadmin.ListAccountsForProvisionedPermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListAccountsForProvisionedPermissionSetOutput, error) {
		accountCallCount++
		switch accountCallCount {
		case 1:
			assert.Nil(t, params.NextToken, "first call should not have NextToken")
			return &ssoadmin.ListAccountsForProvisionedPermissionSetOutput{
				AccountIds: []string{"111111111111", "222222222222"},
				NextToken:  aws.String("page2"),
			}, nil
		case 2:
			assert.Equal(t, "page2", *params.NextToken)
			return &ssoadmin.ListAccountsForProvisionedPermissionSetOutput{
				AccountIds: []string{"333333333333"},
				NextToken:  nil,
			}, nil
		default:
			t.Fatal("ListAccountsForProvisionedPermissionSet called too many times")
			return nil, nil
		}
	}

	instance, err := GetSSOAccountInstance(mock)
	require.NoError(t, err)
	assert.Equal(t, 3, len(instance.PermissionSets[0].Accounts), "should collect accounts across two pages")
	assert.Equal(t, 2, accountCallCount, "ListAccountsForProvisionedPermissionSet should be called twice")
}

func TestListAccountAssignments_Pagination(t *testing.T) {
	mock := newBasicMock()
	// Return one account
	mock.ListAccountsForProvisionedPermissionSetFunc = func(ctx context.Context, params *ssoadmin.ListAccountsForProvisionedPermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListAccountsForProvisionedPermissionSetOutput, error) {
		return &ssoadmin.ListAccountsForProvisionedPermissionSetOutput{
			AccountIds: []string{"111111111111"},
		}, nil
	}

	assignmentCallCount := 0
	mock.ListAccountAssignmentsFunc = func(ctx context.Context, params *ssoadmin.ListAccountAssignmentsInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListAccountAssignmentsOutput, error) {
		assignmentCallCount++
		switch assignmentCallCount {
		case 1:
			assert.Nil(t, params.NextToken, "first call should not have NextToken")
			return &ssoadmin.ListAccountAssignmentsOutput{
				AccountAssignments: []ssotypes.AccountAssignment{
					{PrincipalType: ssotypes.PrincipalTypeUser, PrincipalId: aws.String("user-1")},
					{PrincipalType: ssotypes.PrincipalTypeGroup, PrincipalId: aws.String("group-1")},
				},
				NextToken: aws.String("page2"),
			}, nil
		case 2:
			assert.Equal(t, "page2", *params.NextToken)
			return &ssoadmin.ListAccountAssignmentsOutput{
				AccountAssignments: []ssotypes.AccountAssignment{
					{PrincipalType: ssotypes.PrincipalTypeUser, PrincipalId: aws.String("user-2")},
				},
				NextToken: nil,
			}, nil
		default:
			t.Fatal("ListAccountAssignments called too many times")
			return nil, nil
		}
	}

	instance, err := GetSSOAccountInstance(mock)
	require.NoError(t, err)
	require.Equal(t, 1, len(instance.PermissionSets[0].Accounts))
	assert.Equal(t, 3, len(instance.PermissionSets[0].Accounts[0].AccountAssignments),
		"should collect assignments across two pages")
	assert.Equal(t, 2, assignmentCallCount, "ListAccountAssignments should be called twice")
}

func TestListManagedPoliciesInPermissionSet_Pagination(t *testing.T) {
	mock := newBasicMock()
	policyCallCount := 0
	mock.ListManagedPoliciesInPermissionSetFunc = func(ctx context.Context, params *ssoadmin.ListManagedPoliciesInPermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListManagedPoliciesInPermissionSetOutput, error) {
		policyCallCount++
		switch policyCallCount {
		case 1:
			assert.Nil(t, params.NextToken, "first call should not have NextToken")
			return &ssoadmin.ListManagedPoliciesInPermissionSetOutput{
				AttachedManagedPolicies: []ssotypes.AttachedManagedPolicy{
					{Arn: aws.String("arn:policy/1"), Name: aws.String("Policy1")},
					{Arn: aws.String("arn:policy/2"), Name: aws.String("Policy2")},
				},
				NextToken: aws.String("page2"),
			}, nil
		case 2:
			assert.Equal(t, "page2", *params.NextToken)
			return &ssoadmin.ListManagedPoliciesInPermissionSetOutput{
				AttachedManagedPolicies: []ssotypes.AttachedManagedPolicy{
					{Arn: aws.String("arn:policy/3"), Name: aws.String("Policy3")},
				},
				NextToken: nil,
			}, nil
		default:
			t.Fatal("ListManagedPoliciesInPermissionSet called too many times")
			return nil, nil
		}
	}

	instance, err := GetSSOAccountInstance(mock)
	require.NoError(t, err)
	assert.Equal(t, 3, len(instance.PermissionSets[0].ManagedPolicies),
		"should collect managed policies across two pages")
	assert.Equal(t, 2, policyCallCount, "ListManagedPoliciesInPermissionSet should be called twice")
}

// Error handling regression tests

func TestGetSSOAccountInstance_ListInstancesError_ReturnsError(t *testing.T) {
	mock := newBasicMock()
	mock.ListInstancesFunc = func(ctx context.Context, params *ssoadmin.ListInstancesInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListInstancesOutput, error) {
		return nil, errors.New("access denied")
	}
	_, err := GetSSOAccountInstance(mock)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list SSO instances")
}

func TestGetSSOAccountInstance_NoInstances_ReturnsError(t *testing.T) {
	mock := newBasicMock()
	mock.ListInstancesFunc = func(ctx context.Context, params *ssoadmin.ListInstancesInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListInstancesOutput, error) {
		return &ssoadmin.ListInstancesOutput{Instances: []ssotypes.InstanceMetadata{}}, nil
	}
	_, err := GetSSOAccountInstance(mock)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no SSO instances found")
}

func TestGetSSOAccountInstance_MultipleInstances_ReturnsError(t *testing.T) {
	mock := newBasicMock()
	mock.ListInstancesFunc = func(ctx context.Context, params *ssoadmin.ListInstancesInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListInstancesOutput, error) {
		return &ssoadmin.ListInstancesOutput{
			Instances: []ssotypes.InstanceMetadata{
				{IdentityStoreId: aws.String("d-1"), InstanceArn: aws.String("arn:1")},
				{IdentityStoreId: aws.String("d-2"), InstanceArn: aws.String("arn:2")},
			},
		}, nil
	}
	_, err := GetSSOAccountInstance(mock)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "found multiple SSO instances")
}

func TestGetSSOAccountInstance_ListPermissionSetsError_ReturnsError(t *testing.T) {
	mock := newBasicMock()
	mock.ListPermissionSetsFunc = func(ctx context.Context, params *ssoadmin.ListPermissionSetsInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListPermissionSetsOutput, error) {
		return nil, errors.New("throttled")
	}
	_, err := GetSSOAccountInstance(mock)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list permission sets")
}

func TestGetSSOAccountInstance_DescribePermissionSetError_ReturnsError(t *testing.T) {
	mock := newBasicMock()
	mock.DescribePermissionSetFunc = func(ctx context.Context, params *ssoadmin.DescribePermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.DescribePermissionSetOutput, error) {
		return nil, errors.New("not found")
	}
	_, err := GetSSOAccountInstance(mock)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to describe permission set")
}

func TestGetSSOAccountInstance_ListAccountsError_ReturnsError(t *testing.T) {
	mock := newBasicMock()
	mock.ListAccountsForProvisionedPermissionSetFunc = func(ctx context.Context, params *ssoadmin.ListAccountsForProvisionedPermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListAccountsForProvisionedPermissionSetOutput, error) {
		return nil, errors.New("service error")
	}
	_, err := GetSSOAccountInstance(mock)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list accounts for permission set")
}

func TestGetSSOAccountInstance_ListAssignmentsError_ReturnsError(t *testing.T) {
	mock := newBasicMock()
	mock.ListAccountsForProvisionedPermissionSetFunc = func(ctx context.Context, params *ssoadmin.ListAccountsForProvisionedPermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListAccountsForProvisionedPermissionSetOutput, error) {
		return &ssoadmin.ListAccountsForProvisionedPermissionSetOutput{
			AccountIds: []string{"111111111111"},
		}, nil
	}
	mock.ListAccountAssignmentsFunc = func(ctx context.Context, params *ssoadmin.ListAccountAssignmentsInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListAccountAssignmentsOutput, error) {
		return nil, errors.New("permission denied")
	}
	_, err := GetSSOAccountInstance(mock)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list account assignments")
}

func TestGetSSOAccountInstance_ListManagedPoliciesError_ReturnsError(t *testing.T) {
	mock := newBasicMock()
	mock.ListManagedPoliciesInPermissionSetFunc = func(ctx context.Context, params *ssoadmin.ListManagedPoliciesInPermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListManagedPoliciesInPermissionSetOutput, error) {
		return nil, errors.New("internal error")
	}
	_, err := GetSSOAccountInstance(mock)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list managed policies")
}

func TestGetSSOAccountInstance_GetInlinePolicyError_ReturnsError(t *testing.T) {
	mock := newBasicMock()
	mock.GetInlinePolicyForPermissionSetFunc = func(ctx context.Context, params *ssoadmin.GetInlinePolicyForPermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.GetInlinePolicyForPermissionSetOutput, error) {
		return nil, errors.New("timeout")
	}
	_, err := GetSSOAccountInstance(mock)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get inline policy")
}

// End-to-end test with pagination on all four listing APIs simultaneously

func TestGetSSOAccountInstance_FullPagination(t *testing.T) {
	createdDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Track call counts to verify pagination
	listPSCalls := 0
	listAccountsCalls := 0
	listAssignmentCalls := 0
	listPolicyCalls := 0

	mock := &mockSSOAdminClient{
		ListInstancesFunc: func(ctx context.Context, params *ssoadmin.ListInstancesInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListInstancesOutput, error) {
			return &ssoadmin.ListInstancesOutput{
				Instances: []ssotypes.InstanceMetadata{
					{IdentityStoreId: aws.String("d-test"), InstanceArn: aws.String("arn:instance")},
				},
			}, nil
		},
		// Two pages of permission sets
		ListPermissionSetsFunc: func(ctx context.Context, params *ssoadmin.ListPermissionSetsInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListPermissionSetsOutput, error) {
			listPSCalls++
			if params.NextToken == nil {
				return &ssoadmin.ListPermissionSetsOutput{
					PermissionSets: []string{"arn:ps-1"},
					NextToken:      aws.String("ps-page2"),
				}, nil
			}
			return &ssoadmin.ListPermissionSetsOutput{
				PermissionSets: []string{"arn:ps-2"},
			}, nil
		},
		DescribePermissionSetFunc: func(ctx context.Context, params *ssoadmin.DescribePermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.DescribePermissionSetOutput, error) {
			return &ssoadmin.DescribePermissionSetOutput{
				PermissionSet: &ssotypes.PermissionSet{
					Name:            params.PermissionSetArn,
					CreatedDate:     &createdDate,
					SessionDuration: aws.String("PT1H"),
				},
			}, nil
		},
		// Two pages of accounts per permission set
		ListAccountsForProvisionedPermissionSetFunc: func(ctx context.Context, params *ssoadmin.ListAccountsForProvisionedPermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListAccountsForProvisionedPermissionSetOutput, error) {
			listAccountsCalls++
			if params.NextToken == nil {
				return &ssoadmin.ListAccountsForProvisionedPermissionSetOutput{
					AccountIds: []string{"acct-1"},
					NextToken:  aws.String("acct-page2"),
				}, nil
			}
			return &ssoadmin.ListAccountsForProvisionedPermissionSetOutput{
				AccountIds: []string{"acct-2"},
			}, nil
		},
		// Two pages of assignments per account
		ListAccountAssignmentsFunc: func(ctx context.Context, params *ssoadmin.ListAccountAssignmentsInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListAccountAssignmentsOutput, error) {
			listAssignmentCalls++
			if params.NextToken == nil {
				return &ssoadmin.ListAccountAssignmentsOutput{
					AccountAssignments: []ssotypes.AccountAssignment{
						{PrincipalType: ssotypes.PrincipalTypeUser, PrincipalId: aws.String(fmt.Sprintf("user-%s-1", *params.AccountId))},
					},
					NextToken: aws.String("assign-page2"),
				}, nil
			}
			return &ssoadmin.ListAccountAssignmentsOutput{
				AccountAssignments: []ssotypes.AccountAssignment{
					{PrincipalType: ssotypes.PrincipalTypeGroup, PrincipalId: aws.String(fmt.Sprintf("group-%s-2", *params.AccountId))},
				},
			}, nil
		},
		// Two pages of managed policies per permission set
		ListManagedPoliciesInPermissionSetFunc: func(ctx context.Context, params *ssoadmin.ListManagedPoliciesInPermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListManagedPoliciesInPermissionSetOutput, error) {
			listPolicyCalls++
			if params.NextToken == nil {
				return &ssoadmin.ListManagedPoliciesInPermissionSetOutput{
					AttachedManagedPolicies: []ssotypes.AttachedManagedPolicy{
						{Arn: aws.String("arn:policy/A"), Name: aws.String("PolicyA")},
					},
					NextToken: aws.String("policy-page2"),
				}, nil
			}
			return &ssoadmin.ListManagedPoliciesInPermissionSetOutput{
				AttachedManagedPolicies: []ssotypes.AttachedManagedPolicy{
					{Arn: aws.String("arn:policy/B"), Name: aws.String("PolicyB")},
				},
			}, nil
		},
		GetInlinePolicyForPermissionSetFunc: func(ctx context.Context, params *ssoadmin.GetInlinePolicyForPermissionSetInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.GetInlinePolicyForPermissionSetOutput, error) {
			return &ssoadmin.GetInlinePolicyForPermissionSetOutput{
				InlinePolicy: aws.String(""),
			}, nil
		},
	}

	instance, err := GetSSOAccountInstance(mock)
	require.NoError(t, err)

	// 2 permission sets (from 2 pages)
	assert.Equal(t, 2, len(instance.PermissionSets))

	// Each permission set has 2 accounts (from 2 pages)
	for _, ps := range instance.PermissionSets {
		assert.Equal(t, 2, len(ps.Accounts), "permission set %s should have 2 accounts", ps.Name)

		// Each account has 2 assignments (from 2 pages)
		for _, acct := range ps.Accounts {
			assert.Equal(t, 2, len(acct.AccountAssignments),
				"account %s should have 2 assignments", acct.AccountID)
		}

		// Each permission set has 2 managed policies (from 2 pages)
		assert.Equal(t, 2, len(ps.ManagedPolicies), "permission set %s should have 2 managed policies", ps.Name)
	}

	// Verify pagination was exercised
	assert.Equal(t, 2, listPSCalls, "ListPermissionSets should be called twice")
	assert.Equal(t, 4, listAccountsCalls, "ListAccountsForProvisionedPermissionSet should be called 4 times (2 PS x 2 pages)")
	assert.Equal(t, 8, listAssignmentCalls, "ListAccountAssignments should be called 8 times (2 PS x 2 accounts x 2 pages)")
	assert.Equal(t, 4, listPolicyCalls, "ListManagedPoliciesInPermissionSet should be called 4 times (2 PS x 2 pages)")
}
