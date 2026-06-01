package cmd

import (
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/ArjenSchwarz/go-output/drawio"

	"github.com/spf13/cobra"
)

// tgwroutetablesCmd represents the tgwroutes command
var tgwroutetablesCmd = &cobra.Command{
	Use:   "routetables",
	Short: "Get an overview of connections between Transit Gateway Route Tables and attached resources",
	Long: `Get an overview of connections between Transit Gateway Route Tables and attached resources

	Using the --resource-id (-r) flag, you can limit the output to the provided resource(s).
	Multiple resource IDs can be provided as comma-separated values (e.g., -r vpc-123,vpc-456).
	For a route table that means all the resources it connects to,
	while for a VPC that means all the route tables it connects
	to and through them what other resources can reach it or it can reach.

	Supports a Draw.io output`,
	Run: tgwroutes,
}

var tgwresourceid string
var simplelist bool

// attachedResourceInfo holds information about a resource attached to a TGW route table
type attachedResourceInfo struct {
	RouteTableID string
	ResourceType string
}

func init() {
	tgwCmd.AddCommand(tgwroutetablesCmd)
	tgwroutetablesCmd.Flags().StringVarP(&tgwresourceid, "resource-id", "r", "", "The id(s) of the resource you want to limit to (comma-separated for multiple)")
	tgwroutetablesCmd.Flags().BoolVarP(&simplelist, "list", "l", false, "Only show a simple list of routes for a single route table; requires --resource-id with a tgw-rtb-* ID")
}

