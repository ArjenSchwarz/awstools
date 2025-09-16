package helpers

import (
	"context"
	"fmt"
	"net"
	"slices"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// ENI and service type constants
const (
	vpcEndpointType    = "VPC Endpoint"
	interfaceType      = "interface"
	lambdaFunctionType = "Lambda Function"
	awsServiceType     = "AWS Service"
)

// GetEc2Name returns the name of the provided EC2 Resource
func GetEc2Name(ec2name string, svc *ec2.Client) string {
	params := &ec2.DescribeInstancesInput{
		InstanceIds: []string{ec2name},
	}
	resp, err := svc.DescribeInstances(context.TODO(), params)

	if err != nil {
		panic(err)
	}

	for _, reservation := range resp.Reservations {
		for _, instance := range reservation.Instances {
			return getNameFromTags(instance.Tags)
		}
	}
	return ""
}

// GetAllSecurityGroups returns a list of all securitygroups in the region
func GetAllSecurityGroups(svc *ec2.Client) []types.SecurityGroup {
	resp, err := svc.DescribeSecurityGroups(context.TODO(), &ec2.DescribeSecurityGroupsInput{})
	if err != nil {
		panic(err)
	}

	return resp.SecurityGroups
}

// GetEc2BySecurityGroup retrieves all instances attached to a securitygroup
func GetEc2BySecurityGroup(securitygroupID *string, svc *ec2.Client) []types.Reservation {
	input := &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("instance.group-id"),
				Values: []string{*securitygroupID},
			},
		},
	}
	resp, err := svc.DescribeInstances(context.TODO(), input)
	if err != nil {
		panic(err)
	}

	return resp.Reservations
}

// GetAllEc2Instances retrieves all EC2 instances
func GetAllEc2Instances(svc *ec2.Client) []types.Reservation {
	resp, err := svc.DescribeInstances(context.TODO(), &ec2.DescribeInstancesInput{})
	if err != nil {
		panic(err)
	}

	return resp.Reservations
}

// GetAllEC2ResourceNames retrieves the names of EC2 related objects
func GetAllEC2ResourceNames(svc *ec2.Client) map[string]string {
	result := make(map[string]string)
	result = addAllVPCNames(svc, result)
	result = addAllPeerNames(svc, result)
	result = addAllSubnetNames(svc, result)
	result = addAllRouteTableNames(svc, result)
	result = addAllTransitGatewayNames(svc, result)
	result = addAllVpnNames(svc, result)
	return result
}

// addAllVPCNames returns the names of all vpcs in a map
func addAllVPCNames(svc *ec2.Client, result map[string]string) map[string]string {
	resp, err := svc.DescribeVpcs(context.TODO(), &ec2.DescribeVpcsInput{})
	if err != nil {
		panic(err)
	}
	for _, vpc := range resp.Vpcs {
		result[*vpc.VpcId] = *vpc.VpcId
		if name := getNameFromTags(vpc.Tags); name != "" {
			result[*vpc.VpcId] = name
		}
	}
	return result
}

func addAllPeerNames(svc *ec2.Client, result map[string]string) map[string]string {
	resp, err := svc.DescribeVpcPeeringConnections(context.TODO(), &ec2.DescribeVpcPeeringConnectionsInput{})
	if err != nil {
		panic(err)
	}
	for _, peer := range resp.VpcPeeringConnections {
		result[*peer.VpcPeeringConnectionId] = *peer.VpcPeeringConnectionId
		if name := getNameFromTags(peer.Tags); name != "" {
			result[*peer.VpcPeeringConnectionId] = name
		}
	}
	return result
}

func addAllSubnetNames(svc *ec2.Client, result map[string]string) map[string]string {
	resp, err := svc.DescribeSubnets(context.TODO(), &ec2.DescribeSubnetsInput{})
	if err != nil {
		panic(err)
	}
	for _, subnet := range resp.Subnets {
		result[*subnet.SubnetId] = *subnet.SubnetId
		if name := getNameFromTags(subnet.Tags); name != "" {
			result[*subnet.SubnetId] = name
		}
	}
	return result
}

func addAllRouteTableNames(svc *ec2.Client, result map[string]string) map[string]string {
	resp, err := svc.DescribeRouteTables(context.TODO(), &ec2.DescribeRouteTablesInput{})
	if err != nil {
		panic(err)
	}
	for _, resource := range resp.RouteTables {
		result[*resource.RouteTableId] = *resource.RouteTableId
		if name := getNameFromTags(resource.Tags); name != "" {
			result[*resource.RouteTableId] = name
		}
	}
	return result
}

func addAllTransitGatewayNames(svc *ec2.Client, result map[string]string) map[string]string {
	tgws := GetAllTransitGateways(svc)
	for _, tgw := range tgws {
		result[tgw.ID] = tgw.Name
		for _, rt := range tgw.RouteTables {
			result[rt.ID] = rt.Name
		}
	}
	return result
}

// addAllVPCNames returns the names of all vpns in a map
func addAllVpnNames(svc *ec2.Client, result map[string]string) map[string]string {
	resp, err := svc.DescribeVpnConnections(context.TODO(), &ec2.DescribeVpnConnectionsInput{})
	if err != nil {
		panic(err)
	}
	for _, vpn := range resp.VpnConnections {
		result[*vpn.VpnConnectionId] = *vpn.VpnConnectionId
		if name := getNameFromTags(vpn.Tags); name != "" {
			result[*vpn.VpnConnectionId] = name
		}
	}
	return result
}

// getResourceDisplayNameFromTags provides tiered name lookup for AWS resources
// 1. Checks Name tag on the resource
// 2. Falls back to the resource ID
// Returns either "Name (ID)" or just "ID" if no name found
func getResourceDisplayNameFromTags(resourceID string, tags []types.Tag) string {
	// Try tag-based name lookup first
	nameFromTags := getNameFromTags(tags)

	// Format the display name
	if nameFromTags != "" && nameFromTags != resourceID {
		return nameFromTags + " (" + resourceID + ")"
	}
	return resourceID
}

// GetResourceDisplayNameWithGlobalLookup provides comprehensive tiered name lookup for AWS resources
// 1. First tries global naming lookup using the provided lookup function
// 2. Then checks Name tag on the resource
// 3. Finally falls back to the resource ID
// Returns either "Name (ID)" or just "ID" if no name found
func GetResourceDisplayNameWithGlobalLookup(resourceID string, tags []types.Tag, globalLookupFunc func(string) string) string {
	// Try global lookup first (if provided)
	if globalLookupFunc != nil {
		nameFromGlobal := globalLookupFunc(resourceID)
		if nameFromGlobal != "" && nameFromGlobal != resourceID {
			return nameFromGlobal + " (" + resourceID + ")"
		}
	}

	// Try tag-based name lookup
	nameFromTags := getNameFromTags(tags)
	if nameFromTags != "" && nameFromTags != resourceID {
		return nameFromTags + " (" + resourceID + ")"
	}

	return resourceID
}

// VpcPeering represents a VPC Peering object
type VpcPeering struct {
	RequesterVpc VPCHolder
	AccepterVpc  VPCHolder
	PeeringID    string
}

// VPCHolder represents basic information about a VPC
type VPCHolder struct {
	ID        string
	AccountID string
}

