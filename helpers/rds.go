package helpers

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
)

var rdsSession = rds.New(session.New())

// RDSSession returns a shared RDSSession
func RDSSession() *rds.RDS {
	return rdsSession
}

// GetRDSName returns the name of the provided RDS Resource
func GetRDSName(rdsname *string) string {
	svc := RDSSession()
	params := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: rdsname,
	}
	resp, err := svc.DescribeDBInstances(params)

	if err != nil {
		panic(err)
	}

	for _, instance := range resp.DBInstances {
		return aws.StringValue(instance.DBInstanceIdentifier)
	}
	return ""
}
