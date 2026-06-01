package helpers

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// Regression tests for T-1366: getResourcesByStackName previously called
// panic(err) when the ListStackResources paginator's NextPage returned an
// error. Normal API failures (missing/invalid --stack, access denied,
// throttling, deleted nested stack) therefore aborted the CLI with a panic
// instead of a controlled error. The fix propagates the error so the command
// can render it normally. getNestedCloudFormationResources is covered too
// because it recurses through the same helper.

// errListStackResourcesAPIClient implements
// cloudformation.ListStackResourcesAPIClient and always fails, mirroring an
// AWS API error such as ValidationError, AccessDenied, or Throttling.
type errListStackResourcesAPIClient struct {
	err   error
	calls int
}

func (m *errListStackResourcesAPIClient) ListStackResources(_ context.Context, _ *cloudformation.ListStackResourcesInput, _ ...func(*cloudformation.Options)) (*cloudformation.ListStackResourcesOutput, error) {
	m.calls++
	return nil, m.err
}

// TestGetResourcesByStackName_PropagatesError proves the helper returns the
// underlying API error instead of panicking. Before the fix this test would
// panic in NextPage and never reach the assertions.
func TestGetResourcesByStackName_PropagatesError(t *testing.T) {
	apiErr := errors.New("ValidationError: Stack [missing] does not exist")
	mock := &errListStackResourcesAPIClient{err: apiErr}

	name := "missing"
	result, err := getResourcesByStackName(&name, mock)

	if err == nil {
		t.Fatalf("getResourcesByStackName() returned nil error, want the API error to be propagated")
	}
	if !errors.Is(err, apiErr) {
		t.Errorf("getResourcesByStackName() error = %v, want it to wrap %v", err, apiErr)
	}
	if result != nil {
		t.Errorf("getResourcesByStackName() returned %d resources on error, want nil", len(result))
	}
}

// TestGetNestedCloudFormationResources_PropagatesError proves the recursive
// helper also propagates the API error from the first failing page rather
// than panicking.
func TestGetNestedCloudFormationResources_PropagatesError(t *testing.T) {
	apiErr := errors.New("AccessDenied: not authorized to perform cloudformation:ListStackResources")
	mock := &errListStackResourcesAPIClient{err: apiErr}

	name := "parent"
	result, err := getNestedCloudFormationResources(&name, mock)

	if err == nil {
		t.Fatalf("getNestedCloudFormationResources() returned nil error, want the API error to be propagated")
	}
	if !errors.Is(err, apiErr) {
		t.Errorf("getNestedCloudFormationResources() error = %v, want it to wrap %v", err, apiErr)
	}
	if result != nil {
		t.Errorf("getNestedCloudFormationResources() returned %d resources on error, want nil", len(result))
	}
}

// TestGetNestedCloudFormationResources_PropagatesNestedError proves an error
// raised while listing a nested stack (after the parent listed successfully)
// is propagated rather than panicking mid-recursion.
func TestGetNestedCloudFormationResources_PropagatesNestedError(t *testing.T) {
	const parent = "parent-stack"
	const nested = "nested-stack"
	nestedErr := errors.New("ValidationError: Stack [nested-stack] does not exist")

	mock := &failingNestedAPIClient{
		parent:    parent,
		nested:    nested,
		nestedErr: nestedErr,
	}

	name := parent
	result, err := getNestedCloudFormationResources(&name, mock)

	if err == nil {
		t.Fatalf("getNestedCloudFormationResources() returned nil error, want the nested API error to be propagated")
	}
	if !errors.Is(err, nestedErr) {
		t.Errorf("getNestedCloudFormationResources() error = %v, want it to wrap %v", err, nestedErr)
	}
	if result != nil {
		t.Errorf("getNestedCloudFormationResources() returned %d resources on nested error, want nil", len(result))
	}
}

// failingNestedAPIClient serves the parent stack successfully (with a single
// nested-stack marker) but fails when the nested stack is listed.
type failingNestedAPIClient struct {
	parent    string
	nested    string
	nestedErr error
}

func (m *failingNestedAPIClient) ListStackResources(_ context.Context, input *cloudformation.ListStackResourcesInput, _ ...func(*cloudformation.Options)) (*cloudformation.ListStackResourcesOutput, error) {
	switch aws.ToString(input.StackName) {
	case m.parent:
		return &cloudformation.ListStackResourcesOutput{
			StackResourceSummaries: []types.StackResourceSummary{
				{
					LogicalResourceId:  aws.String("NestedChild"),
					PhysicalResourceId: aws.String(m.nested),
					ResourceType:       aws.String("AWS::CloudFormation::Stack"),
					ResourceStatus:     types.ResourceStatusCreateComplete,
				},
			},
		}, nil
	case m.nested:
		return nil, m.nestedErr
	default:
		return &cloudformation.ListStackResourcesOutput{}, nil
	}
}
