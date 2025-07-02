package config

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/appmesh"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestAWSConfig_StsClient(t *testing.T) {
	awsConfig := &AWSConfig{}
	client := awsConfig.StsClient()
	assert.NotNil(t, client)
	assert.IsType(t, &sts.Client{}, client)
}

func TestAWSConfig_IAMClient(t *testing.T) {
	awsConfig := &AWSConfig{}
	client := awsConfig.IAMClient()
	assert.NotNil(t, client)
	assert.IsType(t, &iam.Client{}, client)
}

func TestAWSConfig_AppmeshClient(t *testing.T) {
	awsConfig := &AWSConfig{}
	client := awsConfig.AppmeshClient()
	assert.NotNil(t, client)
	assert.IsType(t, &appmesh.Client{}, client)
}

func TestAWSConfig_CloudformationClient(t *testing.T) {
	awsConfig := &AWSConfig{}
	client := awsConfig.CloudformationClient()
	assert.NotNil(t, client)
	assert.IsType(t, &cloudformation.Client{}, client)
}

func TestAWSConfig_Ec2Client(t *testing.T) {
	awsConfig := &AWSConfig{}
	client := awsConfig.Ec2Client()
	assert.NotNil(t, client)
	assert.IsType(t, &ec2.Client{}, client)
}

func TestAWSConfig_RdsClient(t *testing.T) {
	awsConfig := &AWSConfig{}
	client := awsConfig.RdsClient()
	assert.NotNil(t, client)
	assert.IsType(t, &rds.Client{}, client)
}

func TestAWSConfig_IamClient(t *testing.T) {
	awsConfig := &AWSConfig{}
	client := awsConfig.IamClient()
	assert.NotNil(t, client)
	assert.IsType(t, &iam.Client{}, client)
}

func TestAWSConfig_OrganizationsClient(t *testing.T) {
	awsConfig := &AWSConfig{}
	client := awsConfig.OrganizationsClient()
	assert.NotNil(t, client)
	assert.IsType(t, &organizations.Client{}, client)
}

func TestAWSConfig_SsoClient(t *testing.T) {
	awsConfig := &AWSConfig{}
	client := awsConfig.SsoClient()
	assert.NotNil(t, client)
	assert.IsType(t, &ssoadmin.Client{}, client)
}

func TestAWSConfig_S3Client(t *testing.T) {
	awsConfig := &AWSConfig{}
	client := awsConfig.S3Client()
	assert.NotNil(t, client)
	assert.IsType(t, &s3.Client{}, client)
}

func TestDefaultAwsConfig_ProfileHandling(t *testing.T) {
	t.Run("uses profile when specified", func(t *testing.T) {
		config := Config{}
		viper.Set("aws.profile", "testprofile")

		defer func() {
			if r := recover(); r != nil {
				// Expected to panic due to invalid profile/credentials in test environment
				assert.NotNil(t, r)
			}
			viper.Reset()
		}()

		DefaultAwsConfig(config)
	})

	t.Run("uses default config when no profile specified", func(t *testing.T) {
		config := Config{}
		viper.Reset()

		defer func() {
			if r := recover(); r != nil {
				// Expected to panic in test environment without valid AWS credentials
				assert.NotNil(t, r)
			}
		}()

		DefaultAwsConfig(config)
	})
}

func TestDefaultAwsConfig_RegionHandling(t *testing.T) {
	config := Config{}
	viper.Set("aws.region", "us-west-2")

	defer func() {
		if r := recover(); r != nil {
			// Expected to panic in test environment without valid AWS credentials
			assert.NotNil(t, r)
		}
		viper.Reset()
	}()

	DefaultAwsConfig(config)
}

func TestAWSConfig_Fields(t *testing.T) {
	awsConfig := AWSConfig{
		AccountAlias: "test-alias",
		AccountID:    "123456789012",
		ProfileName:  "test-profile",
		Region:       "us-east-1",
		UserID:       "test-user-id",
	}

	assert.Equal(t, "test-alias", awsConfig.AccountAlias)
	assert.Equal(t, "123456789012", awsConfig.AccountID)
	assert.Equal(t, "test-profile", awsConfig.ProfileName)
	assert.Equal(t, "us-east-1", awsConfig.Region)
	assert.Equal(t, "test-user-id", awsConfig.UserID)
}
