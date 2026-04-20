package cmd

import (
	"fmt"
	"strings"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/spf13/cobra"
)

// ssoDanglingCmd represents the sso Dangling command
var s3listCmd = &cobra.Command{
	Use:   "list",
	Short: "An overview of S3 buckets",
	Long:  `Lists all S3 buckets.`,
	Run:   s3List,
}

var publicBucketsOnly bool
var unencryptedBucketsOnly bool
var includeTags string

// s3StateUnknown is the label used in the bucket listing whenever a
// per-bucket detail call failed and the true state could not be
// determined (see T-714).
const s3StateUnknown = "Unknown"

func init() {
	s3Cmd.AddCommand(s3listCmd)
	s3listCmd.Flags().BoolVar(&publicBucketsOnly, "public-only", false, "Only show public buckets")
	s3listCmd.Flags().BoolVar(&unencryptedBucketsOnly, "unencrypted-only", false, "Only show unencrypted buckets")
	s3listCmd.Flags().StringVarP(&includeTags, "include-tags", "t", "", "Optional tag values to show in output")
}

func s3List(_ *cobra.Command, _ []string) {
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "S3 Buckets"
	buckets := helpers.GetBucketDetails(awsConfig.S3Client())
	keys := []string{"Name", "AccountID", "AccountName", "Region", "Is Private", "Policy is locked down", "ACLs are locked down", "Public Access Block", "Logs to", "Encryption", "Replication", "Versioning", "Versioning MFA delete"}
	if includeTags != "" {
		taglist := strings.SplitSeq(includeTags, ",")
		for tag := range taglist {
			keys = append(keys, fmt.Sprintf("Tag: %s", tag))
		}
	}
	if settings.IsVerbose() {
		keys = append(keys, "Policy")
	}
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = resultTitle
	for _, bucket := range buckets {
		// --public-only excludes buckets that are confirmed private.
		// Buckets with an unknown public state are kept so the user
		// still sees them (and the rendered "Unknown" makes the state
		// visible); exclude only those we know to be private.
		if publicBucketsOnly && bucket.IsPublic != nil && !*bucket.IsPublic {
			continue
		}
		// --unencrypted-only excludes buckets that are confirmed
		// encrypted. Buckets whose encryption state could not be
		// determined are also excluded so we do not falsely flag them
		// as unencrypted.
		if unencryptedBucketsOnly && (bucket.HasEncryption == nil || *bucket.HasEncryption) {
			continue
		}
		content := make(map[string]any)
		content["Name"] = bucket.Name
		content["AccountID"] = awsConfig.AccountID
		content["AccountName"] = getName(awsConfig.AccountID)
		content["Region"] = bucket.Region
		content["Owner"] = bucket.Owner
		content["Is Private"] = negatedTriState(bucket.IsPublic)
		content["Policy is locked down"] = negatedTriState(bucket.PublicPolicy)
		content["ACLs are locked down"] = negatedTriState(bucket.OpenACLs)

		content["Public Access Block"] = parsePublicAccessBlock(bucket.PublicAccessBlockConfiguration)
		switch {
		case bucket.HasEncryption == nil:
			content["Encryption"] = s3StateUnknown
		case *bucket.HasEncryption:
			content["Encryption"] = s3EncryptionToString(bucket.EncryptionRules)
		default:
			content["Encryption"] = false
		}
		switch {
		case bucket.LoggingEnabled == nil:
			content["Logs to"] = s3StateUnknown
		case *bucket.LoggingEnabled:
			content["Logs to"] = bucket.LoggingBucket
		default:
			content["Logs to"] = false
		}
		if includeTags != "" {
			taglist := strings.SplitSeq(includeTags, ",")
			for tag := range taglist {
				content[fmt.Sprintf("Tag: %s", tag)] = bucket.Tags[tag]
			}
		}
		if len(bucket.Replication.Rules) > 0 {
			content["Replication"] = bucket.GetReplicationStrings()
		} else {
			content["Replication"] = false
		}
		if settings.IsVerbose() {
			content["Policy"] = bucket.Policy
		}
		content["Versioning"] = triState(bucket.Versioning)
		content["Versioning MFA delete"] = triState(bucket.VersioningMFAEnabled)
		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	output.Write()
}

// triState renders an optional boolean for the bucket listing output.
// A nil value means the underlying AWS call failed and is shown as
// s3StateUnknown so the report does not imply a confirmed answer.
func triState(v *bool) any {
	if v == nil {
		return s3StateUnknown
	}
	return *v
}

// negatedTriState renders the negation of an optional boolean. Nil
// stays s3StateUnknown — inverting an unknown answer is still
// unknown.
func negatedTriState(v *bool) any {
	if v == nil {
		return s3StateUnknown
	}
	return !*v
}

func s3EncryptionToString(rules []types.ServerSideEncryptionRule) string {
	result := ""
	for _, rule := range rules {
		return string(rule.ApplyServerSideEncryptionByDefault.SSEAlgorithm)
	}
	return result
}

// parsePublicAccessBlock renders a PublicAccessBlockConfiguration as a human
// readable string. A nil config means the underlying GetPublicAccessBlock call
// failed or returned no configuration (e.g. no PAB set, or access denied) and
// is reported as "Unknown" so it is not confused with a bucket that has all
// four flags explicitly set to false.
func parsePublicAccessBlock(config *types.PublicAccessBlockConfiguration) string {
	if config == nil {
		return "Unknown"
	}
	if aws.ToBool(config.BlockPublicAcls) && aws.ToBool(config.BlockPublicPolicy) && aws.ToBool(config.IgnorePublicAcls) && aws.ToBool(config.RestrictPublicBuckets) {
		return "All true"
	}
	if !aws.ToBool(config.BlockPublicAcls) && !aws.ToBool(config.BlockPublicPolicy) && !aws.ToBool(config.IgnorePublicAcls) && !aws.ToBool(config.RestrictPublicBuckets) {
		return "All false"
	}
	return fmt.Sprintf("Block Public ACLs: %v, Block Public Policy: %v, Ignore Public ACLs: %v, Restrict Public Buckets: %v",
		aws.ToBool(config.BlockPublicAcls), aws.ToBool(config.BlockPublicPolicy), aws.ToBool(config.IgnorePublicAcls), aws.ToBool(config.RestrictPublicBuckets))
}
