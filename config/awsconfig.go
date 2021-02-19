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
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

//AWSConfig is a holder for AWS Config type information
type AWSConfig struct {
	Config aws.Config
}

// DefaultAwsConfig loads default AWS Config
func DefaultAwsConfig(config Config) AWSConfig {
	awsConfig := AWSConfig{}
	if *config.Profile != "" {
		cfg, err := external.LoadDefaultConfig(context.TODO(), external.WithSharedConfigProfile(*config.Profile))
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
	if *config.Region != "" {
		awsConfig.Config.Region = *config.Region
	}
	return awsConfig
}

// StsClient returns an STS Client
func (config *AWSConfig) StsClient() *sts.Client {
	return sts.NewFromConfig(config.Config)
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
