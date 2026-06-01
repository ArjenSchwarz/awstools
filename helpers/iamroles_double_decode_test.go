package helpers

import (
	"context"
	"net/url"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

// doubleDecodeMockIAMClient is a focused IAMClient mock for the T-1094
// regression test. It returns a single attached role policy whose document,
// once URL-decoded, intentionally contains literal "%2F" and "%ZZ" string
// values. A correct implementation decodes the policy document exactly once
// (in getAttachedPolicy); a second decode in getAttachedPoliciesForRole either
// corrupts "%2F" into "/" or panics on the invalid "%ZZ" escape.
type doubleDecodeMockIAMClient struct {
	IAMClient
	// decodedDocument is the policy document as it should appear after exactly
	// one URL decode (i.e. what getAttachedPolicy returns).
	decodedDocument string
}

func (m *doubleDecodeMockIAMClient) ListAttachedRolePolicies(_ context.Context, _ *iam.ListAttachedRolePoliciesInput, _ ...func(*iam.Options)) (*iam.ListAttachedRolePoliciesOutput, error) {
	return &iam.ListAttachedRolePoliciesOutput{
		AttachedPolicies: []types.AttachedPolicy{
			{
				PolicyName: aws.String("RegressionPolicy"),
				PolicyArn:  aws.String("arn:aws:iam::123456789012:policy/RegressionPolicy"),
			},
		},
	}, nil
}

func (m *doubleDecodeMockIAMClient) GetPolicy(_ context.Context, input *iam.GetPolicyInput, _ ...func(*iam.Options)) (*iam.GetPolicyOutput, error) {
	return &iam.GetPolicyOutput{
		Policy: &types.Policy{
			PolicyName:       aws.String("RegressionPolicy"),
			Arn:              input.PolicyArn,
			DefaultVersionId: aws.String("v1"),
		},
	}, nil
}

func (m *doubleDecodeMockIAMClient) GetPolicyVersion(_ context.Context, _ *iam.GetPolicyVersionInput, _ ...func(*iam.Options)) (*iam.GetPolicyVersionOutput, error) {
	// AWS returns the policy document URL-encoded. We encode the intended
	// decoded document once so that a single decode reproduces decodedDocument.
	encoded := url.QueryEscape(m.decodedDocument)
	return &iam.GetPolicyVersionOutput{
		PolicyVersion: &types.PolicyVersion{
			Document:  &encoded,
			VersionId: aws.String("v1"),
		},
	}, nil
}

// TestGetAttachedPoliciesForRole_NoDoubleDecode is the T-1094 regression test.
// It exercises the verbose attached-role-policy path with a document whose
// decoded JSON contains literal "%2F" and "%ZZ" string values. With the
// double-decode bug present this either panics (invalid "%ZZ" escape) or
// silently corrupts "%2F" into "/". After the fix the values survive intact.
func TestGetAttachedPoliciesForRole_NoDoubleDecode(t *testing.T) {
	// Ensure no stale cache entry interferes with this test.
	delete(cachedIAMPolicyDocuments, "RegressionPolicy")
	t.Cleanup(func() { delete(cachedIAMPolicyDocuments, "RegressionPolicy") })

	// Decoded JSON deliberately contains literal "%2F" and "%ZZ" as a Resource
	// value. "%ZZ" is an invalid percent-escape, so a second QueryUnescape
	// would error/panic; "%2F" would be corrupted into "/".
	decoded := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:GetObject","Resource":"literal-%2F-and-%ZZ-value"}]}`

	mock := &doubleDecodeMockIAMClient{decodedDocument: decoded}

	policies := getAttachedPoliciesForRole("regression-role", true, mock)

	policy, ok := policies["RegressionPolicy"]
	if !ok {
		t.Fatalf("expected attached policy %q to be present", "RegressionPolicy")
	}
	if len(policy.Statement) != 1 {
		t.Fatalf("expected exactly one statement, got %d", len(policy.Statement))
	}

	resource, ok := policy.Statement[0].Resource.(string)
	if !ok {
		t.Fatalf("expected Resource to be a string, got %T", policy.Statement[0].Resource)
	}

	want := "literal-%2F-and-%ZZ-value"
	if resource != want {
		t.Errorf("Resource = %q, want %q (double-decode corrupted the value)", resource, want)
	}
}