// GetAllVpcPeers returns the peerings that are present in this region of this account
func GetAllVpcPeers(svc *ec2.Client) []VpcPeering {
	var result []VpcPeering
	resp, err := svc.DescribeVpcPeeringConnections(context.TODO(), &ec2.DescribeVpcPeeringConnectionsInput{})
	if err != nil {
		panic(err)
	}
	for _, connection := range resp.VpcPeeringConnections {
		peering := VpcPeering{
			RequesterVpc: VPCHolder{ID: *connection.RequesterVpcInfo.VpcId,
				AccountID: *connection.RequesterVpcInfo.OwnerId},
			AccepterVpc: VPCHolder{ID: *connection.AccepterVpcInfo.VpcId,
				AccountID: *connection.AccepterVpcInfo.OwnerId},
			PeeringID: *connection.VpcPeeringConnectionId,
		}
		result = append(result, peering)
	}
	return result
}

// VPCRouteTable contains the relevant information for a Route Table
type VPCRouteTable struct {
	Vpc     VPCHolder
	ID      string
	Routes  []VPCRoute
	Subnets []string
	Default bool
}

// VPCRoute represents a Route object
// DestinationTarget shows the target, regardless of the type
type VPCRoute struct {
	DestinationCIDR   string
	State             string
	DestinationTarget string
}

// GetAllVPCRouteTables returns all the Routetables in the account and region
func GetAllVPCRouteTables(svc *ec2.Client) []VPCRouteTable {
	var result []VPCRouteTable
	resp, err := svc.DescribeRouteTables(context.TODO(), &ec2.DescribeRouteTablesInput{})
	if err != nil {
		panic(err)
	}
	for _, routetable := range resp.RouteTables {
		var subnets []string
		for _, assocs := range routetable.Associations {
			if assocs.SubnetId != nil {
				subnets = append(subnets, *assocs.SubnetId)
			}
		}
		table := VPCRouteTable{
			Vpc: VPCHolder{ID: *routetable.VpcId,
				AccountID: *routetable.OwnerId},
			ID:      *routetable.RouteTableId,
			Routes:  parseVPCRoutes(routetable.Routes),
			Subnets: subnets,
		}
		result = append(result, table)
	}
	return result
}

func parseVPCRoutes(routes []types.Route) []VPCRoute {
	var result []VPCRoute
	for _, route := range routes {
		rt := VPCRoute{
			State: string(route.State),
		}
		if route.DestinationCidrBlock != nil {
			rt.DestinationCIDR = *route.DestinationCidrBlock
		}
		if route.DestinationIpv6CidrBlock != nil {
			rt.DestinationCIDR = *route.DestinationIpv6CidrBlock
		}
		if route.VpcPeeringConnectionId != nil {
			rt.DestinationTarget = *route.VpcPeeringConnectionId
		}
		if route.GatewayId != nil {
			rt.DestinationTarget = *route.GatewayId
		}
		if route.NatGatewayId != nil {
			rt.DestinationTarget = *route.NatGatewayId
		}
		if route.NetworkInterfaceId != nil {
			rt.DestinationTarget = *route.NetworkInterfaceId
		}
		if route.EgressOnlyInternetGatewayId != nil {
			rt.DestinationTarget = *route.EgressOnlyInternetGatewayId
		}
		if route.TransitGatewayId != nil {
			rt.DestinationTarget = *route.TransitGatewayId
		}
		result = append(result, rt)
	}
	return result
}

// TransitGateway is a struct for managing TransitGateway objects
type TransitGateway struct {
	ID          string
	AccountID   string
	Name        string
	RouteTables map[string]TransitGatewayRouteTable
}

// TransitGatewayRouteTable is a struct for managing Transit Gateway route table objects
type TransitGatewayRouteTable struct {
	ID                     string
	Name                   string
	Routes                 []TransitGatewayRoute
	SourceAttachments      []TransitGatewayAttachment
	DestinationAttachments []TransitGatewayAttachment
}

// TransitGatewayRoute reflects a Transit Gateway Route object
type TransitGatewayRoute struct {
	State        string
	CIDR         string
	Attachment   TransitGatewayAttachment
	ResourceType string
	RouteType    string
}

// TransitGatewayAttachment reflects a Transit Gateway Attachment
type TransitGatewayAttachment struct {
	ID           string
	ResourceType string
	ResourceID   string
}

// GetAllTransitGateways returns an array of all Transit Gateways in the account
func GetAllTransitGateways(svc *ec2.Client) []TransitGateway {
	var result []TransitGateway
	resp, err := svc.DescribeTransitGateways(context.TODO(), &ec2.DescribeTransitGatewaysInput{})
	if err != nil {
		panic(err)
	}
	for _, tgw := range resp.TransitGateways {
		tgwobject := TransitGateway{
			ID:          *tgw.TransitGatewayId,
			AccountID:   *tgw.OwnerId,
			Name:        getNameFromTags(tgw.Tags),
			RouteTables: GetRouteTablesForTransitGateway(*tgw.TransitGatewayId, svc),
		}
		result = append(result, tgwobject)
	}
	return result
}

// GetRouteTablesForTransitGateway returns all route tables attached to a Transit Gateway
func GetRouteTablesForTransitGateway(tgwID string, svc *ec2.Client) map[string]TransitGatewayRouteTable {
	result := make(map[string]TransitGatewayRouteTable)
	params := &ec2.DescribeTransitGatewayRouteTablesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("transit-gateway-id"),
				Values: []string{tgwID},
			},
		},
	}
	resp, err := svc.DescribeTransitGatewayRouteTables(context.TODO(), params)
	if err != nil {
		panic(err)
	}
	for _, table := range resp.TransitGatewayRouteTables {
		routetable := TransitGatewayRouteTable{
			ID:   *table.TransitGatewayRouteTableId,
			Name: getNameFromTags(table.Tags),
		}
		result[routetable.ID] = routetable
	}
	for _, routetable := range result {
		routetable.Routes = append(GetActiveRoutesForTransitGatewayRouteTable(routetable.ID, svc), GetBlackholeRoutesForTransitGatewayRouteTable(routetable.ID, svc)...)
		routetable.SourceAttachments = GetSourceAttachmentsForTransitGatewayRouteTable(routetable.ID, svc)
		result[routetable.ID] = routetable
	}
	return result
}

// GetSourceAttachmentsForTransitGatewayRouteTable returns all the source attachments attached to a Transit Gateway route table
func GetSourceAttachmentsForTransitGatewayRouteTable(routetableID string, svc *ec2.Client) []TransitGatewayAttachment {
	var result []TransitGatewayAttachment
	params := &ec2.GetTransitGatewayRouteTableAssociationsInput{
		TransitGatewayRouteTableId: &routetableID,
	}
	resp, err := svc.GetTransitGatewayRouteTableAssociations(context.TODO(), params)
	if err != nil {
		panic(err)
	}
	for _, attachment := range resp.Associations {
		tgwattachment := TransitGatewayAttachment{
			ID:           *attachment.TransitGatewayAttachmentId,
			ResourceID:   *attachment.ResourceId,
			ResourceType: string(attachment.ResourceType),
		}
		result = append(result, tgwattachment)
	}
	return result
}

