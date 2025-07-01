package helpers

import (
	"testing"
	"time"
)

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

// Integration tests would require AWS SSO access
func TestGetSSOAccountInstance_Integration(t *testing.T) {
	t.Skip("Skipping integration test - requires SSO Admin client interface implementation")
}
