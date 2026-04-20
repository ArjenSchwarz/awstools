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

// addAllInstanceNames adds every DB instance's display name to the result map.
// AWS's DescribeDBInstances API paginates at 100 instances per page by default,
// so this helper walks NewDescribeDBInstancesPaginator until every page is
// consumed. Accepting the narrow rds.DescribeDBInstancesAPIClient interface
// lets the pagination logic be unit tested without a real *rds.Client.
func addAllInstanceNames(svc rds.DescribeDBInstancesAPIClient, result map[string]string) map[string]string {
	paginator := rds.NewDescribeDBInstancesPaginator(svc, &rds.DescribeDBInstancesInput{})
	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(context.TODO())
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
	}
	return result
}
