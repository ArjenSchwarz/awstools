package helpers

import (
	"reflect"
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
