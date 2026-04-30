package cmd

import (
	"testing"

	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/stretchr/testify/assert"
)

// TestFilterGateway_IncludesNonVPCAttachments verifies that filterGateway
// returns non-VPC resources (VPN, DX gateway, peering) from both route
// destinations and source attachments. This is the regression test for T-924.
func TestFilterGateway_IncludesNonVPCAttachments(t *testing.T) {
	// Reset global filter state
	tgwresourceid = ""

	gateways := []helpers.TransitGateway{
		{
			ID:        "tgw-00000001",
			AccountID: "123456789012",
			Name:      "test-tgw",
			RouteTables: map[string]helpers.TransitGatewayRouteTable{
				"tgw-rtb-00000001": {
					ID:   "tgw-rtb-00000001",
					Name: "rt1",
					Routes: []helpers.TransitGatewayRoute{
						{
							State: "active",
							CIDR:  "10.0.0.0/16",
							Attachment: helpers.TransitGatewayAttachment{
								ID:         "tgw-attach-vpc",
								ResourceID: "vpc-00000001",
							},
						},
						{
							State: "active",
							CIDR:  "192.168.0.0/16",
							Attachment: helpers.TransitGatewayAttachment{
								ID:         "tgw-attach-vpn",
								ResourceID: "vpn-00000001",
							},
						},
					},
					SourceAttachments: []helpers.TransitGatewayAttachment{
						{
							ID:           "tgw-attach-dxgw",
							ResourceID:   "dxgw-00000001",
							ResourceType: "direct-connect-gateway",
						},
					},
				},
			},
		},
	}

	attachedresources, tgwrts := filterGateway(gateways)

	// Route table should list both VPC and VPN as destinations
	assert.Contains(t, tgwrts["tgw-rtb-00000001"], "vpc-00000001")
	assert.Contains(t, tgwrts["tgw-rtb-00000001"], "vpn-00000001")

	// All resource types should appear in attachedresources
	assert.Contains(t, attachedresources, "vpc-00000001", "VPC should be in attached resources")
	assert.Contains(t, attachedresources, "vpn-00000001", "VPN should be in attached resources")
	assert.Contains(t, attachedresources, "dxgw-00000001", "DX gateway should be in attached resources")

	// DX gateway (source attachment only) should have its route table set
	assert.Equal(t, "tgw-rtb-00000001", attachedresources["dxgw-00000001"])
}

// TestFilterGateway_NonVPCSourceAttachmentNotDuplicated verifies that a
// resource appearing as both a route target and a source attachment is not
// duplicated and retains its route table association.
func TestFilterGateway_NonVPCSourceAttachmentNotDuplicated(t *testing.T) {
	tgwresourceid = ""

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
							CIDR:  "172.16.0.0/12",
							Attachment: helpers.TransitGatewayAttachment{
								ID:         "tgw-attach-vpn",
								ResourceID: "vpn-00000001",
							},
						},
					},
					SourceAttachments: []helpers.TransitGatewayAttachment{
						{
							ID:           "tgw-attach-vpn",
							ResourceID:   "vpn-00000001",
							ResourceType: "vpn",
						},
					},
				},
			},
		},
	}

	attachedresources, _ := filterGateway(gateways)

	// VPN appears in both routes and source attachments — should have route table set
	assert.Equal(t, "tgw-rtb-00000001", attachedresources["vpn-00000001"])
}
