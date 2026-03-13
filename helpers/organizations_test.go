package helpers

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	orgtypes "github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockOrganizationsClient implements OrganizationsAPI for testing.
type mockOrganizationsClient struct {
	ListRootsFunc                  func(ctx context.Context, params *organizations.ListRootsInput, optFns ...func(*organizations.Options)) (*organizations.ListRootsOutput, error)
	ListChildrenFunc               func(ctx context.Context, params *organizations.ListChildrenInput, optFns ...func(*organizations.Options)) (*organizations.ListChildrenOutput, error)
	DescribeOrganizationalUnitFunc func(ctx context.Context, params *organizations.DescribeOrganizationalUnitInput, optFns ...func(*organizations.Options)) (*organizations.DescribeOrganizationalUnitOutput, error)
	DescribeAccountFunc            func(ctx context.Context, params *organizations.DescribeAccountInput, optFns ...func(*organizations.Options)) (*organizations.DescribeAccountOutput, error)
}

func (m *mockOrganizationsClient) ListRoots(ctx context.Context, params *organizations.ListRootsInput, optFns ...func(*organizations.Options)) (*organizations.ListRootsOutput, error) {
	return m.ListRootsFunc(ctx, params, optFns...)
}

func (m *mockOrganizationsClient) ListChildren(ctx context.Context, params *organizations.ListChildrenInput, optFns ...func(*organizations.Options)) (*organizations.ListChildrenOutput, error) {
	return m.ListChildrenFunc(ctx, params, optFns...)
}

func (m *mockOrganizationsClient) DescribeOrganizationalUnit(ctx context.Context, params *organizations.DescribeOrganizationalUnitInput, optFns ...func(*organizations.Options)) (*organizations.DescribeOrganizationalUnitOutput, error) {
	return m.DescribeOrganizationalUnitFunc(ctx, params, optFns...)
}

func (m *mockOrganizationsClient) DescribeAccount(ctx context.Context, params *organizations.DescribeAccountInput, optFns ...func(*organizations.Options)) (*organizations.DescribeAccountOutput, error) {
	return m.DescribeAccountFunc(ctx, params, optFns...)
}

func TestOrganizationEntry_Struct(t *testing.T) {
	entry := OrganizationEntry{
		ID:       "r-1234567890",
		Name:     "Root Organization",
		Arn:      "arn:aws:organizations::123456789012:root/r-1234567890",
		Type:     string(orgtypes.TargetTypeRoot),
		Children: []OrganizationEntry{},
	}

	if entry.ID != "r-1234567890" {
		t.Errorf("Expected ID to be 'r-1234567890', got %s", entry.ID)
	}

	if entry.Name != "Root Organization" {
		t.Errorf("Expected Name to be 'Root Organization', got %s", entry.Name)
	}

	if entry.Type != string(orgtypes.TargetTypeRoot) {
		t.Errorf("Expected Type to be '%s', got %s", string(orgtypes.TargetTypeRoot), entry.Type)
	}

	if len(entry.Children) != 0 {
		t.Errorf("Expected Children to be empty, got %d children", len(entry.Children))
	}
}

func TestOrganizationEntry_WithChildren(t *testing.T) {
	child1 := OrganizationEntry{
		ID:   "ou-1234567890-abcdefgh",
		Name: "Production OU",
		Type: string(orgtypes.TargetTypeOrganizationalUnit),
	}

	child2 := OrganizationEntry{
		ID:   "ou-1234567890-ijklmnop",
		Name: "Development OU",
		Type: string(orgtypes.TargetTypeOrganizationalUnit),
	}

	parent := OrganizationEntry{
		ID:       "r-1234567890",
		Name:     "Root Organization",
		Type:     string(orgtypes.TargetTypeRoot),
		Children: []OrganizationEntry{child1, child2},
	}

	if len(parent.Children) != 2 {
		t.Errorf("Expected 2 children, got %d", len(parent.Children))
	}

	if parent.Children[0].Name != "Production OU" {
		t.Errorf("Expected first child name to be 'Production OU', got %s", parent.Children[0].Name)
	}

	if parent.Children[1].Name != "Development OU" {
		t.Errorf("Expected second child name to be 'Development OU', got %s", parent.Children[1].Name)
	}
}

