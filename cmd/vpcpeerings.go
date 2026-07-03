package cmd

import (
	"maps"
	"slices"
	"strings"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	output "github.com/ArjenSchwarz/go-output/v2"
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
	RunE: peerings,
}

func init() {
	vpcCmd.AddCommand(peeringsCmd)
}

func peerings(cmd *cobra.Command, _ []string) error {
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "VPC Peerings for account " + getName(helpers.GetAccountID(awsConfig.StsClient()))
	peerings := helpers.GetAllVpcPeers(awsConfig.Ec2Client())
	keys := []string{"ID", nameColumn, accountIDColumn, "PeeringIDs"}
	if settings.IsDrawIO() {
		keys = append(keys, imageColumn)
	}
	vpcs := make(map[string]helpers.VPCHolder)
	sorted := make(map[string][]string)
	merged := false
	if settings.ShouldCombineAndAppend() {
		parsed, err := output.ParseDrawIOFile(settings.GetString("output.file"))
		if err != nil {
			return err
		}
		for _, record := range parsed.Records {
			id := drawIORecordString(record, "ID")
			accountid := drawIORecordString(record, accountIDColumn)
			peeringids := drawIORecordString(record, "PeeringIDs")
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
		merged = true
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
	rows := []map[string]any{}
	// Sort the map keys before ranging so row order is deterministic between
	// runs (R2.8).
	for _, id := range slices.Sorted(maps.Keys(sorted)) {
		entry := sorted[id]
		peeringIDs := unique(entry)
		content := make(map[string]any)
		content["ID"] = id
		content[nameColumn] = getName(id)
		if len(entry) > 0 {
			content[accountIDColumn] = vpcs[id].AccountID
			content["PeeringIDs"] = peeringIDs
			if settings.IsDrawIO() {
				content[imageColumn] = awsShape("Network Content Delivery", vpcColumn)
			}
		} else if settings.IsDrawIO() {
			content[imageColumn] = awsShape("Network Content Delivery", "Peering Connection")
		}
		rows = append(rows, content)
	}

	docs := config.DocumentSet{
		Table: output.New().
			Table(resultTitle, rows, output.WithKeys(keys...)).
			Build(),
	}
	if settings.NeedsGraphFormat() {
		docs.Graph = output.New().
			Graph(resultTitle, graphEdges(rows, "ID", "PeeringIDs")).
			Build()
	}
	if settings.IsDrawIO() {
		docs.DrawIO = output.New().
			DrawIO(resultTitle, drawIORecords(rows), createVpcPeeringsDrawIOHeader()).
			Build()
	}
	var opts []config.RenderOption
	if merged {
		// The prior file contents are already merged into the document, so
		// the file must be written fresh instead of appended to.
		opts = append(opts, config.WithFileOverwrite())
	}
	return settings.RenderDocuments(cmd.Context(), docs, opts...)
}

func createVpcPeeringsDrawIOHeader() output.DrawIOHeader {
	drawioheader := drawIOBaseHeader("%Name%", "%Image%", imageColumn)
	connection := drawIOConnection()
	connection.From = "PeeringIDs"
	connection.To = "ID"
	connection.Invert = false
	connection.Style = "curved=1;endArrow=none;endFill=1;fontSize=11;"
	drawioheader.Connections = append(drawioheader.Connections, connection)
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
