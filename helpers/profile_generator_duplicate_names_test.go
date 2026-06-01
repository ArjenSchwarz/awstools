package helpers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// TestGenerateProfilesForNonConflictedRoles_DuplicateNames is a regression test
// for T-1167. With a naming pattern like {account_name}, two distinct
// non-conflicted roles in the same account generate the same desired profile
// name. Before the fix the conflict resolver was never applied on this path, so
// both GeneratedProfile.Name values were identical and AppendProfiles silently
// overwrote the first profile with the second, dropping a discovered role.
//
// Expected behaviour: each role yields a distinct generated name and no profile
// is lost.
func TestGenerateProfilesForNonConflictedRoles_DuplicateNames(t *testing.T) {
	// Write a minimal AWS config file containing an SSO template profile so that
	// ValidateTemplateProfile succeeds. Point AWS_CONFIG_FILE at it so both the
	// template lookup and the existing-name seeding read from this file.
	configContent := `[profile test-sso-profile]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_account_id = 222222222222
sso_role_name = AdministratorAccess
region = us-east-1
`
	configPath := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	t.Setenv("AWS_CONFIG_FILE", configPath)

	// Naming pattern {account_name} collides for two roles in the same account.
	pg, err := NewProfileGenerator("test-sso-profile", "{account_name}", false, "", ConflictPrompt, aws.Config{})
	if err != nil {
		t.Fatalf("NewProfileGenerator returned error: %v", err)
	}

	roles := []DiscoveredRole{
		{
			AccountID:         "111111111111",
			AccountName:       "shared-account",
			PermissionSetName: "AdministratorAccess",
			RoleName:          "AWSReservedSSO_AdministratorAccess_a",
		},
		{
			AccountID:         "111111111111",
			AccountName:       "shared-account",
			PermissionSetName: "ReadOnlyAccess",
			RoleName:          "AWSReservedSSO_ReadOnlyAccess_b",
		},
	}

	profiles, err := pg.GenerateProfilesForNonConflictedRoles(roles)
	if err != nil {
		t.Fatalf("GenerateProfilesForNonConflictedRoles returned error: %v", err)
	}

	if len(profiles) != len(roles) {
		t.Fatalf("expected %d generated profiles, got %d", len(roles), len(profiles))
	}

	// Every generated name must be unique, otherwise a role would be silently
	// dropped when appended to the AWS config file via AppendProfiles.
	seen := make(map[string]int)
	for _, p := range profiles {
		seen[p.Name]++
	}
	for name, count := range seen {
		if count > 1 {
			t.Errorf("duplicate generated profile name %q appeared %d times; roles would be overwritten", name, count)
		}
	}

	// Confirm both permission sets are represented (no role lost).
	roleNames := make(map[string]bool)
	for _, p := range profiles {
		roleNames[p.RoleName] = true
	}
	for _, r := range roles {
		if !roleNames[r.PermissionSetName] {
			t.Errorf("role with permission set %q was dropped from generated profiles", r.PermissionSetName)
		}
	}
}