// Regression tests for T-418: error handling must not panic

func TestGetOrganizationRoot_ListRootsError_ReturnsError(t *testing.T) {
	mock := &mockOrganizationsClient{
		ListRootsFunc: func(_ context.Context, _ *organizations.ListRootsInput, _ ...func(*organizations.Options)) (*organizations.ListRootsOutput, error) {
			return nil, errors.New("access denied")
		},
	}

	_, err := getOrganizationRoot(mock)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
}

func TestGetOrganizationRoot_EmptyRoots_ReturnsError(t *testing.T) {
	mock := &mockOrganizationsClient{
		ListRootsFunc: func(_ context.Context, _ *organizations.ListRootsInput, _ ...func(*organizations.Options)) (*organizations.ListRootsOutput, error) {
			return &organizations.ListRootsOutput{Roots: []orgtypes.Root{}}, nil
		},
	}

	_, err := getOrganizationRoot(mock)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no organization roots found")
}

func TestFindChildren_ListOUChildrenError_ReturnsError(t *testing.T) {
	mock := &mockOrganizationsClient{
		ListChildrenFunc: func(_ context.Context, params *organizations.ListChildrenInput, _ ...func(*organizations.Options)) (*organizations.ListChildrenOutput, error) {
			return nil, errors.New("throttling exception")
		},
	}
	entry := &OrganizationEntry{ID: "r-root123"}

	_, err := entry.findChildren(mock)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "throttling exception")
}

func TestFindChildren_ListAccountChildrenError_ReturnsError(t *testing.T) {
	mock := &mockOrganizationsClient{
		ListChildrenFunc: func(_ context.Context, params *organizations.ListChildrenInput, _ ...func(*organizations.Options)) (*organizations.ListChildrenOutput, error) {
			if params.ChildType == orgtypes.ChildType(orgtypes.TargetTypeOrganizationalUnit) {
				return &organizations.ListChildrenOutput{Children: []orgtypes.Child{}}, nil
			}
			// Account list call fails
			return nil, errors.New("service unavailable")
		},
	}
	entry := &OrganizationEntry{ID: "r-root123"}

	_, err := entry.findChildren(mock)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "service unavailable")
}

func TestFormatChild_DescribeOUError_ReturnsError(t *testing.T) {
	mock := &mockOrganizationsClient{
		DescribeOrganizationalUnitFunc: func(_ context.Context, _ *organizations.DescribeOrganizationalUnitInput, _ ...func(*organizations.Options)) (*organizations.DescribeOrganizationalUnitOutput, error) {
			return nil, errors.New("access denied to OU")
		},
	}
	child := orgtypes.Child{
		Id:   aws.String("ou-abc123"),
		Type: orgtypes.ChildType(orgtypes.TargetTypeOrganizationalUnit),
	}

	_, err := formatChild(child, mock)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied to OU")
}

func TestFormatChild_DescribeAccountError_ReturnsError(t *testing.T) {
	mock := &mockOrganizationsClient{
		DescribeAccountFunc: func(_ context.Context, _ *organizations.DescribeAccountInput, _ ...func(*organizations.Options)) (*organizations.DescribeAccountOutput, error) {
			return nil, errors.New("account not found")
		},
	}
	child := orgtypes.Child{
		Id:   aws.String("123456789012"),
		Type: orgtypes.ChildType(orgtypes.TargetTypeAccount),
	}

	_, err := formatChild(child, mock)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account not found")
}

func TestGetFullOrganization_ListRootsError_ReturnsError(t *testing.T) {
	mock := &mockOrganizationsClient{
		ListRootsFunc: func(_ context.Context, _ *organizations.ListRootsInput, _ ...func(*organizations.Options)) (*organizations.ListRootsOutput, error) {
			return nil, errors.New("credentials expired")
		},
	}

	_, err := GetFullOrganization(mock)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "credentials expired")
}

