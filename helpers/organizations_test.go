package helpers

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
)

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
