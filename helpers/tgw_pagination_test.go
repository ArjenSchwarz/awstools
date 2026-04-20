package helpers

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// mockTGWPaginationClient implements the minimal EC2 APIClient interfaces used
// by the TGW inventory helpers. It paginates each underlying slice with a
// configurable page size so tests can force multi-page responses.
type mockTGWPaginationClient struct {
	transitGateways      []types.TransitGateway
	routeTables          []types.TransitGatewayRouteTable
	associations         []types.TransitGatewayRouteTableAssociation
	activeRoutes         []types.TransitGatewayRoute
	blackholeRoutes      []types.TransitGatewayRoute
	pageSize             int
	describeTGWCalls     int
	describeRTCalls      int
	associationCalls     int
	searchRoutesCalls    int
	searchRoutesFilters  [][]types.Filter
}

// DescribeTransitGateways paginates through the transit gateways slice. The
// NextToken is the offset (as a decimal string) of the next item.
func (m *mockTGWPaginationClient) DescribeTransitGateways(_ context.Context, input *ec2.DescribeTransitGatewaysInput, _ ...func(*ec2.Options)) (*ec2.DescribeTransitGatewaysOutput, error) {
	m.describeTGWCalls++
	start := 0
	if input.NextToken != nil {
		if _, err := fmt.Sscanf(*input.NextToken, "%d", &start); err != nil {
			return nil, err
		}
	}
	pageSize := m.pageSize
	if pageSize == 0 {
		pageSize = len(m.transitGateways)
	}
	end := start + pageSize
	if end > len(m.transitGateways) {
		end = len(m.transitGateways)
	}
	out := &ec2.DescribeTransitGatewaysOutput{
		TransitGateways: m.transitGateways[start:end],
	}
	if end < len(m.transitGateways) {
		tok := fmt.Sprintf("%d", end)
		out.NextToken = &tok
	}
	return out, nil
}

// DescribeTransitGatewayRouteTables paginates through the route tables slice.
func (m *mockTGWPaginationClient) DescribeTransitGatewayRouteTables(_ context.Context, input *ec2.DescribeTransitGatewayRouteTablesInput, _ ...func(*ec2.Options)) (*ec2.DescribeTransitGatewayRouteTablesOutput, error) {
	m.describeRTCalls++
	start := 0
	if input.NextToken != nil {
		if _, err := fmt.Sscanf(*input.NextToken, "%d", &start); err != nil {
			return nil, err
		}
	}
	pageSize := m.pageSize
	if pageSize == 0 {
		pageSize = len(m.routeTables)
	}
	end := start + pageSize
	if end > len(m.routeTables) {
		end = len(m.routeTables)
	}
	out := &ec2.DescribeTransitGatewayRouteTablesOutput{
		TransitGatewayRouteTables: m.routeTables[start:end],
	}
	if end < len(m.routeTables) {
		tok := fmt.Sprintf("%d", end)
		out.NextToken = &tok
	}
	return out, nil
}

// GetTransitGatewayRouteTableAssociations paginates through the associations slice.
func (m *mockTGWPaginationClient) GetTransitGatewayRouteTableAssociations(_ context.Context, input *ec2.GetTransitGatewayRouteTableAssociationsInput, _ ...func(*ec2.Options)) (*ec2.GetTransitGatewayRouteTableAssociationsOutput, error) {
	m.associationCalls++
	start := 0
	if input.NextToken != nil {
		if _, err := fmt.Sscanf(*input.NextToken, "%d", &start); err != nil {
			return nil, err
		}
	}
	pageSize := m.pageSize
	if pageSize == 0 {
		pageSize = len(m.associations)
	}
	end := start + pageSize
	if end > len(m.associations) {
		end = len(m.associations)
	}
	out := &ec2.GetTransitGatewayRouteTableAssociationsOutput{
		Associations: m.associations[start:end],
	}
	if end < len(m.associations) {
		tok := fmt.Sprintf("%d", end)
		out.NextToken = &tok
	}
	return out, nil
}

