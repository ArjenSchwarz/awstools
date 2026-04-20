package helpers

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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
		IsPublic:             aws.Bool(true),
		Region:               "us-west-2",
		OpenACLs:             aws.Bool(false),
		PublicPolicy:         aws.Bool(true),
		LoggingEnabled:       aws.Bool(true),
		LoggingBucket:        "logging-bucket",
		HasEncryption:        aws.Bool(true),
		Versioning:           aws.Bool(true),
		VersioningMFAEnabled: aws.Bool(false),
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
	if !aws.ToBool(bucket.IsPublic) {
		t.Errorf("Expected IsPublic to be true, got %v", aws.ToBool(bucket.IsPublic))
	}
	if bucket.Region != "us-west-2" {
		t.Errorf("Expected Region to be 'us-west-2', got %s", bucket.Region)
	}
	if aws.ToBool(bucket.OpenACLs) {
		t.Errorf("Expected OpenACLs to be false, got %v", aws.ToBool(bucket.OpenACLs))
	}
	if !aws.ToBool(bucket.PublicPolicy) {
		t.Errorf("Expected PublicPolicy to be true, got %v", aws.ToBool(bucket.PublicPolicy))
	}
	if !aws.ToBool(bucket.LoggingEnabled) {
		t.Errorf("Expected LoggingEnabled to be true, got %v", aws.ToBool(bucket.LoggingEnabled))
	}
	if bucket.LoggingBucket != "logging-bucket" {
		t.Errorf("Expected LoggingBucket to be 'logging-bucket', got %s", bucket.LoggingBucket)
	}
	if !aws.ToBool(bucket.HasEncryption) {
		t.Errorf("Expected HasEncryption to be true, got %v", aws.ToBool(bucket.HasEncryption))
	}
	if !aws.ToBool(bucket.Versioning) {
		t.Errorf("Expected Versioning to be true, got %v", aws.ToBool(bucket.Versioning))
	}
	if aws.ToBool(bucket.VersioningMFAEnabled) {
		t.Errorf("Expected VersioningMFAEnabled to be false, got %v", aws.ToBool(bucket.VersioningMFAEnabled))
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
		HasEncryption: aws.Bool(true),
		EncryptionRules: []types.ServerSideEncryptionRule{
			{
				ApplyServerSideEncryptionByDefault: &types.ServerSideEncryptionByDefault{
					SSEAlgorithm: types.ServerSideEncryptionAes256,
				},
			},
		},
	}

	if !aws.ToBool(bucket.HasEncryption) {
		t.Errorf("Expected HasEncryption to be true, got %v", aws.ToBool(bucket.HasEncryption))
	}

	if len(bucket.EncryptionRules) != 1 {
		t.Errorf("Expected 1 encryption rule, got %d", len(bucket.EncryptionRules))
	}

	rule := bucket.EncryptionRules[0]
	if rule.ApplyServerSideEncryptionByDefault.SSEAlgorithm != types.ServerSideEncryptionAes256 {
		t.Errorf("Expected SSE algorithm to be AES256, got %v", rule.ApplyServerSideEncryptionByDefault.SSEAlgorithm)
	}
}

