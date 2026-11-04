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

// TestResolveConflicts_ReplacedRolesWithSameNameBothSurvive is a regression
// test for T-1646. Two roles that both already have profiles and are both
// replaced can map to the same pattern-derived name. ResolveConflicts used to
// give both the proposed name, so AppendToConfig removed both old profiles and
// only the last role survived. Each replace action's NewName must match the
// profile that is actually written.
func TestResolveConflicts_ReplacedRolesWithSameNameBothSurvive(t *testing.T) {
	const template = `[profile test-sso-profile]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_account_id = 222222222222
sso_role_name = AdministratorAccess
region = us-east-1
`
	ssoProfile := func(name, accountID, roleName string) string {
		return "\n[profile " + name + "]\nsso_start_url = https://example.awsapps.com/start\nsso_region = us-east-1\nsso_account_id = " +
			accountID + "\nsso_role_name = " + roleName + "\nregion = us-east-1\n"
	}

	tests := map[string]struct {
		existing string
		// wantNames maps permission set to the expected generated profile name.
		wantNames map[string]string
		removed   []string
		// kept maps an unrelated profile name to the account it must still point at.
		kept map[string]string
	}{
		"both old profiles renamed": {
			existing: ssoProfile("old-admin", "111111111111", "AdministratorAccess") +
				ssoProfile("old-readonly", "111111111111", "ReadOnlyAccess"),
			wantNames: map[string]string{"AdministratorAccess": "shared-account", "ReadOnlyAccess": "shared-account_1"},
			removed:   []string{"old-admin", "old-readonly"},
		},
		"in-place replacement keeps name and suffix skips unrelated profile": {
			existing: ssoProfile("shared-account", "111111111111", "AdministratorAccess") +
				ssoProfile("old-readonly", "111111111111", "ReadOnlyAccess") +
				ssoProfile("shared-account_1", "333333333333", "ViewOnlyAccess"),
			wantNames: map[string]string{"AdministratorAccess": "shared-account", "ReadOnlyAccess": "shared-account_2"},
			removed:   []string{"old-readonly"},
			kept:      map[string]string{"shared-account_1": "333333333333"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "config")
			if err := os.WriteFile(configPath, []byte(template+tc.existing), 0600); err != nil {
				t.Fatalf("failed to write temp config: %v", err)
			}
			t.Setenv("AWS_CONFIG_FILE", configPath)

			pg, err := NewProfileGenerator("test-sso-profile", "{account_name}", true, configPath, ConflictReplace, aws.Config{})
			if err != nil {
				t.Fatalf("NewProfileGenerator returned error: %v", err)
			}
			pg.SetLogger(&MockLogger{})

			roles := []DiscoveredRole{
				{AccountID: "111111111111", AccountName: "shared-account", PermissionSetName: "AdministratorAccess", RoleName: "AdministratorAccess"},
				{AccountID: "111111111111", AccountName: "shared-account", PermissionSetName: "ReadOnlyAccess", RoleName: "ReadOnlyAccess"},
			}

			conflicts, err := pg.DetectProfileConflicts(roles)
			if err != nil {
				t.Fatalf("DetectProfileConflicts returned error: %v", err)
			}
			if len(conflicts) != 2 {
				t.Fatalf("expected 2 conflicts, got %d", len(conflicts))
			}

			resolution, err := pg.ResolveConflicts(conflicts)
			if err != nil {
				t.Fatalf("ResolveConflicts returned error: %v", err)
			}

			for i, action := range resolution.Actions {
				if action.NewName != resolution.GeneratedProfiles[i].Name {
					t.Errorf("action %d NewName %q does not match generated profile %q", i, action.NewName, resolution.GeneratedProfiles[i].Name)
				}
			}

			if err := pg.AppendToConfig(resolution.GeneratedProfiles, resolution.Actions); err != nil {
				t.Fatalf("AppendToConfig returned error: %v", err)
			}

			written, err := LoadAWSConfigFile(configPath)
			if err != nil {
				t.Fatalf("LoadAWSConfigFile returned error: %v", err)
			}
			for roleName, profileName := range tc.wantNames {
				got, ok := written.GetProfile(profileName)
				if !ok {
					t.Errorf("profile %q for %s missing from written config", profileName, roleName)
					continue
				}
				if got.SSORoleName != roleName || got.SSOAccountID != "111111111111" {
					t.Errorf("profile %q points at %s/%s, want 111111111111/%s", profileName, got.SSOAccountID, got.SSORoleName, roleName)
				}
			}
			for _, old := range tc.removed {
				if written.HasProfile(old) {
					t.Errorf("old profile %q should have been removed", old)
				}
			}
			for profileName, accountID := range tc.kept {
				got, ok := written.GetProfile(profileName)
				if !ok || got.SSOAccountID != accountID {
					t.Errorf("unrelated profile %q was changed or removed: %+v", profileName, got)
				}
			}
			if !written.HasProfile("test-sso-profile") {
				t.Errorf("template profile was removed")
			}
		})
	}
}