func tgwroutes(_ *cobra.Command, _ []string) {
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "Overview of all routes"
	gateways := helpers.GetAllTransitGateways(awsConfig.Ec2Client())
	keys := []string{"ID", nameColumn, destinationsColumn, targetGatewayColumn}
	if settings.IsDrawIO() {
		keys = append(keys, imageColumn)
	}
	if simplelist {
		simplelistOnly(awsConfig)
		return
	}
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = resultTitle
	output.Settings.SortKey = targetGatewayColumn
	if settings.IsDrawIO() {
		output.Settings.DrawIOHeader = createTgwRoutesDrawIOHeader()
	}
	if output.Settings.NeedsFromToColumns() {
		output.Settings.AddFromToColumns(destinationsColumn, "ID")
	}

	attachedresources, tgwrts := filterGateway(gateways)

	for rt, connectedvpcs := range tgwrts {
		content := make(map[string]any)
		content["ID"] = rt
		content[nameColumn] = getName(rt)
		content[destinationsColumn] = unique(connectedvpcs)
		if settings.IsDrawIO() {
			content[imageColumn] = drawio.AWSShape("Network Content Delivery", routeTableColumn)
		}
		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	for resourceid, resourceInfo := range attachedresources {
		content := make(map[string]any)
		content["ID"] = resourceid
		content[nameColumn] = getName(resourceid)
		if settings.IsDrawIO() {
			// Use raw ID for DrawIO to enable proper connection matching
			content[targetGatewayColumn] = resourceInfo.RouteTableID
		} else {
			// Use composite name for other output formats
			if getName(resourceInfo.RouteTableID) != resourceInfo.RouteTableID && getName(resourceInfo.RouteTableID) != "" {
				content[targetGatewayColumn] = getNameWithID(resourceInfo.RouteTableID)
			} else {
				content[targetGatewayColumn] = resourceInfo.RouteTableID
			}
		}

		if settings.IsDrawIO() {
			// Use actual ResourceType from AWS API when available
			resourceType := resourceInfo.ResourceType
			if resourceType == "" {
				// Fallback to deducing type from resource ID
				resourceType = helpers.TypeByResourceID(resourceid)
			}
			switch resourceType {
			case vpcResourceType:
				content[imageColumn] = drawio.AWSShape("Network Content Delivery", vpcColumn)
			case "vpn":
				content[imageColumn] = drawio.AWSShape("Network Content Delivery", "Site-to-Site VPN")
			case "dxgw":
				content[imageColumn] = drawio.AWSShape("Network Content Delivery", "Direct Connect Gateway")
			case tgwResourceType:
				content[imageColumn] = drawio.AWSShape("Network Content Delivery", "Transit Gateway")
			case "peering":
				content[imageColumn] = drawio.AWSShape("Network Content Delivery", "Transit Gateway")
			case "tgw-peering":
				content[imageColumn] = drawio.AWSShape("Network Content Delivery", "Transit Gateway")
			case "direct-connect-gateway":
				content[imageColumn] = drawio.AWSShape("Network Content Delivery", "Direct Connect")
			case "connect":
				content[imageColumn] = drawio.AWSShape("Network Content Delivery", "Transit Gateway")
			default:
				content[imageColumn] = drawio.AWSShape("General Resources", "General")
			}
		}
		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	output.Write()
}

// validateSimpleListResourceID checks that the resource ID supplied for the
// --list output is a single Transit Gateway route table ID (tgw-rtb-*).
//
// The simple list queries SearchTransitGatewayRoutes for one route table, so
// an empty value, a non-route-table resource (VPC, TGW, attachment, ...), or a
// comma-separated list must be rejected with a clear usage error rather than
// silently calling the AWS API with an empty or wrong ID.
func validateSimpleListResourceID(resourceID string) error {
	trimmed := strings.TrimSpace(resourceID)
	if trimmed == "" {
		return fmt.Errorf("the --list output requires a Transit Gateway route table ID; pass one with --resource-id (e.g. -r tgw-rtb-0123456789abcdef0)")
	}
	if strings.Contains(trimmed, ",") {
		return fmt.Errorf("the --list output supports a single Transit Gateway route table ID, not a list: %q", resourceID)
	}
	if helpers.TypeByResourceID(trimmed) != "tgw-rtb" {
		return fmt.Errorf("the --list output requires a Transit Gateway route table ID (tgw-rtb-*), got %q", trimmed)
	}
	return nil
}

func simplelistOnly(awsConfig config.AWSConfig) {
	if err := validateSimpleListResourceID(tgwresourceid); err != nil {
		log.Fatal(err)
	}
	routetableID := strings.TrimSpace(tgwresourceid)
	keys := []string{cidrColumn, "Target", "Route Type", "State"}
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = fmt.Sprintf("Simple route list for %s", routetableID)
	output.Settings.SortKey = cidrColumn

	activeroutes := helpers.GetActiveRoutesForTransitGatewayRouteTable(routetableID, awsConfig.Ec2Client())
	blackholeroutes := helpers.GetBlackholeRoutesForTransitGatewayRouteTable(routetableID, awsConfig.Ec2Client())

	for _, route := range activeroutes {
		content := make(map[string]any)
		content[cidrColumn] = route.CIDR
		content["Target"] = getName(route.Attachment.ResourceID)
		// content["Target Type"] = getName(route.Attachment.ResourceType)
		content["Route Type"] = route.RouteType
		content["State"] = route.State

		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	for _, route := range blackholeroutes {
		content := make(map[string]any)
		content[cidrColumn] = route.CIDR
		content["Target"] = "-"
		// content["Target Type"] = "-"
		content["Route Type"] = route.RouteType
		content["State"] = route.State

		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	output.Write()
}

func filterGateway(gateways []helpers.TransitGateway) (map[string]attachedResourceInfo, map[string][]string) {
	// Parse comma-separated resource IDs
	var resourceIDs []string
	if tgwresourceid != "" {
		for _, id := range strings.Split(tgwresourceid, ",") {
			if trimmed := strings.TrimSpace(id); trimmed != "" {
				resourceIDs = append(resourceIDs, trimmed)
			}
		}
	}

	// Determine limiter type from first resource ID (all should be same type)
	limitertype := ""
	if len(resourceIDs) > 0 {
		limitertype = helpers.TypeByResourceID(resourceIDs[0])
	}

	attachedresources := make(map[string]attachedResourceInfo)
	tgwrts := make(map[string][]string)

	for _, gateway := range gateways {
		// only add relevant gateway if filtered by gateway
		if limitertype == tgwResourceType && !slices.Contains(resourceIDs, gateway.ID) {
			continue
		}
		for _, routetable := range gateway.RouteTables {
			// only add relevant route tables if filtered by route table
			if limitertype == "tgw-rtb" && !slices.Contains(resourceIDs, routetable.ID) {
				continue
			}
			tgwrts[routetable.ID] = []string{}
			for _, route := range routetable.Routes {
				// Skip routes without attachments (e.g., blackhole routes)
				if route.Attachment.ResourceID == "" {
					continue
				}
				tgwrts[routetable.ID] = append(tgwrts[routetable.ID], route.Attachment.ResourceID)
				if _, ok := attachedresources[route.Attachment.ResourceID]; !ok {
					attachedresources[route.Attachment.ResourceID] = attachedResourceInfo{
						ResourceType: route.Attachment.ResourceType,
					}
				}
			}
			for _, sourceattachment := range routetable.SourceAttachments {
				attachedresources[sourceattachment.ResourceID] = attachedResourceInfo{
					RouteTableID: routetable.ID,
					ResourceType: sourceattachment.ResourceType,
				}
			}
		}
	}
	// For VPC pass over everything and remove what's not relevant
	if limitertype == vpcResourceType {
		attachedtgwrts := []string{}
		for tgwid, destinationvpcs := range tgwrts {
			// Check if any of the resource IDs match
			hasMatch := false
			for _, resourceID := range resourceIDs {
				if contains(destinationvpcs, resourceID) || tgwid == attachedresources[resourceID].RouteTableID {
					hasMatch = true
					break
				}
			}
			if !hasMatch {
				delete(tgwrts, tgwid)
				continue
			}
			// Filter destinations to only include the requested VPCs
			var matchingVPCs []string
			for _, resourceID := range resourceIDs {
				if contains(destinationvpcs, resourceID) {
					matchingVPCs = append(matchingVPCs, resourceID)
					attachedtgwrts = append(attachedtgwrts, tgwid)
				}
			}
			if len(matchingVPCs) > 0 {
				tgwrts[tgwid] = matchingVPCs
			}
		}
		for resourceid, resourceInfo := range attachedresources {
			if !slices.Contains(resourceIDs, resourceid) && !slices.Contains(attachedtgwrts, resourceInfo.RouteTableID) {
				delete(attachedresources, resourceid)
			}
		}
	}
	return attachedresources, tgwrts
}

func createTgwRoutesDrawIOHeader() drawio.Header {
	drawioheader := drawio.NewHeader("%Name%", "%Image%", imageColumn)
	drawioheader.SetHeightAndWidth("78", "78")
	connection := drawio.NewConnection()
	connection.From = destinationsColumn
	connection.To = "ID"
	connection.Invert = false
	connection.Label = "Outbound"
	drawioheader.AddConnection(connection)
	connection2 := drawio.NewConnection()
	connection2.From = targetGatewayColumn
	connection2.To = "ID"
	connection2.Invert = false
	connection2.Label = "Inbound"
	drawioheader.AddConnection(connection2)
	return drawioheader
}
