package helpers

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Bucket represents detailed information about an S3 bucket including security, encryption, and configuration settings
type S3Bucket struct {
	Account                        string
	ACLs                           []types.Grant
	EncryptionRules                []types.ServerSideEncryptionRule
	HasEncryption                  bool
	IsPublic                       bool
	LoggingBucket                  string
	LoggingEnabled                 bool
	Name                           string
	OpenACLs                       bool
	Owner                          string
	Policy                         string
	PublicAccessBlockConfiguration *types.PublicAccessBlockConfiguration
	PublicPolicy                   bool
	Region                         string
	Replication                    types.ReplicationConfiguration
	Tags                           map[string]string
	Versioning                     bool
	VersioningMFAEnabled           bool
}

// normalizeBucketLocation converts the LocationConstraint returned by
// S3's GetBucketLocation API into a canonical AWS region identifier.
// S3 returns an empty constraint for buckets in us-east-1 and the
// legacy value "EU" for buckets originally created in eu-west-1; both
// need to be mapped to their standard region IDs.
func normalizeBucketLocation(constraint types.BucketLocationConstraint) string {
	switch constraint {
	case "":
		return "us-east-1"
	case "EU":
		return "eu-west-1"
	default:
		return string(constraint)
	}
}

// resolveOwnerName safely extracts a display name from an S3 Owner,
// falling back to the Owner ID or empty string when fields are nil.
func resolveOwnerName(owner *types.Owner) string {
	if owner == nil {
		return ""
	}
	if name := aws.ToString(owner.DisplayName); name != "" {
		return name
	}
	return aws.ToString(owner.ID)
}

// GetAllBuckets returns an overview of all buckets
// GetAllBuckets retrieves all S3 buckets and returns them along with the owner name
func GetAllBuckets(svc *s3.Client) ([]types.Bucket, string) {
	params := &s3.ListBucketsInput{}
	resp, err := svc.ListBuckets(context.TODO(), params)

	if err != nil {
		panic(err)
	}

	return resp.Buckets, resolveOwnerName(resp.Owner)
}

// GetBucketDetails retrieves detailed information for all S3 buckets including encryption, versioning, and policies
func GetBucketDetails(svc *s3.Client) []S3Bucket {
	buckets, owner := GetAllBuckets(svc)
	result := make([]S3Bucket, 0)
	for _, bucket := range buckets {
		tags := make(map[string]string)
		statusresp, _ := svc.GetBucketPolicyStatus(context.TODO(), &s3.GetBucketPolicyStatusInput{Bucket: bucket.Name})
		isPublic := false
		policyIsPublic := false
		if statusresp != nil {
			isPublic = aws.ToBool(statusresp.PolicyStatus.IsPublic)
			policyIsPublic = aws.ToBool(statusresp.PolicyStatus.IsPublic)
		}
		aclresp, _ := svc.GetBucketAcl(context.TODO(), &s3.GetBucketAclInput{Bucket: bucket.Name})
		acls := make([]types.Grant, 0)
		if aclresp != nil {
			acls = aclresp.Grants
		}
		openacls := hasOpenACLs(acls)
		if openacls {
			isPublic = true
		}
		// PublicAccessBlock overrides other public making settings. The error
		// is intentionally tolerated: the call routinely fails with
		// NoSuchPublicAccessBlockConfiguration for buckets that simply have
		// no PAB set. When the call fails we leave pabConfig nil so callers
		// can distinguish "unknown" from a real "all false" configuration.
		publicresp, pabErr := svc.GetPublicAccessBlock(context.TODO(), &s3.GetPublicAccessBlockInput{Bucket: bucket.Name})
		var pabConfig *types.PublicAccessBlockConfiguration
		if pabErr == nil && publicresp != nil {
			pabConfig = publicresp.PublicAccessBlockConfiguration
		}
		if pabConfig != nil {
			if (aws.ToBool(pabConfig.IgnorePublicAcls) || !openacls) && aws.ToBool(pabConfig.RestrictPublicBuckets) {
				isPublic = false
			}
		}

		locationresp, err := svc.GetBucketLocation(context.TODO(), &s3.GetBucketLocationInput{Bucket: bucket.Name})
		var region string
		if err != nil {
			region = "ERROR"
		} else {
			region = normalizeBucketLocation(locationresp.LocationConstraint)
		}
		bucketObject := S3Bucket{
			Name:         *bucket.Name,
			Owner:        owner,
			IsPublic:     isPublic,
			Region:       region,
			ACLs:         acls,
			OpenACLs:     openacls,
			PublicPolicy: policyIsPublic,
		}

		bucketObject.PublicAccessBlockConfiguration = pabConfig

		loggingresp, _ := svc.GetBucketLogging(context.TODO(), &s3.GetBucketLoggingInput{Bucket: bucket.Name})
		if loggingresp != nil && loggingresp.LoggingEnabled != nil {
			if aws.ToString(loggingresp.LoggingEnabled.TargetBucket) != "" {
				bucketObject.LoggingEnabled = true
				bucketObject.LoggingBucket = aws.ToString(loggingresp.LoggingEnabled.TargetBucket)
			}
		}

		params := &s3.GetBucketEncryptionInput{Bucket: bucket.Name}
		resp, _ := svc.GetBucketEncryption(context.TODO(), params)

		hasEncryption := false
		if resp != nil {
			hasEncryption = true
			bucketObject.EncryptionRules = resp.ServerSideEncryptionConfiguration.Rules
		}
		bucketObject.HasEncryption = hasEncryption

		tagsResp, _ := svc.GetBucketTagging(context.TODO(), &s3.GetBucketTaggingInput{Bucket: bucket.Name})
		if tagsResp != nil {
			for _, tag := range tagsResp.TagSet {
				tags[*tag.Key] = *tag.Value
			}
		}
		bucketObject.Tags = tags
		policyResp, _ := svc.GetBucketPolicy(context.TODO(), &s3.GetBucketPolicyInput{Bucket: bucket.Name})
		if policyResp != nil && policyResp.Policy != nil {
			bucketObject.Policy = *policyResp.Policy
		}
		replicationResp, _ := svc.GetBucketReplication(context.TODO(), &s3.GetBucketReplicationInput{Bucket: bucket.Name})
		if replicationResp != nil && replicationResp.ReplicationConfiguration != nil {
			bucketObject.Replication = *replicationResp.ReplicationConfiguration
		}
		versioningResp, _ := svc.GetBucketVersioning(context.TODO(), &s3.GetBucketVersioningInput{Bucket: bucket.Name})
		if versioningResp != nil {
			if versioningResp.Status == types.BucketVersioningStatusEnabled {
				bucketObject.Versioning = true
			} else {
				bucketObject.Versioning = false
			}
			if versioningResp.MFADelete == types.MFADeleteStatusEnabled {
				bucketObject.VersioningMFAEnabled = true
			} else {
				bucketObject.VersioningMFAEnabled = false
			}
		}
		result = append(result, bucketObject)

	}
	return result
}

