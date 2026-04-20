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
