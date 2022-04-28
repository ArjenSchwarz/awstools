package cmd

import (
	"strings"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/ArjenSchwarz/awstools/lib/format"
	"github.com/ArjenSchwarz/awstools/lib/format/drawio"
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
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "VPC Peerings for account " + getName(helpers.GetAccountID(awsConfig.StsClient()))
	peerings := helpers.GetAllVpcPeers(awsConfig.Ec2Client())
	keys := []string{"ID", "Name", "AccountID", "PeeringIDs"}
	if settings.IsDrawIO() {
		keys = append(keys, "Image")
	}
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = resultTitle
	if settings.IsDrawIO() {
		output.Settings.DrawIOHeader = createVpcPeeringsDrawIOHeader()
	}
	if output.Settings.NeedsFromToColumns() {
		output.Settings.AddFromToColumns("ID", "PeeringIDs")
	}
	vpcs := make(map[string]helpers.VPCHolder)
	sorted := make(map[string][]string)
	if settings.ShouldCombineAndAppend() {
		headers, previousResults := drawio.GetHeaderAndContentsFromFile(settings.GetString("output.file"))
		for _, row := range previousResults {
			id := row[headers["ID"]]
			accountid := row[headers["AccountID"]]
			peeringids := row[headers["PeeringIDs"]]
			if peeringids != "" {
				sorted[id] = strings.Split(peeringids, ",")
				vpcHolder := helpers.VPCHolder{
					ID:        id,
					AccountID: accountid,
				}
				vpcs[id] = vpcHolder
			} else {
				sorted[id] = []string{}
			}
		}
	}

	for _, peering := range peerings {
		if _, ok := sorted[peering.PeeringID]; !ok {
			sorted[peering.PeeringID] = []string{}
		}
		if _, ok := sorted[peering.AccepterVpc.ID]; !ok {
			sorted[peering.AccepterVpc.ID] = []string{peering.PeeringID}
			vpcs[peering.AccepterVpc.ID] = peering.AccepterVpc
		} else {
			sorted[peering.AccepterVpc.ID] = append(sorted[peering.AccepterVpc.ID], peering.PeeringID)
		}
		if _, ok := sorted[peering.RequesterVpc.ID]; !ok {
			sorted[peering.RequesterVpc.ID] = []string{peering.PeeringID}
			vpcs[peering.RequesterVpc.ID] = peering.RequesterVpc
		} else {
			sorted[peering.RequesterVpc.ID] = append(sorted[peering.RequesterVpc.ID], peering.PeeringID)
		}
	}
	for id, entry := range sorted {
		peeringIDs := unique(entry)
		content := make(map[string]interface{})
		content["ID"] = id
		content["Name"] = getName(id)
		if len(entry) > 0 {
			content["AccountID"] = vpcs[id].AccountID
			content["PeeringIDs"] = strings.Join(peeringIDs, ",")
			if settings.IsDrawIO() {
				content["Image"] = drawio.AWSShape("Network Content Delivery", "VPC")
			}
		} else {
			if settings.IsDrawIO() {
				content["Image"] = drawio.AWSShape("Network Content Delivery", "Peering Connection")
			}
		}
		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	output.Write()
}

func createVpcPeeringsDrawIOHeader() drawio.Header {
	drawioheader := drawio.NewHeader("%Name%", "%Image%", "Image")
	drawioheader.SetHeightAndWidth("78", "78")
	connection := drawio.NewConnection()
	connection.From = "PeeringIDs"
	connection.To = "ID"
	connection.Invert = false
	connection.Style = "curved=1;endArrow=none;endFill=1;fontSize=11;"
	drawioheader.AddConnection(connection)
	return drawioheader
}

func unique(stringSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range stringSlice {
		if _, ok := keys[entry]; !ok {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
