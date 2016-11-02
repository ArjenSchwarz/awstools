package helpers

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

var cfnSession = cloudformation.New(session.New())

// CfnSession returns a shared CfnSession
func CfnSession() *cloudformation.CloudFormation {
	return cfnSession
}

// GetResourcesByStackName returns a slice of the Stack Resources in the provided stack
func GetResourcesByStackName(stackname *string) []*cloudformation.StackResource {
	svc := CfnSession()

	params := &cloudformation.DescribeStackResourcesInput{
		StackName: stackname,
	}
	resp, err := svc.DescribeStackResources(params)

	if err != nil {
		panic(err)
	}

	// Pretty-print the response data.
	return resp.StackResources
}

// GetNestedCloudFormationResources retrieves a slice of the Stack Resources that
// are in the provided stack or in one of its children
func GetNestedCloudFormationResources(stackname *string) []*cloudformation.StackResource {
	resources := GetResourcesByStackName(stackname)
	result := make([]*cloudformation.StackResource, 0, len(resources))
	for _, resource := range resources {
		result = append(result, resource)
		if aws.StringValue(resource.ResourceType) == "AWS::CloudFormation::Stack" {
			for _, subresource := range GetNestedCloudFormationResources(resource.PhysicalResourceId) {
				result = append(result, subresource)
			}
		}
	}
	return result
}
