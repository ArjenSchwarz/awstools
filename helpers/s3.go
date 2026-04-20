package helpers

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3API defines the subset of the S3 client used by helpers.GetBucketDetails.
// It exists so tests can inject per-call failures when exercising the
// "unknown state" handling in the security report.
type S3API interface {
	ListBuckets(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error)
	GetBucketPolicyStatus(ctx context.Context, params *s3.GetBucketPolicyStatusInput, optFns ...func(*s3.Options)) (*s3.GetBucketPolicyStatusOutput, error)
	GetBucketAcl(ctx context.Context, params *s3.GetBucketAclInput, optFns ...func(*s3.Options)) (*s3.GetBucketAclOutput, error)
	GetPublicAccessBlock(ctx context.Context, params *s3.GetPublicAccessBlockInput, optFns ...func(*s3.Options)) (*s3.GetPublicAccessBlockOutput, error)
	GetBucketLocation(ctx context.Context, params *s3.GetBucketLocationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error)
	GetBucketLogging(ctx context.Context, params *s3.GetBucketLoggingInput, optFns ...func(*s3.Options)) (*s3.GetBucketLoggingOutput, error)
	GetBucketEncryption(ctx context.Context, params *s3.GetBucketEncryptionInput, optFns ...func(*s3.Options)) (*s3.GetBucketEncryptionOutput, error)
	GetBucketTagging(ctx context.Context, params *s3.GetBucketTaggingInput, optFns ...func(*s3.Options)) (*s3.GetBucketTaggingOutput, error)
	GetBucketPolicy(ctx context.Context, params *s3.GetBucketPolicyInput, optFns ...func(*s3.Options)) (*s3.GetBucketPolicyOutput, error)
	GetBucketReplication(ctx context.Context, params *s3.GetBucketReplicationInput, optFns ...func(*s3.Options)) (*s3.GetBucketReplicationOutput, error)
	GetBucketVersioning(ctx context.Context, params *s3.GetBucketVersioningInput, optFns ...func(*s3.Options)) (*s3.GetBucketVersioningOutput, error)
}

