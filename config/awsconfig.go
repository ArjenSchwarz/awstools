package config

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	external "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/appmesh"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

//AWSConfig is a holder for AWS Config type information
type AWSConfig struct {
	AccountAlias string
	AccountID    string
	Config       aws.Config
	ProfileName  string
	Region       string
	UserID       string
}

// DefaultAwsConfig loads default AWS Config
func DefaultAwsConfig(config Config) AWSConfig {
	awsConfig := AWSConfig{}
	if config.GetLCString("profile") != "" {
		awsConfig.ProfileName = config.GetLCString("profile")
		cfg, err := external.LoadDefaultConfig(context.TODO(), external.WithSharedConfigProfile(config.GetLCString("profile")))
		if err != nil {
			panic(err)
		}
		awsConfig.Config = cfg
	} else {
		cfg, err := external.LoadDefaultConfig(context.TODO())
		if err != nil {
			panic(err)
		}
		awsConfig.Config = cfg
	}
	if config.GetLCString("region") != "" {
		awsConfig.Config.Region = config.GetLCString("region")
	}
	awsConfig.Region = awsConfig.Config.Region
	awsConfig.setCallerInfo()
	awsConfig.setAlias()
	return awsConfig
}

func (config *AWSConfig) setCallerInfo() {
	c := config.StsClient()
	result, err := c.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		panic(err)
	}
	config.AccountID = *result.Account
	config.UserID = *result.UserId
}

func (config *AWSConfig) setAlias() {
	c := config.IAMClient()
	result, err := c.ListAccountAliases(context.TODO(), &iam.ListAccountAliasesInput{})
	if err != nil || len(result.AccountAliases) == 0 {
		// If the user doesn't have permission to see the aliases or the account has no aliases, continue without
		config.AccountAlias = config.AccountID
	} else {
		config.AccountAlias = result.AccountAliases[0]
	}
}

// StsClient returns an STS Client
func (config *AWSConfig) StsClient() *sts.Client {
	return sts.NewFromConfig(config.Config)
}

// StsClient returns an STS Client
func (config *AWSConfig) IAMClient() *iam.Client {
	return iam.NewFromConfig(config.Config)
}

//AppmeshClient returns an AppMesh Client
func (config *AWSConfig) AppmeshClient() *appmesh.Client {
	return appmesh.NewFromConfig(config.Config)
}

//CloudformationClient returns an cloudformation Client
func (config *AWSConfig) CloudformationClient() *cloudformation.Client {
	return cloudformation.NewFromConfig(config.Config)
}

//Ec2Client returns an ec2 Client
func (config *AWSConfig) Ec2Client() *ec2.Client {
	return ec2.NewFromConfig(config.Config)
}

//RdsClient returns an rds Client
func (config *AWSConfig) RdsClient() *rds.Client {
	return rds.NewFromConfig(config.Config)
}

//IamClient returns an IAM Client
func (config *AWSConfig) IamClient() *iam.Client {
	return iam.NewFromConfig(config.Config)
}

//OrganizationsClient returns an organizations Client
func (config *AWSConfig) OrganizationsClient() *organizations.Client {
	return organizations.NewFromConfig(config.Config)
}

//SsoClient returns an SSO Client
func (config *AWSConfig) SsoClient() *ssoadmin.Client {
	return ssoadmin.NewFromConfig(config.Config)
}

//S3Client returns an S3 Client
func (config *AWSConfig) S3Client() *s3.Client {
	return s3.NewFromConfig(config.Config)
}
