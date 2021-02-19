package cmd

import (
	"strings"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/drawio"
	"github.com/ArjenSchwarz/awstools/helpers"

	"github.com/spf13/cobra"
)

// tgwroutetablesCmd represents the tgwroutes command
var tgwroutetablesCmd = &cobra.Command{
	Use:   "routetables",
	Short: "Get an overview of connections between Transit Gateway Route Tables and attached resources",
	Long: `Get an overview of connections between Transit Gateway Route Tables and attached resources
	This is currently limited to showing VPC attachments only, but that will be fixed soon.

	Using the --resource-id (-r) flag, you can limit the output to the provided resource.
	For a route table that means all the VPCs it connects to,
	while for a VPC that means all the route tables it connects
	to and through them what other VPCs can reach it or it can reach.

	Supports a Draw.io output`,
	Run: tgwroutes,
}

var tgwresourceid string

func init() {
	tgwCmd.AddCommand(tgwroutetablesCmd)
	tgwroutetablesCmd.Flags().StringVarP(&tgwresourceid, "resource-id", "r", "", "The id of the resource you want to limit to")
}

func tgwroutes(cmd *cobra.Command, args []string) {
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "Overview of all routes"
	gateways := helpers.GetAllTransitGateways(awsConfig.Ec2Client())
	keys := []string{"ID", "Name", "Destinations", "TargetGateway"}
	if settings.IsDrawIO() {
		keys = append(keys, "Image")
	}
	output := helpers.OutputArray{Keys: keys, Title: resultTitle}
	switch settings.GetOutputFormat() {
	case "drawio":
		output.DrawIOHeader = createTgwRoutesDrawIOHeader()
	case "dot":
		dotcolumns := config.DotColumns{
			From: "Destinations",
			To:   "ID",
		}
		settings.DotColumns = &dotcolumns
	}

	attachedresources, tgwrts := filterGateway(gateways)

	for rt, connectedvpcs := range tgwrts {
		content := make(map[string]string)
		content["ID"] = rt
		content["Name"] = getName(rt)
		content["Destinations"] = strings.Join(connectedvpcs, ",")
		if settings.IsDrawIO() {
			content["Image"] = drawio.ShapeAWSRoute53RouteTable
		}
		holder := helpers.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	for resourceid, tgw := range attachedresources {
		content := make(map[string]string)
		content["ID"] = resourceid
		content["Name"] = getName(resourceid)
		content["TargetGateway"] = tgw
		if settings.IsDrawIO() {
			switch helpers.TypeByResourceID(resourceid) {
			case "vpc":
				content["Image"] = drawio.ShapeAWSVPC
			case "vpn":
				content["Image"] = drawio.ShapeAWSClientVPN
			}
		}
		holder := helpers.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	output.Write(*settings)
}

func filterGateway(gateways []helpers.TransitGateway) (map[string]string, map[string][]string) {
	limitertype := helpers.TypeByResourceID(tgwresourceid)
	attachedresources := make(map[string]string)
	tgwrts := make(map[string][]string)

	for _, gateway := range gateways {
		// only add relevant gateway if filtered by gateway
		if limitertype == "tgw" && gateway.ID != tgwresourceid {
			continue
		}
		for _, routetable := range gateway.RouteTables {
			// only add relevant route tables if filtered by route table
			if limitertype == "tgw-rtb" && routetable.ID != tgwresourceid {
				continue
			}
			tgwrts[routetable.ID] = []string{}
			for _, route := range routetable.Routes {
				tgwrts[routetable.ID] = append(tgwrts[routetable.ID], route.Attachment.ResourceID)
				if _, ok := attachedresources[route.Attachment.ResourceID]; !ok {
					attachedresources[route.Attachment.ResourceID] = ""
				}
			}
			for _, sourceattachment := range routetable.SourceAttachments {
				attachedresources[sourceattachment.ResourceID] = routetable.ID
			}
		}
	}
	// For VPC pass over everything and remove what's not relevant
	if limitertype == "vpc" {
		attachedtgwrts := []string{}
		for tgwid, destinationvpcs := range tgwrts {
			if !contains(destinationvpcs, tgwresourceid) && tgwid != attachedresources[tgwresourceid] {
				delete(tgwrts, tgwid)
			}
			if contains(destinationvpcs, tgwresourceid) {
				tgwrts[tgwid] = []string{tgwresourceid}
				attachedtgwrts = append(attachedtgwrts, tgwid)
			}
		}
		for resourceid, tgwrt := range attachedresources {
			if resourceid != tgwresourceid && !contains(attachedtgwrts, tgwrt) {
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
