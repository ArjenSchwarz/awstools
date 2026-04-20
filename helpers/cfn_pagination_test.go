package helpers

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// Regression tests for T-784: `helpers.GetResourcesByStackName` previously
// issued a single `DescribeStackResources` call. The AWS API caps that
// response at 100 resources and the SDK exposes no paginator for it, so
// stacks with more than 100 resources were silently truncated — both for the
// top-level stack and for any nested stack discovered through
// `GetNestedCloudFormationResources`. The fix is to switch to
// `ListStackResources`, which paginates via `NextToken`.

// mockListStackResourcesAPIClient implements
// cloudformation.ListStackResourcesAPIClient and paginates the provided
// resource summaries using a fixed page size. It records how many pages
// were served per stack so tests can prove the helper walks every page.
type mockListStackResourcesAPIClient struct {
	// pages maps a stack name to the full slice of summaries for that stack.
	pages map[string][]types.StackResourceSummary
	// pageSize controls how many summaries are returned per response. A
	// value of 0 means "return everything in one page".
	pageSize int
	// callsByStack records the number of ListStackResources calls made per
	// stack name so tests can assert multi-page traversal.
	callsByStack map[string]int
}

func newMockListStackResourcesAPIClient(pageSize int) *mockListStackResourcesAPIClient {
	return &mockListStackResourcesAPIClient{
		pages:        map[string][]types.StackResourceSummary{},
		pageSize:     pageSize,
		callsByStack: map[string]int{},
	}
}

func (m *mockListStackResourcesAPIClient) ListStackResources(_ context.Context, input *cloudformation.ListStackResourcesInput, _ ...func(*cloudformation.Options)) (*cloudformation.ListStackResourcesOutput, error) {
	stack := aws.ToString(input.StackName)
	m.callsByStack[stack]++
	summaries, ok := m.pages[stack]
	if !ok {
		return &cloudformation.ListStackResourcesOutput{}, nil
	}
	start := 0
	if input.NextToken != nil {
		if _, err := fmt.Sscanf(*input.NextToken, "%d", &start); err != nil {
			return nil, err
		}
	}
	pageSize := m.pageSize
	if pageSize == 0 {
		pageSize = len(summaries)
	}
	end := start + pageSize
	if end > len(summaries) {
		end = len(summaries)
	}
	out := &cloudformation.ListStackResourcesOutput{
		StackResourceSummaries: summaries[start:end],
	}
	if end < len(summaries) {
		token := fmt.Sprintf("%d", end)
		out.NextToken = &token
	}
	return out, nil
}

// makeStackResourceSummaries generates n synthetic resource summaries with
// predictable logical and physical ids so tests can assert completeness
// across pages.
func makeStackResourceSummaries(n int) []types.StackResourceSummary {
	result := make([]types.StackResourceSummary, n)
	for i := 0; i < n; i++ {
		result[i] = types.StackResourceSummary{
			LogicalResourceId:  aws.String(fmt.Sprintf("Logical%04d", i)),
			PhysicalResourceId: aws.String(fmt.Sprintf("physical-%04d", i)),
			ResourceType:       aws.String("AWS::S3::Bucket"),
			ResourceStatus:     types.ResourceStatusCreateComplete,
		}
	}
	return result
}

// TestGetResourcesByStackName_Pagination proves that the helper walks every
// page of ListStackResources. Before the fix it called DescribeStackResources
// once and returned at most 100 resources regardless of how many the stack
// actually contained.
func TestGetResourcesByStackName_Pagination(t *testing.T) {
	const stack = "test-stack"
	// 250 resources forces three pages at page size 100 — analogous to the
	// real-world 100-resource AWS cap that triggered the bug.
	total := 250
	mock := newMockListStackResourcesAPIClient(100)
	mock.pages[stack] = makeStackResourceSummaries(total)

	name := stack
	result := getResourcesByStackName(&name, mock)

	if len(result) != total {
		t.Fatalf("getResourcesByStackName() returned %d resources, want %d (pagination bug: only first page returned)", len(result), total)
	}
	if mock.callsByStack[stack] != 3 {
		t.Errorf("ListStackResources called %d times for %q, want 3 (one per page)", mock.callsByStack[stack], stack)
	}
	for i, resource := range result {
		wantLogical := fmt.Sprintf("Logical%04d", i)
		wantPhysical := fmt.Sprintf("physical-%04d", i)
		if aws.ToString(resource.LogicalResourceId) != wantLogical {
			t.Errorf("result[%d].LogicalResourceId = %s, want %s", i, aws.ToString(resource.LogicalResourceId), wantLogical)
		}
		if aws.ToString(resource.PhysicalResourceId) != wantPhysical {
			t.Errorf("result[%d].PhysicalResourceId = %s, want %s", i, aws.ToString(resource.PhysicalResourceId), wantPhysical)
		}
		// StackName is populated from the input so callers and the
		// `cfn resources` output retain the originating stack for each
		// resource (StackResourceSummary itself has no StackName field).
		if aws.ToString(resource.StackName) != stack {
			t.Errorf("result[%d].StackName = %q, want %q", i, aws.ToString(resource.StackName), stack)
		}
	}
}

