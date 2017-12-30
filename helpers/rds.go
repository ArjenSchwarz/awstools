package helpers

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
)

// GetRDSName returns the name of the provided RDS Resource
func GetRDSName(rdsname *string, config aws.Config) string {
	svc := rds.New(config)
	params := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: rdsname,
	}
	req := svc.DescribeDBInstancesRequest(params)
	resp, err := req.Send()

	if err != nil {
		panic(err)
	}

	for _, instance := range resp.DBInstances {
		return aws.StringValue(instance.DBInstanceIdentifier)
	}
	return ""
}