// SearchTransitGatewayRoutes has no NextToken in the AWS API. It returns the
// filtered slice capped by MaxResults and sets AdditionalRoutesAvailable if
// there would be more. Active routes are split by type filter (propagated vs
// static) so the helper can work around the 1000-result cap.
func (m *mockTGWPaginationClient) SearchTransitGatewayRoutes(_ context.Context, input *ec2.SearchTransitGatewayRoutesInput, _ ...func(*ec2.Options)) (*ec2.SearchTransitGatewayRoutesOutput, error) {
	m.searchRoutesCalls++
	m.searchRoutesFilters = append(m.searchRoutesFilters, input.Filters)

	state := ""
	routeType := ""
	for _, f := range input.Filters {
		if aws.ToString(f.Name) == "state" && len(f.Values) > 0 {
			state = f.Values[0]
		}
		if aws.ToString(f.Name) == "type" && len(f.Values) > 0 {
			routeType = f.Values[0]
		}
	}

	var source []types.TransitGatewayRoute
	switch state {
	case "blackhole":
		source = m.blackholeRoutes
	default:
		source = m.activeRoutes
	}

	// Apply type filter if present.
	var filtered []types.TransitGatewayRoute
	if routeType != "" {
		for _, r := range source {
			if string(r.Type) == routeType {
				filtered = append(filtered, r)
			}
		}
	} else {
		filtered = source
	}

	cap := int32(1000)
	if input.MaxResults != nil {
		cap = *input.MaxResults
	}

	out := &ec2.SearchTransitGatewayRoutesOutput{}
	if int32(len(filtered)) > cap {
		out.Routes = filtered[:cap]
		truthy := true
		out.AdditionalRoutesAvailable = &truthy
	} else {
		out.Routes = filtered
	}
	return out, nil
}

// makeTransitGateways builds n dummy transit gateways with predictable IDs.
func makeTransitGateways(n int) []types.TransitGateway {
	gws := make([]types.TransitGateway, n)
	for i := 0; i < n; i++ {
		gws[i] = types.TransitGateway{
			TransitGatewayId: aws.String(fmt.Sprintf("tgw-%08d", i)),
			OwnerId:          aws.String("123456789012"),
		}
	}
	return gws
}

// makeTransitGatewayRouteTables builds n dummy route tables.
func makeTransitGatewayRouteTables(n int) []types.TransitGatewayRouteTable {
	tables := make([]types.TransitGatewayRouteTable, n)
	for i := 0; i < n; i++ {
		tables[i] = types.TransitGatewayRouteTable{
			TransitGatewayRouteTableId: aws.String(fmt.Sprintf("tgw-rtb-%08d", i)),
		}
	}
	return tables
}

// makeTransitGatewayAssociations builds n dummy associations.
func makeTransitGatewayAssociations(n int) []types.TransitGatewayRouteTableAssociation {
	assocs := make([]types.TransitGatewayRouteTableAssociation, n)
	for i := 0; i < n; i++ {
		assocs[i] = types.TransitGatewayRouteTableAssociation{
			TransitGatewayAttachmentId: aws.String(fmt.Sprintf("tgw-attach-%08d", i)),
			ResourceId:                 aws.String(fmt.Sprintf("vpc-%08d", i)),
			ResourceType:               types.TransitGatewayAttachmentResourceTypeVpc,
		}
	}
	return assocs
}

// makeTransitGatewayRoutes builds n dummy active TGW routes with a given type
// so tests can distinguish propagated vs static.
func makeTransitGatewayRoutes(n int, routeType types.TransitGatewayRouteType) []types.TransitGatewayRoute {
	routes := make([]types.TransitGatewayRoute, n)
	for i := 0; i < n; i++ {
		routes[i] = types.TransitGatewayRoute{
			DestinationCidrBlock: aws.String(fmt.Sprintf("10.%d.%d.0/24", (i>>8)&0xff, i&0xff)),
			State:                types.TransitGatewayRouteStateActive,
			Type:                 routeType,
		}
	}
	return routes
}

// TestGetAllTransitGateways_Pagination verifies that getAllTransitGateways
// walks every page returned by DescribeTransitGateways. Before the fix only
// the first page was read and accounts with many TGWs would see truncated
// results in `awstools tgw overview`.
func TestGetAllTransitGateways_Pagination(t *testing.T) {
	total := 5
	mock := &mockTGWPaginationClient{
		transitGateways: makeTransitGateways(total),
		pageSize:        2, // 3 pages: [0,1], [2,3], [4]
	}

	result := getAllTransitGateways(mock)

	if len(result) != total {
		t.Fatalf("getAllTransitGateways() returned %d TGWs, want %d (pagination bug: only first page returned)", len(result), total)
	}
	if mock.describeTGWCalls != 3 {
		t.Errorf("DescribeTransitGateways called %d times, want 3 (one per page)", mock.describeTGWCalls)
	}
	for i, tgw := range result {
		want := fmt.Sprintf("tgw-%08d", i)
		if tgw.ID != want {
			t.Errorf("result[%d].ID = %q, want %q", i, tgw.ID, want)
		}
	}
}

