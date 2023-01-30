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

func tgwoverview(cmd *cobra.Command, args []string) {
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "Transit Gateway Routes in account " + getName(helpers.GetAccountID(awsConfig.StsClient()))
	gateways := helpers.GetAllTransitGateways(awsConfig.Ec2Client())
	keys := []string{"Transit Gateway Account", "Transit Gateway", "Route Table", "CIDR", "Target", "Target Type", "State"}
	if settings.IsDrawIO() {
		keys = []string{"ID", "Name", "Destinations", "Image"}
	}
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = resultTitle
	output.Settings.SortKey = "Route Table"
	if settings.IsDrawIO() {
		createTgwOverviewDrawIO(&output, gateways)
	} else {
		for _, gateway := range gateways {
			for _, routetable := range gateway.RouteTables {
				for _, route := range routetable.Routes {
					if excludeRouteTarget == route.Attachment.ResourceID {
						continue
					}
					if !includeBlackhole && route.State == "blackhole" {
						continue
					}
					content := make(map[string]interface{})
					content["Transit Gateway Account"] = getNameWithId(gateway.AccountID)
					content["Transit Gateway"] = getNameWithId(gateway.ID)
					content["Route Table"] = getNameWithId(routetable.ID)
					content["CIDR"] = route.CIDR
					if route.Attachment.ResourceID != "" {
						content["Target"] = getNameWithId(route.Attachment.ResourceID)
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
			}
		}
	}
	output.Write()
}

func createTgwOverviewDrawIO(output *format.OutputArray, gateways []helpers.TransitGateway) {
	drawioheader := drawio.NewHeader("%Name%", "%Image%", "Image")
	drawioheader.SetHeightAndWidth("78", "78")
	connection := drawio.NewConnection()
	connection.From = "Destinations"
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
				Name:         row[headers["Name"]],
				Destinations: strings.Split(row[headers["Destinations"]], ","),
				Image:        row[headers["Image"]],
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
			case "vpc":
				image = drawio.AWSShape("Network Content Delivery", "VPC")
			case "vpn":
				image = drawio.AWSShape("Network Content Delivery", "Site-to-Site VPN")
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
		content := make(map[string]interface{})
		content["ID"] = mapping.ID
		content["Name"] = mapping.Name
		content["Destinations"] = mapping.Destinations
		content["Image"] = mapping.Image
		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
}
