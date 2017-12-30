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

func GetRDSBySecurityGroup(securityGroupId *string) []*rds.DBInstance {
	var result []*rds.DBInstance
	svc := RDSSession()
	resp, err := svc.DescribeDBInstances(&rds.DescribeDBInstancesInput{})
	if err != nil {
		panic(err)
	}
	for _, instance := range resp.DBInstances {
		for _, vpcsecgroup := range instance.VpcSecurityGroups {
			if aws.StringValue(vpcsecgroup.Status) == "active" && aws.StringValue(vpcsecgroup.VpcSecurityGroupId) == aws.StringValue(securityGroupId) {
				result = append(result, instance)
			}
		}
	}
	return result
}

func GetAllRDS() []*rds.DBInstance {
	svc := RDSSession()
	resp, err := svc.DescribeDBInstances(&rds.DescribeDBInstancesInput{})
	if err != nil {
		panic(err)
	}
	return resp.DBInstances
}
