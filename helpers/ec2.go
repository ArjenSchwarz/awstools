package helpers

import (
	"context"

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
			for _, tag := range instance.Tags {
				if aws.ToString(tag.Key) == "Name" {
					return aws.ToString(tag.Value)
				}
			}
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
	return result
}

//addAllVPCNames returns the names of all vpcs in a map
func addAllVPCNames(svc *ec2.Client, result map[string]string) map[string]string {
	resp, err := svc.DescribeVpcs(context.TODO(), &ec2.DescribeVpcsInput{})
	if err != nil {
		panic(err)
	}
	for _, vpc := range resp.Vpcs {
		result[*vpc.VpcId] = *vpc.VpcId
		if vpc.Tags != nil {
			for _, tag := range vpc.Tags {
				if *tag.Key == "Name" {
					result[*vpc.VpcId] = *tag.Value
					break
				}
			}
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
		if peer.Tags != nil {
			for _, tag := range peer.Tags {
				if *tag.Key == "Name" {
					result[*peer.VpcPeeringConnectionId] = *tag.Value
					break
				}
			}
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
		if subnet.Tags != nil {
			for _, tag := range subnet.Tags {
				if *tag.Key == "Name" {
					result[*subnet.SubnetId] = *tag.Value
					break
				}
			}
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
		if resource.Tags != nil {
			for _, tag := range resource.Tags {
				if *tag.Key == "Name" {
					result[*resource.RouteTableId] = *tag.Value
					break
				}
			}
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

//GetAllVPCRouteTables returns all the Routetables in the account and region
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
		routetable.Routes = GetActiveRoutesForTransitGatewayRouteTable(routetable.ID, svc)
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
		tgwroute := TransitGatewayRoute{
			State: string(route.State),
			CIDR:  *route.DestinationCidrBlock,
			Attachment: TransitGatewayAttachment{
				ID:         *route.TransitGatewayAttachments[0].TransitGatewayAttachmentId,
				ResourceID: *route.TransitGatewayAttachments[0].ResourceId,
			},
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
	for _, tag := range tags {
		if aws.ToString(tag.Key) == "Name" {
			return aws.ToString(tag.Value)
		}
	}
	return ""
}
