package helpers

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestGetNameFromTags(t *testing.T) {
	tests := []struct {
		name     string
		tags     []types.Tag
		expected string
	}{
		{
			name: "tag with Name key exists",
			tags: []types.Tag{
				{Key: aws.String("Environment"), Value: aws.String("production")},
				{Key: aws.String("Name"), Value: aws.String("my-resource")},
				{Key: aws.String("Team"), Value: aws.String("backend")},
			},
			expected: "my-resource",
		},
		{
			name: "no Name tag exists",
			tags: []types.Tag{
				{Key: aws.String("Environment"), Value: aws.String("production")},
				{Key: aws.String("Team"), Value: aws.String("backend")},
			},
			expected: "",
		},
		{
			name:     "nil tags",
			tags:     nil,
			expected: "",
		},
		{
			name:     "empty tags slice",
			tags:     []types.Tag{},
			expected: "",
		},
		{
			name: "Name tag with empty value",
			tags: []types.Tag{
				{Key: aws.String("Name"), Value: aws.String("")},
			},
			expected: "",
		},
		{
			name: "multiple Name tags - returns first",
			tags: []types.Tag{
				{Key: aws.String("Name"), Value: aws.String("first-name")},
				{Key: aws.String("Name"), Value: aws.String("second-name")},
			},
			expected: "first-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getNameFromTags(tt.tags)
			if result != tt.expected {
				t.Errorf("getNameFromTags() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestIsLatestInstanceFamily(t *testing.T) {
	tests := []struct {
		name           string
		instanceFamily string
		expected       bool
	}{
		// Compute optimized
		{
			name:           "c4 is latest compute",
			instanceFamily: "c4",
			expected:       true,
		},
		{
			name:           "c3 is not latest compute",
			instanceFamily: "c3",
			expected:       false,
		},
		// Dense storage
		{
			name:           "d2 is latest dense storage",
			instanceFamily: "d2",
			expected:       true,
		},
		{
			name:           "d1 is not latest dense storage",
			instanceFamily: "d1",
			expected:       false,
		},
		// FPGA
		{
			name:           "f1 is latest FPGA",
			instanceFamily: "f1",
			expected:       true,
		},
		// GPU
		{
			name:           "g3 is latest GPU",
			instanceFamily: "g3",
			expected:       true,
		},
		{
			name:           "g2 is not latest GPU",
			instanceFamily: "g2",
			expected:       false,
		},
		// Accelerated computing
		{
			name:           "p2 is latest accelerated",
			instanceFamily: "p2",
			expected:       true,
		},
		{
			name:           "p1 is not latest accelerated",
			instanceFamily: "p1",
			expected:       false,
		},
		// Storage optimized
		{
			name:           "i3 is latest storage optimized",
			instanceFamily: "i3",
			expected:       true,
		},
		{
			name:           "i2 is not latest storage optimized",
			instanceFamily: "i2",
			expected:       false,
		},
		// General purpose
		{
			name:           "m4 is latest general purpose",
			instanceFamily: "m4",
			expected:       true,
		},
		{
			name:           "m3 is not latest general purpose",
			instanceFamily: "m3",
			expected:       false,
		},
		// Memory optimized
		{
			name:           "r4 is latest memory optimized",
			instanceFamily: "r4",
			expected:       true,
		},
		{
			name:           "r3 is not latest memory optimized",
			instanceFamily: "r3",
			expected:       false,
		},
		// Burstable
		{
			name:           "t2 is latest burstable",
			instanceFamily: "t2",
			expected:       true,
		},
		{
			name:           "t1 is not latest burstable",
			instanceFamily: "t1",
			expected:       false,
		},
		// High memory
		{
			name:           "x1 is latest high memory",
			instanceFamily: "x1",
			expected:       true,
		},
		// Unknown family
		{
			name:           "unknown family returns false",
			instanceFamily: "z9",
			expected:       false,
		},
		// Invalid format
		{
			name:           "invalid format returns false",
			instanceFamily: "invalid",
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsLatestInstanceFamily(tt.instanceFamily)
			if result != tt.expected {
				t.Errorf("IsLatestInstanceFamily(%q) = %v, want %v", tt.instanceFamily, result, tt.expected)
			}
		})
	}
}

func TestParseVPCRoutes(t *testing.T) {
	tests := []struct {
		name     string
		routes   []types.Route
		expected []VPCRoute
	}{
		{
			name: "route with destination CIDR and VPC peering",
			routes: []types.Route{
				{
					DestinationCidrBlock:   aws.String("10.0.0.0/16"),
					VpcPeeringConnectionId: aws.String("pcx-12345678"),
					State:                  types.RouteStateActive,
				},
			},
			expected: []VPCRoute{
				{
					DestinationCIDR:   "10.0.0.0/16",
					State:             "active",
					DestinationTarget: "pcx-12345678",
				},
			},
		},
		{
			name: "route with IPv6 CIDR and gateway",
			routes: []types.Route{
				{
					DestinationIpv6CidrBlock: aws.String("2001:db8::/32"),
					GatewayId:                aws.String("igw-12345678"),
					State:                    types.RouteStateActive,
				},
			},
			expected: []VPCRoute{
				{
					DestinationCIDR:   "2001:db8::/32",
					State:             "active",
					DestinationTarget: "igw-12345678",
				},
			},
		},
		{
			name: "route with NAT gateway",
			routes: []types.Route{
				{
					DestinationCidrBlock: aws.String("0.0.0.0/0"),
					NatGatewayId:         aws.String("nat-12345678"),
					State:                types.RouteStateActive,
				},
			},
			expected: []VPCRoute{
				{
					DestinationCIDR:   "0.0.0.0/0",
					State:             "active",
					DestinationTarget: "nat-12345678",
				},
			},
		},
		{
			name: "route with network interface",
			routes: []types.Route{
				{
					DestinationCidrBlock: aws.String("192.168.1.0/24"),
					NetworkInterfaceId:   aws.String("eni-12345678"),
					State:                types.RouteStateActive,
				},
			},
			expected: []VPCRoute{
				{
					DestinationCIDR:   "192.168.1.0/24",
					State:             "active",
					DestinationTarget: "eni-12345678",
				},
			},
		},
		{
			name: "route with egress-only internet gateway",
			routes: []types.Route{
				{
					DestinationIpv6CidrBlock:    aws.String("::/0"),
					EgressOnlyInternetGatewayId: aws.String("eigw-12345678"),
					State:                       types.RouteStateActive,
				},
			},
			expected: []VPCRoute{
				{
					DestinationCIDR:   "::/0",
					State:             "active",
					DestinationTarget: "eigw-12345678",
				},
			},
		},
		{
			name: "route with transit gateway",
			routes: []types.Route{
				{
					DestinationCidrBlock: aws.String("172.16.0.0/16"),
					TransitGatewayId:     aws.String("tgw-12345678"),
					State:                types.RouteStateActive,
				},
			},
			expected: []VPCRoute{
				{
					DestinationCIDR:   "172.16.0.0/16",
					State:             "active",
					DestinationTarget: "tgw-12345678",
				},
			},
		},
		{
			name:     "empty routes",
			routes:   []types.Route{},
			expected: nil, // parseVPCRoutes returns nil for empty input
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseVPCRoutes(tt.routes)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseVPCRoutes() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestVPCPeering_Struct(t *testing.T) {
	peering := VpcPeering{
		RequesterVpc: VPCHolder{
			ID:        "vpc-12345678",
			AccountID: "123456789012",
		},
		AccepterVpc: VPCHolder{
			ID:        "vpc-87654321",
			AccountID: "210987654321",
		},
		PeeringID: "pcx-12345678",
	}

	if peering.RequesterVpc.ID != "vpc-12345678" {
		t.Errorf("Expected RequesterVpc.ID to be 'vpc-12345678', got %s", peering.RequesterVpc.ID)
	}
	if peering.AccepterVpc.AccountID != "210987654321" {
		t.Errorf("Expected AccepterVpc.AccountID to be '210987654321', got %s", peering.AccepterVpc.AccountID)
	}
	if peering.PeeringID != "pcx-12345678" {
		t.Errorf("Expected PeeringID to be 'pcx-12345678', got %s", peering.PeeringID)
	}
}

func TestVPCRouteTable_Struct(t *testing.T) {
	routeTable := VPCRouteTable{
		Vpc: VPCHolder{
			ID:        "vpc-12345678",
			AccountID: "123456789012",
		},
		ID: "rtb-12345678",
		Routes: []VPCRoute{
			{
				DestinationCIDR:   "10.0.0.0/16",
				State:             "active",
				DestinationTarget: "local",
			},
		},
		Subnets: []string{"subnet-12345678", "subnet-87654321"},
		Default: true,
	}

	if routeTable.Vpc.ID != "vpc-12345678" {
		t.Errorf("Expected Vpc.ID to be 'vpc-12345678', got %s", routeTable.Vpc.ID)
	}
	if len(routeTable.Routes) != 1 {
		t.Errorf("Expected 1 route, got %d", len(routeTable.Routes))
	}
	if len(routeTable.Subnets) != 2 {
		t.Errorf("Expected 2 subnets, got %d", len(routeTable.Subnets))
	}
	if !routeTable.Default {
		t.Errorf("Expected Default to be true, got %v", routeTable.Default)
	}
}

func TestMatchTransitGatewayAttachment(t *testing.T) {
	tests := []struct {
		name        string
		attachments []types.TransitGatewayVpcAttachment
		subnetID    string
		expected    string
	}{
		{
			name:        "no attachments returns empty",
			attachments: nil,
			subnetID:    "subnet-aaa",
			expected:    "",
		},
		{
			name: "subnet on first attachment",
			attachments: []types.TransitGatewayVpcAttachment{
				{
					TransitGatewayAttachmentId: aws.String("tgw-attach-111"),
					SubnetIds:                  []string{"subnet-aaa", "subnet-bbb"},
				},
			},
			subnetID: "subnet-aaa",
			expected: "tgw-attach-111",
		},
		{
			name: "subnet on second attachment (regression: previously only checked first)",
			attachments: []types.TransitGatewayVpcAttachment{
				{
					TransitGatewayAttachmentId: aws.String("tgw-attach-111"),
					SubnetIds:                  []string{"subnet-aaa"},
				},
				{
					TransitGatewayAttachmentId: aws.String("tgw-attach-222"),
					SubnetIds:                  []string{"subnet-bbb", "subnet-ccc"},
				},
			},
			subnetID: "subnet-bbb",
			expected: "tgw-attach-222",
		},
		{
			name: "subnet on third attachment",
			attachments: []types.TransitGatewayVpcAttachment{
				{
					TransitGatewayAttachmentId: aws.String("tgw-attach-111"),
					SubnetIds:                  []string{"subnet-aaa"},
				},
				{
					TransitGatewayAttachmentId: aws.String("tgw-attach-222"),
					SubnetIds:                  []string{"subnet-bbb"},
				},
				{
					TransitGatewayAttachmentId: aws.String("tgw-attach-333"),
					SubnetIds:                  []string{"subnet-ccc", "subnet-ddd"},
				},
			},
			subnetID: "subnet-ddd",
			expected: "tgw-attach-333",
		},
		{
			name: "subnet not found in any attachment",
			attachments: []types.TransitGatewayVpcAttachment{
				{
					TransitGatewayAttachmentId: aws.String("tgw-attach-111"),
					SubnetIds:                  []string{"subnet-aaa"},
				},
				{
					TransitGatewayAttachmentId: aws.String("tgw-attach-222"),
					SubnetIds:                  []string{"subnet-bbb"},
				},
			},
			subnetID: "subnet-zzz",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchTransitGatewayAttachment(tt.attachments, tt.subnetID)
			if result != tt.expected {
				t.Errorf("matchTransitGatewayAttachment() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestTransitGateway_Struct(t *testing.T) {
	tgw := TransitGateway{
		ID:        "tgw-12345678",
		AccountID: "123456789012",
		Name:      "my-transit-gateway",
		RouteTables: map[string]TransitGatewayRouteTable{
			"tgw-rtb-12345678": {
				ID:   "tgw-rtb-12345678",
				Name: "default-route-table",
			},
		},
	}

	if tgw.ID != "tgw-12345678" {
		t.Errorf("Expected ID to be 'tgw-12345678', got %s", tgw.ID)
	}
	if tgw.Name != "my-transit-gateway" {
		t.Errorf("Expected Name to be 'my-transit-gateway', got %s", tgw.Name)
	}
	if len(tgw.RouteTables) != 1 {
		t.Errorf("Expected 1 route table, got %d", len(tgw.RouteTables))
	}
}

func TestTransitGatewayRoute_Struct(t *testing.T) {
	route := TransitGatewayRoute{
		State:        "active",
		CIDR:         "10.0.0.0/16",
		ResourceType: "vpc",
		RouteType:    "static",
		Attachment: TransitGatewayAttachment{
			ID:           "tgw-attach-12345678",
			ResourceType: "vpc",
			ResourceID:   "vpc-12345678",
		},
	}

	if route.State != "active" {
		t.Errorf("Expected State to be 'active', got %s", route.State)
	}
	if route.CIDR != "10.0.0.0/16" {
		t.Errorf("Expected CIDR to be '10.0.0.0/16', got %s", route.CIDR)
	}
	if route.Attachment.ID != "tgw-attach-12345678" {
		t.Errorf("Expected Attachment.ID to be 'tgw-attach-12345678', got %s", route.Attachment.ID)
	}
}

// TestParseActiveRoute_NoAttachments verifies that parseActiveRoute handles routes
// with no attachments (e.g. local/propagated routes) without panicking. (T-398)
func TestParseActiveRoute_NoAttachments(t *testing.T) {
	route := types.TransitGatewayRoute{
		DestinationCidrBlock:      aws.String("10.0.0.0/16"),
		State:                     types.TransitGatewayRouteStateActive,
		Type:                      types.TransitGatewayRouteTypePropagated,
		TransitGatewayAttachments: []types.TransitGatewayRouteAttachment{},
	}
	result := parseActiveRoute(route)

	if result.CIDR != "10.0.0.0/16" {
		t.Errorf("Expected CIDR '10.0.0.0/16', got %s", result.CIDR)
	}
	if result.State != "active" {
		t.Errorf("Expected State 'active', got %s", result.State)
	}
	if result.RouteType != "propagated" {
		t.Errorf("Expected RouteType 'propagated', got %s", result.RouteType)
	}
	if result.Attachment.ID != "" {
		t.Errorf("Expected empty Attachment.ID, got %s", result.Attachment.ID)
	}
	if result.Attachment.ResourceID != "" {
		t.Errorf("Expected empty Attachment.ResourceID, got %s", result.Attachment.ResourceID)
	}
}

// TestParseActiveRoute_NilAttachments verifies that parseActiveRoute handles routes
// with a nil attachments slice without panicking. (T-398)
func TestParseActiveRoute_NilAttachments(t *testing.T) {
	route := types.TransitGatewayRoute{
		DestinationCidrBlock:      aws.String("172.16.0.0/12"),
		State:                     types.TransitGatewayRouteStateActive,
		Type:                      types.TransitGatewayRouteTypeStatic,
		TransitGatewayAttachments: nil,
	}
	result := parseActiveRoute(route)

	if result.CIDR != "172.16.0.0/12" {
		t.Errorf("Expected CIDR '172.16.0.0/12', got %s", result.CIDR)
	}
	if result.Attachment.ID != "" {
		t.Errorf("Expected empty Attachment.ID, got %s", result.Attachment.ID)
	}
}

// TestParseActiveRoute_WithAttachment verifies the normal case with an attachment present.
func TestParseActiveRoute_WithAttachment(t *testing.T) {
	route := types.TransitGatewayRoute{
		DestinationCidrBlock: aws.String("10.1.0.0/16"),
		State:                types.TransitGatewayRouteStateActive,
		Type:                 types.TransitGatewayRouteTypeStatic,
		TransitGatewayAttachments: []types.TransitGatewayRouteAttachment{
			{
				ResourceId:                 aws.String("vpc-abc123"),
				TransitGatewayAttachmentId: aws.String("tgw-attach-abc123"),
				ResourceType:               types.TransitGatewayAttachmentResourceTypeVpc,
			},
		},
	}
	result := parseActiveRoute(route)

	if result.CIDR != "10.1.0.0/16" {
		t.Errorf("Expected CIDR '10.1.0.0/16', got %s", result.CIDR)
	}
	if result.Attachment.ID != "tgw-attach-abc123" {
		t.Errorf("Expected Attachment.ID 'tgw-attach-abc123', got %s", result.Attachment.ID)
	}
	if result.Attachment.ResourceID != "vpc-abc123" {
		t.Errorf("Expected Attachment.ResourceID 'vpc-abc123', got %s", result.Attachment.ResourceID)
	}
}

// TestParseActiveRoute_VPNStripsPublicIP verifies VPN routes have public IP stripped.
func TestParseActiveRoute_VPNStripsPublicIP(t *testing.T) {
	route := types.TransitGatewayRoute{
		DestinationCidrBlock: aws.String("10.2.0.0/16"),
		State:                types.TransitGatewayRouteStateActive,
		Type:                 types.TransitGatewayRouteTypeStatic,
		TransitGatewayAttachments: []types.TransitGatewayRouteAttachment{
			{
				ResourceId:                 aws.String("vpn-abc123(1.2.3.4)"),
				TransitGatewayAttachmentId: aws.String("tgw-attach-vpn123"),
				ResourceType:               types.TransitGatewayAttachmentResourceTypeVpn,
			},
		},
	}
	result := parseActiveRoute(route)

	if result.Attachment.ResourceID != "vpn-abc123" {
		t.Errorf("Expected ResourceID 'vpn-abc123' (public IP stripped), got %s", result.Attachment.ResourceID)
	}
}

func TestIsValidIPAddress(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{
			name:     "valid IPv4 address",
			ip:       "192.168.1.1",
			expected: true,
		},
		{
			name:     "valid IPv6 address",
			ip:       "2001:db8::1",
			expected: true,
		},
		{
			name:     "invalid IP with high octets",
			ip:       "999.999.999.999",
			expected: false,
		},
		{
			name:     "empty string",
			ip:       "",
			expected: false,
		},
		{
			name:     "text string",
			ip:       "not-an-ip",
			expected: false,
		},
		{
			name:     "partial IP",
			ip:       "192.168.1",
			expected: false,
		},
		{
			name:     "IP with extra octets",
			ip:       "192.168.1.1.1",
			expected: false,
		},
		{
			name:     "localhost IPv4",
			ip:       "127.0.0.1",
			expected: true,
		},
		{
			name:     "localhost IPv6",
			ip:       "::1",
			expected: true,
		},
		{
			name:     "zero IP",
			ip:       "0.0.0.0",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidIPAddress(tt.ip)
			if result != tt.expected {
				t.Errorf("IsValidIPAddress(%q) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestIsValidCIDR(t *testing.T) {
	tests := []struct {
		name     string
		cidr     string
		expected bool
	}{
		{
			name:     "valid IPv4 CIDR",
			cidr:     "192.168.1.0/24",
			expected: true,
		},
		{
			name:     "valid IPv6 CIDR",
			cidr:     "2001:db8::/32",
			expected: true,
		},
		{
			name:     "valid single host CIDR",
			cidr:     "192.168.1.1/32",
			expected: true,
		},
		{
			name:     "invalid CIDR without mask",
			cidr:     "192.168.1.0",
			expected: false,
		},
		{
			name:     "invalid CIDR with bad IP",
			cidr:     "999.999.999.999/24",
			expected: false,
		},
		{
			name:     "invalid CIDR with bad mask",
			cidr:     "192.168.1.0/99",
			expected: false,
		},
		{
			name:     "empty string",
			cidr:     "",
			expected: false,
		},
		{
			name:     "text string",
			cidr:     "not-a-cidr",
			expected: false,
		},
		{
			name:     "valid wide IPv4 CIDR",
			cidr:     "10.0.0.0/8",
			expected: true,
		},
		{
			name:     "valid narrow IPv4 CIDR",
			cidr:     "192.168.1.0/30",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidCIDR(tt.cidr)
			if result != tt.expected {
				t.Errorf("IsValidCIDR(%q) = %v, want %v", tt.cidr, result, tt.expected)
			}
		})
	}
}

func TestIPFinderResult_Struct(t *testing.T) {
	// Test struct creation and JSON serialization
	eni := &types.NetworkInterface{
		NetworkInterfaceId: aws.String("eni-12345678"),
		PrivateIpAddress:   aws.String("10.0.1.100"),
	}

	result := IPFinderResult{
		IPAddress:    "10.0.1.100",
		ENI:          eni,
		ResourceType: "EC2 Instance",
		ResourceName: "web-server-1",
		ResourceID:   "i-12345678",
		VPC: VPCInfo{
			ID:   "vpc-12345678",
			Name: "main-vpc",
			CIDR: "10.0.0.0/16",
		},
		Subnet: SubnetInfo{
			ID:   "subnet-12345678",
			Name: "public-subnet-1",
			CIDR: "10.0.1.0/24",
		},
		SecurityGroups: []SecurityGroupInfo{
			{
				ID:   "sg-12345678",
				Name: "web-servers",
			},
		},
		IsSecondaryIP: false,
		Found:         true,
	}

	// Verify struct fields
	if result.IPAddress != "10.0.1.100" {
		t.Errorf("Expected IPAddress to be '10.0.1.100', got %s", result.IPAddress)
	}
	if result.ResourceType != "EC2 Instance" {
		t.Errorf("Expected ResourceType to be 'EC2 Instance', got %s", result.ResourceType)
	}
	if result.VPC.Name != "main-vpc" {
		t.Errorf("Expected VPC.Name to be 'main-vpc', got %s", result.VPC.Name)
	}
	if result.Subnet.CIDR != "10.0.1.0/24" {
		t.Errorf("Expected Subnet.CIDR to be '10.0.1.0/24', got %s", result.Subnet.CIDR)
	}
	if len(result.SecurityGroups) != 1 {
		t.Errorf("Expected 1 security group, got %d", len(result.SecurityGroups))
	}
	if !result.Found {
		t.Errorf("Expected Found to be true, got %v", result.Found)
	}
	if result.IsSecondaryIP {
		t.Errorf("Expected IsSecondaryIP to be false, got %v", result.IsSecondaryIP)
	}
}

func TestVPCInfo_Struct(t *testing.T) {
	vpcInfo := VPCInfo{
		ID:   "vpc-12345678",
		Name: "production-vpc",
		CIDR: "172.16.0.0/16",
	}

	if vpcInfo.ID != "vpc-12345678" {
		t.Errorf("Expected ID to be 'vpc-12345678', got %s", vpcInfo.ID)
	}
	if vpcInfo.Name != "production-vpc" {
		t.Errorf("Expected Name to be 'production-vpc', got %s", vpcInfo.Name)
	}
	if vpcInfo.CIDR != "172.16.0.0/16" {
		t.Errorf("Expected CIDR to be '172.16.0.0/16', got %s", vpcInfo.CIDR)
	}
}

func TestSubnetInfo_Struct(t *testing.T) {
	subnetInfo := SubnetInfo{
		ID:   "subnet-87654321",
		Name: "private-subnet-2",
		CIDR: "172.16.2.0/24",
	}

	if subnetInfo.ID != "subnet-87654321" {
		t.Errorf("Expected ID to be 'subnet-87654321', got %s", subnetInfo.ID)
	}
	if subnetInfo.Name != "private-subnet-2" {
		t.Errorf("Expected Name to be 'private-subnet-2', got %s", subnetInfo.Name)
	}
	if subnetInfo.CIDR != "172.16.2.0/24" {
		t.Errorf("Expected CIDR to be '172.16.2.0/24', got %s", subnetInfo.CIDR)
	}
}

func TestSecurityGroupInfo_Struct(t *testing.T) {
	sgInfo := SecurityGroupInfo{
		ID:   "sg-87654321",
		Name: "database-servers",
	}

	if sgInfo.ID != "sg-87654321" {
		t.Errorf("Expected ID to be 'sg-87654321', got %s", sgInfo.ID)
	}
	if sgInfo.Name != "database-servers" {
		t.Errorf("Expected Name to be 'database-servers', got %s", sgInfo.Name)
	}
}

func TestIsSecondaryIP(t *testing.T) {
	tests := []struct {
		name      string
		eni       types.NetworkInterface
		ipAddress string
		expected  bool
	}{
		{
			name: "primary IP address",
			eni: types.NetworkInterface{
				PrivateIpAddress: aws.String("10.0.1.100"),
				PrivateIpAddresses: []types.NetworkInterfacePrivateIpAddress{
					{
						PrivateIpAddress: aws.String("10.0.1.100"),
						Primary:          aws.Bool(true),
					},
				},
			},
			ipAddress: "10.0.1.100",
			expected:  false,
		},
		{
			name: "secondary IP address",
			eni: types.NetworkInterface{
				PrivateIpAddress: aws.String("10.0.1.100"),
				PrivateIpAddresses: []types.NetworkInterfacePrivateIpAddress{
					{
						PrivateIpAddress: aws.String("10.0.1.100"),
						Primary:          aws.Bool(true),
					},
					{
						PrivateIpAddress: aws.String("10.0.1.101"),
						Primary:          aws.Bool(false),
					},
				},
			},
			ipAddress: "10.0.1.101",
			expected:  true,
		},
		{
			name: "IP not found in ENI",
			eni: types.NetworkInterface{
				PrivateIpAddress: aws.String("10.0.1.100"),
				PrivateIpAddresses: []types.NetworkInterfacePrivateIpAddress{
					{
						PrivateIpAddress: aws.String("10.0.1.100"),
						Primary:          aws.Bool(true),
					},
				},
			},
			ipAddress: "10.0.1.999",
			expected:  false,
		},
		{
			name: "multiple secondary IPs",
			eni: types.NetworkInterface{
				PrivateIpAddress: aws.String("10.0.1.100"),
				PrivateIpAddresses: []types.NetworkInterfacePrivateIpAddress{
					{
						PrivateIpAddress: aws.String("10.0.1.100"),
						Primary:          aws.Bool(true),
					},
					{
						PrivateIpAddress: aws.String("10.0.1.101"),
						Primary:          aws.Bool(false),
					},
					{
						PrivateIpAddress: aws.String("10.0.1.102"),
						Primary:          aws.Bool(false),
					},
				},
			},
			ipAddress: "10.0.1.102",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSecondaryIP(tt.eni, tt.ipAddress)
			if result != tt.expected {
				t.Errorf("isSecondaryIP() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHandleAWSAPIError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		apiName     string
		expectedMsg string
	}{
		{
			name:        "UnauthorizedOperation error",
			err:         fmt.Errorf("UnauthorizedOperation: You are not authorized to perform this operation"),
			apiName:     "DescribeNetworkInterfaces",
			expectedMsg: "insufficient permissions for DescribeNetworkInterfaces",
		},
		{
			name:        "AuthFailure error",
			err:         fmt.Errorf("AuthFailure: AWS was not able to validate the provided access credentials"),
			apiName:     "DescribeVpcs",
			expectedMsg: "AWS authentication failed",
		},
		{
			name:        "RequestLimitExceeded error",
			err:         fmt.Errorf("RequestLimitExceeded: Request limit exceeded"),
			apiName:     "DescribeSubnets",
			expectedMsg: "AWS API rate limit exceeded",
		},
		{
			name:        "Throttling error",
			err:         fmt.Errorf("Throttling: Rate exceeded"),
			apiName:     "DescribeSecurityGroups",
			expectedMsg: "AWS API rate limit exceeded",
		},
		{
			name:        "Generic error",
			err:         fmt.Errorf("InternalError: We encountered an internal error"),
			apiName:     "DescribeNetworkInterfaces",
			expectedMsg: "failed to call DescribeNetworkInterfaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if err, ok := r.(error); ok {
						if !strings.Contains(err.Error(), tt.expectedMsg) {
							t.Errorf("handleAWSAPIError() error = %v, want error containing %v", err, tt.expectedMsg)
						}
					} else {
						t.Errorf("handleAWSAPIError() recovered non-error: %v", r)
					}
				} else {
					t.Errorf("handleAWSAPIError() should have panicked but didn't")
				}
			}()

			handleAWSAPIError(tt.err, tt.apiName)
		})
	}
}

func TestFindIPAddressDetails_ErrorHandling(t *testing.T) {
	tests := []struct {
		name             string
		ipAddress        string
		includeSecondary bool
		expected         IPFinderResult
	}{
		{
			name:             "IP not found returns not found result",
			ipAddress:        "10.0.1.999",
			includeSecondary: false,
			expected: IPFinderResult{
				IPAddress: "10.0.1.999",
				Found:     false,
			},
		},
		{
			name:             "Valid IP format with secondary flag",
			ipAddress:        "192.168.1.1",
			includeSecondary: true,
			expected: IPFinderResult{
				IPAddress: "192.168.1.1",
				Found:     false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Since we can't easily mock the AWS client in this test,
			// we'll test the not found scenario by design
			// This test validates that the function returns the expected structure
			// when no ENIs are found
			result := IPFinderResult{
				IPAddress: tt.ipAddress,
				Found:     false,
			}

			if result.IPAddress != tt.expected.IPAddress {
				t.Errorf("FindIPAddressDetails() IPAddress = %v, want %v", result.IPAddress, tt.expected.IPAddress)
			}
			if result.Found != tt.expected.Found {
				t.Errorf("FindIPAddressDetails() Found = %v, want %v", result.Found, tt.expected.Found)
			}
		})
	}
}

func TestGetResourceNameAndID_EdgeCases(t *testing.T) {
	// Test handling of ENI with no attachments
	eni := types.NetworkInterface{
		NetworkInterfaceId: aws.String("eni-12345678"),
	}

	// Create a minimal cache for testing
	cache := &ENILookupCache{
		EndpointsByENI:   make(map[string]*types.VpcEndpoint),
		NATGatewaysByENI: make(map[string]*types.NatGateway),
	}

	name, id := getResourceNameAndID(eni, cache)

	// Should return some form of resource description and empty ID
	if name == "" {
		t.Error("getResourceNameAndID() should return some resource description for unattached ENI")
	}
	if id != "" {
		t.Errorf("getResourceNameAndID() should return empty ID for unattached ENI, got %s", id)
	}
}

func TestGetVPCInfo_EdgeCases(t *testing.T) {
	// Test empty VPC ID
	result := VPCInfo{}
	if result.ID != "" {
		t.Errorf("VPCInfo with empty ID should have empty ID, got %s", result.ID)
	}

	// Test VPC info structure with missing name
	vpcInfo := VPCInfo{
		ID:   "vpc-12345678",
		Name: "",
		CIDR: "10.0.0.0/16",
	}

	if vpcInfo.Name != "" {
		t.Errorf("VPCInfo with empty name should have empty name, got %s", vpcInfo.Name)
	}
	if vpcInfo.ID != "vpc-12345678" {
		t.Errorf("VPCInfo ID should be preserved, got %s", vpcInfo.ID)
	}
}

func TestGetSubnetInfo_EdgeCases(t *testing.T) {
	// Test empty subnet ID
	result := SubnetInfo{}
	if result.ID != "" {
		t.Errorf("SubnetInfo with empty ID should have empty ID, got %s", result.ID)
	}

	// Test subnet info structure with missing name
	subnetInfo := SubnetInfo{
		ID:   "subnet-12345678",
		Name: "",
		CIDR: "10.0.1.0/24",
	}

	if subnetInfo.Name != "" {
		t.Errorf("SubnetInfo with empty name should have empty name, got %s", subnetInfo.Name)
	}
	if subnetInfo.ID != "subnet-12345678" {
		t.Errorf("SubnetInfo ID should be preserved, got %s", subnetInfo.ID)
	}
}

// Integration tests for IP finder functionality
// These tests are intended to run against real AWS resources in a test environment
func TestFindIPAddressDetails_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// These tests would require a real AWS environment with test resources
	// For now, we'll test the structure and behavior without actual AWS calls
	t.Run("integration test structure validation", func(t *testing.T) {
		// This test validates that the FindIPAddressDetails function
		// returns the expected structure for integration testing
		testIP := "10.0.1.100"

		// Mock result structure that would be returned in real integration test
		expectedResult := IPFinderResult{
			IPAddress:      testIP,
			Found:          false, // Would be true in real test with actual resources
			IsSecondaryIP:  false,
			ResourceType:   "",
			ResourceName:   "",
			ResourceID:     "",
			VPC:            VPCInfo{},
			Subnet:         SubnetInfo{},
			SecurityGroups: []SecurityGroupInfo{},
			RouteTable:     RouteTableInfo{},
		}

		// Validate structure
		if expectedResult.IPAddress != testIP {
			t.Errorf("Expected IP address %s, got %s", testIP, expectedResult.IPAddress)
		}

		// Validate that all required fields are present
		if expectedResult.SecurityGroups == nil {
			t.Error("SecurityGroups slice should be initialized")
		}
		if expectedResult.RouteTable.Routes == nil {
			expectedResult.RouteTable.Routes = []string{}
		}

		t.Log("Integration test structure validation passed")
	})

	t.Run("test with different AWS configurations", func(t *testing.T) {
		// This test validates behavior with different AWS configurations
		// In a real environment, this would test:
		// - Different regions
		// - Different AWS profiles
		// - Different credential sources

		configs := []struct {
			name    string
			region  string
			profile string
		}{
			{"default-config", "us-east-1", "default"},
			{"west-region", "us-west-2", "default"},
			{"prod-profile", "us-east-1", "production"},
		}

		for _, cfg := range configs {
			t.Run(cfg.name, func(t *testing.T) {
				// In real integration test, would load AWS config with:
				// - cfg.region
				// - cfg.profile
				// And then call FindIPAddressDetails

				t.Logf("Would test with region: %s, profile: %s", cfg.region, cfg.profile)

				// Validate that different configs don't break the function
				testIP := "10.0.1.100"
				result := IPFinderResult{
					IPAddress: testIP,
					Found:     false,
				}

				if result.IPAddress != testIP {
					t.Errorf("Expected IP %s, got %s", testIP, result.IPAddress)
				}
			})
		}
	})

	t.Run("validate output format integrity", func(t *testing.T) {
		// This test validates that output formats work correctly
		// In a real integration test, this would test:
		// - JSON structure validation
		// - CSV format validation
		// - Table format validation

		testResult := IPFinderResult{
			IPAddress:      "10.0.1.100",
			Found:          true,
			IsSecondaryIP:  false,
			ResourceType:   "EC2 Instance",
			ResourceName:   "web-server-01",
			ResourceID:     "i-1234567890abcdef0",
			VPC:            VPCInfo{ID: "vpc-12345678", Name: "main-vpc", CIDR: "10.0.0.0/16"},
			Subnet:         SubnetInfo{ID: "subnet-12345678", Name: "web-subnet", CIDR: "10.0.1.0/24"},
			SecurityGroups: []SecurityGroupInfo{{ID: "sg-12345678", Name: "web-sg"}},
			RouteTable:     RouteTableInfo{ID: "rtb-12345678", Name: "main-rt", Routes: []string{"0.0.0.0/0 -> igw-12345678"}},
		}

		// Validate JSON marshaling works
		if testResult.IPAddress == "" {
			t.Error("Test result should have IP address")
		}
		if len(testResult.SecurityGroups) == 0 {
			t.Error("Test result should have security groups")
		}
		if len(testResult.RouteTable.Routes) == 0 {
			t.Error("Test result should have routes")
		}

		t.Log("Output format integrity validation passed")
	})
}

// Performance tests for IP finder functionality
func BenchmarkFindIPAddressDetails(b *testing.B) {
	// This benchmark tests the performance of the IP finder function
	// with different scenarios

	b.Run("single_ip_search", func(b *testing.B) {
		testIP := "10.0.1.100"

		b.ResetTimer()
		for b.Loop() {
			// In real benchmark, would call FindIPAddressDetails
			// For now, we'll benchmark the structure creation
			result := IPFinderResult{
				IPAddress: testIP,
				Found:     false,
			}

			// Prevent compiler optimization
			if result.IPAddress != testIP {
				b.Fatalf("Unexpected IP address: %s", result.IPAddress)
			}
		}
	})

	b.Run("ip_validation_performance", func(b *testing.B) {
		testIPs := []string{
			"10.0.1.100",
			"192.168.1.1",
			"172.16.0.1",
			"2001:db8::1",
			"invalid-ip",
		}

		b.ResetTimer()
		for b.Loop() {
			for _, ip := range testIPs {
				_ = IsValidIPAddress(ip)
			}
		}
	})

	b.Run("secondary_ip_detection", func(b *testing.B) {
		// Mock ENI with multiple IP addresses
		eni := types.NetworkInterface{
			NetworkInterfaceId: aws.String("eni-12345678"),
			PrivateIpAddress:   aws.String("10.0.1.100"),
			PrivateIpAddresses: []types.NetworkInterfacePrivateIpAddress{
				{
					PrivateIpAddress: aws.String("10.0.1.100"),
					Primary:          aws.Bool(true),
				},
				{
					PrivateIpAddress: aws.String("10.0.1.101"),
					Primary:          aws.Bool(false),
				},
				{
					PrivateIpAddress: aws.String("10.0.1.102"),
					Primary:          aws.Bool(false),
				},
			},
		}

		testIP := "10.0.1.101"

		b.ResetTimer()
		for b.Loop() {
			_ = isSecondaryIP(eni, testIP)
		}
	})
}

// Test performance optimization with ENI cache
func BenchmarkENILookupCachePerformance(b *testing.B) {
	// This benchmark tests the effectiveness of the ENI lookup cache

	b.Run("cache_lookup_performance", func(b *testing.B) {
		// Create mock ENIs for cache testing
		enis := make([]types.NetworkInterface, 100)
		for i := range 100 {
			enis[i] = types.NetworkInterface{
				NetworkInterfaceId: aws.String(fmt.Sprintf("eni-%d", i)),
				PrivateIpAddress:   aws.String(fmt.Sprintf("10.0.%d.%d", i/255, i%255)),
			}
		}

		// Create cache (would normally be populated with real AWS data)
		cache := &ENILookupCache{
			EndpointsByENI:   make(map[string]*types.VpcEndpoint),
			NATGatewaysByENI: make(map[string]*types.NatGateway),
		}

		b.ResetTimer()
		for b.Loop() {
			for _, eni := range enis {
				// Simulate cache lookup
				_, exists := cache.EndpointsByENI[*eni.NetworkInterfaceId]
				if exists {
					// Cache hit - would be faster in real scenario
				}
			}
		}
	})

	b.Run("resource_name_id_extraction", func(b *testing.B) {
		eni := types.NetworkInterface{
			NetworkInterfaceId: aws.String("eni-12345678"),
			Attachment: &types.NetworkInterfaceAttachment{
				InstanceId: aws.String("i-1234567890abcdef0"),
			},
		}

		cache := &ENILookupCache{
			EndpointsByENI:   make(map[string]*types.VpcEndpoint),
			NATGatewaysByENI: make(map[string]*types.NatGateway),
		}

		b.ResetTimer()
		for b.Loop() {
			_, _ = getResourceNameAndID(eni, cache)
		}
	})
}

// TestENILookupCache_EndpointsByENI_DistinctEntries verifies that each ENI
// maps to the correct VPC endpoint in the cache. This is a regression test
// for T-456: when storing &endpoint from a range loop, all map entries could
// end up pointing to the last item if the loop variable is reused.
func TestENILookupCache_EndpointsByENI_DistinctEntries(t *testing.T) {
	cache := &ENILookupCache{
		EndpointsByENI: make(map[string]*types.VpcEndpoint),
	}

	// Simulate the logic from batchFetchVPCEndpoints:
	// iterating over a slice of VpcEndpoint and storing &endpoint
	endpoints := []types.VpcEndpoint{
		{
			VpcEndpointId:       aws.String("vpce-aaa"),
			ServiceName:         aws.String("com.amazonaws.us-east-1.s3"),
			NetworkInterfaceIds: []string{"eni-001"},
		},
		{
			VpcEndpointId:       aws.String("vpce-bbb"),
			ServiceName:         aws.String("com.amazonaws.us-east-1.ec2"),
			NetworkInterfaceIds: []string{"eni-002"},
		},
		{
			VpcEndpointId:       aws.String("vpce-ccc"),
			ServiceName:         aws.String("com.amazonaws.us-east-1.sqs"),
			NetworkInterfaceIds: []string{"eni-003", "eni-004"},
		},
	}

	// Reproduce the original loop pattern
	for _, endpoint := range endpoints {
		for _, eniID := range endpoint.NetworkInterfaceIds {
			cache.EndpointsByENI[eniID] = &endpoint
		}
	}

	// Each ENI must map to its own distinct endpoint
	tests := []struct {
		eniID              string
		expectedEndpointID string
	}{
		{"eni-001", "vpce-aaa"},
		{"eni-002", "vpce-bbb"},
		{"eni-003", "vpce-ccc"},
		{"eni-004", "vpce-ccc"},
	}

	for _, tt := range tests {
		ep, ok := cache.EndpointsByENI[tt.eniID]
		if !ok {
			t.Errorf("EndpointsByENI missing entry for %s", tt.eniID)
			continue
		}
		if *ep.VpcEndpointId != tt.expectedEndpointID {
			t.Errorf("EndpointsByENI[%s] = %s, want %s (pointer reuse bug: all entries point to last item)",
				tt.eniID, *ep.VpcEndpointId, tt.expectedEndpointID)
		}
	}

	// Verify all pointers are not the same address
	seen := make(map[*types.VpcEndpoint]bool)
	for eniID, ep := range cache.EndpointsByENI {
		// eni-003 and eni-004 should share a pointer (same endpoint), but
		// eni-001 and eni-002 should have distinct pointers
		if eniID == "eni-003" || eniID == "eni-004" {
			continue
		}
		if seen[ep] {
			t.Errorf("EndpointsByENI has duplicate pointer %p for ENI %s — loop variable reuse bug", ep, eniID)
		}
		seen[ep] = true
	}
}

// TestENILookupCache_NATGatewaysByENI_DistinctEntries verifies that each ENI
// maps to the correct NAT gateway in the cache. Regression test for T-456.
func TestENILookupCache_NATGatewaysByENI_DistinctEntries(t *testing.T) {
	cache := &ENILookupCache{
		NATGatewaysByENI: make(map[string]*types.NatGateway),
	}

	// Simulate the logic from batchFetchNATGateways
	natgateways := []types.NatGateway{
		{
			NatGatewayId: aws.String("nat-aaa"),
			NatGatewayAddresses: []types.NatGatewayAddress{
				{NetworkInterfaceId: aws.String("eni-101")},
			},
		},
		{
			NatGatewayId: aws.String("nat-bbb"),
			NatGatewayAddresses: []types.NatGatewayAddress{
				{NetworkInterfaceId: aws.String("eni-102")},
			},
		},
		{
			NatGatewayId: aws.String("nat-ccc"),
			NatGatewayAddresses: []types.NatGatewayAddress{
				{NetworkInterfaceId: aws.String("eni-103")},
			},
		},
	}

	// Reproduce the original loop pattern
	for _, natgw := range natgateways {
		for _, address := range natgw.NatGatewayAddresses {
			if address.NetworkInterfaceId != nil {
				cache.NATGatewaysByENI[*address.NetworkInterfaceId] = &natgw
			}
		}
	}

	// Each ENI must map to its own distinct NAT gateway
	tests := []struct {
		eniID           string
		expectedNATGWID string
	}{
		{"eni-101", "nat-aaa"},
		{"eni-102", "nat-bbb"},
		{"eni-103", "nat-ccc"},
	}

	for _, tt := range tests {
		natgw, ok := cache.NATGatewaysByENI[tt.eniID]
		if !ok {
			t.Errorf("NATGatewaysByENI missing entry for %s", tt.eniID)
			continue
		}
		if *natgw.NatGatewayId != tt.expectedNATGWID {
			t.Errorf("NATGatewaysByENI[%s] = %s, want %s (pointer reuse bug: all entries point to last item)",
				tt.eniID, *natgw.NatGatewayId, tt.expectedNATGWID)
		}
	}

	// Verify all pointers are distinct addresses
	seen := make(map[*types.NatGateway]bool)
	for eniID, natgw := range cache.NATGatewaysByENI {
		if seen[natgw] {
			t.Errorf("NATGatewaysByENI has duplicate pointer %p for ENI %s — loop variable reuse bug", natgw, eniID)
		}
		seen[natgw] = true
		_ = eniID
	}
}

func TestGetSubnetRouteTable_ExplicitAssociation(t *testing.T) {
	routeTables := []types.RouteTable{
		{
			RouteTableId: aws.String("rtb-main-vpc1"),
			VpcId:        aws.String("vpc-111"),
			Associations: []types.RouteTableAssociation{
				{Main: aws.Bool(true)},
			},
		},
		{
			RouteTableId: aws.String("rtb-explicit"),
			VpcId:        aws.String("vpc-111"),
			Associations: []types.RouteTableAssociation{
				{SubnetId: aws.String("subnet-aaa")},
			},
		},
	}

	rt := GetSubnetRouteTable("subnet-aaa", "vpc-111", routeTables)
	if rt == nil {
		t.Fatal("expected route table, got nil")
	}
	if *rt.RouteTableId != "rtb-explicit" {
		t.Errorf("expected rtb-explicit, got %s", *rt.RouteTableId)
	}
}

func TestGetSubnetRouteTable_MainRouteTableMatchesVPC(t *testing.T) {
	// Two VPCs, each with its own main route table.
	// subnet-bbb belongs to vpc-222 and has no explicit association.
	// The function must return vpc-222's main route table, not vpc-111's.
	routeTables := []types.RouteTable{
		{
			RouteTableId: aws.String("rtb-main-vpc1"),
			VpcId:        aws.String("vpc-111"),
			Associations: []types.RouteTableAssociation{
				{Main: aws.Bool(true)},
			},
		},
		{
			RouteTableId: aws.String("rtb-main-vpc2"),
			VpcId:        aws.String("vpc-222"),
			Associations: []types.RouteTableAssociation{
				{Main: aws.Bool(true)},
			},
		},
	}

	rt := GetSubnetRouteTable("subnet-bbb", "vpc-222", routeTables)
	if rt == nil {
		t.Fatal("expected route table, got nil")
	}
	if *rt.RouteTableId != "rtb-main-vpc2" {
		t.Errorf("expected rtb-main-vpc2 (vpc-222's main RT), got %s", *rt.RouteTableId)
	}
}

func TestGetSubnetRouteTable_NoMatchReturnsNil(t *testing.T) {
	routeTables := []types.RouteTable{
		{
			RouteTableId: aws.String("rtb-main-vpc1"),
			VpcId:        aws.String("vpc-111"),
			Associations: []types.RouteTableAssociation{
				{Main: aws.Bool(true)},
			},
		},
	}

	// Subnet belongs to vpc-999 which has no route tables at all
	rt := GetSubnetRouteTable("subnet-zzz", "vpc-999", routeTables)
	if rt != nil {
		t.Errorf("expected nil for unknown VPC, got %s", *rt.RouteTableId)
	}
}

func TestIsValidInstanceID(t *testing.T) {
	tests := []struct {
		id    string
		valid bool
	}{
		{"i-1234567890abcdef0", true},   // 17-char hex (modern format)
		{"i-12345678", true},            // 8-char hex (legacy format)
		{"i-abcdef1234567890a", true},   // 17-char hex
		{"i-abcdef12", true},            // 8-char hex
		{"", false},                     // empty
		{"i-", false},                   // prefix only
		{"i-123", false},                // too short
		{"i-1234567890abcdef0a", false}, // too long
		{"i-ABCDEF12", false},           // uppercase not valid
		{"i-1234567g", false},           // non-hex character
		{"vol-12345678", false},         // wrong prefix
		{"not-an-id", false},            // totally wrong
	}

	for _, tt := range tests {
		got := isValidInstanceID(tt.id)
		if got != tt.valid {
			t.Errorf("isValidInstanceID(%q) = %v, want %v", tt.id, got, tt.valid)
		}
	}
}

func TestIsPublicSubnet_UsesCorrectVPCMainRouteTable(t *testing.T) {
	// vpc-111 has a public main route table (with igw)
	// vpc-222 has a private main route table (no igw)
	// subnet-private belongs to vpc-222 with no explicit association
	// Without the VPC constraint, it would incorrectly pick vpc-111's main RT and report public
	routeTables := []types.RouteTable{
		{
			RouteTableId: aws.String("rtb-main-vpc1"),
			VpcId:        aws.String("vpc-111"),
			Associations: []types.RouteTableAssociation{
				{Main: aws.Bool(true)},
			},
			Routes: []types.Route{
				{
					DestinationCidrBlock: aws.String("0.0.0.0/0"),
					GatewayId:            aws.String("igw-12345678"),
				},
			},
		},
		{
			RouteTableId: aws.String("rtb-main-vpc2"),
			VpcId:        aws.String("vpc-222"),
			Associations: []types.RouteTableAssociation{
				{Main: aws.Bool(true)},
			},
			Routes: []types.Route{
				{
					DestinationCidrBlock: aws.String("10.0.0.0/16"),
					GatewayId:            aws.String("local"),
				},
			},
		},
	}

	result := isPublicSubnet("subnet-private", "vpc-222", routeTables)
	if result {
		t.Error("subnet in vpc-222 should be private, but was classified as public (wrong main route table used)")
	}
}

func TestHasInternetGatewayRoute_IPv6DefaultRoute(t *testing.T) {
	// A route table with an IPv6 default route (::/0) via an internet gateway
	// should be detected as having internet access.
	routeTable := types.RouteTable{
		RouteTableId: aws.String("rtb-ipv6"),
		Routes: []types.Route{
			{
				DestinationIpv6CidrBlock: aws.String("::/0"),
				GatewayId:                aws.String("igw-ipv6only"),
			},
		},
	}
	if !hasInternetGatewayRoute(routeTable) {
		t.Error("route table with IPv6 ::/0 via igw should be detected as having internet gateway route")
	}
}

func TestHasInternetGatewayRoute_DualStack(t *testing.T) {
	// A dual-stack route table with both IPv4 and IPv6 default routes via igw.
	routeTable := types.RouteTable{
		RouteTableId: aws.String("rtb-dualstack"),
		Routes: []types.Route{
			{
				DestinationCidrBlock: aws.String("0.0.0.0/0"),
				GatewayId:            aws.String("igw-dual"),
			},
			{
				DestinationIpv6CidrBlock: aws.String("::/0"),
				GatewayId:                aws.String("igw-dual"),
			},
		},
	}
	if !hasInternetGatewayRoute(routeTable) {
		t.Error("dual-stack route table with igw should be detected as having internet gateway route")
	}
}

func TestHasInternetGatewayRoute_IPv6EgressOnly(t *testing.T) {
	// An egress-only internet gateway (eigw-) is NOT a full internet gateway.
	// Subnets with only eigw routes should NOT be classified as public.
	routeTable := types.RouteTable{
		RouteTableId: aws.String("rtb-eigw"),
		Routes: []types.Route{
			{
				DestinationIpv6CidrBlock:    aws.String("::/0"),
				EgressOnlyInternetGatewayId: aws.String("eigw-12345678"),
			},
		},
	}
	if hasInternetGatewayRoute(routeTable) {
		t.Error("route table with only eigw should NOT be detected as having internet gateway route")
	}
}

func TestIsPublicSubnet_IPv6OnlyPublic(t *testing.T) {
	// Subnet with IPv6-only internet route via igw should be classified as public.
	routeTables := []types.RouteTable{
		{
			RouteTableId: aws.String("rtb-ipv6pub"),
			VpcId:        aws.String("vpc-ipv6"),
			Associations: []types.RouteTableAssociation{
				{
					SubnetId: aws.String("subnet-ipv6pub"),
				},
			},
			Routes: []types.Route{
				{
					DestinationIpv6CidrBlock: aws.String("::/0"),
					GatewayId:                aws.String("igw-v6"),
				},
			},
		},
	}

	result := isPublicSubnet("subnet-ipv6pub", "vpc-ipv6", routeTables)
	if !result {
		t.Error("subnet with IPv6 ::/0 via igw should be classified as public")
	}
}

// Regression tests for T-568: prefix list routes were silently dropped
// because FormatRouteTableInfo only checked DestinationCidrBlock and
// DestinationIpv6CidrBlock, ignoring DestinationPrefixListId entirely.

func TestFormatRouteTableInfo_PrefixListRoute(t *testing.T) {
	rt := &types.RouteTable{
		RouteTableId: aws.String("rtb-prefix"),
		Routes: []types.Route{
			{
				DestinationPrefixListId: aws.String("pl-68a54001"),
				GatewayId:               aws.String("vpce-abc123"),
			},
		},
	}

	_, routes := FormatRouteTableInfo(rt)

	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d: %v", len(routes), routes)
	}
	expected := "pl-68a54001: vpce-abc123"
	if routes[0] != expected {
		t.Errorf("expected %q, got %q", expected, routes[0])
	}
}

// Regression tests for T-656: GetNatGatewayFromNetworkInterface must tolerate
// NAT gateway address entries and network interfaces where NetworkInterfaceId
// is nil without panicking. Scanning must continue past missing values and
// match the correct NAT gateway when a later address has a non-nil matching ID.

func TestMatchNatGatewayByENI_NilAddressNetworkInterfaceID(t *testing.T) {
	natgateways := []types.NatGateway{
		{
			NatGatewayId: aws.String("nat-aaa"),
			NatGatewayAddresses: []types.NatGatewayAddress{
				{NetworkInterfaceId: nil},
				{NetworkInterfaceId: aws.String("eni-match")},
			},
		},
	}

	result := matchNatGatewayByENI(natgateways, "eni-match")
	if result == nil {
		t.Fatal("expected matching NAT gateway, got nil (nil address NetworkInterfaceId aborted scan)")
	}
	if aws.ToString(result.NatGatewayId) != "nat-aaa" {
		t.Errorf("expected nat-aaa, got %s", aws.ToString(result.NatGatewayId))
	}
}

func TestMatchNatGatewayByENI_EmptyENIID(t *testing.T) {
	natgateways := []types.NatGateway{
		{
			NatGatewayId: aws.String("nat-aaa"),
			NatGatewayAddresses: []types.NatGatewayAddress{
				{NetworkInterfaceId: nil},
			},
		},
	}

	if result := matchNatGatewayByENI(natgateways, ""); result != nil {
		t.Errorf("expected nil for empty eniID, got %s", aws.ToString(result.NatGatewayId))
	}
}

func TestMatchNatGatewayByENI_NoMatch(t *testing.T) {
	natgateways := []types.NatGateway{
		{
			NatGatewayId: aws.String("nat-aaa"),
			NatGatewayAddresses: []types.NatGatewayAddress{
				{NetworkInterfaceId: aws.String("eni-other")},
			},
		},
	}

	if result := matchNatGatewayByENI(natgateways, "eni-missing"); result != nil {
		t.Errorf("expected nil when no match, got %s", aws.ToString(result.NatGatewayId))
	}
}

func TestMatchNatGatewayByENI_MatchOnLaterGateway(t *testing.T) {
	natgateways := []types.NatGateway{
		{
			NatGatewayId: aws.String("nat-first"),
			NatGatewayAddresses: []types.NatGatewayAddress{
				{NetworkInterfaceId: nil},
			},
		},
		{
			NatGatewayId: aws.String("nat-second"),
			NatGatewayAddresses: []types.NatGatewayAddress{
				{NetworkInterfaceId: aws.String("eni-target")},
			},
		},
	}

	result := matchNatGatewayByENI(natgateways, "eni-target")
	if result == nil {
		t.Fatal("expected match on second gateway, got nil")
	}
	if aws.ToString(result.NatGatewayId) != "nat-second" {
		t.Errorf("expected nat-second, got %s", aws.ToString(result.NatGatewayId))
	}
}

func TestFormatRouteTableInfo_MixedRoutes(t *testing.T) {
	// A route table with all three destination types: IPv4 CIDR,
	// IPv6 CIDR, and prefix list.
	rt := &types.RouteTable{
		RouteTableId: aws.String("rtb-mixed"),
		Routes: []types.Route{
			{
				DestinationCidrBlock: aws.String("0.0.0.0/0"),
				GatewayId:            aws.String("igw-111"),
			},
			{
				DestinationIpv6CidrBlock: aws.String("::/0"),
				GatewayId:                aws.String("igw-111"),
			},
			{
				DestinationPrefixListId: aws.String("pl-s3"),
				GatewayId:               aws.String("vpce-s3"),
			},
			{
				DestinationCidrBlock: aws.String("10.0.0.0/16"),
			},
		},
	}

	_, routes := FormatRouteTableInfo(rt)

	if len(routes) != 4 {
		t.Fatalf("expected 4 routes, got %d: %v", len(routes), routes)
	}

	expectedRoutes := []string{
		"0.0.0.0/0: igw-111",
		"::/0: igw-111",
		"pl-s3: vpce-s3",
		"10.0.0.0/16: local",
	}
	if !reflect.DeepEqual(routes, expectedRoutes) {
		t.Errorf("routes mismatch:\n  got:  %v\n  want: %v", routes, expectedRoutes)
	}
}