// S3Bucket represents detailed information about an S3 bucket including
// security, encryption, and configuration settings.
//
// The tri-state boolean fields (`IsPublic`, `PublicPolicy`, `OpenACLs`,
// `LoggingEnabled`, `HasEncryption`, `Versioning`,
// `VersioningMFAEnabled`) are `*bool`: a nil value means the relevant
// AWS call failed and the state is unknown. Callers must treat nil
// distinctly from `false` — in particular, security-report filters
// should not assume "unknown" means "confirmed off".
type S3Bucket struct {
	Account                        string
	ACLs                           []types.Grant
	EncryptionRules                []types.ServerSideEncryptionRule
	HasEncryption                  *bool
	IsPublic                       *bool
	LoggingBucket                  string
	LoggingEnabled                 *bool
	Name                           string
	OpenACLs                       *bool
	Owner                          string
	Policy                         string
	PublicAccessBlockConfiguration *types.PublicAccessBlockConfiguration
	PublicPolicy                   *bool
	Region                         string
	Replication                    types.ReplicationConfiguration
	Tags                           map[string]string
	Versioning                     *bool
	VersioningMFAEnabled           *bool
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

// warnS3DetailError emits a stderr warning that a per-bucket detail
// call failed. The bucket is still included in the result set with the
// affected field left as "unknown" (nil).
func warnS3DetailError(bucket, call string, err error) {
	fmt.Fprintf(os.Stderr, "Warning: S3 %s for bucket %q failed, marking state unknown: %v\n", call, bucket, err)
}

// GetAllBuckets retrieves all S3 buckets and returns them along with
// the owner name.
func GetAllBuckets(svc S3API) ([]types.Bucket, string) {
	params := &s3.ListBucketsInput{}
	resp, err := svc.ListBuckets(context.TODO(), params)

	if err != nil {
		panic(err)
	}

	return resp.Buckets, resolveOwnerName(resp.Owner)
}

// GetBucketDetails retrieves detailed information for all S3 buckets
// including encryption, versioning, and policies. Detail calls that
// fail leave the relevant tri-state field on S3Bucket as nil
// ("unknown") rather than defaulting to false.
func GetBucketDetails(svc S3API) []S3Bucket {
	buckets, owner := GetAllBuckets(svc)
	result := make([]S3Bucket, 0)
	for _, bucket := range buckets {
		bucketName := aws.ToString(bucket.Name)
		tags := make(map[string]string)

		var policyIsPublic *bool
		statusresp, err := svc.GetBucketPolicyStatus(context.TODO(), &s3.GetBucketPolicyStatusInput{Bucket: bucket.Name})
		switch {
		case err != nil:
			warnS3DetailError(bucketName, "GetBucketPolicyStatus", err)
		case statusresp != nil && statusresp.PolicyStatus != nil:
			policyIsPublic = boolPtr(aws.ToBool(statusresp.PolicyStatus.IsPublic))
		default:
			// Response was empty; S3 returns this when no bucket
			// policy exists, which is a definitive "not public".
			policyIsPublic = boolPtr(false)
		}

		var openacls *bool
		aclresp, err := svc.GetBucketAcl(context.TODO(), &s3.GetBucketAclInput{Bucket: bucket.Name})
		acls := make([]types.Grant, 0)
		if err != nil {
			warnS3DetailError(bucketName, "GetBucketAcl", err)
		} else {
			if aclresp != nil {
				acls = aclresp.Grants
			}
			openacls = boolPtr(hasOpenACLs(acls))
		}

		// PublicAccessBlock overrides other public-making settings.
		publicresp, err := svc.GetPublicAccessBlock(context.TODO(), &s3.GetPublicAccessBlockInput{Bucket: bucket.Name})
		var pab *types.PublicAccessBlockConfiguration
		if err != nil {
			// T-693 covers PAB's own "unknown" handling specifically;
			// here we only record that we could not read it.
			warnS3DetailError(bucketName, "GetPublicAccessBlock", err)
		} else if publicresp != nil && publicresp.PublicAccessBlockConfiguration != nil {
			pab = publicresp.PublicAccessBlockConfiguration
		}

		isPublic := computeBucketIsPublic(policyIsPublic, openacls, pab)

		locationresp, err := svc.GetBucketLocation(context.TODO(), &s3.GetBucketLocationInput{Bucket: bucket.Name})
		var region string
		if err != nil {
			region = "ERROR"
		} else {
			region = normalizeBucketLocation(locationresp.LocationConstraint)
		}
		bucketObject := S3Bucket{
			Name:         bucketName,
			Owner:        owner,
			IsPublic:     isPublic,
			Region:       region,
			ACLs:         acls,
			OpenACLs:     openacls,
			PublicPolicy: policyIsPublic,
		}

		bucketObject.PublicAccessBlockConfiguration = pab

		loggingresp, err := svc.GetBucketLogging(context.TODO(), &s3.GetBucketLoggingInput{Bucket: bucket.Name})
		if err != nil {
			warnS3DetailError(bucketName, "GetBucketLogging", err)
		} else {
			enabled := false
			if loggingresp != nil && loggingresp.LoggingEnabled != nil {
				target := aws.ToString(loggingresp.LoggingEnabled.TargetBucket)
				if target != "" {
					enabled = true
					bucketObject.LoggingBucket = target
				}
			}
			bucketObject.LoggingEnabled = boolPtr(enabled)
		}

		params := &s3.GetBucketEncryptionInput{Bucket: bucket.Name}
		resp, err := svc.GetBucketEncryption(context.TODO(), params)
		if err != nil {
			warnS3DetailError(bucketName, "GetBucketEncryption", err)
		} else {
			if resp != nil && resp.ServerSideEncryptionConfiguration != nil {
				bucketObject.EncryptionRules = resp.ServerSideEncryptionConfiguration.Rules
				bucketObject.HasEncryption = boolPtr(true)
			} else {
				bucketObject.HasEncryption = boolPtr(false)
			}
		}

		tagsResp, err := svc.GetBucketTagging(context.TODO(), &s3.GetBucketTaggingInput{Bucket: bucket.Name})
		if err != nil {
			// Tag retrieval commonly fails (NoSuchTagSet) for buckets
			// with no tags; treat as "no tags" rather than an error.
			// Other failures are logged but not fatal.
			warnS3DetailError(bucketName, "GetBucketTagging", err)
		} else if tagsResp != nil {
			for _, tag := range tagsResp.TagSet {
				tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
			}
		}
		bucketObject.Tags = tags

		policyResp, err := svc.GetBucketPolicy(context.TODO(), &s3.GetBucketPolicyInput{Bucket: bucket.Name})
		if err != nil {
			warnS3DetailError(bucketName, "GetBucketPolicy", err)
		} else if policyResp != nil && policyResp.Policy != nil {
			bucketObject.Policy = *policyResp.Policy
		}

		replicationResp, err := svc.GetBucketReplication(context.TODO(), &s3.GetBucketReplicationInput{Bucket: bucket.Name})
		if err != nil {
			warnS3DetailError(bucketName, "GetBucketReplication", err)
		} else if replicationResp != nil && replicationResp.ReplicationConfiguration != nil {
			bucketObject.Replication = *replicationResp.ReplicationConfiguration
		}

		versioningResp, err := svc.GetBucketVersioning(context.TODO(), &s3.GetBucketVersioningInput{Bucket: bucket.Name})
		if err != nil {
			warnS3DetailError(bucketName, "GetBucketVersioning", err)
		} else if versioningResp != nil {
			bucketObject.Versioning = boolPtr(versioningResp.Status == types.BucketVersioningStatusEnabled)
			bucketObject.VersioningMFAEnabled = boolPtr(versioningResp.MFADelete == types.MFADeleteStatusEnabled)
		}

		result = append(result, bucketObject)

	}
	return result
}

// boolPtr returns a pointer to the given boolean value.
func boolPtr(v bool) *bool {
	return &v
}

// computeBucketIsPublic returns the bucket's overall public state
// based on policy status, ACL state, and Public Access Block
// configuration. The result is nil ("unknown") whenever a contributing
// input is unknown and could flip the answer:
//   - If either the policy status or ACL state is unknown, the only
//     way to claim "not public" is if the Public Access Block both
//     restricts public buckets AND either ignores public ACLs or is
//     irrelevant because the ACLs are known-clean.
//   - A confirmed public input (policy or ACL) always makes the
//     bucket public unless the PAB restricts public buckets AND
//     blocks the corresponding path (IgnorePublicAcls for ACLs,
//     BlockPublicPolicy for the policy).
func computeBucketIsPublic(policyIsPublic, openACLs *bool, pab *types.PublicAccessBlockConfiguration) *bool {
	// Extract the PAB booleans once. A missing PAB means "no block
	// applied" for the purposes of this computation.
	restrictPublicBuckets := false
	ignorePublicAcls := false
	blockPublicPolicy := false
	if pab != nil {
		restrictPublicBuckets = aws.ToBool(pab.RestrictPublicBuckets)
		ignorePublicAcls = aws.ToBool(pab.IgnorePublicAcls)
		blockPublicPolicy = aws.ToBool(pab.BlockPublicPolicy)
	}

	// When PAB fully locks the bucket down we can answer "not public"
	// regardless of policy/ACL state, even if they are unknown.
	if restrictPublicBuckets && ignorePublicAcls && blockPublicPolicy {
		return boolPtr(false)
	}

	// Otherwise, evaluate each source and factor in the PAB.
	policyContribution := policyIsPublic
	if policyContribution != nil && *policyContribution && restrictPublicBuckets && blockPublicPolicy {
		// PAB neutralises a public policy.
		f := false
		policyContribution = &f
	}

	aclContribution := openACLs
	if aclContribution != nil && *aclContribution && restrictPublicBuckets && ignorePublicAcls {
		// PAB neutralises public ACLs.
		f := false
		aclContribution = &f
	}

	// Any confirmed-true makes the bucket public.
	if policyContribution != nil && *policyContribution {
		return boolPtr(true)
	}
	if aclContribution != nil && *aclContribution {
		return boolPtr(true)
	}

	// If either input is unknown, the overall answer is unknown —
	// the unknown side could still be public.
	if policyContribution == nil || aclContribution == nil {
		return nil
	}

	// Both inputs confirmed not public.
	return boolPtr(false)
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
