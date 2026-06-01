package cmd

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// TestParsePublicAccessBlock verifies that parsePublicAccessBlock distinguishes
// between "unknown" (no PAB configured or GetPublicAccessBlock failed) and the
// legitimate "all four flags false" state. Regression test for T-693.
func TestParsePublicAccessBlock(t *testing.T) {
	tests := []struct {
		name   string
		config *types.PublicAccessBlockConfiguration
		want   string
	}{
		{
			name:   "unknown when nil",
			config: nil,
			want:   "Unknown",
		},
		{
			name: "all true",
			config: &types.PublicAccessBlockConfiguration{
				BlockPublicAcls:       aws.Bool(true),
				BlockPublicPolicy:     aws.Bool(true),
				IgnorePublicAcls:      aws.Bool(true),
				RestrictPublicBuckets: aws.Bool(true),
			},
			want: "All true",
		},
		{
			name: "all false",
			config: &types.PublicAccessBlockConfiguration{
				BlockPublicAcls:       aws.Bool(false),
				BlockPublicPolicy:     aws.Bool(false),
				IgnorePublicAcls:      aws.Bool(false),
				RestrictPublicBuckets: aws.Bool(false),
			},
			want: "All false",
		},
		{
			name: "mixed",
			config: &types.PublicAccessBlockConfiguration{
				BlockPublicAcls:       aws.Bool(true),
				BlockPublicPolicy:     aws.Bool(false),
				IgnorePublicAcls:      aws.Bool(true),
				RestrictPublicBuckets: aws.Bool(false),
			},
			want: "Block Public ACLs: true, Block Public Policy: false, Ignore Public ACLs: true, Restrict Public Buckets: false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePublicAccessBlock(tt.config)
			if got != tt.want {
				t.Errorf("parsePublicAccessBlock() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestS3EncryptionToString verifies that s3EncryptionToString tolerates rules
// with a nil ApplyServerSideEncryptionByDefault block. A malformed or partial
// GetBucketEncryption response can return a rule without the default
// encryption block; before the fix this dereferenced a nil pointer and
// panicked while rendering `awstools s3 list`. Regression test for T-1176.
func TestS3EncryptionToString(t *testing.T) {
	tests := []struct {
		name  string
		rules []types.ServerSideEncryptionRule
		want  string
	}{
		{
			name:  "no rules",
			rules: nil,
			want:  "",
		},
		{
			name: "nil default rule does not panic",
			rules: []types.ServerSideEncryptionRule{
				{ApplyServerSideEncryptionByDefault: nil},
			},
			want: s3StateUnknown,
		},
		{
			name: "aes256 algorithm",
			rules: []types.ServerSideEncryptionRule{
				{
					ApplyServerSideEncryptionByDefault: &types.ServerSideEncryptionByDefault{
						SSEAlgorithm: types.ServerSideEncryptionAes256,
					},
				},
			},
			want: string(types.ServerSideEncryptionAes256),
		},
		{
			name: "kms algorithm",
			rules: []types.ServerSideEncryptionRule{
				{
					ApplyServerSideEncryptionByDefault: &types.ServerSideEncryptionByDefault{
						SSEAlgorithm: types.ServerSideEncryptionAwsKms,
					},
				},
			},
			want: string(types.ServerSideEncryptionAwsKms),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s3EncryptionToString(tt.rules)
			if got != tt.want {
				t.Errorf("s3EncryptionToString() = %q, want %q", got, tt.want)
			}
		})
	}
}
