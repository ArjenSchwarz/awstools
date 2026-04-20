package helpers

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// mockDescribeRouteTablesClient simulates DescribeRouteTables pagination by
// splitting a pre-configured slice of route tables across multiple pages
// based on the NextToken. It satisfies ec2.DescribeRouteTablesAPIClient.
type mockDescribeRouteTablesClient struct {
	routeTables []types.RouteTable
	pageSize    int
	callCount   int
}

func (m *mockDescribeRouteTablesClient) DescribeRouteTables(_ context.Context, input *ec2.DescribeRouteTablesInput, _ ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error) {
	m.callCount++
	start := 0
	if input.NextToken != nil {
		if _, err := fmt.Sscanf(*input.NextToken, "%d", &start); err != nil {
			return nil, err
		}
	}
	pageSize := m.pageSize
	if pageSize == 0 {
		pageSize = 100
	}
	end := start + pageSize
	if end > len(m.routeTables) {
		end = len(m.routeTables)
	}
	output := &ec2.DescribeRouteTablesOutput{
		RouteTables: m.routeTables[start:end],
	}
	if end < len(m.routeTables) {
		next := fmt.Sprintf("%d", end)
		output.NextToken = &next
	}
	return output, nil
}

// makeRouteTables builds n dummy route tables with unique IDs.
func makeRouteTables(n int) []types.RouteTable {
	tables := make([]types.RouteTable, n)
	for i := range n {
		id := fmt.Sprintf("rtb-%08d", i)
		vpc := fmt.Sprintf("vpc-%08d", i)
		tables[i] = types.RouteTable{
			RouteTableId: aws.String(id),
			VpcId:        aws.String(vpc),
			OwnerId:      aws.String("123456789012"),
		}
	}
	return tables
}

// TestGetAllVPCRouteTables_Pagination verifies that GetAllVPCRouteTables
// retrieves every route table across multiple pages. Before the fix it
// only returned the contents of the first page.
func TestGetAllVPCRouteTables_Pagination(t *testing.T) {
	totalTables := 5
	mock := &mockDescribeRouteTablesClient{
		routeTables: makeRouteTables(totalTables),
		pageSize:    2, // force 3 pages: [0,1], [2,3], [4]
	}

	result := getAllVPCRouteTables(mock)

	if len(result) != totalTables {
		t.Errorf("getAllVPCRouteTables() returned %d route tables, want %d (pagination bug: only first page returned)", len(result), totalTables)
	}

	if mock.callCount < 3 {
		t.Errorf("expected at least 3 DescribeRouteTables calls for %d tables at page size %d, got %d", totalTables, mock.pageSize, mock.callCount)
	}

	for i, rt := range result {
		expectedID := fmt.Sprintf("rtb-%08d", i)
		if rt.ID != expectedID {
			t.Errorf("result[%d].ID = %q, want %q", i, rt.ID, expectedID)
		}
	}
}
