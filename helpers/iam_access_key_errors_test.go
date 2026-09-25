package helpers

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

type accessKeyErrorIAMClient struct {
	IAMClient
	listErr     error
	lastUsedErr error
}

func (m accessKeyErrorIAMClient) ListAccessKeys(_ context.Context, _ *iam.ListAccessKeysInput, _ ...func(*iam.Options)) (*iam.ListAccessKeysOutput, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return &iam.ListAccessKeysOutput{AccessKeyMetadata: []types.AccessKeyMetadata{{AccessKeyId: aws.String("AKIAEXAMPLE")}}}, nil
}

func (m accessKeyErrorIAMClient) GetAccessKeyLastUsed(_ context.Context, _ *iam.GetAccessKeyLastUsedInput, _ ...func(*iam.Options)) (*iam.GetAccessKeyLastUsedOutput, error) {
	return nil, m.lastUsedErr
}

// API failures are expected command errors, not reasons to crash the process.
func TestIAMUser_AccessKeyAPIErrorsDoNotPanic_T1549(t *testing.T) {
	user := IAMUser{Name: "test-user"}
	accessDenied := errors.New("access denied")
	throttled := errors.New("throttled")
	tests := []struct {
		name    string
		call    func(accessKeyErrorIAMClient) error
		mock    accessKeyErrorIAMClient
		wantErr error
	}{
		{
			name:    "HasAccessKeys ListAccessKeys error",
			call:    func(mock accessKeyErrorIAMClient) error { _, err := user.HasAccessKeys(mock); return err },
			mock:    accessKeyErrorIAMClient{listErr: accessDenied},
			wantErr: accessDenied,
		},
		{
			name:    "GetLastAccessKeyDate ListAccessKeys error",
			call:    func(mock accessKeyErrorIAMClient) error { _, err := user.GetLastAccessKeyDate(mock); return err },
			mock:    accessKeyErrorIAMClient{listErr: throttled},
			wantErr: throttled,
		},
		{
			name:    "GetLastAccessKeyDate GetAccessKeyLastUsed error",
			call:    func(mock accessKeyErrorIAMClient) error { _, err := user.GetLastAccessKeyDate(mock); return err },
			mock:    accessKeyErrorIAMClient{lastUsedErr: accessDenied},
			wantErr: accessDenied,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call(tc.mock)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want wrapped %v", err, tc.wantErr)
			}
			if !strings.Contains(err.Error(), user.Name) {
				t.Errorf("error %q does not name the affected user", err)
			}
		})
	}
}