// GetActiveRoutesForTransitGatewayRouteTable returns all routes that are currently active for a Transit Gateway route table
func GetActiveRoutesForTransitGatewayRouteTable(routetableID string, svc *ec2.Client) []TransitGatewayRoute {
	var result []TransitGatewayRoute
	desiredState := "active"
	params := &ec2.SearchTransitGatewayRoutesInput{
		TransitGatewayRouteTableId: &routetableID,
		Filters: []types.Filter{
			{
				Name:   aws.String("state"),
				Values: []string{desiredState},
			},
		},
	}
	resp, err := svc.SearchTransitGatewayRoutes(context.TODO(), params)
	if err != nil {
		panic(err)
	}
	for _, route := range resp.Routes {
		resourceid := *route.TransitGatewayAttachments[0].ResourceId
		// We don't care about the public IPs of the routes, so strip those off
		if route.TransitGatewayAttachments[0].ResourceType == types.TransitGatewayAttachmentResourceTypeVpn {
			resourceid = strings.Split(resourceid, "(")[0]
		}
		tgwroute := TransitGatewayRoute{
			State: string(route.State),
			CIDR:  *route.DestinationCidrBlock,
			Attachment: TransitGatewayAttachment{
				ID:         *route.TransitGatewayAttachments[0].TransitGatewayAttachmentId,
				ResourceID: resourceid,
			},
			RouteType: string(route.Type),
		}
		result = append(result, tgwroute)
	}
	return result
}

// GetBlackholeRoutesForTransitGatewayRouteTable returns all routes that are currently active for a Transit Gateway route table
func GetBlackholeRoutesForTransitGatewayRouteTable(routetableID string, svc *ec2.Client) []TransitGatewayRoute {
	var result []TransitGatewayRoute
	desiredState := "blackhole"
	params := &ec2.SearchTransitGatewayRoutesInput{
		TransitGatewayRouteTableId: &routetableID,
		Filters: []types.Filter{
			{
				Name:   aws.String("state"),
				Values: []string{desiredState},
			},
		},
	}
	resp, err := svc.SearchTransitGatewayRoutes(context.TODO(), params)
	if err != nil {
		panic(err)
	}
	for _, route := range resp.Routes {
		tgwroute := TransitGatewayRoute{
			State:     string(route.State),
			CIDR:      *route.DestinationCidrBlock,
			RouteType: string(route.Type),
		}
		result = append(result, tgwroute)
	}
	return result
}

// IsLatestInstanceFamily checks if an instance is part of the la
// test family is running in the latest instance family.
// TODO: Automate this to work properly
func IsLatestInstanceFamily(instanceFamily string) bool {
	family := instanceFamily[0:1]
	version := instanceFamily[1:]
	switch family {
	case "c":
		return version == "4"
	case "d":
		return version == "2"
	case "f":
		return version == "1"
	case "g":
		return version == "3"
	case "p":
		return version == "2"
	case "i":
		return version == "3"
	case "m":
		return version == "4"
	case "r":
		return version == "4"
	case "t":
		return version == "2"
	case "x":
		return version == "1"
	default:
		return false
	}
}

func getNameFromTags(tags []types.Tag) string {
	if tags == nil {
		return ""
	}
	for _, tag := range tags {
		if aws.ToString(tag.Key) == "Name" {
			name := aws.ToString(tag.Value)
			// Handle empty or whitespace-only names
			if strings.TrimSpace(name) == "" {
				return ""
			}
			// Truncate very long names for display purposes (keep up to 100 chars)
			if len(name) > 100 {
				return name[:97] + "..."
			}
			return name
		}
	}
	return ""
}

// GetNetworkInterfaces retrieves all network interfaces in the region
func GetNetworkInterfaces(svc *ec2.Client) []types.NetworkInterface {
	params := &ec2.DescribeNetworkInterfacesInput{}
	resp, err := svc.DescribeNetworkInterfaces(context.TODO(), params)
	if err != nil {
		panic(err)
	}
	return resp.NetworkInterfaces
}

// GetTransitGatewayFromNetworkInterface returns the Transit Gateway attachment ID for a network interface
func GetTransitGatewayFromNetworkInterface(netinterface types.NetworkInterface, svc *ec2.Client) string {
	params := &ec2.DescribeTransitGatewayVpcAttachmentsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{*netinterface.VpcId},
			},
		},
	}
	resp, err := svc.DescribeTransitGatewayVpcAttachments(context.Background(), params)
	if err != nil {
		panic(err)
	}
	if len(resp.TransitGatewayVpcAttachments) > 0 {
		if slices.Contains(resp.TransitGatewayVpcAttachments[0].SubnetIds, *netinterface.SubnetId) {
			return *resp.TransitGatewayVpcAttachments[0].TransitGatewayAttachmentId
		}
	}
	return ""
}

// GetVPCEndpointFromNetworkInterface returns the VPC endpoint associated with a network interface
func GetVPCEndpointFromNetworkInterface(netinterface types.NetworkInterface, svc *ec2.Client) *types.VpcEndpoint {
	// TODO: Consider caching this
	params := &ec2.DescribeVpcEndpointsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{*netinterface.VpcId},
			},
		},
	}
	resp, err := svc.DescribeVpcEndpoints(context.Background(), params)
	if err != nil {
		panic(err)
	}
	if len(resp.VpcEndpoints) > 0 {
		for _, endpoint := range resp.VpcEndpoints {
			if slices.Contains(endpoint.NetworkInterfaceIds, *netinterface.NetworkInterfaceId) {
				return &endpoint
			}
		}
	}
	return nil
}

// GetNatGatewayFromNetworkInterface returns the NAT gateway associated with a network interface
func GetNatGatewayFromNetworkInterface(netinterface types.NetworkInterface, svc *ec2.Client) *types.NatGateway {
	params := &ec2.DescribeNatGatewaysInput{
		Filter: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{*netinterface.VpcId},
			},
		},
	}
	resp, err := svc.DescribeNatGateways(context.Background(), params)
	if err != nil {
		panic(err)
	}
	if len(resp.NatGateways) > 0 {
		for _, natgw := range resp.NatGateways {
			for _, address := range natgw.NatGatewayAddresses {
				if *address.NetworkInterfaceId == *netinterface.NetworkInterfaceId {
					return &natgw
				}
			}
		}
	}
	return nil
}

// VPCOverview represents the complete VPC usage analysis
type VPCOverview struct {
	VPCs    []VPCUsageInfo  `json:"vpcs"`
	Summary VPCUsageSummary `json:"summary"`
}

// VPCUsageInfo contains detailed information about a single VPC
type VPCUsageInfo struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	CIDR    string            `json:"cidr"`
	Tags    []types.Tag       `json:"tags,omitempty"`
	Subnets []SubnetUsageInfo `json:"subnets"`
}

// SubnetUsageInfo contains detailed subnet usage information
type SubnetUsageInfo struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	CIDR         string          `json:"cidr"`
	VPCId        string          `json:"vpc_id"`
	VPCName      string          `json:"vpc_name"`
	Tags         []types.Tag     `json:"tags,omitempty"`
	IsPublic     bool            `json:"is_public"`
	TotalIPs     int             `json:"total_ips"`
	AvailableIPs int             `json:"available_ips"`
	UsedIPs      int             `json:"used_ips"`
	IPDetails    []IPAddressInfo `json:"ip_details,omitempty"`
}

// IPAddressInfo contains information about individual IP addresses
type IPAddressInfo struct {
	IPAddress      string `json:"ip_address"`
	UsageType      string `json:"usage_type"`
	AttachmentInfo string `json:"attachment_info"`
	PublicIP       string `json:"public_ip,omitempty"`
}

