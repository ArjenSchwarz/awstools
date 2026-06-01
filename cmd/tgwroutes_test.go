package cmd

import (
	"slices"
	"testing"

	"github.com/ArjenSchwarz/awstools/helpers"
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
