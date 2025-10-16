package cmd

import (
	"fmt"

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

	Using the --resource-id (-r) flag, you can limit the output to the provided resource.
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
	tgwroutetablesCmd.Flags().StringVarP(&tgwresourceid, "resource-id", "r", "", "The id of the resource you want to limit to")
	tgwroutetablesCmd.Flags().BoolVarP(&simplelist, "list", "l", false, "Only show a simple list of routes")
}

func tgwroutes(_ *cobra.Command, _ []string) {
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "Overview of all routes"
	gateways := helpers.GetAllTransitGateways(awsConfig.Ec2Client())
	keys := []string{"ID", "Name", "Destinations", "TargetGateway"}
	if settings.IsDrawIO() {
		keys = append(keys, "Image")
	}
	if simplelist {
		simplelistOnly(awsConfig)
		return
	}
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = resultTitle
	output.Settings.SortKey = "TargetGateway"
	if settings.IsDrawIO() {
		output.Settings.DrawIOHeader = createTgwRoutesDrawIOHeader()
	}
	if output.Settings.NeedsFromToColumns() {
		output.Settings.AddFromToColumns("Destinations", "ID")
	}

	attachedresources, tgwrts := filterGateway(gateways)

	for rt, connectedvpcs := range tgwrts {
		content := make(map[string]any)
		content["ID"] = rt
		content["Name"] = getName(rt)
		content["Destinations"] = connectedvpcs
		if settings.IsDrawIO() {
			content["Image"] = drawio.AWSShape("Network Content Delivery", "Route Table")
		}
		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	for resourceid, resourceInfo := range attachedresources {
		content := make(map[string]any)
		content["ID"] = resourceid
		content["Name"] = getName(resourceid)
		if settings.IsDrawIO() {
			// Use raw ID for DrawIO to enable proper connection matching
			content["TargetGateway"] = resourceInfo.RouteTableID
		} else {
			// Use composite name for other output formats
			if getName(resourceInfo.RouteTableID) != resourceInfo.RouteTableID && getName(resourceInfo.RouteTableID) != "" {
				content["TargetGateway"] = getNameWithID(resourceInfo.RouteTableID)
			} else {
				content["TargetGateway"] = resourceInfo.RouteTableID
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
				content["Image"] = drawio.AWSShape("Network Content Delivery", "VPC")
			case "vpn":
				content["Image"] = drawio.AWSShape("Network Content Delivery", "Site-to-Site VPN")
			case "dxgw":
				content["Image"] = drawio.AWSShape("Network Content Delivery", "Direct Connect Gateway")
			case tgwResourceType:
				content["Image"] = drawio.AWSShape("Network Content Delivery", "Transit Gateway")
			case "peering":
				content["Image"] = drawio.AWSShape("Network Content Delivery", "Transit Gateway")
			case "tgw-peering":
				content["Image"] = drawio.AWSShape("Network Content Delivery", "Transit Gateway")
			case "direct-connect-gateway":
				content["Image"] = drawio.AWSShape("Network Content Delivery", "Direct Connect")
			case "connect":
				content["Image"] = drawio.AWSShape("Network Content Delivery", "Transit Gateway")
			default:
				content["Image"] = drawio.AWSShape("General Resources", "General")
			}
		}
		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	output.Write()
}

func simplelistOnly(awsConfig config.AWSConfig) {
	keys := []string{"CIDR", "Target", "Route Type", "State"}
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = fmt.Sprintf("Simple route list for %s", tgwresourceid)
	output.Settings.SortKey = "CIDR"

	activeroutes := helpers.GetActiveRoutesForTransitGatewayRouteTable(tgwresourceid, awsConfig.Ec2Client())
	blackholeroutes := helpers.GetBlackholeRoutesForTransitGatewayRouteTable(tgwresourceid, awsConfig.Ec2Client())

	for _, route := range activeroutes {
		content := make(map[string]any)
		content["CIDR"] = route.CIDR
		content["Target"] = getName(route.Attachment.ResourceID)
		// content["Target Type"] = getName(route.Attachment.ResourceType)
		content["Route Type"] = route.RouteType
		content["State"] = route.State

		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	for _, route := range blackholeroutes {
		content := make(map[string]any)
		content["CIDR"] = route.CIDR
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
	limitertype := helpers.TypeByResourceID(tgwresourceid)
	attachedresources := make(map[string]attachedResourceInfo)
	tgwrts := make(map[string][]string)

	for _, gateway := range gateways {
		// only add relevant gateway if filtered by gateway
		if limitertype == tgwResourceType && gateway.ID != tgwresourceid {
			continue
		}
		for _, routetable := range gateway.RouteTables {
			// only add relevant route tables if filtered by route table
			if limitertype == "tgw-rtb" && routetable.ID != tgwresourceid {
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
			if !contains(destinationvpcs, tgwresourceid) && tgwid != attachedresources[tgwresourceid].RouteTableID {
				delete(tgwrts, tgwid)
			}
			if contains(destinationvpcs, tgwresourceid) {
				tgwrts[tgwid] = []string{tgwresourceid}
				attachedtgwrts = append(attachedtgwrts, tgwid)
			}
		}
		for resourceid, resourceInfo := range attachedresources {
			if resourceid != tgwresourceid && !contains(attachedtgwrts, resourceInfo.RouteTableID) {
				delete(attachedresources, resourceid)
			}
		}
	}
	return attachedresources, tgwrts
}

func createTgwRoutesDrawIOHeader() drawio.Header {
	drawioheader := drawio.NewHeader("%Name%", "%Image%", "Image")
	drawioheader.SetHeightAndWidth("78", "78")
	connection := drawio.NewConnection()
	connection.From = "Destinations"
	connection.To = "ID"
	connection.Invert = false
	connection.Label = "Outbound"
	drawioheader.AddConnection(connection)
	connection2 := drawio.NewConnection()
	connection2.From = "TargetGateway"
	connection2.To = "ID"
	connection2.Invert = false
	connection2.Label = "Inbound"
	drawioheader.AddConnection(connection2)
	return drawioheader
}