func TestGetFullOrganization_Success(t *testing.T) {
	mock := &mockOrganizationsClient{
		ListRootsFunc: func(_ context.Context, _ *organizations.ListRootsInput, _ ...func(*organizations.Options)) (*organizations.ListRootsOutput, error) {
			return &organizations.ListRootsOutput{
				Roots: []orgtypes.Root{
					{
						Id:   aws.String("r-root1"),
						Arn:  aws.String("arn:aws:organizations::123456789012:root/r-root1"),
						Name: aws.String("Root"),
					},
				},
			}, nil
		},
		ListChildrenFunc: func(_ context.Context, params *organizations.ListChildrenInput, _ ...func(*organizations.Options)) (*organizations.ListChildrenOutput, error) {
			if *params.ParentId == "r-root1" && params.ChildType == orgtypes.ChildType(orgtypes.TargetTypeOrganizationalUnit) {
				return &organizations.ListChildrenOutput{
					Children: []orgtypes.Child{
						{Id: aws.String("ou-prod1"), Type: orgtypes.ChildType(orgtypes.TargetTypeOrganizationalUnit)},
					},
				}, nil
			}
			if *params.ParentId == "r-root1" && params.ChildType == orgtypes.ChildType(orgtypes.TargetTypeAccount) {
				return &organizations.ListChildrenOutput{
					Children: []orgtypes.Child{
						{Id: aws.String("111111111111"), Type: orgtypes.ChildType(orgtypes.TargetTypeAccount)},
					},
				}, nil
			}
			// Nested OU has no children
			return &organizations.ListChildrenOutput{Children: []orgtypes.Child{}}, nil
		},
		DescribeOrganizationalUnitFunc: func(_ context.Context, params *organizations.DescribeOrganizationalUnitInput, _ ...func(*organizations.Options)) (*organizations.DescribeOrganizationalUnitOutput, error) {
			return &organizations.DescribeOrganizationalUnitOutput{
				OrganizationalUnit: &orgtypes.OrganizationalUnit{
					Id:   params.OrganizationalUnitId,
					Arn:  aws.String("arn:aws:organizations::123456789012:ou/r-root1/" + *params.OrganizationalUnitId),
					Name: aws.String("Production"),
				},
			}, nil
		},
		DescribeAccountFunc: func(_ context.Context, params *organizations.DescribeAccountInput, _ ...func(*organizations.Options)) (*organizations.DescribeAccountOutput, error) {
			return &organizations.DescribeAccountOutput{
				Account: &orgtypes.Account{
					Id:   params.AccountId,
					Arn:  aws.String("arn:aws:organizations::123456789012:account/" + *params.AccountId),
					Name: aws.String("Main Account"),
				},
			}, nil
		},
	}

	org, err := GetFullOrganization(mock)
	require.NoError(t, err)
	assert.Equal(t, "Root", org.Name)
	assert.Equal(t, "r-root1", org.ID)
	require.Len(t, org.Children, 2)
	assert.Equal(t, "Production", org.Children[0].Name)
	assert.Equal(t, "Main Account", org.Children[1].Name)
}

func TestFindChildren_DescribeOUErrorDuringTraversal_ReturnsError(t *testing.T) {
	mock := &mockOrganizationsClient{
		ListChildrenFunc: func(_ context.Context, params *organizations.ListChildrenInput, _ ...func(*organizations.Options)) (*organizations.ListChildrenOutput, error) {
			if params.ChildType == orgtypes.ChildType(orgtypes.TargetTypeOrganizationalUnit) {
				return &organizations.ListChildrenOutput{
					Children: []orgtypes.Child{
						{Id: aws.String("ou-fail"), Type: orgtypes.ChildType(orgtypes.TargetTypeOrganizationalUnit)},
					},
				}, nil
			}
			return &organizations.ListChildrenOutput{Children: []orgtypes.Child{}}, nil
		},
		DescribeOrganizationalUnitFunc: func(_ context.Context, _ *organizations.DescribeOrganizationalUnitInput, _ ...func(*organizations.Options)) (*organizations.DescribeOrganizationalUnitOutput, error) {
			return nil, errors.New("OU describe failed")
		},
	}
	entry := &OrganizationEntry{ID: "r-root123"}

	_, err := entry.findChildren(mock)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "OU describe failed")
}
