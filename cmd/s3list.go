package cmd

import (
	"fmt"
	"strings"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	format "github.com/ArjenSchwarz/go-output"
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

func init() {
	s3Cmd.AddCommand(s3listCmd)
	s3listCmd.Flags().BoolVar(&publicBucketsOnly, "public-only", false, "Only show public buckets")
	s3listCmd.Flags().BoolVar(&unencryptedBucketsOnly, "unencrypted-only", false, "Only show unencrypted buckets")
	s3listCmd.Flags().StringVarP(&includeTags, "include-tags", "t", "", "Optional tag values to show in output")
}

func s3List(cmd *cobra.Command, args []string) {
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "S3 Buckets"
	buckets := helpers.GetBucketDetails(awsConfig.S3Client())
	keys := []string{"Name", "AccountID", "AccountName", "Region", "Is Private", "Policy is locked down", "ACLs are locked down", "Public Access Block", "Logs to", "Encryption", "Replication"}
	if includeTags != "" {
		taglist := strings.Split(includeTags, ",")
		for _, tag := range taglist {
			keys = append(keys, fmt.Sprintf("Tag: %s", tag))
		}
	}
	if settings.IsVerbose() {
		keys = append(keys, "Policy")
	}
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = resultTitle
	for _, bucket := range buckets {
		if publicBucketsOnly && !bucket.IsPublic {
			continue
		}
		if unencryptedBucketsOnly && bucket.HasEncryption {
			continue
		}
		content := make(map[string]interface{})
		content["Name"] = bucket.Name
		content["AccountID"] = awsConfig.AccountID
		content["AccountName"] = getName(awsConfig.AccountID)
		content["Region"] = bucket.Region
		content["Owner"] = bucket.Owner
		content["Is Private"] = !bucket.IsPublic
		content["Policy is locked down"] = !bucket.PublicPolicy
		content["ACLs are locked down"] = !bucket.OpenACLs

		content["Public Access Block"] = parsePublicAccessBlock(bucket.PublicAccessBlockConfiguration)
		if bucket.HasEncryption {
			content["Encryption"] = s3EncryptionToString(bucket.EncryptionRules)
		} else {
			content["Encryption"] = false
		}
		if bucket.LoggingEnabled {
			content["Logs to"] = bucket.LoggingBucket
		} else {
			content["Logs to"] = false
		}
		if includeTags != "" {
			taglist := strings.Split(includeTags, ",")
			for _, tag := range taglist {
				content[fmt.Sprintf("Tag: %s", tag)] = bucket.Tags[tag]
			}
		}
		if len(bucket.Replication.Rules) > 0 {
			content["Replication"] = strings.Join(bucket.GetReplicationStrings(), settings.GetSeparator())
		} else {
			content["Replication"] = false
		}
		if settings.IsVerbose() {
			content["Policy"] = bucket.Policy
		}
		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	output.Write()
}

func s3EncryptionToString(rules []types.ServerSideEncryptionRule) string {
	result := ""
	for _, rule := range rules {
		return string(rule.ApplyServerSideEncryptionByDefault.SSEAlgorithm)
	}
	return result
}

func parsePublicAccessBlock(config types.PublicAccessBlockConfiguration) string {
	if config.BlockPublicAcls && config.BlockPublicPolicy && config.IgnorePublicAcls && config.RestrictPublicBuckets {
		return "All true"
	}
	if !config.BlockPublicAcls && !config.BlockPublicPolicy && !config.IgnorePublicAcls && !config.RestrictPublicBuckets {
		return "All false"
	}
	return fmt.Sprintf("Block Public ACLs: %v, Block Public Policy: %v, Ignore Public ACLs: %v, Restrict Public Buckets: %v",
		config.BlockPublicAcls, config.BlockPublicPolicy, config.IgnorePublicAcls, config.RestrictPublicBuckets)
}
