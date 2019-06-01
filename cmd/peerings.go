package cmd

import (
	"strings"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/drawio"
	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/spf13/cobra"
)

// peeringsCmd represents the peerings command
var peeringsCmd = &cobra.Command{
	Use:   "peerings",
	Short: "Get VPC Peerings",
	Long: `Get an overview of Peerings. For a graphical option consider using
	the dot or drawio output formats.

	awstools vpc peerings -o dot | dot -Tpng  -o peerings.png
	awstools vpc peerings -o drawio | pbcopy`,
	Run: peerings,
}

func init() {
	vpcCmd.AddCommand(peeringsCmd)
}

func peerings(cmd *cobra.Command, args []string) {
	switch strings.ToLower(*settings.OutputFormat) {
	case "drawio":
		*settings.Verbose = true
		drawioheader := drawio.NewHeader("%Name%", "%Image%", "Image")
		connection := drawio.NewConnection()
		connection.From = "VPCIDs"
		connection.To = "Name"
		connection.Invert = false
		// connection.Label = "Connects"
		connection.Style = "curved=1;endArrow=none;endFill=1;fontSize=11;"
		drawioheader.AddConnection(connection)
		header := drawioheader.String()
		settings.OutputHeaders = &header
	case "dot":
		dotcolumns := config.DotColumns{
			From: "Name",
			To:   "VPCIDs",
		}
		settings.DotColumns = &dotcolumns
	}
	svc := helpers.Ec2Session()
	peerings := helpers.GetAllVpcPeers(svc)
	keys := []string{"Name", "VPCIDs"}
	if *settings.Verbose {
		keys = append(keys, "Image")
	}
	output := helpers.OutputArray{Keys: keys}
	var vpcs []string
	for _, peering := range peerings {
		content := make(map[string]string)
		content["Name"] = peering.PeeringName
		content["VPCIDs"] = peering.RequesterVpc + " (" + peering.RequesterAccount + ")," + peering.AccepterVpc + " (" + peering.AccepterAccount + ")"
		if *settings.Verbose {
			content["Image"] = drawio.ShapeAWSVPCPeering
		}
		holder := helpers.OutputHolder{Contents: content}
		output.AddHolder(holder)
		if !stringInSlice(peering.RequesterVpc, vpcs) {
			vpcs = append(vpcs, peering.RequesterVpc)
			content := make(map[string]string)
			content["Name"] = peering.RequesterVpc + " (" + peering.RequesterAccount + ")"
			if *settings.Verbose {
				content["Image"] = drawio.ShapeAWSVPC
			}
			holder := helpers.OutputHolder{Contents: content}
			output.AddHolder(holder)
		}
		if !stringInSlice(peering.AccepterVpc, vpcs) {
			vpcs = append(vpcs, peering.AccepterVpc)
			content := make(map[string]string)
			content["Name"] = peering.AccepterVpc + " (" + peering.AccepterAccount + ")"
			if *settings.Verbose {
				content["Image"] = drawio.ShapeAWSVPC
			}
			holder := helpers.OutputHolder{Contents: content}
			output.AddHolder(holder)
		}
	}
	output.Write(*settings)
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
