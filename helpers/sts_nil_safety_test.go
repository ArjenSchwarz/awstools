package helpers

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Regression tests for T-734: GetAccountID must not panic when the STS
// GetCallerIdentity response contains nil Account, Arn, or UserId fields.
//
// In edge cases (e.g. SSO sessions in specific states) the AWS SDK can
// return a *sts.GetCallerIdentityOutput with one or more of these pointer
// fields unset. Previously the code dereferenced `*result.Account`
// directly, which panicked.

func TestAccountIDFromIdentity_Populated(t *testing.T) {
	out := &sts.GetCallerIdentityOutput{
		Account: aws.String("123456789012"),
		Arn:     aws.String("arn:aws:iam::123456789012:user/test"),
		UserId:  aws.String("AIDACKCEVSQ6C2EXAMPLE"),
	}
	if got := accountIDFromIdentity(out); got != "123456789012" {
		t.Fatalf("expected account ID 123456789012, got %q", got)
	}
}

func TestAccountIDFromIdentity_NilAccount(t *testing.T) {
	// Expected: empty string instead of panic when Account is nil.
	out := &sts.GetCallerIdentityOutput{
		Arn:    aws.String("arn:aws:iam::123456789012:user/test"),
		UserId: aws.String("AIDACKCEVSQ6C2EXAMPLE"),
	}
	if got := accountIDFromIdentity(out); got != "" {
		t.Fatalf("expected empty string for nil Account, got %q", got)
	}
}

func TestAccountIDFromIdentity_NilOutput(t *testing.T) {
	// Expected: empty string instead of panic when the whole output is nil.
	if got := accountIDFromIdentity(nil); got != "" {
		t.Fatalf("expected empty string for nil output, got %q", got)
	}
}

func TestAccountIDFromIdentity_AllNilFields(t *testing.T) {
	// Expected: empty string when every field is nil (SSO edge case).
	out := &sts.GetCallerIdentityOutput{}
	if got := accountIDFromIdentity(out); got != "" {
		t.Fatalf("expected empty string for all-nil fields, got %q", got)
	}
}