// Regression test for T-354: GetAllBuckets panics when Owner or DisplayName is nil.
// ListBuckets can omit Owner/DisplayName depending on permissions.
func TestResolveOwnerName(t *testing.T) {
	tests := []struct {
		name     string
		owner    *types.Owner
		expected string
	}{
		{
			name:     "nil owner returns empty string",
			owner:    nil,
			expected: "",
		},
		{
			name:     "owner with nil DisplayName and nil ID returns empty string",
			owner:    &types.Owner{},
			expected: "",
		},
		{
			name:     "owner with nil DisplayName falls back to ID",
			owner:    &types.Owner{ID: aws.String("123456789012")},
			expected: "123456789012",
		},
		{
			name:     "owner with empty DisplayName falls back to ID",
			owner:    &types.Owner{DisplayName: aws.String(""), ID: aws.String("123456789012")},
			expected: "123456789012",
		},
		{
			name:     "owner with DisplayName returns DisplayName",
			owner:    &types.Owner{DisplayName: aws.String("my-account"), ID: aws.String("123456789012")},
			expected: "my-account",
		},
		{
			name:     "owner with DisplayName and nil ID returns DisplayName",
			owner:    &types.Owner{DisplayName: aws.String("my-account")},
			expected: "my-account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveOwnerName(tt.owner)
			if result != tt.expected {
				t.Errorf("resolveOwnerName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestHasOpenACLs_NilGranteeURI is a regression test for T-338.
// Grants with a nil URI (e.g. TypeAmazonCustomerByEmail) must not panic.
func TestHasOpenACLs_NilGranteeURI(t *testing.T) {
	tests := []struct {
		name     string
		acls     []types.Grant
		expected bool
	}{
		{
			name:     "empty grants",
			acls:     []types.Grant{},
			expected: false,
		},
		{
			name: "canonical user only",
			acls: []types.Grant{
				{
					Grantee: &types.Grantee{
						Type: types.TypeCanonicalUser,
						ID:   aws.String("owner-id"),
					},
					Permission: types.PermissionFullControl,
				},
			},
			expected: false,
		},
		{
			name: "log delivery group (not open)",
			acls: []types.Grant{
				{
					Grantee: &types.Grantee{
						Type: types.TypeGroup,
						URI:  aws.String("http://acs.amazonaws.com/groups/s3/LogDelivery"),
					},
					Permission: types.PermissionWrite,
				},
			},
			expected: false,
		},
		{
			name: "public all-users group (open)",
			acls: []types.Grant{
				{
					Grantee: &types.Grantee{
						Type: types.TypeGroup,
						URI:  aws.String("http://acs.amazonaws.com/groups/global/AllUsers"),
					},
					Permission: types.PermissionRead,
				},
			},
			expected: true,
		},
		{
			name: "email grantee with nil URI must not panic (T-338)",
			acls: []types.Grant{
				{
					Grantee: &types.Grantee{
						Type:         types.TypeAmazonCustomerByEmail,
						EmailAddress: aws.String("user@example.com"),
						// URI is nil — this caused the original panic
					},
					Permission: types.PermissionRead,
				},
			},
			expected: true,
		},
		{
			name: "nil Grantee pointer must not panic",
			acls: []types.Grant{
				{
					Grantee:    nil,
					Permission: types.PermissionRead,
				},
			},
			expected: false,
		},
		{
			name: "mixed grants with nil URI among others",
			acls: []types.Grant{
				{
					Grantee: &types.Grantee{
						Type: types.TypeCanonicalUser,
						ID:   aws.String("owner-id"),
					},
					Permission: types.PermissionFullControl,
				},
				{
					Grantee: &types.Grantee{
						Type:         types.TypeAmazonCustomerByEmail,
						EmailAddress: aws.String("user@example.com"),
					},
					Permission: types.PermissionRead,
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasOpenACLs(tt.acls)
			if result != tt.expected {
				t.Errorf("hasOpenACLs() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestNormalizeBucketLocation is a regression test for T-690.
// S3's GetBucketLocation returns the legacy value "EU" for buckets originally
// created in eu-west-1, and an empty LocationConstraint for us-east-1. Both
// must be normalised to their canonical region IDs.
func TestNormalizeBucketLocation(t *testing.T) {
	tests := []struct {
		name       string
		constraint string
		expected   string
	}{
		{
			name:       "empty constraint maps to us-east-1",
			constraint: "",
			expected:   "us-east-1",
		},
		{
			name:       "legacy EU constraint maps to eu-west-1",
			constraint: "EU",
			expected:   "eu-west-1",
		},
		{
			name:       "standard region passes through unchanged",
			constraint: "eu-west-1",
			expected:   "eu-west-1",
		},
		{
			name:       "us-west-2 passes through unchanged",
			constraint: "us-west-2",
			expected:   "us-west-2",
		},
		{
			name:       "ap-southeast-2 passes through unchanged",
			constraint: "ap-southeast-2",
			expected:   "ap-southeast-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeBucketLocation(types.BucketLocationConstraint(tt.constraint))
			if result != tt.expected {
				t.Errorf("normalizeBucketLocation(%q) = %q, want %q", tt.constraint, result, tt.expected)
			}
		})
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

// mockS3Client implements S3API. Each method delegates to a function
// field so individual tests can return specific responses or errors
// per call.
type mockS3Client struct {
	listBuckets           func(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error)
	getBucketPolicyStatus func(ctx context.Context, params *s3.GetBucketPolicyStatusInput, optFns ...func(*s3.Options)) (*s3.GetBucketPolicyStatusOutput, error)
	getBucketAcl          func(ctx context.Context, params *s3.GetBucketAclInput, optFns ...func(*s3.Options)) (*s3.GetBucketAclOutput, error)
	getPublicAccessBlock  func(ctx context.Context, params *s3.GetPublicAccessBlockInput, optFns ...func(*s3.Options)) (*s3.GetPublicAccessBlockOutput, error)
	getBucketLocation     func(ctx context.Context, params *s3.GetBucketLocationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error)
	getBucketLogging      func(ctx context.Context, params *s3.GetBucketLoggingInput, optFns ...func(*s3.Options)) (*s3.GetBucketLoggingOutput, error)
	getBucketEncryption   func(ctx context.Context, params *s3.GetBucketEncryptionInput, optFns ...func(*s3.Options)) (*s3.GetBucketEncryptionOutput, error)
	getBucketTagging      func(ctx context.Context, params *s3.GetBucketTaggingInput, optFns ...func(*s3.Options)) (*s3.GetBucketTaggingOutput, error)
	getBucketPolicy       func(ctx context.Context, params *s3.GetBucketPolicyInput, optFns ...func(*s3.Options)) (*s3.GetBucketPolicyOutput, error)
	getBucketReplication  func(ctx context.Context, params *s3.GetBucketReplicationInput, optFns ...func(*s3.Options)) (*s3.GetBucketReplicationOutput, error)
	getBucketVersioning   func(ctx context.Context, params *s3.GetBucketVersioningInput, optFns ...func(*s3.Options)) (*s3.GetBucketVersioningOutput, error)
}

func (m *mockS3Client) ListBuckets(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {
	return m.listBuckets(ctx, params, optFns...)
}

func (m *mockS3Client) GetBucketPolicyStatus(ctx context.Context, params *s3.GetBucketPolicyStatusInput, optFns ...func(*s3.Options)) (*s3.GetBucketPolicyStatusOutput, error) {
	return m.getBucketPolicyStatus(ctx, params, optFns...)
}

func (m *mockS3Client) GetBucketAcl(ctx context.Context, params *s3.GetBucketAclInput, optFns ...func(*s3.Options)) (*s3.GetBucketAclOutput, error) {
	return m.getBucketAcl(ctx, params, optFns...)
}

func (m *mockS3Client) GetPublicAccessBlock(ctx context.Context, params *s3.GetPublicAccessBlockInput, optFns ...func(*s3.Options)) (*s3.GetPublicAccessBlockOutput, error) {
	return m.getPublicAccessBlock(ctx, params, optFns...)
}

func (m *mockS3Client) GetBucketLocation(ctx context.Context, params *s3.GetBucketLocationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error) {
	return m.getBucketLocation(ctx, params, optFns...)
}

func (m *mockS3Client) GetBucketLogging(ctx context.Context, params *s3.GetBucketLoggingInput, optFns ...func(*s3.Options)) (*s3.GetBucketLoggingOutput, error) {
	return m.getBucketLogging(ctx, params, optFns...)
}

func (m *mockS3Client) GetBucketEncryption(ctx context.Context, params *s3.GetBucketEncryptionInput, optFns ...func(*s3.Options)) (*s3.GetBucketEncryptionOutput, error) {
	return m.getBucketEncryption(ctx, params, optFns...)
}

func (m *mockS3Client) GetBucketTagging(ctx context.Context, params *s3.GetBucketTaggingInput, optFns ...func(*s3.Options)) (*s3.GetBucketTaggingOutput, error) {
	return m.getBucketTagging(ctx, params, optFns...)
}

func (m *mockS3Client) GetBucketPolicy(ctx context.Context, params *s3.GetBucketPolicyInput, optFns ...func(*s3.Options)) (*s3.GetBucketPolicyOutput, error) {
	return m.getBucketPolicy(ctx, params, optFns...)
}

func (m *mockS3Client) GetBucketReplication(ctx context.Context, params *s3.GetBucketReplicationInput, optFns ...func(*s3.Options)) (*s3.GetBucketReplicationOutput, error) {
	return m.getBucketReplication(ctx, params, optFns...)
}

func (m *mockS3Client) GetBucketVersioning(ctx context.Context, params *s3.GetBucketVersioningInput, optFns ...func(*s3.Options)) (*s3.GetBucketVersioningOutput, error) {
	return m.getBucketVersioning(ctx, params, optFns...)
}

// healthyS3Mock returns a mock where every detail call succeeds with
// a benign default response. Individual tests override the fields
// they want to fail.
func healthyS3Mock(bucketName string) *mockS3Client {
	return &mockS3Client{
		listBuckets: func(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {
			return &s3.ListBucketsOutput{
				Buckets: []types.Bucket{{Name: aws.String(bucketName)}},
				Owner:   &types.Owner{DisplayName: aws.String("owner"), ID: aws.String("owner-id")},
			}, nil
		},
		getBucketPolicyStatus: func(ctx context.Context, params *s3.GetBucketPolicyStatusInput, optFns ...func(*s3.Options)) (*s3.GetBucketPolicyStatusOutput, error) {
			return &s3.GetBucketPolicyStatusOutput{PolicyStatus: &types.PolicyStatus{IsPublic: aws.Bool(false)}}, nil
		},
		getBucketAcl: func(ctx context.Context, params *s3.GetBucketAclInput, optFns ...func(*s3.Options)) (*s3.GetBucketAclOutput, error) {
			return &s3.GetBucketAclOutput{Grants: []types.Grant{}}, nil
		},
		getPublicAccessBlock: func(ctx context.Context, params *s3.GetPublicAccessBlockInput, optFns ...func(*s3.Options)) (*s3.GetPublicAccessBlockOutput, error) {
			return &s3.GetPublicAccessBlockOutput{PublicAccessBlockConfiguration: &types.PublicAccessBlockConfiguration{}}, nil
		},
		getBucketLocation: func(ctx context.Context, params *s3.GetBucketLocationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error) {
			return &s3.GetBucketLocationOutput{LocationConstraint: types.BucketLocationConstraintEuWest1}, nil
		},
		getBucketLogging: func(ctx context.Context, params *s3.GetBucketLoggingInput, optFns ...func(*s3.Options)) (*s3.GetBucketLoggingOutput, error) {
			return &s3.GetBucketLoggingOutput{LoggingEnabled: &types.LoggingEnabled{TargetBucket: aws.String("log-target")}}, nil
		},
		getBucketEncryption: func(ctx context.Context, params *s3.GetBucketEncryptionInput, optFns ...func(*s3.Options)) (*s3.GetBucketEncryptionOutput, error) {
			return &s3.GetBucketEncryptionOutput{
				ServerSideEncryptionConfiguration: &types.ServerSideEncryptionConfiguration{
					Rules: []types.ServerSideEncryptionRule{{
						ApplyServerSideEncryptionByDefault: &types.ServerSideEncryptionByDefault{
							SSEAlgorithm: types.ServerSideEncryptionAes256,
						},
					}},
				},
			}, nil
		},
		getBucketTagging: func(ctx context.Context, params *s3.GetBucketTaggingInput, optFns ...func(*s3.Options)) (*s3.GetBucketTaggingOutput, error) {
			return &s3.GetBucketTaggingOutput{TagSet: []types.Tag{}}, nil
		},
		getBucketPolicy: func(ctx context.Context, params *s3.GetBucketPolicyInput, optFns ...func(*s3.Options)) (*s3.GetBucketPolicyOutput, error) {
			return &s3.GetBucketPolicyOutput{Policy: aws.String("")}, nil
		},
		getBucketReplication: func(ctx context.Context, params *s3.GetBucketReplicationInput, optFns ...func(*s3.Options)) (*s3.GetBucketReplicationOutput, error) {
			return &s3.GetBucketReplicationOutput{}, nil
		},
		getBucketVersioning: func(ctx context.Context, params *s3.GetBucketVersioningInput, optFns ...func(*s3.Options)) (*s3.GetBucketVersioningOutput, error) {
			return &s3.GetBucketVersioningOutput{Status: types.BucketVersioningStatusEnabled, MFADelete: types.MFADeleteStatusEnabled}, nil
		},
	}
}

// silenceStderr redirects os.Stderr to /dev/null for the duration of
// a test so warning logs emitted by GetBucketDetails don't pollute
// the test output. It restores the original file descriptor on
// cleanup.
func silenceStderr(t *testing.T) {
	t.Helper()
	orig := os.Stderr
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open /dev/null: %v", err)
	}
	os.Stderr = devNull
	t.Cleanup(func() {
		os.Stderr = orig
		devNull.Close()
	})
}

// TestGetBucketDetails_UnknownOnDetailErrors is the regression test
// for T-714. When any of the per-bucket detail calls fails, the
// matching tri-state field on the returned S3Bucket must be nil
// ("unknown") rather than falsely defaulting to false.
func TestGetBucketDetails_UnknownOnDetailErrors(t *testing.T) {
	const bucketName = "test-bucket"
	errBoom := errors.New("boom")

	tests := []struct {
		name       string
		breakMock  func(m *mockS3Client)
		expectNil  func(b S3Bucket) bool
		fieldLabel string
	}{
		{
			name: "GetBucketEncryption failure => HasEncryption is unknown",
			breakMock: func(m *mockS3Client) {
				m.getBucketEncryption = func(ctx context.Context, params *s3.GetBucketEncryptionInput, optFns ...func(*s3.Options)) (*s3.GetBucketEncryptionOutput, error) {
					return nil, errBoom
				}
			},
			expectNil:  func(b S3Bucket) bool { return b.HasEncryption == nil },
			fieldLabel: "HasEncryption",
		},
		{
			name: "GetBucketVersioning failure => Versioning and VersioningMFAEnabled are unknown",
			breakMock: func(m *mockS3Client) {
				m.getBucketVersioning = func(ctx context.Context, params *s3.GetBucketVersioningInput, optFns ...func(*s3.Options)) (*s3.GetBucketVersioningOutput, error) {
					return nil, errBoom
				}
			},
			expectNil:  func(b S3Bucket) bool { return b.Versioning == nil && b.VersioningMFAEnabled == nil },
			fieldLabel: "Versioning/VersioningMFAEnabled",
		},
		{
			name: "GetBucketLogging failure => LoggingEnabled is unknown",
			breakMock: func(m *mockS3Client) {
				m.getBucketLogging = func(ctx context.Context, params *s3.GetBucketLoggingInput, optFns ...func(*s3.Options)) (*s3.GetBucketLoggingOutput, error) {
					return nil, errBoom
				}
			},
			expectNil:  func(b S3Bucket) bool { return b.LoggingEnabled == nil },
			fieldLabel: "LoggingEnabled",
		},
		{
			name: "GetBucketAcl failure => OpenACLs and IsPublic are unknown",
			breakMock: func(m *mockS3Client) {
				m.getBucketAcl = func(ctx context.Context, params *s3.GetBucketAclInput, optFns ...func(*s3.Options)) (*s3.GetBucketAclOutput, error) {
					return nil, errBoom
				}
			},
			// IsPublic must also be unknown because an unknown ACL
			// could flip the answer.
			expectNil:  func(b S3Bucket) bool { return b.OpenACLs == nil && b.IsPublic == nil },
			fieldLabel: "OpenACLs/IsPublic",
		},
		{
			name: "GetBucketPolicyStatus failure => PublicPolicy and IsPublic are unknown",
			breakMock: func(m *mockS3Client) {
				m.getBucketPolicyStatus = func(ctx context.Context, params *s3.GetBucketPolicyStatusInput, optFns ...func(*s3.Options)) (*s3.GetBucketPolicyStatusOutput, error) {
					return nil, errBoom
				}
			},
			expectNil:  func(b S3Bucket) bool { return b.PublicPolicy == nil && b.IsPublic == nil },
			fieldLabel: "PublicPolicy/IsPublic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			silenceStderr(t)
			mock := healthyS3Mock(bucketName)
			tt.breakMock(mock)

			buckets := GetBucketDetails(mock)
			if len(buckets) != 1 {
				t.Fatalf("expected 1 bucket, got %d", len(buckets))
			}
			if !tt.expectNil(buckets[0]) {
				t.Errorf("expected %s to be nil (unknown); got bucket=%+v", tt.fieldLabel, buckets[0])
			}
		})
	}
}

// TestGetBucketDetails_HealthyPathSetsPointers verifies that when all
// detail calls succeed the tri-state pointers are populated (not nil).
func TestGetBucketDetails_HealthyPathSetsPointers(t *testing.T) {
	silenceStderr(t)
	mock := healthyS3Mock("ok-bucket")

	buckets := GetBucketDetails(mock)
	if len(buckets) != 1 {
		t.Fatalf("expected 1 bucket, got %d", len(buckets))
	}
	b := buckets[0]
	if b.HasEncryption == nil || !*b.HasEncryption {
		t.Errorf("HasEncryption expected true, got %v", b.HasEncryption)
	}
	if b.Versioning == nil || !*b.Versioning {
		t.Errorf("Versioning expected true, got %v", b.Versioning)
	}
	if b.VersioningMFAEnabled == nil || !*b.VersioningMFAEnabled {
		t.Errorf("VersioningMFAEnabled expected true, got %v", b.VersioningMFAEnabled)
	}
	if b.LoggingEnabled == nil || !*b.LoggingEnabled {
		t.Errorf("LoggingEnabled expected true, got %v", b.LoggingEnabled)
	}
	if b.LoggingBucket != "log-target" {
		t.Errorf("LoggingBucket expected log-target, got %q", b.LoggingBucket)
	}
	if b.OpenACLs == nil || *b.OpenACLs {
		t.Errorf("OpenACLs expected false, got %v", b.OpenACLs)
	}
	if b.PublicPolicy == nil || *b.PublicPolicy {
		t.Errorf("PublicPolicy expected false, got %v", b.PublicPolicy)
	}
	if b.IsPublic == nil || *b.IsPublic {
		t.Errorf("IsPublic expected false, got %v", b.IsPublic)
	}
}

// TestComputeBucketIsPublic covers the tri-state aggregation logic so
// that unknown policy/ACL inputs don't silently become "not public".
func TestComputeBucketIsPublic(t *testing.T) {
	tests := []struct {
		name     string
		policy   *bool
		acls     *bool
		pab      *types.PublicAccessBlockConfiguration
		expected *bool
	}{
		{
			name:     "both confirmed not public, no PAB => not public",
			policy:   aws.Bool(false),
			acls:     aws.Bool(false),
			pab:      nil,
			expected: aws.Bool(false),
		},
		{
			name:     "confirmed public policy => public",
			policy:   aws.Bool(true),
			acls:     aws.Bool(false),
			pab:      nil,
			expected: aws.Bool(true),
		},
		{
			name:     "confirmed public ACLs => public",
			policy:   aws.Bool(false),
			acls:     aws.Bool(true),
			pab:      nil,
			expected: aws.Bool(true),
		},
		{
			name:     "unknown policy, clean ACLs, no PAB => unknown",
			policy:   nil,
			acls:     aws.Bool(false),
			pab:      nil,
			expected: nil,
		},
		{
			name:     "clean policy, unknown ACLs, no PAB => unknown",
			policy:   aws.Bool(false),
			acls:     nil,
			pab:      nil,
			expected: nil,
		},
		{
			name:   "unknown inputs but PAB fully locks down => not public",
			policy: nil,
			acls:   nil,
			pab: &types.PublicAccessBlockConfiguration{
				BlockPublicAcls:       aws.Bool(true),
				BlockPublicPolicy:     aws.Bool(true),
				IgnorePublicAcls:      aws.Bool(true),
				RestrictPublicBuckets: aws.Bool(true),
			},
			expected: aws.Bool(false),
		},
		{
			name:   "public policy neutralised by PAB => not public",
			policy: aws.Bool(true),
			acls:   aws.Bool(false),
			pab: &types.PublicAccessBlockConfiguration{
				BlockPublicAcls:       aws.Bool(true),
				BlockPublicPolicy:     aws.Bool(true),
				IgnorePublicAcls:      aws.Bool(true),
				RestrictPublicBuckets: aws.Bool(true),
			},
			expected: aws.Bool(false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeBucketIsPublic(tt.policy, tt.acls, tt.pab)
			switch {
			case got == nil && tt.expected == nil:
				// ok
			case got == nil || tt.expected == nil:
				t.Fatalf("got=%v, want=%v", got, tt.expected)
			case *got != *tt.expected:
				t.Fatalf("got=%v, want=%v", *got, *tt.expected)
			}
		})
	}
}
