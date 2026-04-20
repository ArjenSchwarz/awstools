package helpers

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// GetResourcesByStackName returns every resource in the provided stack. It
// walks the ListStackResources paginator so stacks with more than 100
// resources are handled correctly; DescribeStackResources silently caps its
// response at 100 and offers no continuation token (T-784).
func GetResourcesByStackName(stackname *string, svc *cloudformation.Client) []types.StackResource {
	return getResourcesByStackName(stackname, svc)
}

// GetNestedCloudFormationResources retrieves every resource in the provided
// stack plus those of any nested stacks it contains. Each level is paginated
// independently.
func GetNestedCloudFormationResources(stackname *string, svc *cloudformation.Client) []types.StackResource {
	return getNestedCloudFormationResources(stackname, svc)
}

// getResourcesByStackName is the testable implementation behind
// GetResourcesByStackName. It takes the narrow ListStackResourcesAPIClient
// interface so tests can provide a paginated mock without a real SDK client.
//
// ListStackResources returns StackResourceSummary values, which omit the
// StackName field; the helper backfills it from the input so downstream
// consumers (notably cmd/cfnresources.go's buildCfnResource) keep the
// originating stack name for each resource.
func getResourcesByStackName(stackname *string, svc cloudformation.ListStackResourcesAPIClient) []types.StackResource {
	paginator := cloudformation.NewListStackResourcesPaginator(svc, &cloudformation.ListStackResourcesInput{
		StackName: stackname,
	})
	var result []types.StackResource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			panic(err)
		}
		for _, summary := range page.StackResourceSummaries {
			result = append(result, stackResourceFromSummary(summary, stackname))
		}
	}
	return result
}

// getNestedCloudFormationResources is the testable implementation behind
// GetNestedCloudFormationResources. A nested-stack entry with a nil
// PhysicalResourceId is skipped rather than recursed into — passing a nil
// StackName to ListStackResources would otherwise re-describe the parent
// stack, leading to duplicates or infinite recursion.
func getNestedCloudFormationResources(stackname *string, svc cloudformation.ListStackResourcesAPIClient) []types.StackResource {
	resources := getResourcesByStackName(stackname, svc)
	result := make([]types.StackResource, 0, len(resources))
	for _, resource := range resources {
		result = append(result, resource)
		if aws.ToString(resource.ResourceType) != "AWS::CloudFormation::Stack" {
			continue
		}
		if resource.PhysicalResourceId == nil || aws.ToString(resource.PhysicalResourceId) == "" {
			continue
		}
		result = append(result, getNestedCloudFormationResources(resource.PhysicalResourceId, svc)...)
	}
	return result
}

// stackResourceFromSummary converts a ListStackResources summary into the
// StackResource shape the rest of the codebase already consumes. StackName is
// taken from the input because ListStackResources does not repeat it per row.
func stackResourceFromSummary(summary types.StackResourceSummary, stackname *string) types.StackResource {
	return types.StackResource{
		StackName:            stackname,
		LogicalResourceId:    summary.LogicalResourceId,
		PhysicalResourceId:   summary.PhysicalResourceId,
		ResourceType:         summary.ResourceType,
		ResourceStatus:       summary.ResourceStatus,
		Timestamp:            summary.LastUpdatedTimestamp,
		ResourceStatusReason: summary.ResourceStatusReason,
		DriftInformation:     driftInfoFromSummary(summary.DriftInformation),
		ModuleInfo:           summary.ModuleInfo,
	}
}

// driftInfoFromSummary converts the summary-shaped drift information into the
// resource-shaped drift information expected by StackResource. Returns nil
// when there is nothing to convert.
func driftInfoFromSummary(summary *types.StackResourceDriftInformationSummary) *types.StackResourceDriftInformation {
	if summary == nil {
		return nil
	}
	return &types.StackResourceDriftInformation{
		StackResourceDriftStatus: summary.StackResourceDriftStatus,
		LastCheckTimestamp:       summary.LastCheckTimestamp,
	}
}
