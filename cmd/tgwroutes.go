package cmd

import (
	"strings"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/drawio"
	"github.com/ArjenSchwarz/awstools/helpers"

	"github.com/spf13/cobra"
)

// tgwroutesCmd represents the tgwroutes command
var tgwroutesCmd = &cobra.Command{
	Use:   "routes",
	Short: "Get an overview of connections between Transit Gateway Route Tables and attached resources",
	Long: `This is currently limited to showing VPC attachments only, but that will be fixed soon.

	Supports a Draw.io output`,
	Run: tgwroutes,
}

func init() {
	tgwCmd.AddCommand(tgwroutesCmd)
}

func tgwroutes(cmd *cobra.Command, args []string) {
	resultTitle := "Overview of all routes"
	svc := helpers.Ec2Session()
	gateways := helpers.GetAllTransitGateways(svc)
	keys := []string{"ID", "Name", "DestinationVPCs", "TargetGateway"}
	if settings.IsDrawIO() {
		keys = append(keys, "Image")
	}
	output := helpers.OutputArray{Keys: keys, Title: resultTitle}
	switch settings.GetOutputFormat() {
	case "drawio":
		output.DrawIOHeader = createTgwRoutesDrawIOHeader()
	case "dot":
		dotcolumns := config.DotColumns{
			From: "DestinationVPCs",
			To:   "ID",
		}
		settings.DotColumns = &dotcolumns
	}
	vpcs := make(map[string]string)
	tgwrts := make(map[string][]string)
	for _, gateway := range gateways {
		for _, routetable := range gateway.RouteTables {
			tgwrts[routetable.ID] = []string{}
			for _, route := range routetable.Routes {
				tgwrts[routetable.ID] = append(tgwrts[routetable.ID], route.Attachment.ResourceID)
				if _, ok := vpcs[route.Attachment.ResourceID]; !ok {
					vpcs[route.Attachment.ResourceID] = ""
				}
			}
			for _, sourceattachment := range routetable.SourceAttachments {
				vpcs[sourceattachment.ResourceID] = routetable.ID
			}
		}
	}
	for rt, connectedvpcs := range tgwrts {
		content := make(map[string]string)
		content["ID"] = rt
		content["Name"] = getName(rt)
		content["DestinationVPCs"] = strings.Join(connectedvpcs, ",")
		if settings.IsDrawIO() {
			content["Image"] = drawio.ShapeAWSRoute53RouteTable
		}
		holder := helpers.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	for vpcid, tgw := range vpcs {
		content := make(map[string]string)
		content["ID"] = vpcid
		content["Name"] = getName(vpcid)
		content["TargetGateway"] = tgw
		if settings.IsDrawIO() {
			content["Image"] = drawio.ShapeAWSVPC
		}
		holder := helpers.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	output.Write(*settings)
}

func createTgwRoutesDrawIOHeader() drawio.Header {
	drawioheader := drawio.NewHeader("%Name%", "%Image%", "Image")
	drawioheader.SetHeightAndWidth("78", "78")
	connection := drawio.NewConnection()
	connection.From = "DestinationVPCs"
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
