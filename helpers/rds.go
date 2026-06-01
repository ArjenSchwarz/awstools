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
			// DbiResourceId is the map key; without it there is nothing to
			// store, so skip instances that lack one rather than panic.
			resourceID := aws.ToString(dbinstance.DbiResourceId)
			if resourceID == "" {
				continue
			}
			// Fall back to the instance identifier (which may itself be
			// empty) and prefer a Name tag when one is present.
			result[resourceID] = aws.ToString(dbinstance.DBInstanceIdentifier)
			for _, tag := range dbinstance.TagList {
				if aws.ToString(tag.Key) == "Name" && tag.Value != nil {
					result[resourceID] = aws.ToString(tag.Value)
					break
				}
			}
		}
	}
	return result
}