// VPCUsageSummary contains aggregate VPC usage statistics
type VPCUsageSummary struct {
	TotalVPCs      int `json:"total_vpcs"`
	TotalSubnets   int `json:"total_subnets"`
	TotalIPs       int `json:"total_ips"`
	UsedIPs        int `json:"used_ips"`
	AWSReservedIPs int `json:"aws_reserved_ips"`
	ServiceIPs     int `json:"service_ips"`
	AvailableIPs   int `json:"available_ips"`
}

// GetVPCUsageOverview retrieves comprehensive VPC usage information
func GetVPCUsageOverview(svc *ec2.Client) VPCOverview {
	vpcs := retrieveVPCData(svc)
	subnets := retrieveSubnetData(svc)
	networkInterfaces := retrieveNetworkInterfaces(svc)
	routeTables := retrieveRouteTables(svc)

	var vpcUsageInfos []VPCUsageInfo
	var summary VPCUsageSummary

	for _, vpc := range vpcs {
		vpcInfo := VPCUsageInfo{
			ID:   *vpc.VpcId,
			Name: getNameFromTags(vpc.Tags),
			CIDR: *vpc.CidrBlock,
			Tags: vpc.Tags,
		}

		var vpcSubnets []SubnetUsageInfo
		for _, subnet := range subnets {
			if *subnet.VpcId == *vpc.VpcId {
				// Calculate total IPs
				totalIPs, _, err := calculateSubnetStats(*subnet.CidrBlock)
				if err != nil {
					panic(err)
				}

				// Analyze detailed IP usage
				ipDetails, usedIPs, availableIPs, awsReservedIPs, serviceIPs, err := analyzeSubnetIPUsage(subnet, networkInterfaces, svc)
				if err != nil {
					panic(err)
				}

				subnetInfo := SubnetUsageInfo{
					ID:           *subnet.SubnetId,
					Name:         getNameFromTags(subnet.Tags),
					CIDR:         *subnet.CidrBlock,
					VPCId:        *subnet.VpcId,
					VPCName:      vpcInfo.Name,
					Tags:         subnet.Tags,
					IsPublic:     isPublicSubnet(*subnet.SubnetId, routeTables),
					TotalIPs:     totalIPs,
					AvailableIPs: availableIPs,
					UsedIPs:      usedIPs,
					IPDetails:    ipDetails,
				}
				vpcSubnets = append(vpcSubnets, subnetInfo)

				// Update summary
				summary.TotalIPs += totalIPs
				summary.UsedIPs += usedIPs
				summary.AvailableIPs += availableIPs
				summary.AWSReservedIPs += awsReservedIPs
				summary.ServiceIPs += serviceIPs
			}
		}
		vpcInfo.Subnets = vpcSubnets
		vpcUsageInfos = append(vpcUsageInfos, vpcInfo)
	}

	summary.TotalVPCs = len(vpcUsageInfos)
	for _, vpc := range vpcUsageInfos {
		summary.TotalSubnets += len(vpc.Subnets)
	}

	return VPCOverview{
		VPCs:    vpcUsageInfos,
		Summary: summary,
	}
}

// retrieveVPCData fetches all VPCs using DescribeVpcs API
func retrieveVPCData(svc *ec2.Client) []types.Vpc {
	resp, err := svc.DescribeVpcs(context.TODO(), &ec2.DescribeVpcsInput{})
	if err != nil {
		panic(err)
	}
	return resp.Vpcs
}

// retrieveSubnetData fetches all subnets using DescribeSubnets API
func retrieveSubnetData(svc *ec2.Client) []types.Subnet {
	resp, err := svc.DescribeSubnets(context.TODO(), &ec2.DescribeSubnetsInput{})
	if err != nil {
		panic(err)
	}
	return resp.Subnets
}

// retrieveNetworkInterfaces fetches all network interfaces using DescribeNetworkInterfaces API
func retrieveNetworkInterfaces(svc *ec2.Client) []types.NetworkInterface {
	resp, err := svc.DescribeNetworkInterfaces(context.TODO(), &ec2.DescribeNetworkInterfacesInput{})
	if err != nil {
		panic(err)
	}
	return resp.NetworkInterfaces
}

// retrieveRouteTables fetches all route tables using DescribeRouteTables API
func retrieveRouteTables(svc *ec2.Client) []types.RouteTable {
	resp, err := svc.DescribeRouteTables(context.TODO(), &ec2.DescribeRouteTablesInput{})
	if err != nil {
		panic(err)
	}
	return resp.RouteTables
}

// GetSubnetRouteTable finds the route table associated with a specific subnet
func GetSubnetRouteTable(subnetID string, routeTables []types.RouteTable) *types.RouteTable {
	// First check for explicit subnet associations
	for _, routeTable := range routeTables {
		for _, association := range routeTable.Associations {
			if association.SubnetId != nil && *association.SubnetId == subnetID {
				return &routeTable
			}
		}
	}

	// If no explicit association found, use the main route table
	for _, routeTable := range routeTables {
		for _, association := range routeTable.Associations {
			if association.Main != nil && *association.Main {
				return &routeTable
			}
		}
	}

	return nil
}

// FormatRouteTableInfo formats route table information similar to vpc routes command
func FormatRouteTableInfo(routeTable *types.RouteTable) (string, []string) {
	if routeTable == nil {
		return "No route table", []string{}
	}

	// Use the same tiered lookup as other resources
	rtDisplay := getResourceDisplayNameFromTags(*routeTable.RouteTableId, routeTable.Tags)

	var routeList []string
	for _, route := range routeTable.Routes {
		destCIDR := ""
		if route.DestinationCidrBlock != nil {
			destCIDR = *route.DestinationCidrBlock
		} else if route.DestinationIpv6CidrBlock != nil {
			destCIDR = *route.DestinationIpv6CidrBlock
		}

		target := ""
		switch {
		case route.GatewayId != nil:
			target = *route.GatewayId
		case route.NatGatewayId != nil:
			target = *route.NatGatewayId
		case route.VpcPeeringConnectionId != nil:
			target = *route.VpcPeeringConnectionId
		case route.NetworkInterfaceId != nil:
			target = *route.NetworkInterfaceId
		case route.TransitGatewayId != nil:
			target = *route.TransitGatewayId
		case route.EgressOnlyInternetGatewayId != nil:
			target = *route.EgressOnlyInternetGatewayId
		default:
			target = "local"
		}

		if destCIDR != "" && target != "" {
			routeList = append(routeList, destCIDR+": "+target)
		}
	}

	return rtDisplay, routeList
}

// isPublicSubnet determines if a subnet is public based on route table analysis
func isPublicSubnet(subnetID string, routeTables []types.RouteTable) bool {
	routeTable := GetSubnetRouteTable(subnetID, routeTables)
	if routeTable == nil {
		return false
	}

	// Check routes for internet gateway
	return hasInternetGatewayRoute(*routeTable)
}

// hasInternetGatewayRoute checks if a route table has a route to an internet gateway
func hasInternetGatewayRoute(routeTable types.RouteTable) bool {
	for _, route := range routeTable.Routes {
		// Check for internet gateway route
		if route.GatewayId != nil && strings.HasPrefix(*route.GatewayId, "igw-") {
			// Check if it's a default route (0.0.0.0/0) or covers broad ranges
			if route.DestinationCidrBlock != nil {
				cidr := *route.DestinationCidrBlock
				if cidr == "0.0.0.0/0" {
					return true
				}
			}
		}
	}
	return false
}

