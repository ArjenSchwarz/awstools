package helpers

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func TestS3Bucket_GetReplicationStrings(t *testing.T) {
	tests := []struct {
		name     string
		bucket   S3Bucket
		expected []string
	}{
		{
			name: "replication with simple prefix filter",
			bucket: S3Bucket{
				Replication: types.ReplicationConfiguration{
					Rules: []types.ReplicationRule{
						{
							Filter: &types.ReplicationRuleFilter{
								Prefix: aws.String("documents/"),
							},
							Destination: &types.Destination{
								Bucket: aws.String("arn:aws:s3:::backup-bucket"),
							},
						},
					},
				},
			},
			expected: []string{"Prefix: documents/ => arn:aws:s3:::backup-bucket"},
		},
		{
			name: "replication with empty prefix (entire bucket)",
			bucket: S3Bucket{
				Replication: types.ReplicationConfiguration{
					Rules: []types.ReplicationRule{
						{
							Filter: &types.ReplicationRuleFilter{
								Prefix: aws.String(""),
							},
							Destination: &types.Destination{
								Bucket: aws.String("arn:aws:s3:::backup-bucket"),
							},
						},
					},
				},
			},
			expected: []string{"Entire bucket => arn:aws:s3:::backup-bucket"},
		},
		{
			name: "replication with tag filter",
			bucket: S3Bucket{
				Replication: types.ReplicationConfiguration{
					Rules: []types.ReplicationRule{
						{
							Filter: &types.ReplicationRuleFilter{
								Tag: &types.Tag{
									Key:   aws.String("Environment"),
									Value: aws.String("Production"),
								},
							},
							Destination: &types.Destination{
								Bucket: aws.String("arn:aws:s3:::prod-backup"),
							},
						},
					},
				},
			},
			expected: []string{"Tag Environment:Production => arn:aws:s3:::prod-backup"},
		},
		{
			name: "replication with And filter (prefix and tags)",
			bucket: S3Bucket{
				Replication: types.ReplicationConfiguration{
					Rules: []types.ReplicationRule{
						{
							Filter: &types.ReplicationRuleFilter{
								And: &types.ReplicationRuleAndOperator{
									Prefix: aws.String("logs/"),
									Tags: []types.Tag{
										{
											Key:   aws.String("Department"),
											Value: aws.String("IT"),
										},
										{
											Key:   aws.String("Project"),
											Value: aws.String("WebApp"),
										},
									},
								},
							},
							Destination: &types.Destination{
								Bucket: aws.String("arn:aws:s3:::complex-backup"),
							},
						},
					},
				},
			},
			expected: []string{"Prefix: logs/, and Tag Department:IT and Tag Project:WebApp => arn:aws:s3:::complex-backup"},
		},
		{
			name: "replication with And filter (no prefix, only tags)",
			bucket: S3Bucket{
				Replication: types.ReplicationConfiguration{
					Rules: []types.ReplicationRule{
						{
							Filter: &types.ReplicationRuleFilter{
								And: &types.ReplicationRuleAndOperator{
									Tags: []types.Tag{
										{
											Key:   aws.String("Backup"),
											Value: aws.String("Required"),
										},
									},
								},
							},
							Destination: &types.Destination{
								Bucket: aws.String("arn:aws:s3:::tag-backup"),
							},
						},
					},
				},
			},
			expected: []string{"Tag Backup:Required => arn:aws:s3:::tag-backup"},
		},
		{
			name: "replication with no filter",
			bucket: S3Bucket{
				Replication: types.ReplicationConfiguration{
					Rules: []types.ReplicationRule{
						{
							Filter: nil,
							Destination: &types.Destination{
								Bucket: aws.String("arn:aws:s3:::no-filter-backup"),
							},
						},
					},
				},
			},
			expected: []string{"Entire bucket => arn:aws:s3:::no-filter-backup"},
		},
		{
			name: "multiple replication rules",
			bucket: S3Bucket{
				Replication: types.ReplicationConfiguration{
					Rules: []types.ReplicationRule{
						{
							Filter: &types.ReplicationRuleFilter{
								Prefix: aws.String("documents/"),
							},
							Destination: &types.Destination{
								Bucket: aws.String("arn:aws:s3:::doc-backup"),
							},
						},
						{
							Filter: &types.ReplicationRuleFilter{
								Tag: &types.Tag{
									Key:   aws.String("Critical"),
									Value: aws.String("Yes"),
								},
							},
							Destination: &types.Destination{
								Bucket: aws.String("arn:aws:s3:::critical-backup"),
							},
						},
					},
				},
			},
			expected: []string{
				"Prefix: documents/ => arn:aws:s3:::doc-backup",
				"Tag Critical:Yes => arn:aws:s3:::critical-backup",
			},
		},
		{
			name: "no replication rules",
			bucket: S3Bucket{
				Replication: types.ReplicationConfiguration{
					Rules: []types.ReplicationRule{},
				},
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.bucket.GetReplicationStrings()
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("S3Bucket.GetReplicationStrings() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestS3Bucket_Struct(t *testing.T) {
	bucket := S3Bucket{
		Account:              "123456789012",
		Name:                 "my-test-bucket",
		Owner:                "bucket-owner",
		IsPublic:             true,
		Region:               "us-west-2",
		OpenACLs:             false,
		PublicPolicy:         true,
		LoggingEnabled:       true,
		LoggingBucket:        "logging-bucket",
		HasEncryption:        true,
		Versioning:           true,
		VersioningMFAEnabled: false,
		Tags: map[string]string{
			"Environment": "Production",
			"Team":        "DevOps",
		},
		Policy: `{"Version":"2012-10-17","Statement":[]}`,
	}

	// Test basic properties
	if bucket.Account != "123456789012" {
		t.Errorf("Expected Account to be '123456789012', got %s", bucket.Account)
	}
	if bucket.Name != "my-test-bucket" {
		t.Errorf("Expected Name to be 'my-test-bucket', got %s", bucket.Name)
	}
	if !bucket.IsPublic {
		t.Errorf("Expected IsPublic to be true, got %v", bucket.IsPublic)
	}
	if bucket.Region != "us-west-2" {
		t.Errorf("Expected Region to be 'us-west-2', got %s", bucket.Region)
	}
	if bucket.OpenACLs {
		t.Errorf("Expected OpenACLs to be false, got %v", bucket.OpenACLs)
	}
	if !bucket.PublicPolicy {
		t.Errorf("Expected PublicPolicy to be true, got %v", bucket.PublicPolicy)
	}
	if !bucket.LoggingEnabled {
		t.Errorf("Expected LoggingEnabled to be true, got %v", bucket.LoggingEnabled)
	}
	if bucket.LoggingBucket != "logging-bucket" {
		t.Errorf("Expected LoggingBucket to be 'logging-bucket', got %s", bucket.LoggingBucket)
	}
	if !bucket.HasEncryption {
		t.Errorf("Expected HasEncryption to be true, got %v", bucket.HasEncryption)
	}
	if !bucket.Versioning {
		t.Errorf("Expected Versioning to be true, got %v", bucket.Versioning)
	}
	if bucket.VersioningMFAEnabled {
		t.Errorf("Expected VersioningMFAEnabled to be false, got %v", bucket.VersioningMFAEnabled)
	}

	// Test tags
	expectedTags := map[string]string{
		"Environment": "Production",
		"Team":        "DevOps",
	}
	if !reflect.DeepEqual(bucket.Tags, expectedTags) {
		t.Errorf("Expected Tags to be %v, got %v", expectedTags, bucket.Tags)
	}

	// Test policy
	expectedPolicy := `{"Version":"2012-10-17","Statement":[]}`
	if bucket.Policy != expectedPolicy {
		t.Errorf("Expected Policy to be %s, got %s", expectedPolicy, bucket.Policy)
	}
}

func TestS3Bucket_ACLsAndGrants(t *testing.T) {
	bucket := S3Bucket{
		ACLs: []types.Grant{
			{
				Grantee: &types.Grantee{
					Type: types.TypeCanonicalUser,
					ID:   aws.String("owner-canonical-user-id"),
				},
				Permission: types.PermissionFullControl,
			},
			{
				Grantee: &types.Grantee{
					Type: types.TypeGroup,
					URI:  aws.String("http://acs.amazonaws.com/groups/global/AllUsers"),
				},
				Permission: types.PermissionRead,
			},
		},
	}

	if len(bucket.ACLs) != 2 {
		t.Errorf("Expected 2 ACLs, got %d", len(bucket.ACLs))
	}

	// Test first grant (canonical user)
	firstGrant := bucket.ACLs[0]
	if firstGrant.Grantee.Type != types.TypeCanonicalUser {
		t.Errorf("Expected first grant type to be CanonicalUser, got %v", firstGrant.Grantee.Type)
	}
	if firstGrant.Permission != types.PermissionFullControl {
		t.Errorf("Expected first grant permission to be FullControl, got %v", firstGrant.Permission)
	}

	// Test second grant (group)
	secondGrant := bucket.ACLs[1]
	if secondGrant.Grantee.Type != types.TypeGroup {
		t.Errorf("Expected second grant type to be Group, got %v", secondGrant.Grantee.Type)
	}
	if secondGrant.Permission != types.PermissionRead {
		t.Errorf("Expected second grant permission to be Read, got %v", secondGrant.Permission)
	}
}

func TestS3Bucket_EncryptionRules(t *testing.T) {
	bucket := S3Bucket{
		HasEncryption: true,
		EncryptionRules: []types.ServerSideEncryptionRule{
			{
				ApplyServerSideEncryptionByDefault: &types.ServerSideEncryptionByDefault{
					SSEAlgorithm: types.ServerSideEncryptionAes256,
				},
			},
		},
	}

	if !bucket.HasEncryption {
		t.Errorf("Expected HasEncryption to be true, got %v", bucket.HasEncryption)
	}

	if len(bucket.EncryptionRules) != 1 {
		t.Errorf("Expected 1 encryption rule, got %d", len(bucket.EncryptionRules))
	}

	rule := bucket.EncryptionRules[0]
	if rule.ApplyServerSideEncryptionByDefault.SSEAlgorithm != types.ServerSideEncryptionAes256 {
		t.Errorf("Expected SSE algorithm to be AES256, got %v", rule.ApplyServerSideEncryptionByDefault.SSEAlgorithm)
	}
}

func TestS3Bucket_PublicAccessBlockConfiguration(t *testing.T) {
	bucket := S3Bucket{
		PublicAccessBlockConfiguration: types.PublicAccessBlockConfiguration{
			BlockPublicAcls:       aws.Bool(true),
			BlockPublicPolicy:     aws.Bool(true),
			IgnorePublicAcls:      aws.Bool(true),
			RestrictPublicBuckets: aws.Bool(true),
		},
	}

	config := bucket.PublicAccessBlockConfiguration
	if !aws.ToBool(config.BlockPublicAcls) {
		t.Errorf("Expected BlockPublicAcls to be true, got %v", aws.ToBool(config.BlockPublicAcls))
	}
	if !aws.ToBool(config.BlockPublicPolicy) {
		t.Errorf("Expected BlockPublicPolicy to be true, got %v", aws.ToBool(config.BlockPublicPolicy))
	}
	if !aws.ToBool(config.IgnorePublicAcls) {
		t.Errorf("Expected IgnorePublicAcls to be true, got %v", aws.ToBool(config.IgnorePublicAcls))
	}
	if !aws.ToBool(config.RestrictPublicBuckets) {
		t.Errorf("Expected RestrictPublicBuckets to be true, got %v", aws.ToBool(config.RestrictPublicBuckets))
	}
}
