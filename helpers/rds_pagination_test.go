package helpers

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
)

// mockDescribeDBInstancesClient simulates DescribeDBInstances pagination by
// splitting a pre-configured slice of DB instances across multiple pages based
// on the Marker. It satisfies rds.DescribeDBInstancesAPIClient.
type mockDescribeDBInstancesClient struct {
	instances []types.DBInstance
	pageSize  int
	callCount int
}

func (m *mockDescribeDBInstancesClient) DescribeDBInstances(_ context.Context, input *rds.DescribeDBInstancesInput, _ ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error) {
	m.callCount++
	start := 0
	if input.Marker != nil {
		if _, err := fmt.Sscanf(*input.Marker, "%d", &start); err != nil {
			return nil, err
		}
	}
	pageSize := m.pageSize
	if pageSize == 0 {
		pageSize = 100
	}
	end := start + pageSize
	if end > len(m.instances) {
		end = len(m.instances)
	}
	output := &rds.DescribeDBInstancesOutput{
		DBInstances: m.instances[start:end],
	}
	if end < len(m.instances) {
		next := fmt.Sprintf("%d", end)
		output.Marker = &next
	}
	return output, nil
}

// makeDBInstances builds n DB instances with predictable IDs and Name tags.
// The first half have Name tags; the second half only have IDs so the fallback
// path is also exercised.
func makeDBInstances(n int) []types.DBInstance {
	instances := make([]types.DBInstance, n)
	for i := range n {
		id := fmt.Sprintf("db-instance-%04d", i)
		resourceID := fmt.Sprintf("db-ABCDEFGHIJK%05d", i)
		inst := types.DBInstance{
			DBInstanceIdentifier: aws.String(id),
			DbiResourceId:        aws.String(resourceID),
		}
		if i < n/2 {
			inst.TagList = []types.Tag{
				{Key: aws.String("Environment"), Value: aws.String("production")},
				{Key: aws.String("Name"), Value: aws.String(fmt.Sprintf("db-name-%d", i))},
			}
		}
		instances[i] = inst
	}
	return instances
}

// TestAddAllInstanceNames_Pagination verifies that addAllInstanceNames
// retrieves every DB instance across multiple pages. Before the fix it only
// returned the first page of DescribeDBInstances, which silently truncated the
// result for accounts with more than ~100 DB instances.
func TestAddAllInstanceNames_Pagination(t *testing.T) {
	total := 5
	mock := &mockDescribeDBInstancesClient{
		instances: makeDBInstances(total),
		pageSize:  2, // force 3 pages: [0,1], [2,3], [4]
	}

	result := addAllInstanceNames(mock, map[string]string{})

	if len(result) != total {
		t.Fatalf("addAllInstanceNames() returned %d entries, want %d (pagination bug: only first page returned)", len(result), total)
	}
	if mock.callCount < 3 {
		t.Errorf("expected at least 3 DescribeDBInstances calls for %d instances at page size %d, got %d", total, mock.pageSize, mock.callCount)
	}
	for i := range total {
		resourceID := fmt.Sprintf("db-ABCDEFGHIJK%05d", i)
		got, ok := result[resourceID]
		if !ok {
			t.Errorf("addAllInstanceNames() missing entry for %s", resourceID)
			continue
		}
		want := fmt.Sprintf("db-instance-%04d", i)
		if i < total/2 {
			want = fmt.Sprintf("db-name-%d", i)
		}
		if got != want {
			t.Errorf("addAllInstanceNames()[%s] = %q, want %q", resourceID, got, want)
		}
	}
}
