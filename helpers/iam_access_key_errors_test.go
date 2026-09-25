package helpers

import (
	"context"
	"errors"
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
	tests := []struct {
		name string
		call func(accessKeyErrorIAMClient)
		mock accessKeyErrorIAMClient
	}{
		{
			name: "HasAccessKeys ListAccessKeys error",
			call: func(mock accessKeyErrorIAMClient) { user.HasAccessKeys(mock) },
			mock: accessKeyErrorIAMClient{listErr: errors.New("access denied")},
		},
		{
			name: "GetLastAccessKeyDate ListAccessKeys error",
			call: func(mock accessKeyErrorIAMClient) { user.GetLastAccessKeyDate(mock) },
			mock: accessKeyErrorIAMClient{listErr: errors.New("throttled")},
		},
		{
			name: "GetLastAccessKeyDate GetAccessKeyLastUsed error",
			call: func(mock accessKeyErrorIAMClient) { user.GetLastAccessKeyDate(mock) },
			mock: accessKeyErrorIAMClient{lastUsedErr: errors.New("access denied")},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.call(tc.mock)
		})
	}
}
