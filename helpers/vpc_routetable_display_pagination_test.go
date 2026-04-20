package helpers

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// associationForSubnet returns a route table association linking the given
// subnet ID (for regression-test setup).
func associationForSubnet(subnetID string) types.RouteTableAssociation {
	return types.RouteTableAssociation{SubnetId: aws.String(subnetID)}
}

// TestGetAllRouteTables_Pagination verifies that GetAllRouteTables walks every
// page of DescribeRouteTables. Before T-805 the vpc overview command called
// DescribeRouteTables directly without pagination, so subnets whose route
// table landed on a later page rendered as "No route table". This test drives
// the paginated helper that the command now relies on.
func TestGetAllRouteTables_Pagination(t *testing.T) {
	totalTables := 5
	mock := &mockDescribeRouteTablesClient{
		routeTables: makeRouteTables(totalTables),
		pageSize:    2, // force 3 pages: [0,1], [2,3], [4]
	}

	result := getAllRouteTables(mock)

	if len(result) != totalTables {
		t.Errorf("getAllRouteTables() returned %d route tables, want %d (pagination bug: only first page returned)", len(result), totalTables)
	}

	if mock.callCount < 3 {
		t.Errorf("expected at least 3 DescribeRouteTables calls for %d tables at page size %d, got %d", totalTables, mock.pageSize, mock.callCount)
	}

	for i, rt := range result {
		expectedID := fmt.Sprintf("rtb-%08d", i)
		if rt.RouteTableId == nil || *rt.RouteTableId != expectedID {
			got := ""
			if rt.RouteTableId != nil {
				got = *rt.RouteTableId
			}
			t.Errorf("result[%d].RouteTableId = %q, want %q", i, got, expectedID)
		}
	}
}

// TestGetAllRouteTables_UsedBySubnetLookupAcrossPages simulates the bug scenario
// where a subnet's route table only appears on a later page of the
// DescribeRouteTables response. Before T-805 the vpc overview command pulled
// the raw route table list with a single API call, so GetSubnetRouteTable
// returned nil for any subnet whose route table was not on page 1.
func TestGetAllRouteTables_UsedBySubnetLookupAcrossPages(t *testing.T) {
	// Build 5 route tables; the target (index 4) is on page 3 with page size 2.
	tables := makeRouteTables(5)

	// Make the last route table the explicit association for a specific subnet
	// in the same VPC as that route table.
	targetSubnetID := "subnet-target"
	targetVPCID := *tables[4].VpcId
	tables[4].Associations = append(tables[4].Associations, associationForSubnet(targetSubnetID))

	mock := &mockDescribeRouteTablesClient{
		routeTables: tables,
		pageSize:    2,
	}

	allTables := getAllRouteTables(mock)

	rt := GetSubnetRouteTable(targetSubnetID, targetVPCID, allTables)
	if rt == nil {
		t.Fatalf("GetSubnetRouteTable returned nil for subnet %q; expected the route table on page 3 to be found", targetSubnetID)
	}
	if rt.RouteTableId == nil || *rt.RouteTableId != *tables[4].RouteTableId {
		got := ""
		if rt.RouteTableId != nil {
			got = *rt.RouteTableId
		}
		t.Errorf("GetSubnetRouteTable returned route table %q, want %q", got, *tables[4].RouteTableId)
	}
}
