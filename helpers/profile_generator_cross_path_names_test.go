package helpers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// TestGenerateProfiles_ConflictAndNonConflictedNamesDoNotCollide is a
// regression test for T-1646. A replace conflict and a non-conflicted role that
// map to the same pattern-derived name (here {account_name}) used to get the
// same GeneratedProfile.Name, because the two paths resolved names
// independently. AppendToConfig then removed the old profile for the replace
// and the later profile overwrote the earlier one, silently dropping a role.
//
// Roles use RoleName == PermissionSetName, as role discovery produces them.
// The test follows the steps GenerateProfilesWorkflow takes after role
// discovery (which needs AWS) against a temporary config file.
func TestGenerateProfiles_ConflictAndNonConflictedNamesDoNotCollide(t *testing.T) {
	configContent := `[profile test-sso-profile]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_account_id = 222222222222
sso_role_name = AdministratorAccess
region = us-east-1

[profile old-admin]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_account_id = 111111111111
sso_role_name = AdministratorAccess
region = us-east-1
`
	configPath := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	t.Setenv("AWS_CONFIG_FILE", configPath)

	pg, err := NewProfileGenerator("test-sso-profile", "{account_name}", true, configPath, ConflictReplace, aws.Config{})
	if err != nil {
		t.Fatalf("NewProfileGenerator returned error: %v", err)
	}
	pg.SetLogger(&MockLogger{})

	roles := []DiscoveredRole{
		{
			AccountID:         "111111111111",
			AccountName:       "shared-account",
			PermissionSetName: "AdministratorAccess",
			RoleName:          "AdministratorAccess",
		},
		{
			AccountID:         "111111111111",
			AccountName:       "shared-account",
			PermissionSetName: "ReadOnlyAccess",
			RoleName:          "ReadOnlyAccess",
		},
	}

	conflicts, err := pg.DetectProfileConflicts(roles)
	if err != nil {
		t.Fatalf("DetectProfileConflicts returned error: %v", err)
	}
	if len(conflicts) != 1 || conflicts[0].DiscoveredRole.PermissionSetName != "AdministratorAccess" {
		t.Fatalf("expected a single conflict for AdministratorAccess, got %+v", conflicts)
	}

	_, nonConflictedRoles := pg.FilterRolesByConflicts(roles, conflicts)
	resolution, err := pg.ResolveConflicts(conflicts)
	if err != nil {
		t.Fatalf("ResolveConflicts returned error: %v", err)
	}
	nonConflictedProfiles, err := pg.generateProfilesForNonConflictedRoles(nonConflictedRoles, resolution.GeneratedProfiles)
	if err != nil {
		t.Fatalf("generateProfilesForNonConflictedRoles returned error: %v", err)
	}

	profiles := append(resolution.GeneratedProfiles, nonConflictedProfiles...)
	if len(profiles) != 2 {
		t.Fatalf("expected 2 generated profiles, got %d", len(profiles))
	}
	if profiles[0].Name == profiles[1].Name {
		t.Fatalf("generated profiles share the name %q", profiles[0].Name)
	}

	if err := pg.AppendToConfig(profiles, resolution.Actions); err != nil {
		t.Fatalf("AppendToConfig returned error: %v", err)
	}

	written, err := LoadAWSConfigFile(configPath)
	if err != nil {
		t.Fatalf("LoadAWSConfigFile returned error: %v", err)
	}
	if written.HasProfile("old-admin") {
		t.Errorf("old-admin should have been removed by the replace action")
	}
	for _, p := range profiles {
		got, ok := written.GetProfile(p.Name)
		if !ok {
			t.Errorf("profile %q missing from written config", p.Name)
			continue
		}
		if got.SSORoleName != p.RoleName {
			t.Errorf("profile %q has role %q, want %q", p.Name, got.SSORoleName, p.RoleName)
		}
	}
}
