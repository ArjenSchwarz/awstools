package helpers

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Bucket struct {
	Name                           string
	Account                        string
	Owner                          string
	IsPublic                       bool
	HasEncryption                  bool
	PublicAccessBlockConfiguration types.PublicAccessBlockConfiguration
	EncryptionRules                []types.ServerSideEncryptionRule
	Tags                           map[string]string
	Region                         string
	ACLs                           []types.Grant
	OpenACLs                       bool
	PublicPolicy                   bool
	LoggingEnabled                 bool
	LoggingBucket                  string
	Policy                         string
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
			isPublic = statusresp.PolicyStatus.IsPublic
			policyIsPublic = statusresp.PolicyStatus.IsPublic
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
			if (publicresp.PublicAccessBlockConfiguration.IgnorePublicAcls || !openacls) && publicresp.PublicAccessBlockConfiguration.RestrictPublicBuckets {
				isPublic = false
			}
		}

		locationresp, err := svc.GetBucketLocation(context.TODO(), &s3.GetBucketLocationInput{Bucket: bucket.Name})
		if err != nil {
			panic(err)
		}

		region := "us-east-1"
		if locationresp.LocationConstraint != "" {
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

		result = append(result, bucketObject)

	}
	return result
}
