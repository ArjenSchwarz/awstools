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

// TestGetAllVPCRouteTables_MainAssociationSetsDefault verifies that a route
// table whose only association is the implicit/main association (Main == true,
// no SubnetId) is mapped with Default == true and no Subnets. EC2 represents a
// VPC's main route table this way; before the fix the mapper only looked at
// Association.SubnetId, so the main route table came back with Default == false
// and an empty Subnets slice, hiding that it applies to all unassociated
// subnets (T-1270).
func TestGetAllVPCRouteTables_MainAssociationSetsDefault(t *testing.T) {
	mock := &mockDescribeRouteTablesClient{
		routeTables: []types.RouteTable{
			{
				RouteTableId: aws.String("rtb-main0001"),
				VpcId:        aws.String("vpc-00000001"),
				OwnerId:      aws.String("123456789012"),
				Associations: []types.RouteTableAssociation{
					{Main: aws.Bool(true)},
				},
			},
			{
				RouteTableId: aws.String("rtb-explicit1"),
				VpcId:        aws.String("vpc-00000001"),
				OwnerId:      aws.String("123456789012"),
				Associations: []types.RouteTableAssociation{
					{Main: aws.Bool(false), SubnetId: aws.String("subnet-11111111")},
				},
			},
		},
	}

	result := getAllVPCRouteTables(mock)
	if len(result) != 2 {
		t.Fatalf("getAllVPCRouteTables() returned %d route tables, want 2", len(result))
	}

	main := result[0]
	if !main.Default {
		t.Errorf("main route table %q: Default = false, want true (main association not detected)", main.ID)
	}
	if len(main.Subnets) != 0 {
		t.Errorf("main route table %q: Subnets = %v, want empty (main association has no SubnetId)", main.ID, main.Subnets)
	}

	explicit := result[1]
	if explicit.Default {
		t.Errorf("explicit route table %q: Default = true, want false (no main association)", explicit.ID)
	}
	if len(explicit.Subnets) != 1 || explicit.Subnets[0] != "subnet-11111111" {
		t.Errorf("explicit route table %q: Subnets = %v, want [subnet-11111111]", explicit.ID, explicit.Subnets)
	}
}
