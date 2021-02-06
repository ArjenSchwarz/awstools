package helpers

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// GetResourcesByStackName returns a slice of the Stack Resources in the provided stack
func GetResourcesByStackName(stackname *string, config aws.Config) []types.StackResource {
	svc := cloudformation.NewFromConfig(config)

	params := &cloudformation.DescribeStackResourcesInput{
		StackName: stackname,
	}
	resp, err := svc.DescribeStackResources(context.TODO(), params)

	if err != nil {
		panic(err)
	}

	// Pretty-print the response data.
	return resp.StackResources
}

// GetNestedCloudFormationResources retrieves a slice of the Stack Resources that
// are in the provided stack or in one of its children
func GetNestedCloudFormationResources(stackname *string, config aws.Config) []types.StackResource {
	resources := GetResourcesByStackName(stackname, config)
	result := make([]types.StackResource, 0, len(resources))
	for _, resource := range resources {
		result = append(result, resource)
		if aws.ToString(resource.ResourceType) == "AWS::CloudFormation::Stack" {
			for _, subresource := range GetNestedCloudFormationResources(resource.PhysicalResourceId, config) {
				result = append(result, subresource)
			}
		}
	}
	return result
}
