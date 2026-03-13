package helpers

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
)

// mockListRootsClient is a mock for the organizationsListRootsAPI interface.
type mockListRootsClient struct {
	output *organizations.ListRootsOutput
	err    error
}

func (m *mockListRootsClient) ListRoots(_ context.Context, _ *organizations.ListRootsInput, _ ...func(*organizations.Options)) (*organizations.ListRootsOutput, error) {
	return m.output, m.err
}

func TestGetOrganizationRoot_ReturnsErrorOnAPIFailure(t *testing.T) {
	mock := &mockListRootsClient{
		output: nil,
		err:    errors.New("AccessDeniedException: not authorized"),
	}

	_, err := getOrganizationRoot(mock)
	if err == nil {
		t.Fatal("Expected error when ListRoots fails, got nil")
	}
	if !errors.Is(err, mock.err) {
		t.Errorf("Expected wrapped error containing %q, got %q", mock.err, err)
	}
}

func TestGetOrganizationRoot_ReturnsErrorOnEmptyRoots(t *testing.T) {
	mock := &mockListRootsClient{
		output: &organizations.ListRootsOutput{
			Roots: []types.Root{},
		},
		err: nil,
	}

	_, err := getOrganizationRoot(mock)
	if err == nil {
		t.Fatal("Expected error when ListRoots returns empty roots, got nil")
	}
}

func TestGetOrganizationRoot_Success(t *testing.T) {
	mock := &mockListRootsClient{
		output: &organizations.ListRootsOutput{
			Roots: []types.Root{
				{
					Id:   aws.String("r-ab12"),
					Arn:  aws.String("arn:aws:organizations::123456789012:root/o-abc123/r-ab12"),
					Name: aws.String("Root"),
				},
			},
		},
		err: nil,
	}

	entry, err := getOrganizationRoot(mock)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if entry.ID != "r-ab12" {
		t.Errorf("Expected ID 'r-ab12', got %q", entry.ID)
	}
	if entry.Name != "Root" {
		t.Errorf("Expected Name 'Root', got %q", entry.Name)
	}
	if entry.Type != string(types.TargetTypeRoot) {
		t.Errorf("Expected Type %q, got %q", string(types.TargetTypeRoot), entry.Type)
	}
}

func TestOrganizationEntry_Struct(t *testing.T) {
	entry := OrganizationEntry{
		ID:       "r-1234567890",
		Name:     "Root Organization",
		Arn:      "arn:aws:organizations::123456789012:root/r-1234567890",
		Type:     string(types.TargetTypeRoot),
		Children: []OrganizationEntry{},
	}

	if entry.ID != "r-1234567890" {
		t.Errorf("Expected ID to be 'r-1234567890', got %s", entry.ID)
	}

	if entry.Name != "Root Organization" {
		t.Errorf("Expected Name to be 'Root Organization', got %s", entry.Name)
	}

	if entry.Type != string(types.TargetTypeRoot) {
		t.Errorf("Expected Type to be '%s', got %s", string(types.TargetTypeRoot), entry.Type)
	}

	if len(entry.Children) != 0 {
		t.Errorf("Expected Children to be empty, got %d children", len(entry.Children))
	}
}

func TestOrganizationEntry_WithChildren(t *testing.T) {
	child1 := OrganizationEntry{
		ID:   "ou-1234567890-abcdefgh",
		Name: "Production OU",
		Type: string(types.TargetTypeOrganizationalUnit),
	}

	child2 := OrganizationEntry{
		ID:   "ou-1234567890-ijklmnop",
		Name: "Development OU",
		Type: string(types.TargetTypeOrganizationalUnit),
	}

	parent := OrganizationEntry{
		ID:       "r-1234567890",
		Name:     "Root Organization",
		Type:     string(types.TargetTypeRoot),
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

// Integration tests would require AWS Organizations access
func TestGetFullOrganization_Integration(t *testing.T) {
	t.Skip("Skipping integration test - requires Organizations client interface implementation")
}

func TestGetOrganizationRoot_Integration(t *testing.T) {
	t.Skip("Skipping integration test - requires Organizations client interface implementation")
}