// parseCIDR parses a CIDR block and returns network information
func parseCIDR(cidr string) (*net.IPNet, error) {
	_, ipnet, err := net.ParseCIDR(cidr)
	return ipnet, err
}

// generateIPRange generates all IP addresses in a subnet CIDR block
func generateIPRange(cidr string) ([]net.IP, error) {
	ipnet, err := parseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	var ips []net.IP
	for ip := ipnet.IP.Mask(ipnet.Mask); ipnet.Contains(ip); incrementIP(ip) {
		ips = append(ips, net.IP(make([]byte, len(ip))))
		copy(ips[len(ips)-1], ip)
	}
	return ips, nil
}

// incrementIP increments an IP address by one
func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// calculateSubnetStats calculates total and available IP counts for a subnet
func calculateSubnetStats(cidr string) (int, int, error) {
	ipnet, err := parseCIDR(cidr)
	if err != nil {
		return 0, 0, err
	}

	ones, bits := ipnet.Mask.Size()
	totalIPs := 1 << uint(bits-ones)

	// AWS reserves first 4 and last IP in each subnet
	availableIPs := max(totalIPs-5, 0)

	return totalIPs, availableIPs, nil
}

// sortIPAddresses sorts IP addresses in ascending numerical order
func sortIPAddresses(ips []net.IP) {
	sort.Slice(ips, func(i, j int) bool {
		return compareIPs(ips[i], ips[j]) < 0
	})
}

// compareIPs compares two IP addresses for sorting
func compareIPs(ip1, ip2 net.IP) int {
	ip1 = ip1.To4()
	ip2 = ip2.To4()
	if ip1 == nil || ip2 == nil {
		return 0
	}

	for i := range 4 {
		if ip1[i] < ip2[i] {
			return -1
		}
		if ip1[i] > ip2[i] {
			return 1
		}
	}
	return 0
}

// identifyAWSReservedIPs identifies AWS reserved IP addresses in a subnet
// Returns a map with IP address as key and usage description as value
// Takes a pre-generated list of IPs to avoid duplicating IP range generation
func identifyAWSReservedIPs(ips []net.IP) map[string]string {
	reserved := make(map[string]string)
	if len(ips) == 0 {
		return reserved
	}

	// First IP (network address)
	reserved[ips[0].String()] = "Network address"

	// Second IP (VPC router)
	if len(ips) > 1 {
		reserved[ips[1].String()] = "VPC router"
	}

	// Third IP (DNS server)
	if len(ips) > 2 {
		reserved[ips[2].String()] = "DNS server"
	}

	// Fourth IP (future use)
	if len(ips) > 3 {
		reserved[ips[3].String()] = "Reserved for future use"
	}

	// Last IP (broadcast address)
	if len(ips) > 4 {
		reserved[ips[len(ips)-1].String()] = "Broadcast address"
	}

	return reserved
}

// mapNetworkInterfacesToIPs maps network interfaces to their IP addresses
// Optimized version that creates a lookup cache once to avoid N+1 API calls
// when processing many ENIs (e.g., DescribeVpcEndpoints, DescribeInstances, etc.)
func mapNetworkInterfacesToIPs(networkInterfaces []types.NetworkInterface, subnetID string, svc *ec2.Client) map[string]IPAddressInfo {
	// Create cache once for all ENI lookups to avoid N+1 API calls
	cache := NewENILookupCache(svc, networkInterfaces)

	ipMap := make(map[string]IPAddressInfo)

	for _, eni := range networkInterfaces {
		if eni.SubnetId != nil && *eni.SubnetId == subnetID {
			for _, privateIP := range eni.PrivateIpAddresses {
				if privateIP.PrivateIpAddress != nil {
					ipInfo := IPAddressInfo{
						IPAddress:      *privateIP.PrivateIpAddress,
						UsageType:      getENIUsageTypeOptimized(eni, cache),
						AttachmentInfo: getENIAttachmentDetailsOptimized(eni, cache),
					}

					// Add public IP if present
					if privateIP.Association != nil && privateIP.Association.PublicIp != nil {
						ipInfo.PublicIP = *privateIP.Association.PublicIp
					}

					ipMap[*privateIP.PrivateIpAddress] = ipInfo
				}
			}
		}
	}

	return ipMap
}

// getENIUsageTypeOptimized returns the general category of what the ENI is used for
// Uses cache to avoid repeated API calls
func getENIUsageTypeOptimized(eni types.NetworkInterface, cache *ENILookupCache) string {
	// Check if it's a VPC endpoint first (highest priority)
	if _, exists := cache.EndpointsByENI[*eni.NetworkInterfaceId]; exists {
		return vpcEndpointType
	}

	// Handle EC2 instances
	if eni.Attachment != nil && eni.Attachment.InstanceId != nil {
		return "EC2 Instance"
	}

	// Handle specific interface types (if not 'interface')
	if eni.InterfaceType != interfaceType {
		switch eni.InterfaceType {
		case types.NetworkInterfaceTypeTransitGateway:
			return "Transit Gateway"
		case types.NetworkInterfaceTypeNatGateway:
			return "NAT Gateway"
		case types.NetworkInterfaceTypeVpcEndpoint:
			return vpcEndpointType
		case types.NetworkInterfaceTypeLambda:
			return lambdaFunctionType
		case "quicksight":
			return "QuickSight"
		case "network_load_balancer":
			return "Network Load Balancer"
		case "gateway_load_balancer":
			return "Gateway Load Balancer"
		default:
			return awsServiceType
		}
	}

	// Handle by description for common services (following JS script logic)
	if eni.Description != nil {
		desc := strings.ToLower(*eni.Description)
		if strings.Contains(desc, "elb") {
			return "Load Balancer"
		}
		if strings.Contains(desc, "rds") {
			return "RDS Database"
		}
		if strings.Contains(desc, "lambda") {
			return lambdaFunctionType
		}
		if strings.Contains(desc, "vpc") {
			return "VPC Service"
		}
		if strings.Contains(desc, "elasticache") {
			return "ElastiCache"
		}
		if strings.Contains(desc, "efs") {
			return "EFS Mount Target"
		}
		if strings.Contains(desc, "redshift") {
			return "Redshift Cluster"
		}
		if strings.Contains(desc, "apigateway") {
			return "API Gateway"
		}
		if strings.Contains(desc, "codebuild") {
			return "CodeBuild"
		}
		// If description contains service keywords, it's likely a service
		return awsServiceType
	}

	// Unattached or unknown
	if eni.Attachment == nil {
		return "Unattached ENI"
	}

	return "Unknown"
}

