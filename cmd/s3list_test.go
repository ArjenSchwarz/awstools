package cmd

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// TestUnencryptedOnlyFilter verifies that the --unencrypted-only filter
// excludes only buckets that are confirmed encrypted, while keeping both
// confirmed-unencrypted and unknown-encryption buckets visible. Unknown
// states must stay visible so users can investigate them (see T-714).
// Regression test for T-1090.
func TestUnencryptedOnlyFilter(t *testing.T) {
	tests := []struct {
		name          string
		hasEncryption *bool
		wantSkip      bool
	}{
		{
			name:          "confirmed encrypted is excluded",
			hasEncryption: aws.Bool(true),
			wantSkip:      true,
		},
		{
			name:          "confirmed unencrypted is kept",
			hasEncryption: aws.Bool(false),
			wantSkip:      false,
		},
		{
			name:          "unknown encryption state is kept",
			hasEncryption: nil,
			wantSkip:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := skipForUnencryptedOnly(tt.hasEncryption)
			if got != tt.wantSkip {
				t.Errorf("skipForUnencryptedOnly(%v) = %v, want %v", tt.hasEncryption, got, tt.wantSkip)
			}
		})
	}
}

// TestPublicOnlyFilter verifies that the --public-only filter excludes only
// buckets confirmed private, while keeping confirmed-public and
// unknown-public buckets visible. Companion coverage for T-1090.
func TestPublicOnlyFilter(t *testing.T) {
	tests := []struct {
		name     string
		isPublic *bool
		wantSkip bool
	}{
		{
			name:     "confirmed private is excluded",
			isPublic: aws.Bool(false),
			wantSkip: true,
		},
		{
			name:     "confirmed public is kept",
			isPublic: aws.Bool(true),
			wantSkip: false,
		},
		{
			name:     "unknown public state is kept",
			isPublic: nil,
			wantSkip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := skipForPublicOnly(tt.isPublic)
			if got != tt.wantSkip {
				t.Errorf("skipForPublicOnly(%v) = %v, want %v", tt.isPublic, got, tt.wantSkip)
			}
		})
	}
}

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