// hasOpenACLs checks whether any grant in the list represents an open (public) ACL.
// A grant is considered open if it targets a non-canonical-user grantee whose URI
// is not the S3 LogDelivery group. Grants with a nil URI (e.g. email-based grants)
// are also treated as open because they indicate a non-owner grant.
func hasOpenACLs(acls []types.Grant) bool {
	for _, acl := range acls {
		if acl.Grantee == nil || acl.Grantee.Type == types.TypeCanonicalUser {
			continue
		}
		uri := aws.ToString(acl.Grantee.URI)
		if uri != "http://acs.amazonaws.com/groups/s3/LogDelivery" {
			return true
		}
	}
	return false
}

// GetReplicationStrings returns a slice of string representations of the bucket's replication rules
func (bucket *S3Bucket) GetReplicationStrings() []string {
	ruleslist := make([]string, 0)
	for _, rule := range bucket.Replication.Rules {
		var filter string
		if rule.Filter != nil {
			// Check if it's an And filter (complex)
			switch {
			case rule.Filter.And != nil:
				prefixPortion := ""
				// Prefix is optional
				if aws.ToString(rule.Filter.And.Prefix) != "" {
					prefixPortion = fmt.Sprintf("Prefix: %s, and ", aws.ToString(rule.Filter.And.Prefix))
				}
				tagsSlice := make([]string, 0)
				for _, replicationtag := range rule.Filter.And.Tags {
					tagsSlice = append(tagsSlice, fmt.Sprintf("Tag %s:%s", aws.ToString(replicationtag.Key), aws.ToString(replicationtag.Value)))
				}
				filter = fmt.Sprintf("%s%s", prefixPortion, strings.Join(tagsSlice, " and "))
			case rule.Filter.Prefix != nil:
				// Simple prefix filter
				if aws.ToString(rule.Filter.Prefix) == "" {
					filter = "Entire bucket"
				} else {
					filter = fmt.Sprintf("Prefix: %s", aws.ToString(rule.Filter.Prefix))
				}
			case rule.Filter.Tag != nil:
				// Simple tag filter
				filter = fmt.Sprintf("Tag %s:%s", aws.ToString(rule.Filter.Tag.Key), aws.ToString(rule.Filter.Tag.Value))
			default:
				filter = "Unknown filter"
			}
		} else {
			// No filter specified
			filter = "Entire bucket"
		}
		ruleslist = append(ruleslist, fmt.Sprintf("%s => %s", filter, aws.ToString(rule.Destination.Bucket)))
	}
	return ruleslist
}