// getENIAttachmentDetailsOptimized returns specific details about what the ENI is attached to
// Uses cache to avoid repeated API calls
func getENIAttachmentDetailsOptimized(eni types.NetworkInterface, cache *ENILookupCache) string {
	// Priority 1: Check if it's a VPC endpoint (following JS script logic)
	if endpoint, exists := cache.EndpointsByENI[*eni.NetworkInterfaceId]; exists {
		// Extract service name (last part after dots, like 's3', 'ec2')
		serviceParts := strings.Split(*endpoint.ServiceName, ".")
		shortServiceName := serviceParts[len(serviceParts)-1]
		return *endpoint.VpcEndpointId + " (" + shortServiceName + ")"
	}

	// Priority 2: Handle EC2 instances
	if eni.Attachment != nil && eni.Attachment.InstanceId != nil {
		instanceID := *eni.Attachment.InstanceId
		if instanceName, exists := cache.InstanceNames[instanceID]; exists && instanceName != "" && instanceName != instanceID {
			return instanceID + " (" + instanceName + ")"
		}
		return instanceID
	}

	// Priority 3: Handle specific interface types (if not 'interface' and not empty)
	if eni.InterfaceType != "" && eni.InterfaceType != interfaceType {
		switch eni.InterfaceType {
		case types.NetworkInterfaceTypeTransitGateway:
			if eni.VpcId != nil {
				if tgwID, exists := cache.TransitGateways[*eni.VpcId]; exists {
					return tgwID
				}
			}
			return "Unknown Transit Gateway"

		case types.NetworkInterfaceTypeNatGateway:
			if natgw, exists := cache.NATGatewaysByENI[*eni.NetworkInterfaceId]; exists {
				natName := getNameFromTags(natgw.Tags)
				if natName != "" && natName != *natgw.NatGatewayId {
					return *natgw.NatGatewayId + " (" + natName + ")"
				}
				return *natgw.NatGatewayId
			}
			return "Unknown NAT Gateway"

		default:
			// For other interface types, return the description if available
			if eni.Description != nil {
				return *eni.Description
			}
			return string(eni.InterfaceType)
		}
	}

	// Priority 4: Check description for service keywords (following JS script logic)
	if eni.Description != nil {
		desc := strings.ToLower(*eni.Description)
		if strings.Contains(desc, "elb") || strings.Contains(desc, "rds") ||
			strings.Contains(desc, "lambda") || strings.Contains(desc, "vpc") {
			return *eni.Description
		}
		// Return description for any service-like ENI
		return *eni.Description
	}

	// Priority 5: Unattached or unknown
	if eni.Attachment == nil {
		return "Unattached"
	}

	return "N/A"
}

// analyzeSubnetIPUsage provides comprehensive IP usage analysis for a subnet
//
// This function performs detailed analysis of IP address allocation within a subnet by:
// 1. Generating all possible IP addresses in the subnet's CIDR range
// 2. Identifying AWS reserved IPs (network, router, DNS, future use, broadcast)
// 3. Mapping active network interfaces to their IP addresses
// 4. Categorizing each IP as either AWS reserved or service-related
//
// Parameters:
//   - subnet: The AWS subnet to analyze
//   - networkInterfaces: All network interfaces to check for IP usage
//   - svc: EC2 client for additional AWS API calls (used for ENI lookups)
//
// Returns:
//   - ipDetails: Detailed information about each IP address in use
//   - usedIPs: Total count of IP addresses currently allocated
//   - availableIPs: Count of IP addresses available for new allocations
//   - awsReservedIPs: Count of IPs reserved by AWS (network, router, DNS, etc.)
//   - serviceIPs: Count of IPs allocated to AWS services or EC2 instances
//   - error: Any error encountered during analysis
//
// Note: This function uses optimized batch API calls via ENILookupCache to avoid N+1 queries
// when analyzing large numbers of network interfaces.
func analyzeSubnetIPUsage(subnet types.Subnet, networkInterfaces []types.NetworkInterface, svc *ec2.Client) ([]IPAddressInfo, int, int, int, int, error) {
	cidr := *subnet.CidrBlock

	// Get all IPs in the subnet
	allIPs, err := generateIPRange(cidr)
	if err != nil {
		return nil, 0, 0, 0, 0, err
	}

	// Get AWS reserved IPs using the same IP list to avoid duplication
	reservedIPs := identifyAWSReservedIPs(allIPs)

	// Map ENIs to IPs
	eniIPMap := mapNetworkInterfacesToIPs(networkInterfaces, *subnet.SubnetId, svc)

	var ipDetails []IPAddressInfo
	usedCount := 0
	awsReservedCount := 0
	serviceIPsCount := 0

	// Sort IPs for ordered output
	sortIPAddresses(allIPs)

	for _, ip := range allIPs {
		ipStr := ip.String()

		if awsUsage, isReserved := reservedIPs[ipStr]; isReserved {
			ipDetails = append(ipDetails, IPAddressInfo{
				IPAddress:      ipStr,
				UsageType:      "RESERVED BY AWS",
				AttachmentInfo: awsUsage,
			})
			usedCount++
			awsReservedCount++
		} else if eniInfo, exists := eniIPMap[ipStr]; exists {
			ipDetails = append(ipDetails, eniInfo)
			usedCount++
			serviceIPsCount++
		}
	}

	totalIPs := len(allIPs)
	availableIPs := totalIPs - usedCount

	return ipDetails, usedCount, availableIPs, awsReservedCount, serviceIPsCount, nil
}

// ENILookupCache contains pre-fetched AWS resource data to avoid N+1 API calls
// This cache dramatically improves performance when analyzing many ENIs by batching
// API calls instead of making individual requests for each ENI's attachment details.
//
// Performance benefits:
// - VPC Endpoints: 1 batched DescribeVpcEndpoints call vs N individual calls
// - EC2 Instances: Batched DescribeInstances calls vs N individual calls
// - NAT Gateways: 1 batched DescribeNatGateways call vs N individual calls
// - Transit Gateways: 1 batched DescribeTransitGatewayVpcAttachments call vs N individual calls
//
// For 100 ENIs, this reduces API calls from ~400 to ~4, significantly improving
// performance and reducing the chance of hitting AWS API rate limits.
type ENILookupCache struct {
	VPCEndpoints     map[string]*types.VpcEndpoint // VPC ID -> endpoints in that VPC
	InstanceNames    map[string]string             // Instance ID -> name
	TransitGateways  map[string]string             // VPC ID -> TGW attachment ID
	NATGateways      map[string]*types.NatGateway  // VPC ID -> NAT gateways in that VPC
	EndpointsByENI   map[string]*types.VpcEndpoint // ENI ID -> VPC endpoint
	NATGatewaysByENI map[string]*types.NatGateway  // ENI ID -> NAT gateway
}

// NewENILookupCache creates and populates a cache with all required AWS resource data
func NewENILookupCache(svc *ec2.Client, networkInterfaces []types.NetworkInterface) *ENILookupCache {
	cache := &ENILookupCache{
		VPCEndpoints:     make(map[string]*types.VpcEndpoint),
		InstanceNames:    make(map[string]string),
		TransitGateways:  make(map[string]string),
		NATGateways:      make(map[string]*types.NatGateway),
		EndpointsByENI:   make(map[string]*types.VpcEndpoint),
		NATGatewaysByENI: make(map[string]*types.NatGateway),
	}

	// Collect unique VPC IDs and instance IDs from ENIs
	vpcIDs := make(map[string]bool)
	instanceIDs := make(map[string]bool)

	for _, eni := range networkInterfaces {
		if eni.VpcId != nil {
			vpcIDs[*eni.VpcId] = true
		}
		if eni.Attachment != nil && eni.Attachment.InstanceId != nil {
			instanceIDs[*eni.Attachment.InstanceId] = true
		}
	}

	// Batch fetch VPC endpoints for all unique VPCs
	cache.batchFetchVPCEndpoints(svc, vpcIDs)

	// Batch fetch instance names for all unique instances
	cache.batchFetchInstanceNames(svc, instanceIDs)

	// Batch fetch NAT gateways for all unique VPCs
	cache.batchFetchNATGateways(svc, vpcIDs)

	// Batch fetch transit gateway attachments for all unique VPCs
	cache.batchFetchTransitGateways(svc, vpcIDs)

	return cache
}

