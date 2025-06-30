package helpers

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

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
	PublicAccessBlockConfiguration types.PublicAccessBlockConfiguration
	PublicPolicy                   bool
	Region                         string
	Replication                    types.ReplicationConfiguration
	Tags                           map[string]string
	Versioning                     bool
	VersioningMFAEnabled           bool
}

// GetAllBuckets returns an overview of all buckets
func GetAllBuckets(svc *s3.Client) ([]types.Bucket, string) {
	params := &s3.ListBucketsInput{}
	resp, err := svc.ListBuckets(context.TODO(), params)

	if err != nil {
		panic(err)
	}

	// Pretty-print the response data.
	return resp.Buckets, *resp.Owner.DisplayName
}

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
		openacls := false
		for _, acl := range acls {
			if acl.Grantee != nil && acl.Grantee.Type != types.TypeCanonicalUser &&
				*acl.Grantee.URI != "http://acs.amazonaws.com/groups/s3/LogDelivery" {
				openacls = true
				isPublic = true
			}
		}
		// PublicAccessBlock overrides other public making settings
		publicresp, _ := svc.GetPublicAccessBlock(context.TODO(), &s3.GetPublicAccessBlockInput{Bucket: bucket.Name})
		if publicresp != nil {
			if (aws.ToBool(publicresp.PublicAccessBlockConfiguration.IgnorePublicAcls) || !openacls) && aws.ToBool(publicresp.PublicAccessBlockConfiguration.RestrictPublicBuckets) {
				isPublic = false
			}
		}

		locationresp, err := svc.GetBucketLocation(context.TODO(), &s3.GetBucketLocationInput{Bucket: bucket.Name})
		region := "us-east-1"
		if err != nil {
			region = "ERROR"
		} else if locationresp.LocationConstraint != "" {
			region = string(locationresp.LocationConstraint)
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

		if publicresp != nil {
			bucketObject.PublicAccessBlockConfiguration = *publicresp.PublicAccessBlockConfiguration
		}

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

func (bucket *S3Bucket) GetReplicationStrings() []string {
	ruleslist := make([]string, 0)
	for _, rule := range bucket.Replication.Rules {
		var filter string
		if rule.Filter != nil {
			// Check if it's an And filter (complex)
			if rule.Filter.And != nil {
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
			} else if rule.Filter.Prefix != nil {
				// Simple prefix filter
				if aws.ToString(rule.Filter.Prefix) == "" {
					filter = "Entire bucket"
				} else {
					filter = fmt.Sprintf("Prefix: %s", aws.ToString(rule.Filter.Prefix))
				}
			} else if rule.Filter.Tag != nil {
				// Simple tag filter
				filter = fmt.Sprintf("Tag %s:%s", aws.ToString(rule.Filter.Tag.Key), aws.ToString(rule.Filter.Tag.Value))
			} else {
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
