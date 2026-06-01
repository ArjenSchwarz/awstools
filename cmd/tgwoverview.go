package cmd

import (
	"strings"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/ArjenSchwarz/go-output/drawio"
	"github.com/spf13/cobra"
)

// tgwoverviewCmd represents the tgwoverview command
var tgwoverviewCmd = &cobra.Command{
	Use:   "overview",
	Short: "A basic overview of the Transit Gateway",
	Long: `Provides an overview of all the route tables and routes in the Transit Gateway.
This can be improved on, but offers a simple text based overview with all relevant information

If you choose the drawio output instead, you get a simple diagram showing the Transit Gateway and all resources (VPCs, VPNs, Direct Connect) attached to it.
	`,
	Run: tgwoverview,
}

var excludeRouteTarget string
var includeBlackhole bool

func init() {
	tgwCmd.AddCommand(tgwoverviewCmd)
	tgwoverviewCmd.Flags().StringVarP(&excludeRouteTarget, "exclude-target", "e", "", "Optional value to exclude a specific target from the output")
	tgwoverviewCmd.Flags().BoolVarP(&includeBlackhole, "blackhole-routes", "b", false, "Optional value to include blackhole routes")
}

func tgwoverview(_ *cobra.Command, _ []string) {
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "Transit Gateway Routes in account " + getName(helpers.GetAccountID(awsConfig.StsClient()))
	gateways := helpers.GetAllTransitGateways(awsConfig.Ec2Client())
	keys := []string{"Transit Gateway Account", "Transit Gateway", routeTableColumn, cidrColumn, "Target", "Target Type", "State"}
	if settings.IsDrawIO() {
		keys = []string{"ID", nameColumn, destinationsColumn, imageColumn}
	}
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = resultTitle
	output.Settings.SortKey = routeTableColumn
	if settings.IsDrawIO() {
		createTgwOverviewDrawIO(&output, gateways)
	} else {
		for _, gateway := range gateways {
			for _, routetable := range gateway.RouteTables {
				// Track which resources appear as route targets so we can
				// identify source attachments that are otherwise invisible.
				routeTargets := make(map[string]bool)
				for _, route := range routetable.Routes {
					if excludeRouteTarget == route.Attachment.ResourceID {
						continue
					}
					if !includeBlackhole && route.State == "blackhole" {
						continue
					}
					if route.Attachment.ResourceID != "" {
						routeTargets[route.Attachment.ResourceID] = true
					}
					content := make(map[string]any)
					content["Transit Gateway Account"] = getNameWithID(gateway.AccountID)
					content["Transit Gateway"] = getNameWithID(gateway.ID)
					content[routeTableColumn] = getNameWithID(routetable.ID)
					content[cidrColumn] = route.CIDR
					if route.Attachment.ResourceID != "" {
						content["Target"] = getNameWithID(route.Attachment.ResourceID)
					} else {
						content["Target"] = ""
					}
					content["Target Type"] = helpers.TypeByResourceID(route.Attachment.ResourceID)
					state := route.State
					if output.Settings.UseEmoji {
						if route.State == "blackhole" {
							state = "❌ " + state
						} else {
							state = "✅ " + state
						}
					}
					content["State"] = state
					holder := format.OutputHolder{Contents: content}
					output.AddHolder(holder)
				}
				// Show source attachments (associations) that don't already
				// appear as route targets so non-VPC attachments are visible.
				for _, attachment := range routetable.SourceAttachments {
					if attachment.ResourceID == "" || routeTargets[attachment.ResourceID] {
						continue
					}
					content := make(map[string]any)
					content["Transit Gateway Account"] = getNameWithID(gateway.AccountID)
					content["Transit Gateway"] = getNameWithID(gateway.ID)
					content[routeTableColumn] = getNameWithID(routetable.ID)
					content[cidrColumn] = "-"
					content["Target"] = getNameWithID(attachment.ResourceID)
					content["Target Type"] = attachment.ResourceType
					content["State"] = "associated"
					holder := format.OutputHolder{Contents: content}
					output.AddHolder(holder)
				}
			}
		}
	}
	output.Write()
}

func createTgwOverviewDrawIO(output *format.OutputArray, gateways []helpers.TransitGateway) {
	drawioheader := drawio.NewHeader("%Name%", "%Image%", imageColumn)
	drawioheader.SetHeightAndWidth("78", "78")
	connection := drawio.NewConnection()
	connection.From = destinationsColumn
	connection.To = "ID"
	connection.Invert = false
	connection.Style = drawio.BidirectionalConnectionStyle
	drawioheader.AddConnection(connection)
	output.Settings.DrawIOHeader = drawioheader
	type targetTgwMap struct {
		ID           string
		Name         string
		Destinations []string
		Image        string
	}
	targetTgwMapping := make(map[string]targetTgwMap)
	if settings.ShouldCombineAndAppend() {
		headers, previousResults := drawio.GetHeaderAndContentsFromFile(settings.GetString("output.file"))
		for _, row := range previousResults {
			targetTgwMapping[row[headers["ID"]]] = targetTgwMap{
				ID:           row[headers["ID"]],
				Name:         row[headers[nameColumn]],
				Destinations: strings.Split(row[headers[destinationsColumn]], ","),
				Image:        row[headers[imageColumn]],
			}
		}
	}
	for _, gateway := range gateways {
		targetTgwMapping[gateway.ID] = targetTgwMap{
			ID:    gateway.ID,
			Name:  gateway.Name,
			Image: drawio.AWSShape("Network Content Delivery", "Transit Gateway"),
		}
		attachedresources, _ := filterGateway([]helpers.TransitGateway{gateway})
		for resourceid := range attachedresources {
			destinations := []string{gateway.ID}
			if val, ok := targetTgwMapping[resourceid]; ok {
				destinations = unique(append(destinations, val.Destinations...))
			}
			image := ""
			switch helpers.TypeByResourceID(resourceid) {
			case vpcResourceType:
				image = drawio.AWSShape("Network Content Delivery", vpcColumn)
			case "vpn":
				image = drawio.AWSShape("Network Content Delivery", "Site-to-Site VPN")
			case "dxgw":
				image = drawio.AWSShape("Network Content Delivery", "Direct Connect Gateway")
			case tgwResourceType:
				image = drawio.AWSShape("Network Content Delivery", "Transit Gateway")
			}
			targetTgwMapping[resourceid] = targetTgwMap{
				ID:           resourceid,
				Name:         getName(resourceid),
				Destinations: destinations,
				Image:        image,
			}
		}
	}
	for _, mapping := range targetTgwMapping {
		content := make(map[string]any)
		content["ID"] = mapping.ID
		content[nameColumn] = mapping.Name
		content[destinationsColumn] = mapping.Destinations
		content[imageColumn] = mapping.Image
		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
}