// batchFetchVPCEndpoints fetches all VPC endpoints for the given VPCs in batches
func (cache *ENILookupCache) batchFetchVPCEndpoints(svc *ec2.Client, vpcIDs map[string]bool) {
	if len(vpcIDs) == 0 {
		return
	}

	// Convert VPC IDs to slice for API call
	vpcIDList := make([]string, 0, len(vpcIDs))
	for vpcID := range vpcIDs {
		vpcIDList = append(vpcIDList, vpcID)
	}

	// Fetch all VPC endpoints for these VPCs
	params := &ec2.DescribeVpcEndpointsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: vpcIDList,
			},
		},
	}

	resp, err := svc.DescribeVpcEndpoints(context.Background(), params)
	if err != nil {
		panic(err)
	}

	// Index endpoints by ENI ID for fast lookup
	for _, endpoint := range resp.VpcEndpoints {
		for _, eniID := range endpoint.NetworkInterfaceIds {
			cache.EndpointsByENI[eniID] = &endpoint
		}
	}
}

// batchFetchInstanceNames fetches all instance names for the given instances
func (cache *ENILookupCache) batchFetchInstanceNames(svc *ec2.Client, instanceIDs map[string]bool) {
	if len(instanceIDs) == 0 {
		return
	}

	// Convert instance IDs to slice for API call
	instanceIDList := make([]string, 0, len(instanceIDs))
	for instanceID := range instanceIDs {
		instanceIDList = append(instanceIDList, instanceID)
	}

	// Fetch all instances in batches (DescribeInstances has a limit)
	const batchSize = 100 // AWS limit for DescribeInstances
	for i := 0; i < len(instanceIDList); i += batchSize {
		end := min(i+batchSize, len(instanceIDList))

		params := &ec2.DescribeInstancesInput{
			InstanceIds: instanceIDList[i:end],
		}

		resp, err := svc.DescribeInstances(context.TODO(), params)
		if err != nil {
			// If an instance doesn't exist, continue with others
			continue
		}

		// Extract names from instances
		for _, reservation := range resp.Reservations {
			for _, instance := range reservation.Instances {
				if instance.InstanceId != nil {
					cache.InstanceNames[*instance.InstanceId] = getNameFromTags(instance.Tags)
				}
			}
		}
	}
}

// batchFetchNATGateways fetches all NAT gateways for the given VPCs
func (cache *ENILookupCache) batchFetchNATGateways(svc *ec2.Client, vpcIDs map[string]bool) {
	if len(vpcIDs) == 0 {
		return
	}

	// Convert VPC IDs to slice for API call
	vpcIDList := make([]string, 0, len(vpcIDs))
	for vpcID := range vpcIDs {
		vpcIDList = append(vpcIDList, vpcID)
	}

	params := &ec2.DescribeNatGatewaysInput{
		Filter: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: vpcIDList,
			},
		},
	}

	resp, err := svc.DescribeNatGateways(context.Background(), params)
	if err != nil {
		panic(err)
	}

	// Index NAT gateways by ENI ID for fast lookup
	for _, natgw := range resp.NatGateways {
		for _, address := range natgw.NatGatewayAddresses {
			if address.NetworkInterfaceId != nil {
				cache.NATGatewaysByENI[*address.NetworkInterfaceId] = &natgw
			}
		}
	}
}

// batchFetchTransitGateways fetches transit gateway attachments for the given VPCs
func (cache *ENILookupCache) batchFetchTransitGateways(svc *ec2.Client, vpcIDs map[string]bool) {
	if len(vpcIDs) == 0 {
		return
	}

	// Convert VPC IDs to slice for API call
	vpcIDList := make([]string, 0, len(vpcIDs))
	for vpcID := range vpcIDs {
		vpcIDList = append(vpcIDList, vpcID)
	}

	params := &ec2.DescribeTransitGatewayVpcAttachmentsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: vpcIDList,
			},
		},
	}

	resp, err := svc.DescribeTransitGatewayVpcAttachments(context.Background(), params)
	if err != nil {
		panic(err)
	}

	// Index TGW attachments by VPC ID for lookup
	for _, attachment := range resp.TransitGatewayVpcAttachments {
		if attachment.VpcId != nil && attachment.TransitGatewayAttachmentId != nil {
			cache.TransitGateways[*attachment.VpcId] = *attachment.TransitGatewayAttachmentId
		}
	}
}

// IPFinderResult contains the result of IP address search
type IPFinderResult struct {
	IPAddress      string                  `json:"ip_address"`
	ENI            *types.NetworkInterface `json:"eni,omitempty"`
	ResourceType   string                  `json:"resource_type"`
	ResourceName   string                  `json:"resource_name"`
	ResourceID     string                  `json:"resource_id"`
	VPC            VPCInfo                 `json:"vpc"`
	Subnet         SubnetInfo              `json:"subnet"`
	SecurityGroups []SecurityGroupInfo     `json:"security_groups"`
	RouteTable     RouteTableInfo          `json:"route_table"`
	IsSecondaryIP  bool                    `json:"is_secondary_ip"`
	Found          bool                    `json:"found"`
}

// VPCInfo contains VPC information for IP finder
type VPCInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	CIDR string `json:"cidr"`
}

// SubnetInfo contains subnet information for IP finder
type SubnetInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	CIDR string `json:"cidr"`
}

// SecurityGroupInfo contains security group information for IP finder
type SecurityGroupInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// RouteTableInfo contains route table information for IP finder
type RouteTableInfo struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Routes []string `json:"routes"`
}

// IsValidIPAddress validates if the provided string is a valid IP address
func IsValidIPAddress(ip string) bool {
	return net.ParseIP(ip) != nil
}

// IsValidCIDR validates if the provided string is a valid CIDR block
func IsValidCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)
	return err == nil
}

