package helpers

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
)

// RDSSession returns a shared RDSSession
func RDSSession(config aws.Config) *rds.Client {
	return rds.NewFromConfig(config)
}

// GetRDSName returns the name of the provided RDS Resource
func GetRDSName(rdsname *string, svc *rds.Client) string {
	params := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: rdsname,
	}
	resp, err := svc.DescribeDBInstances(context.TODO(), params)

	if err != nil {
		panic(err)
	}

	for _, instance := range resp.DBInstances {
		return aws.ToString(instance.DBInstanceIdentifier)
	}
	return ""
}
