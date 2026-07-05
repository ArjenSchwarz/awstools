package cmd

import (
	"slices"
	"testing"

	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/stretchr/testify/assert"
)

// TestFilterGatewaySkipsBlackholeRoutes is the regression test for T-1124.
//
// `tgw routetables` full graph mode calls filterGateway, which iterates over
// TransitGatewayRouteTable.Routes. The helpers append active routes plus
// blackhole routes, and blackhole routes have no attachment, so their
// Attachment.ResourceID is the empty string. The bug was that filterGateway
// appended that empty string to the route table destinations and created an
// attachedresources[""] entry, which rendered a blank destination/resource row
// and produced bad Draw.io edges.
//
// Expected behaviour: blackhole routes (empty attachment resource IDs) are
// skipped, so neither the route table destinations nor the attached resources
// contain an empty string.
func TestFilterGatewaySkipsBlackholeRoutes(t *testing.T) {
	// Ensure no resource-id filter is applied for this test.
	originalResourceID := tgwresourceid
	tgwresourceid = ""
	t.Cleanup(func() { tgwresourceid = originalResourceID })

	const (
		routeTableID  = "tgw-rtb-test"
		activeVPCID   = "vpc-active"
		gatewayID     = "tgw-test"
		blackholeCIDR = "10.0.0.0/8"
	)

	gateways := []helpers.TransitGateway{
		{
			ID: gatewayID,
			RouteTables: map[string]helpers.TransitGatewayRouteTable{
				routeTableID: {
					ID: routeTableID,
					Routes: []helpers.TransitGatewayRoute{
						{
							// Active route with a real attachment.
							State:     "active",
							RouteType: "static",
							Attachment: helpers.TransitGatewayAttachment{
								ID:           "tgw-attach-active",
								ResourceID:   activeVPCID,
								ResourceType: vpcResourceType,
							},
						},
						{
							// Blackhole route: no attachment, so ResourceID is "".
							State:     "blackhole",
							RouteType: "static",
							CIDR:      blackholeCIDR,
						},
					},
				},
			},
		},
	}

	attachedresources, tgwrts := filterGateway(gateways)

	destinations, ok := tgwrts[routeTableID]
	if !ok {
		t.Fatalf("expected route table %q to be present in destinations map", routeTableID)
	}

	if slices.Contains(destinations, "") {
		t.Errorf("route table destinations must not contain an empty string; got %#v", destinations)
	}
	if !slices.Contains(destinations, activeVPCID) {
		t.Errorf("route table destinations should contain the active VPC %q; got %#v", activeVPCID, destinations)
	}

	if _, exists := attachedresources[""]; exists {
		t.Errorf("attached resources must not contain an empty resource ID entry; got %#v", attachedresources)
	}
	if _, exists := attachedresources[activeVPCID]; !exists {
		t.Errorf("attached resources should contain the active VPC %q; got %#v", activeVPCID, attachedresources)
	}
}

// TestDrawIORecords pins the conversion of shared table rows into draw.io
// records: []string cells are joined with "," (matching v1's drawio CSV
// cells, which the combine read-back splits on ","), everything else passes
// through untouched.
func TestDrawIORecords(t *testing.T) {
	rows := []map[string]any{
		{
			"ID":               "tgw-rtb-1",
			nameColumn:         "route table",
			destinationsColumn: []string{"vpc-1", "vpc-2"},
		},
		{
			"ID":               "vpc-1",
			destinationsColumn: []string{},
			"Count":            3,
		},
	}

	records := drawIORecords(rows)

	assert.Len(t, records, 2)
	assert.Equal(t, "tgw-rtb-1", records[0]["ID"])
	assert.Equal(t, "route table", records[0][nameColumn])
	assert.Equal(t, "vpc-1,vpc-2", records[0][destinationsColumn], "[]string cells must be comma-joined")
	assert.Equal(t, "", records[1][destinationsColumn], "empty []string cells must become empty strings")
	assert.Equal(t, 3, records[1]["Count"], "non-string cells pass through untouched")
}

// TestValidateSimpleListResourceID is the regression test for T-1255.
//
// Bug: simplelistOnly used the global tgwresourceid value directly as a
// Transit Gateway route table ID and passed it to
// GetActiveRoutesForTransitGatewayRouteTable /
// GetBlackholeRoutesForTransitGatewayRouteTable without validating it. Running
// `awstools tgw routetables --list` without --resource-id, or with a VPC/TGW
// ID, would call SearchTransitGatewayRoutes with an empty or wrong route table
// ID instead of rejecting the input with a clear usage error.
//
// Expected: validation requires a non-empty tgw-rtb-* route table ID and
// rejects everything else with an error.
func TestValidateSimpleListResourceID(t *testing.T) {
	tests := []struct {
		name       string
		resourceID string
		wantErr    bool
	}{
		{name: "empty rejected", resourceID: "", wantErr: true},
		{name: "whitespace rejected", resourceID: "   ", wantErr: true},
		{name: "vpc id rejected", resourceID: "vpc-00000001", wantErr: true},
		{name: "tgw id rejected", resourceID: "tgw-00000001", wantErr: true},
		{name: "tgw attachment rejected", resourceID: "tgw-attach-00000001", wantErr: true},
		{name: "multiple ids rejected", resourceID: "tgw-rtb-1,tgw-rtb-2", wantErr: true},
		{name: "valid route table accepted", resourceID: "tgw-rtb-00000001", wantErr: false},
		{name: "valid route table with surrounding whitespace accepted", resourceID: "  tgw-rtb-00000001  ", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSimpleListResourceID(tt.resourceID)
			if tt.wantErr {
				assert.Error(t, err, "expected validation error for %q", tt.resourceID)
			} else {
				assert.NoError(t, err, "expected no validation error for %q", tt.resourceID)
			}
		})
	}
}