// FindIPAddressDetails searches for an IP address across ENIs and returns detailed information
// Searches both primary and secondary IP addresses on all ENIs
func FindIPAddressDetails(svc *ec2.Client, ipAddress string) IPFinderResult {
	// Create filter for IP address search
	// Note: addresses.private-ip-address filter includes both primary and secondary IPs
	filters := []types.Filter{
		{
			Name:   aws.String("addresses.private-ip-address"),
			Values: []string{ipAddress},
		},
	}

	// Search for ENIs with the IP address (includes both primary and secondary IPs)
	enis := searchENIsByIP(svc, filters)

	if len(enis) == 0 {
		return IPFinderResult{
			IPAddress: ipAddress,
			Found:     false,
		}
	}

	// Handle multiple ENIs with the same IP (rare but possible)
	if len(enis) > 1 {
		// Log warning about multiple matches - following awstools pattern of using panic for warnings
		// This is a rare scenario but can happen in some edge cases
		fmt.Printf("Warning: Multiple ENIs found with IP %s. Returning details for first ENI (%s)\n",
			ipAddress, *enis[0].NetworkInterfaceId)
	}

	// Process the first matching ENI
	eni := enis[0]

	// Create ENI cache for efficient resource lookup
	cache := NewENILookupCache(svc, []types.NetworkInterface{eni})

	// Build detailed result
	result := IPFinderResult{
		IPAddress:     ipAddress,
		ENI:           &eni,
		Found:         true,
		IsSecondaryIP: isSecondaryIP(eni, ipAddress),
	}

	// Populate resource information
	result.ResourceType = getENIUsageTypeOptimized(eni, cache)
	result.ResourceName, result.ResourceID = getResourceNameAndID(eni, cache)
	result.VPC = getVPCInfo(svc, aws.ToString(eni.VpcId))
	result.Subnet = getSubnetInfo(svc, aws.ToString(eni.SubnetId))
	result.SecurityGroups = getSecurityGroupInfo(svc, eni.Groups)
	result.RouteTable = getRouteTableInfo(svc, aws.ToString(eni.SubnetId))

	return result
}

// handleAWSAPIError provides better error messages for common AWS API errors
func handleAWSAPIError(err error, apiName string) {
	if strings.Contains(err.Error(), "UnauthorizedOperation") {
		panic(fmt.Errorf("insufficient permissions for %s\n\nRequired permissions:\n  - ec2:%s\n\nOriginal error: %v", apiName, apiName, err))
	}
	if strings.Contains(err.Error(), "AuthFailure") {
		panic(fmt.Errorf("AWS authentication failed\n\nPlease check your AWS credentials:\n  - AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables\n  - ~/.aws/credentials file\n  - IAM role (if running on EC2)\n\nOriginal error: %v", err))
	}
	if strings.Contains(err.Error(), "RequestLimitExceeded") || strings.Contains(err.Error(), "Throttling") {
		panic(fmt.Errorf("AWS API rate limit exceeded\n\nThe request was throttled. Please wait a moment and try again.\n\nOriginal error: %v", err))
	}
	// For other errors, provide the original error with context
	panic(fmt.Errorf("failed to call %s: %v", apiName, err))
}

// searchENIsByIP searches for ENIs with a specific IP address
func searchENIsByIP(svc *ec2.Client, filters []types.Filter) []types.NetworkInterface {
	input := &ec2.DescribeNetworkInterfacesInput{
		Filters: filters,
	}

	resp, err := svc.DescribeNetworkInterfaces(context.TODO(), input)
	if err != nil {
		handleAWSAPIError(err, "DescribeNetworkInterfaces")
	}

	return resp.NetworkInterfaces
}

// isSecondaryIP checks if the IP address is a secondary IP on the ENI
func isSecondaryIP(eni types.NetworkInterface, ipAddress string) bool {
	primaryIP := aws.ToString(eni.PrivateIpAddress)
	if primaryIP == ipAddress {
		return false
	}

	for _, privateIP := range eni.PrivateIpAddresses {
		if aws.ToString(privateIP.PrivateIpAddress) == ipAddress && !aws.ToBool(privateIP.Primary) {
			return true
		}
	}

	return false
}

// getResourceNameAndID extracts resource name and ID from ENI attachment details
func getResourceNameAndID(eni types.NetworkInterface, cache *ENILookupCache) (string, string) {
	attachmentDetails := getENIAttachmentDetailsOptimized(eni, cache)

	// Handle EC2 instances
	if eni.Attachment != nil && eni.Attachment.InstanceId != nil {
		return attachmentDetails, *eni.Attachment.InstanceId
	}

	// Handle VPC endpoints
	if endpoint, exists := cache.EndpointsByENI[*eni.NetworkInterfaceId]; exists {
		return attachmentDetails, *endpoint.VpcEndpointId
	}

	// Handle NAT gateways
	if natgw, exists := cache.NATGatewaysByENI[*eni.NetworkInterfaceId]; exists {
		return attachmentDetails, *natgw.NatGatewayId
	}

	// Default to attachment details
	return attachmentDetails, ""
}

// getVPCInfo retrieves VPC information
func getVPCInfo(svc *ec2.Client, vpcID string) VPCInfo {
	if vpcID == "" {
		return VPCInfo{}
	}

	input := &ec2.DescribeVpcsInput{
		VpcIds: []string{vpcID},
	}

	resp, err := svc.DescribeVpcs(context.TODO(), input)
	if err != nil {
		handleAWSAPIError(err, "DescribeVpcs")
	}

	if len(resp.Vpcs) == 0 {
		return VPCInfo{ID: vpcID}
	}

	vpc := resp.Vpcs[0]
	return VPCInfo{
		ID:   vpcID,
		Name: getNameFromTags(vpc.Tags),
		CIDR: aws.ToString(vpc.CidrBlock),
	}
}

// getSubnetInfo retrieves subnet information
func getSubnetInfo(svc *ec2.Client, subnetID string) SubnetInfo {
	if subnetID == "" {
		return SubnetInfo{}
	}

	input := &ec2.DescribeSubnetsInput{
		SubnetIds: []string{subnetID},
	}

	resp, err := svc.DescribeSubnets(context.TODO(), input)
	if err != nil {
		handleAWSAPIError(err, "DescribeSubnets")
	}

	if len(resp.Subnets) == 0 {
		return SubnetInfo{ID: subnetID}
	}

	subnet := resp.Subnets[0]
	return SubnetInfo{
		ID:   subnetID,
		Name: getNameFromTags(subnet.Tags),
		CIDR: aws.ToString(subnet.CidrBlock),
	}
}

// getSecurityGroupInfo retrieves security group information
func getSecurityGroupInfo(svc *ec2.Client, groups []types.GroupIdentifier) []SecurityGroupInfo {
	if len(groups) == 0 {
		return []SecurityGroupInfo{}
	}

	// Extract group IDs
	groupIDs := make([]string, len(groups))
	for i, group := range groups {
		groupIDs[i] = aws.ToString(group.GroupId)
	}

	input := &ec2.DescribeSecurityGroupsInput{
		GroupIds: groupIDs,
	}

	resp, err := svc.DescribeSecurityGroups(context.TODO(), input)
	if err != nil {
		handleAWSAPIError(err, "DescribeSecurityGroups")
	}

	var result []SecurityGroupInfo
	for _, sg := range resp.SecurityGroups {
		result = append(result, SecurityGroupInfo{
			ID:   aws.ToString(sg.GroupId),
			Name: aws.ToString(sg.GroupName),
		})
	}

	return result
}

// getRouteTableInfo retrieves route table information for a subnet
func getRouteTableInfo(svc *ec2.Client, subnetID string) RouteTableInfo {
	if subnetID == "" {
		return RouteTableInfo{}
	}

	// Get all route tables
	routeTables := retrieveRouteTables(svc)

	// Find the route table associated with this subnet
	routeTable := GetSubnetRouteTable(subnetID, routeTables)
	if routeTable == nil {
		return RouteTableInfo{
			ID:     "No route table",
			Name:   "No route table",
			Routes: []string{},
		}
	}

	// Get formatted route table information
	rtDisplay, routeList := FormatRouteTableInfo(routeTable)

	return RouteTableInfo{
		ID:     aws.ToString(routeTable.RouteTableId),
		Name:   rtDisplay,
		Routes: routeList,
	}
}
