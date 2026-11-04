package helpers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Regression tests for T-1428: naming patterns containing {region} must go
// through conflict detection with the template profile's region instead of
// silently bypassing it.

const regionConflictConfig = `[profile sso-template]
region = ap-southeast-2
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_account_id = 999999999999
sso_role_name = ReadOnly

[profile my-existing-admin]
region = ap-southeast-2
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_account_id = 123456789012
sso_role_name = AdminAccess
`

var regionConflictRole = DiscoveredRole{
	AccountID:         "123456789012",
	AccountName:       "prod",
	PermissionSetName: "AdminAccess",
	RoleName:          "AdminAccess",
}

// newRegionConflictGenerator writes the config to an explicit output file and
// points AWS_CONFIG_FILE at an empty temp file so the user's real AWS
// configuration is never read.
func newRegionConflictGenerator(t *testing.T, strategy ConflictResolutionStrategy) *ProfileGenerator {
	t.Helper()
	t.Setenv("AWS_CONFIG_FILE", CreateTempConfigFile(t, ""))

	outputPath := filepath.Join(t.TempDir(), "config")
	require.NoError(t, os.WriteFile(outputPath, []byte(regionConflictConfig), 0600))

	pg, err := NewProfileGenerator("sso-template", "{account_name}-{region}-{role_name}", false, outputPath, strategy, aws.Config{})
	require.NoError(t, err)
	return pg
}

func TestDetectProfileConflicts_RegionPatternDetectsSameRole(t *testing.T) {
	pg := newRegionConflictGenerator(t, ConflictSkip)
	roles := []DiscoveredRole{regionConflictRole}

	conflicts, err := pg.DetectProfileConflicts(roles)
	require.NoError(t, err)
	require.Len(t, conflicts, 1)
	assert.Equal(t, ConflictSameRole, conflicts[0].ConflictType)
	assert.Equal(t, "prod-ap-southeast-2-AdminAccess", conflicts[0].ProposedName)
	require.Len(t, conflicts[0].ExistingProfiles, 1)
	assert.Equal(t, "my-existing-admin", conflicts[0].ExistingProfiles[0].Name)

	_, nonConflicted := pg.FilterRolesByConflicts(roles, conflicts)
	assert.Empty(t, nonConflicted, "conflicted role must not be generated as non-conflicted")
}

func TestResolveConflicts_RegionPattern(t *testing.T) {
	tests := map[string]struct {
		strategy      ConflictResolutionStrategy
		wantAction    ActionType
		wantGenerated []string
		wantSkipped   int
	}{
		"skip keeps existing profile": {
			strategy:    ConflictSkip,
			wantAction:  ActionSkip,
			wantSkipped: 1,
		},
		"replace uses region-based name": {
			strategy:      ConflictReplace,
			wantAction:    ActionReplace,
			wantGenerated: []string{"prod-ap-southeast-2-AdminAccess"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			pg := newRegionConflictGenerator(t, tc.strategy)

			conflicts, err := pg.DetectProfileConflicts([]DiscoveredRole{regionConflictRole})
			require.NoError(t, err)
			require.Len(t, conflicts, 1)

			got, err := pg.ResolveConflicts(conflicts)
			require.NoError(t, err)
			require.Len(t, got.Actions, 1)
			assert.Equal(t, tc.wantAction, got.Actions[0].Action)
			assert.Equal(t, "my-existing-admin", got.Actions[0].OldName)
			assert.Len(t, got.SkippedRoles, tc.wantSkipped)

			var gotNames []string
			for _, p := range got.GeneratedProfiles {
				gotNames = append(gotNames, p.Name)
			}
			assert.Equal(t, tc.wantGenerated, gotNames)
		})
	}
}

func TestDetectConflicts_AnalysisFailureIsReturned(t *testing.T) {
	configFile := &AWSConfigFile{Profiles: map[string]Profile{}}
	namingPattern, err := NewNamingPattern("{account_name}-{region}-{role_name}")
	require.NoError(t, err)

	// No region set: the {region} placeholder cannot be filled, so analysis fails.
	detector := NewProfileConflictDetector(configFile, namingPattern)

	conflicts, err := detector.DetectConflicts([]DiscoveredRole{regionConflictRole})
	require.Error(t, err, "failed analysis must not be reported as no conflicts")
	assert.Nil(t, conflicts)
}
