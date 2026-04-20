package config

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Regression tests for T-734: setCallerInfo and its helper must not panic
// when STS GetCallerIdentity returns nil Account, Arn, or UserId fields.
//
// In some edge cases (notably SSO sessions in specific states) the AWS
// SDK returns a *sts.GetCallerIdentityOutput with one or more of these
// pointer fields unset. The previous code dereferenced them directly,
// which panicked.

func TestResolveCallerIdentity_Populated(t *testing.T) {
	out := &sts.GetCallerIdentityOutput{
		Account: aws.String("123456789012"),
		Arn:     aws.String("arn:aws:iam::123456789012:user/test"),
		UserId:  aws.String("AIDACKCEVSQ6C2EXAMPLE"),
	}
	accountID, userID, arn := resolveCallerIdentity(out)
	if accountID != "123456789012" {
		t.Errorf("expected account ID 123456789012, got %q", accountID)
	}
	if userID != "AIDACKCEVSQ6C2EXAMPLE" {
		t.Errorf("expected user ID AIDACKCEVSQ6C2EXAMPLE, got %q", userID)
	}
	if arn != "arn:aws:iam::123456789012:user/test" {
		t.Errorf("expected arn arn:aws:iam::123456789012:user/test, got %q", arn)
	}
}

func TestResolveCallerIdentity_NilAccount(t *testing.T) {
	// Account nil — must return empty string for AccountID without panicking.
	out := &sts.GetCallerIdentityOutput{
		Arn:    aws.String("arn:aws:iam::123456789012:user/test"),
		UserId: aws.String("AIDACKCEVSQ6C2EXAMPLE"),
	}
	accountID, userID, arn := resolveCallerIdentity(out)
	if accountID != "" {
		t.Errorf("expected empty accountID for nil Account, got %q", accountID)
	}
	if userID != "AIDACKCEVSQ6C2EXAMPLE" {
		t.Errorf("expected user ID AIDACKCEVSQ6C2EXAMPLE, got %q", userID)
	}
	if arn != "arn:aws:iam::123456789012:user/test" {
		t.Errorf("expected arn arn:aws:iam::123456789012:user/test, got %q", arn)
	}
}

func TestResolveCallerIdentity_NilUserID(t *testing.T) {
	// UserId nil — must return empty string for UserID without panicking.
	out := &sts.GetCallerIdentityOutput{
		Account: aws.String("123456789012"),
		Arn:     aws.String("arn:aws:iam::123456789012:user/test"),
	}
	accountID, userID, arn := resolveCallerIdentity(out)
	if accountID != "123456789012" {
		t.Errorf("expected accountID 123456789012, got %q", accountID)
	}
	if userID != "" {
		t.Errorf("expected empty userID for nil UserId, got %q", userID)
	}
	if arn != "arn:aws:iam::123456789012:user/test" {
		t.Errorf("expected arn arn:aws:iam::123456789012:user/test, got %q", arn)
	}
}

func TestResolveCallerIdentity_NilArn(t *testing.T) {
	// Arn nil — must return empty string for arn without panicking.
	out := &sts.GetCallerIdentityOutput{
		Account: aws.String("123456789012"),
		UserId:  aws.String("AIDACKCEVSQ6C2EXAMPLE"),
	}
	accountID, userID, arn := resolveCallerIdentity(out)
	if accountID != "123456789012" {
		t.Errorf("expected accountID 123456789012, got %q", accountID)
	}
	if userID != "AIDACKCEVSQ6C2EXAMPLE" {
		t.Errorf("expected userID AIDACKCEVSQ6C2EXAMPLE, got %q", userID)
	}
	if arn != "" {
		t.Errorf("expected empty arn for nil Arn, got %q", arn)
	}
}

func TestResolveCallerIdentity_AllNilFields(t *testing.T) {
	// All fields nil — every returned value must be empty, no panic.
	out := &sts.GetCallerIdentityOutput{}
	accountID, userID, arn := resolveCallerIdentity(out)
	if accountID != "" || userID != "" || arn != "" {
		t.Errorf("expected all empty strings for nil fields, got accountID=%q userID=%q arn=%q", accountID, userID, arn)
	}
}

func TestResolveCallerIdentity_NilOutput(t *testing.T) {
	// Nil output — must return empty strings without panicking.
	accountID, userID, arn := resolveCallerIdentity(nil)
	if accountID != "" || userID != "" || arn != "" {
		t.Errorf("expected all empty strings for nil output, got accountID=%q userID=%q arn=%q", accountID, userID, arn)
	}
}
