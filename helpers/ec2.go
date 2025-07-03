package helpers

import (
	"context"
	"net"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
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

// getResourceDisplayNameFromTags provides tiered name lookup for AWS resources using getName and tags
// This is a helper version for use within the helpers package
func getResourceDisplayNameFromTags(resourceID string, tags []types.Tag) string {
	// Try using existing tag-based name lookup first
	nameFromTags := getNameFromTags(tags)

	// Format the display name
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
			return aws.ToString(tag.Value)
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
		for _, subnetID := range resp.TransitGatewayVpcAttachments[0].SubnetIds {
			if subnetID == *netinterface.SubnetId {
				return *resp.TransitGatewayVpcAttachments[0].TransitGatewayAttachmentId
			}
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
			for _, eniID := range endpoint.NetworkInterfaceIds {
				if eniID == *netinterface.NetworkInterfaceId {
					return &endpoint
				}
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
	availableIPs := totalIPs - 5
	if availableIPs < 0 {
		availableIPs = 0
	}

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

	for i := 0; i < 4; i++ {
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
func identifyAWSReservedIPs(cidr string) (map[string]string, error) {
	ips, err := generateIPRange(cidr)
	if err != nil {
		return nil, err
	}

	reserved := make(map[string]string)
	if len(ips) == 0 {
		return reserved, nil
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

	return reserved, nil
}

// mapNetworkInterfacesToIPs maps network interfaces to their IP addresses
func mapNetworkInterfacesToIPs(networkInterfaces []types.NetworkInterface, subnetID string, svc *ec2.Client) map[string]IPAddressInfo {
	ipMap := make(map[string]IPAddressInfo)

	for _, eni := range networkInterfaces {
		if eni.SubnetId != nil && *eni.SubnetId == subnetID {
			for _, privateIP := range eni.PrivateIpAddresses {
				if privateIP.PrivateIpAddress != nil {
					ipInfo := IPAddressInfo{
						IPAddress:      *privateIP.PrivateIpAddress,
						UsageType:      getENIUsageType(eni, svc),
						AttachmentInfo: getENIAttachmentDetails(eni, svc),
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

// getENIUsageType returns the general category of what the ENI is used for
func getENIUsageType(eni types.NetworkInterface, svc *ec2.Client) string {
	// Check if it's a VPC endpoint first (highest priority)
	endpoint := GetVPCEndpointFromNetworkInterface(eni, svc)
	if endpoint != nil {
		return "VPC Endpoint"
	}

	// Handle EC2 instances
	if eni.Attachment != nil && eni.Attachment.InstanceId != nil {
		return "EC2 Instance"
	}

	// Handle specific interface types (if not 'interface')
	if eni.InterfaceType != "interface" {
		switch eni.InterfaceType {
		case types.NetworkInterfaceTypeTransitGateway:
			return "Transit Gateway"
		case types.NetworkInterfaceTypeNatGateway:
			return "NAT Gateway"
		case types.NetworkInterfaceTypeVpcEndpoint:
			return "VPC Endpoint"
		case types.NetworkInterfaceTypeLambda:
			return "Lambda Function"
		case "quicksight":
			return "QuickSight"
		case "network_load_balancer":
			return "Network Load Balancer"
		case "gateway_load_balancer":
			return "Gateway Load Balancer"
		default:
			return "AWS Service"
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
			return "Lambda Function"
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
		return "AWS Service"
	}

	// Unattached or unknown
	if eni.Attachment == nil {
		return "Unattached ENI"
	}

	return "Unknown"
}

// getENIAttachmentDetails returns specific details about what the ENI is attached to
func getENIAttachmentDetails(eni types.NetworkInterface, svc *ec2.Client) string {
	// Priority 1: Check if it's a VPC endpoint (following JS script logic)
	endpoint := GetVPCEndpointFromNetworkInterface(eni, svc)
	if endpoint != nil {
		// Extract service name (last part after dots, like 's3', 'ec2')
		serviceParts := strings.Split(*endpoint.ServiceName, ".")
		shortServiceName := serviceParts[len(serviceParts)-1]
		return *endpoint.VpcEndpointId + " (" + shortServiceName + ")"
	}

	// Priority 2: Handle EC2 instances
	if eni.Attachment != nil && eni.Attachment.InstanceId != nil {
		instanceName := GetEc2Name(*eni.Attachment.InstanceId, svc)
		if instanceName != "" && instanceName != *eni.Attachment.InstanceId {
			return *eni.Attachment.InstanceId + " (" + instanceName + ")"
		}
		return *eni.Attachment.InstanceId
	}

	// Priority 3: Handle specific interface types (if not 'interface')
	if eni.InterfaceType != "interface" {
		switch eni.InterfaceType {
		case types.NetworkInterfaceTypeTransitGateway:
			tgwID := GetTransitGatewayFromNetworkInterface(eni, svc)
			if tgwID != "" {
				return tgwID
			}
			return "Unknown Transit Gateway"

		case types.NetworkInterfaceTypeNatGateway:
			natgw := GetNatGatewayFromNetworkInterface(eni, svc)
			if natgw != nil {
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
// Returns: ipDetails, usedIPs, availableIPs, awsReservedIPs, serviceIPs, error
func analyzeSubnetIPUsage(subnet types.Subnet, networkInterfaces []types.NetworkInterface, svc *ec2.Client) ([]IPAddressInfo, int, int, int, int, error) {
	cidr := *subnet.CidrBlock

	// Get all IPs in the subnet
	allIPs, err := generateIPRange(cidr)
	if err != nil {
		return nil, 0, 0, 0, 0, err
	}

	// Get AWS reserved IPs
	reservedIPs, err := identifyAWSReservedIPs(cidr)
	if err != nil {
		return nil, 0, 0, 0, 0, err
	}

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