// TestGetNestedCloudFormationResources_PaginatesNested proves that
// GetNestedCloudFormationResources paginates both the parent stack and any
// nested stacks it discovers. Previously the recursion truncated each level
// at 100 resources.
func TestGetNestedCloudFormationResources_PaginatesNested(t *testing.T) {
	const parent = "parent-stack"
	const nested = "nested-stack"

	mock := newMockListStackResourcesAPIClient(100)

	// Build a parent with 150 regular resources plus one nested-stack
	// resource pointing at `nested`.
	parentResources := makeStackResourceSummaries(150)
	parentResources = append(parentResources, types.StackResourceSummary{
		LogicalResourceId:  aws.String("NestedChild"),
		PhysicalResourceId: aws.String(nested),
		ResourceType:       aws.String("AWS::CloudFormation::Stack"),
		ResourceStatus:     types.ResourceStatusCreateComplete,
	})
	mock.pages[parent] = parentResources

	// Build a nested stack with 150 resources of its own.
	mock.pages[nested] = makeStackResourceSummaries(150)

	name := parent
	result := getNestedCloudFormationResources(&name, mock)

	// Expect every resource from both levels: 150 + 1 (nested marker) + 150.
	wantTotal := 150 + 1 + 150
	if len(result) != wantTotal {
		t.Fatalf("getNestedCloudFormationResources() returned %d resources, want %d", len(result), wantTotal)
	}

	// Count resources by stack to make sure each level was fully walked.
	byStack := map[string]int{}
	for _, r := range result {
		byStack[aws.ToString(r.StackName)]++
	}
	if byStack[parent] != 151 {
		t.Errorf("parent stack contributed %d resources, want 151", byStack[parent])
	}
	if byStack[nested] != 150 {
		t.Errorf("nested stack contributed %d resources, want 150", byStack[nested])
	}

	// Parent needs 2 pages (150 items at pageSize 100 = pages of 100 + 50 +
	// the nested marker on the second page), nested needs 2 pages.
	if mock.callsByStack[parent] != 2 {
		t.Errorf("ListStackResources for parent called %d times, want 2", mock.callsByStack[parent])
	}
	if mock.callsByStack[nested] != 2 {
		t.Errorf("ListStackResources for nested called %d times, want 2", mock.callsByStack[nested])
	}
}

// TestGetNestedCloudFormationResources_SkipsNestedWithoutPhysicalID ensures a
// nested stack entry missing its PhysicalResourceId does not trigger a
// recursive call (which would otherwise re-describe the parent because a nil
// StackName defaults to the current stack on the AWS side).
func TestGetNestedCloudFormationResources_SkipsNestedWithoutPhysicalID(t *testing.T) {
	const parent = "parent-stack"
	mock := newMockListStackResourcesAPIClient(0)
	mock.pages[parent] = []types.StackResourceSummary{
		{
			LogicalResourceId: aws.String("OrphanNested"),
			// PhysicalResourceId intentionally nil.
			ResourceType:   aws.String("AWS::CloudFormation::Stack"),
			ResourceStatus: types.ResourceStatusCreateInProgress,
		},
	}

	name := parent
	result := getNestedCloudFormationResources(&name, mock)

	if len(result) != 1 {
		t.Fatalf("expected only the orphan nested entry, got %d resources", len(result))
	}
	if mock.callsByStack[parent] != 1 {
		t.Errorf("parent stack called %d times, want 1 (no recursion for nil PhysicalResourceId)", mock.callsByStack[parent])
	}
	// Ensure no attempt was made to recurse using an empty string key either.
	for stack, calls := range mock.callsByStack {
		if stack != parent && calls > 0 {
			if strings.TrimSpace(stack) == "" {
				t.Errorf("helper recursed with empty stack name")
			}
		}
	}
}
