package helpers

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

// GetResourcesByStackName returns a slice of the Stack Resources in the provided stack
func GetResourcesByStackName(stackname *string, config aws.Config) []cloudformation.StackResource {
	svc := cloudformation.New(config)

	params := &cloudformation.DescribeStackResourcesInput{
		StackName: stackname,
	}
	req := svc.DescribeStackResourcesRequest(params)
	resp, err := req.Send()
	if err != nil {
		panic(err)
	}

	return resp.StackResources
}

// GetNestedCloudFormationResources retrieves a slice of the Stack Resources that
// are in the provided stack or in one of its children
func GetNestedCloudFormationResources(stackname *string, config aws.Config) []cloudformation.StackResource {
	resources := GetResourcesByStackName(stackname, config)
	result := make([]cloudformation.StackResource, 0, len(resources))
	for _, resource := range resources {
		result = append(result, resource)
		if aws.StringValue(resource.ResourceType) == "AWS::CloudFormation::Stack" {
			for _, subresource := range GetNestedCloudFormationResources(resource.PhysicalResourceId, config) {
				result = append(result, subresource)
			}
		}
	}
	return result
}
