package helpers

import (
	"context"
	"maps"
	"net/url"
	"slices"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

const (
	t1656AWSManagedArn      = "arn:aws:iam::aws:policy/ReadOnlyAccess"
	t1656CustomerManagedArn = "arn:aws:iam::123456789012:policy/ReadOnlyAccess"
)

// sameNameAttachedPolicyMockIAMClient serves two roles that each attach a
// different managed policy sharing the PolicyName "ReadOnlyAccess" (T-1656).
type sameNameAttachedPolicyMockIAMClient struct {
	IAMClient
	attached  map[string]types.AttachedPolicy // role name -> attached policy
	documents map[string]string               // policy ARN -> decoded document
}

func newSameNameAttachedPolicyMock() *sameNameAttachedPolicyMockIAMClient {
	return &sameNameAttachedPolicyMockIAMClient{
		attached: map[string]types.AttachedPolicy{
			"role-a": {PolicyName: aws.String("ReadOnlyAccess"), PolicyArn: aws.String(t1656AWSManagedArn)},
			"role-b": {PolicyName: aws.String("ReadOnlyAccess"), PolicyArn: aws.String(t1656CustomerManagedArn)},
		},
		documents: map[string]string{
			t1656AWSManagedArn:      `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"*:Get*","Resource":"*"}]}`,
			t1656CustomerManagedArn: `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:GetObject","Resource":"arn:aws:s3:::team-bucket/*"}]}`,
		},
	}
}

func (m *sameNameAttachedPolicyMockIAMClient) ListRoles(_ context.Context, _ *iam.ListRolesInput, _ ...func(*iam.Options)) (*iam.ListRolesOutput, error) {
	return &iam.ListRolesOutput{Roles: []types.Role{
		{RoleName: aws.String("role-a"), RoleId: aws.String("AROAA"), Path: aws.String("/")},
		{RoleName: aws.String("role-b"), RoleId: aws.String("AROAB"), Path: aws.String("/")},
	}}, nil
}

func (m *sameNameAttachedPolicyMockIAMClient) ListRolePolicies(_ context.Context, _ *iam.ListRolePoliciesInput, _ ...func(*iam.Options)) (*iam.ListRolePoliciesOutput, error) {
	return &iam.ListRolePoliciesOutput{}, nil
}

func (m *sameNameAttachedPolicyMockIAMClient) ListAttachedRolePolicies(_ context.Context, input *iam.ListAttachedRolePoliciesInput, _ ...func(*iam.Options)) (*iam.ListAttachedRolePoliciesOutput, error) {
	return &iam.ListAttachedRolePoliciesOutput{
		AttachedPolicies: []types.AttachedPolicy{m.attached[aws.ToString(input.RoleName)]},
	}, nil
}

func (m *sameNameAttachedPolicyMockIAMClient) GetPolicy(_ context.Context, input *iam.GetPolicyInput, _ ...func(*iam.Options)) (*iam.GetPolicyOutput, error) {
	return &iam.GetPolicyOutput{Policy: &types.Policy{
		Arn:              input.PolicyArn,
		PolicyName:       aws.String("ReadOnlyAccess"),
		DefaultVersionId: aws.String("v1"),
	}}, nil
}

func (m *sameNameAttachedPolicyMockIAMClient) GetPolicyVersion(_ context.Context, input *iam.GetPolicyVersionInput, _ ...func(*iam.Options)) (*iam.GetPolicyVersionOutput, error) {
	encoded := url.QueryEscape(m.documents[aws.ToString(input.PolicyArn)])
	return &iam.GetPolicyVersionOutput{PolicyVersion: &types.PolicyVersion{
		Document:  &encoded,
		VersionId: aws.String("v1"),
	}}, nil
}

// TestGetRolesAndPolicies_SameNamedAttachedPolicies is the T-1656 regression
// test. Two roles attach different managed policies that share a name. Each
// must keep its own document and role relationship instead of the second role
// reusing the first policy's cached document.
func TestGetRolesAndPolicies_SameNamedAttachedPolicies(t *testing.T) {
	clearCache := func() {
		delete(cachedIAMPolicyDocuments, t1656AWSManagedArn)
		delete(cachedIAMPolicyDocuments, t1656CustomerManagedArn)
		delete(cachedIAMPolicyDocuments, "ReadOnlyAccess")
	}
	clearCache()
	t.Cleanup(clearCache)

	roles, policies := GetRolesAndPolicies(true, newSameNameAttachedPolicyMock())

	if len(policies) != 2 {
		t.Fatalf("got %d policies, want 2 (same-named attached policies were merged): %v", len(policies), slices.Collect(maps.Keys(policies)))
	}

	tests := map[string]struct {
		arn      string
		role     string
		resource string
	}{
		"AWS-managed policy":      {arn: t1656AWSManagedArn, role: "role-a", resource: "*"},
		"customer-managed policy": {arn: t1656CustomerManagedArn, role: "role-b", resource: "arn:aws:s3:::team-bucket/*"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			policy, ok := policies[tc.arn]
			if !ok {
				t.Fatalf("policy %q missing from output", tc.arn)
			}
			if policy.Name != "ReadOnlyAccess" {
				t.Errorf("Name = %q, want %q", policy.Name, "ReadOnlyAccess")
			}
			if got := policy.GetRoleNames(); !slices.Equal(got, []string{tc.role}) {
				t.Errorf("Roles = %v, want [%s]", got, tc.role)
			}
			if len(policy.Statement) != 1 {
				t.Fatalf("got %d statements, want 1", len(policy.Statement))
			}
			if got := policy.Statement[0].Resource; got != tc.resource {
				t.Errorf("Resource = %v, want %q (wrong policy document)", got, tc.resource)
			}
		})
	}

	for _, role := range roles {
		if got := role.GetPolicyNames(); !slices.Equal(got, []string{"ReadOnlyAccess"}) {
			t.Errorf("role %s GetPolicyNames() = %v, want [ReadOnlyAccess]", role.Name, got)
		}
		wantIDs := map[string][]string{"role-a": {t1656AWSManagedArn}, "role-b": {t1656CustomerManagedArn}}[role.Name]
		if got := role.GetPolicyIDs(); !slices.Equal(got, wantIDs) {
			t.Errorf("role %s GetPolicyIDs() = %v, want %v", role.Name, got, wantIDs)
		}
	}
}
