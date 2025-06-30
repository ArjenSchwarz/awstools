package helpers

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
)

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

// GetAllRdsResourceNames gets a list of all names for RDS objects
// TODO: clusters, subnet groups, parameter groups, option groups
func GetAllRdsResourceNames(svc *rds.Client) map[string]string {
	result := make(map[string]string)
	result = addAllInstanceNames(svc, result)
	return result
}

func addAllInstanceNames(svc *rds.Client, result map[string]string) map[string]string {
	resp, err := svc.DescribeDBInstances(context.TODO(), &rds.DescribeDBInstancesInput{})
	if err != nil {
		panic(err)
	}
	for _, dbinstance := range resp.DBInstances {
		result[*dbinstance.DbiResourceId] = *dbinstance.DBInstanceIdentifier
		if dbinstance.TagList != nil {
			for _, tag := range dbinstance.TagList {
				if *tag.Key == "Name" {
					result[*dbinstance.DbiResourceId] = *tag.Value
					break
				}
			}
		}
	}
	return result
}