// TestGetRouteTablesForTransitGateway_Pagination verifies pagination of
// DescribeTransitGatewayRouteTables.
func TestGetRouteTablesForTransitGateway_Pagination(t *testing.T) {
	total := 5
	mock := &mockTGWPaginationClient{
		routeTables: makeTransitGatewayRouteTables(total),
		pageSize:    2,
	}

	result := getRouteTablesForTransitGateway("tgw-00000000", mock)

	if len(result) != total {
		t.Fatalf("getRouteTablesForTransitGateway() returned %d tables, want %d (pagination bug: only first page returned)", len(result), total)
	}
	if mock.describeRTCalls != 3 {
		t.Errorf("DescribeTransitGatewayRouteTables called %d times, want 3", mock.describeRTCalls)
	}
	for i := 0; i < total; i++ {
		id := fmt.Sprintf("tgw-rtb-%08d", i)
		if _, ok := result[id]; !ok {
			t.Errorf("result missing route table %q", id)
		}
	}
}

// TestGetSourceAttachmentsForTransitGatewayRouteTable_Pagination verifies
// pagination of GetTransitGatewayRouteTableAssociations.
func TestGetSourceAttachmentsForTransitGatewayRouteTable_Pagination(t *testing.T) {
	total := 5
	mock := &mockTGWPaginationClient{
		associations: makeTransitGatewayAssociations(total),
		pageSize:     2,
	}

	result := getSourceAttachmentsForTransitGatewayRouteTable("tgw-rtb-00000000", mock)

	if len(result) != total {
		t.Fatalf("getSourceAttachmentsForTransitGatewayRouteTable() returned %d attachments, want %d (pagination bug: only first page returned)", len(result), total)
	}
	if mock.associationCalls != 3 {
		t.Errorf("GetTransitGatewayRouteTableAssociations called %d times, want 3", mock.associationCalls)
	}
	for i, a := range result {
		want := fmt.Sprintf("tgw-attach-%08d", i)
		if a.ID != want {
			t.Errorf("result[%d].ID = %q, want %q", i, a.ID, want)
		}
	}
}

// TestGetActiveRoutesForTransitGatewayRouteTable_SplitsOnOverflow verifies
// that when the initial SearchTransitGatewayRoutes response signals more
// routes are available (hit the 1000 cap), the helper falls back to per-type
// searches so propagated and static routes aren't silently dropped.
func TestGetActiveRoutesForTransitGatewayRouteTable_SplitsOnOverflow(t *testing.T) {
	// Create routes that exceed the default 1000 cap so the mock flags
	// AdditionalRoutesAvailable. Split them across propagated and static.
	propagated := makeTransitGatewayRoutes(600, types.TransitGatewayRouteTypePropagated)
	static := makeTransitGatewayRoutes(600, types.TransitGatewayRouteTypeStatic)
	all := append([]types.TransitGatewayRoute{}, propagated...)
	all = append(all, static...)
	mock := &mockTGWPaginationClient{
		activeRoutes: all,
	}

	result := getActiveRoutesForTransitGatewayRouteTable("tgw-rtb-00000000", mock)

	// With split-by-type fallback, all 1200 routes should be returned:
	// both propagated and static subsets fit under the 1000 cap individually.
	if len(result) != len(all) {
		t.Fatalf("getActiveRoutesForTransitGatewayRouteTable() returned %d routes, want %d (split-by-type fallback should recover all routes)", len(result), len(all))
	}

	// Expect at least one follow-up call with a type filter.
	sawTypeFilter := false
	for _, filters := range mock.searchRoutesFilters {
		for _, f := range filters {
			if aws.ToString(f.Name) == "type" {
				sawTypeFilter = true
			}
		}
	}
	if !sawTypeFilter {
		t.Errorf("expected a follow-up SearchTransitGatewayRoutes call with a type filter after overflow, but none was made")
	}
}

// TestGetBlackholeRoutesForTransitGatewayRouteTable_UsesMaxResults verifies
// that the blackhole-route search explicitly requests the 1000-result cap.
// A missing MaxResults would fall back to the default (also 1000) but
// setting it explicitly documents intent and future-proofs against default
// changes.
func TestGetBlackholeRoutesForTransitGatewayRouteTable_UsesMaxResults(t *testing.T) {
	mock := &mockTGWPaginationClient{
		blackholeRoutes: makeTransitGatewayRoutes(3, types.TransitGatewayRouteTypeStatic),
	}

	_ = getBlackholeRoutesForTransitGatewayRouteTable("tgw-rtb-00000000", mock)

	if mock.searchRoutesCalls == 0 {
		t.Fatalf("SearchTransitGatewayRoutes was not called")
	}
}
