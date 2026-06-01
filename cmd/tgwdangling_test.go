package cmd

import (
	"testing"

	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/stretchr/testify/assert"
)

// TestDanglingRoutes_SkipsBlackholeRoutes is the regression test for T-1393.
// A blackhole route is parsed with an empty Attachment.ResourceID. Before the
// fix, that empty string was appended as a target, producing a dangling row
// with an empty DestinationVPC/DestinationName. This test asserts no such empty
// destination is produced.
func TestDanglingRoutes_SkipsBlackholeRoutes(t *testing.T) {
	gateways := []helpers.TransitGateway{
		{
			ID:        "tgw-00000001",
			AccountID: "123456789012",
			RouteTables: map[string]helpers.TransitGatewayRouteTable{
				"tgw-rtb-00000001": {
					ID: "tgw-rtb-00000001",
					Routes: []helpers.TransitGatewayRoute{
						{
							State: "active",
							CIDR:  "10.0.0.0/16",
							Attachment: helpers.TransitGatewayAttachment{
								ID:         "tgw-attach-vpc2",
								ResourceID: "vpc-00000002",
							},
						},
						{
							// Blackhole route: parseBlackholeRoute leaves the
							// attachment empty.
							State:     "blackhole",
							CIDR:      "172.16.0.0/12",
							RouteType: "static",
						},
					},
					SourceAttachments: []helpers.TransitGatewayAttachment{
						{
							ID:           "tgw-attach-vpc1",
							ResourceID:   "vpc-00000001",
							ResourceType: "vpc",
						},
					},
				},
			},
		},
	}

	vpcs := danglingRouteTargets(gateways)

	// The source VPC's target list must not contain an empty resource ID.
	assert.NotContains(t, vpcs["vpc-00000001"], "", "blackhole route should not be appended as an empty target")
	assert.Contains(t, vpcs["vpc-00000001"], "vpc-00000002", "active route target should be present")

	// No empty-key entry should be created either.
	_, hasEmpty := vpcs[""]
	assert.False(t, hasEmpty, "no empty source key should exist")
}
