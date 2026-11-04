package cmd

import (
	"testing"

	"github.com/ArjenSchwarz/awstools/helpers"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/spf13/viper"
)

// newSSOOverviewTestInstance builds an SSO instance with two permission sets
// assigned to two accounts.
func newSSOOverviewTestInstance() helpers.SSOInstance {
	admin := &helpers.SSOPermissionSet{Arn: "arn:aws:sso:::permissionSet/ssoins-1/ps-admin", Name: "Admin"}
	readonly := &helpers.SSOPermissionSet{Arn: "arn:aws:sso:::permissionSet/ssoins-1/ps-readonly", Name: "ReadOnly"}

	accountA := helpers.SSOAccount{
		AccountID: "111111111111",
		AccountAssignments: []helpers.SSOAccountAssignment{
			{PrincipalType: "USER", PrincipalID: "user-a", PermissionSet: admin},
		},
	}
	accountB := helpers.SSOAccount{
		AccountID: "222222222222",
		AccountAssignments: []helpers.SSOAccountAssignment{
			{PrincipalType: "GROUP", PrincipalID: "group-b", PermissionSet: readonly},
		},
	}
	admin.Accounts = []helpers.SSOAccount{accountA}
	readonly.Accounts = []helpers.SSOAccount{accountB}

	return helpers.SSOInstance{
		Arn:            "arn:aws:sso:::instance/ssoins-1",
		PermissionSets: []helpers.SSOPermissionSet{*admin, *readonly},
		Accounts: map[string]helpers.SSOAccount{
			accountA.AccountID: accountA,
			accountB.AccountID: accountB,
		},
	}
}

// isolateSSOOverviewTest sets the resource filter and ensures no naming file
// from the user's configuration is used.
func isolateSSOOverviewTest(t *testing.T, resourceID string) {
	t.Helper()
	oldResourceID := ssoresourceid
	oldNameFile := viper.Get("output.namefile")
	ssoresourceid = resourceID
	viper.Set("output.namefile", "")
	t.Cleanup(func() {
		ssoresourceid = oldResourceID
		viper.Set("output.namefile", oldNameFile)
	})
}

// assertRootEdgesMatchNodes checks that the root node's children are exactly
// the expected IDs and that each of them is emitted as a node.
func assertRootEdgesMatchNodes(t *testing.T, output format.OutputArray, rootID string, want []string) {
	t.Helper()
	nodeIDs := map[string]bool{}
	var root map[string]any
	for _, holder := range output.Contents {
		id, _ := holder.Contents[drawIOIDColumn].(string)
		nodeIDs[id] = true
		if id == rootID {
			root = holder.Contents
		}
	}
	if root == nil {
		t.Fatalf("root node %q not found", rootID)
	}
	got, _ := root[childrenColumn].([]string)
	if len(got) != len(want) {
		t.Fatalf("root children = %v, want %v", got, want)
	}
	gotSet := map[string]bool{}
	for _, child := range got {
		gotSet[child] = true
		if !nodeIDs[child] {
			t.Errorf("root edge to %q has no matching node", child)
		}
	}
	for _, child := range want {
		if !gotSet[child] {
			t.Errorf("root children = %v, missing %q", got, child)
		}
	}
}

func TestCreateSSOAccountDrawIOContents_RootChildrenFollowFilter(t *testing.T) {
	tests := map[string]struct {
		resourceID string
		want       []string
	}{
		"no filter":         {resourceID: "", want: []string{"111111111111", "222222222222"}},
		"filter by account": {resourceID: "222222222222", want: []string{"222222222222"}},
		"unknown account":   {resourceID: "999999999999", want: []string{}},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			isolateSSOOverviewTest(t, tc.resourceID)
			instance := newSSOOverviewTestInstance()
			output := format.OutputArray{Settings: format.NewOutputSettings()}
			createSSOAccountDrawIOContents(instance, &output)
			assertRootEdgesMatchNodes(t, output, instance.Arn, tc.want)
		})
	}
}

func TestCreateSSOPermissionsetsDrawIOContents_RootChildrenFollowFilter(t *testing.T) {
	tests := map[string]struct {
		resourceID string
		want       []string
	}{
		"no filter":      {resourceID: "", want: []string{"Admin", "ReadOnly"}},
		"filter by name": {resourceID: "ReadOnly", want: []string{"ReadOnly"}},
		"filter by arn":  {resourceID: "arn:aws:sso:::permissionSet/ssoins-1/ps-admin", want: []string{"Admin"}},
		"unknown name":   {resourceID: "Missing", want: []string{}},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			isolateSSOOverviewTest(t, tc.resourceID)
			instance := newSSOOverviewTestInstance()
			output := format.OutputArray{Settings: format.NewOutputSettings()}
			createSSOPermissionsetsDrawIOContents(instance, &output)
			assertRootEdgesMatchNodes(t, output, instance.Arn, tc.want)
		})
	}
}
