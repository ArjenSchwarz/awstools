package config

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
)

// Config holds the global configuration settings
type Config struct {
	Verbose      *bool
	OutputFile   *string
	OutputFormat *string
}

// DefaultAwsConfig loads default AWS Config
func DefaultAwsConfig() aws.Config {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		panic(err)
	}
	return cfg
}
