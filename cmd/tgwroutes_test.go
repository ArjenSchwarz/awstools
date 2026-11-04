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

// TestFilterGatewayVPCFilterKeepsDestinations is the regression test for T-1717.
//
// Bug: when `tgw routetables --resource-id vpc-...` matched a route table only
// because the VPC is one of its source attachments, the route table was kept
// with all of its destinations, but the destination resources were pruned from
// attachedresources. Draw.io/dot output then had route-table edges pointing at
// nodes that were never emitted.
//
// Expected: every destination of a retained route table is also present in
// attachedresources, and destination matches keep their existing behaviour.
func TestFilterGatewayVPCFilterKeepsDestinations(t *testing.T) {
	route := func(resourceID string) helpers.TransitGatewayRoute {
		return helpers.TransitGatewayRoute{
			State: "active",
			Attachment: helpers.TransitGatewayAttachment{
				ID:           "tgw-attach-" + resourceID,
				ResourceID:   resourceID,
				ResourceType: vpcResourceType,
			},
		}
	}
	source := func(resourceID string) helpers.TransitGatewayAttachment {
		return helpers.TransitGatewayAttachment{
			ID:           "tgw-attach-" + resourceID,
			ResourceID:   resourceID,
			ResourceType: vpcResourceType,
		}
	}
	// vpc-app is associated with tgw-rtb-app, which routes to vpc-shared and
	// vpc-db. vpc-shared is associated with tgw-rtb-shared and vpc-other with
	// tgw-rtb-other; neither of those route tables routes to vpc-app, so for a
	// vpc-app filter vpc-app only matches as a source and both are removed.
	gateways := []helpers.TransitGateway{
		{
			ID: "tgw-test",
			RouteTables: map[string]helpers.TransitGatewayRouteTable{
				"tgw-rtb-app": {
					ID:                "tgw-rtb-app",
					Routes:            []helpers.TransitGatewayRoute{route("vpc-shared"), route("vpc-db")},
					SourceAttachments: []helpers.TransitGatewayAttachment{source("vpc-app")},
				},
				"tgw-rtb-shared": {
					ID:                "tgw-rtb-shared",
					Routes:            []helpers.TransitGatewayRoute{route("vpc-other")},
					SourceAttachments: []helpers.TransitGatewayAttachment{source("vpc-shared")},
				},
				"tgw-rtb-other": {
					ID:                "tgw-rtb-other",
					Routes:            []helpers.TransitGatewayRoute{route("vpc-shared")},
					SourceAttachments: []helpers.TransitGatewayAttachment{source("vpc-other")},
				},
			},
		},
	}

	tests := map[string]struct {
		resourceID      string
		wantRouteTables map[string][]string
		wantResources   []string
		// wantInbound maps each emitted resource to its graph association
		// (Target Gateway column in Draw.io output).
		wantInbound map[string]string
	}{
		"source-only match keeps destination resources": {
			resourceID:      "vpc-app",
			wantRouteTables: map[string][]string{"tgw-rtb-app": {"vpc-shared", "vpc-db"}},
			wantResources:   []string{"vpc-app", "vpc-db", "vpc-shared"},
			// vpc-shared is associated with the filtered-out tgw-rtb-shared,
			// so it has no inbound graph edge.
			wantInbound: map[string]string{"vpc-app": "tgw-rtb-app", "vpc-db": "", "vpc-shared": ""},
		},
		"destination match keeps only requested destination and its sources": {
			resourceID:      "vpc-db",
			wantRouteTables: map[string][]string{"tgw-rtb-app": {"vpc-db"}},
			wantResources:   []string{"vpc-app", "vpc-db"},
			wantInbound:     map[string]string{"vpc-app": "tgw-rtb-app", "vpc-db": ""},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			originalResourceID := tgwresourceid
			tgwresourceid = tc.resourceID
			t.Cleanup(func() { tgwresourceid = originalResourceID })

			attachedresources, tgwrts := filterGateway(gateways)

			assert.Equal(t, tc.wantRouteTables, tgwrts)
			gotResources := make([]string, 0, len(attachedresources))
			for id := range attachedresources {
				gotResources = append(gotResources, id)
			}
			slices.Sort(gotResources)
			assert.Equal(t, tc.wantResources, gotResources)

			// Outbound edges: route table -> destination resource.
			for rt, destinations := range tgwrts {
				for _, destination := range destinations {
					assert.Containsf(t, attachedresources, destination, "route table %s points at %s, which is not emitted", rt, destination)
				}
			}
			// Inbound edges: resource -> associated route table.
			gotInbound := make(map[string]string, len(attachedresources))
			for id, info := range attachedresources {
				rt := graphRouteTableID(info, tgwrts)
				gotInbound[id] = rt
				if rt != "" {
					assert.Containsf(t, tgwrts, rt, "resource %s is associated with %s, which is not emitted", id, rt)
				}
			}
			assert.Equal(t, tc.wantInbound, gotInbound)
		})
	}
}
