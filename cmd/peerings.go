package cmd

import (
	"encoding/csv"
	"encoding/json"
	"io/ioutil"
	"log"
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
	setPeeringConfigs()
	svc := helpers.Ec2Session()
	peerings := helpers.GetAllVpcPeers(svc)
	keys := []string{"ID", "Name", "AccountID", "PeeringIDs"}
	if *settings.Verbose {
		keys = append(keys, "Image")
	}
	output := helpers.OutputArray{Keys: keys}
	vpcs := make(map[string]helpers.VPCHolder)
	sorted := make(map[string][]string)
	if *settings.AppendToOutput {
		previousResults := appendToDrawIO()
		for _, row := range previousResults {
			// TODO fix magic numbers
			id := row[0]
			accountid := row[2]
			peeringids := row[3]
			if peeringids != "" {
				sorted[id] = strings.Split(peeringids, ",")
				vpcHolder := helpers.VPCHolder{
					ID:        id,
					AccountID: accountid,
				}
				vpcs[row[0]] = vpcHolder
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
		content := make(map[string]string)
		content["ID"] = id
		content["Name"] = setNames(id)
		if len(entry) > 0 {
			content["AccountID"] = vpcs[id].AccountID
			content["PeeringIDs"] = strings.Join(peeringIDs, ",")
			if *settings.Verbose {
				content["Image"] = drawio.ShapeAWSVPC
			}
		} else {
			if *settings.Verbose {
				content["Image"] = drawio.ShapeAWSVPCPeering
			}
		}
		holder := helpers.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	output.Write(*settings)
}

func setPeeringConfigs() {
	switch strings.ToLower(*settings.OutputFormat) {
	case "drawio":
		*settings.Verbose = true
		drawioheader := drawio.NewHeader("%Name%", "%Image%", "Image")
		connection := drawio.NewConnection()
		connection.From = "PeeringIDs"
		connection.To = "ID"
		connection.Invert = false
		connection.Style = "curved=1;endArrow=none;endFill=1;fontSize=11;"
		drawioheader.AddConnection(connection)
		header := drawioheader.String()
		settings.OutputHeaders = &header
	case "dot":
		dotcolumns := config.DotColumns{
			From: "ID",
			To:   "PeeringIDs",
		}
		settings.DotColumns = &dotcolumns
	}
}

func setNames(id string) string {
	if *settings.NameFile != "" {
		nameFile, err := ioutil.ReadFile(*settings.NameFile)
		if err != nil {
			panic(err)
		}
		values := make(map[string]string)
		err = json.Unmarshal(nameFile, &values)
		if err != nil {
			panic(err)
		}
		if val, ok := values[id]; ok {
			return val
		}
	}
	return id
}

func appendToDrawIO() [][]string {
	originalfile, err := ioutil.ReadFile(*settings.OutputFile)
	if err != nil {
		panic(err)
	}
	originalString := string(originalfile)
	r := csv.NewReader(strings.NewReader(originalString))
	r.Comment = '#'
	records, err := r.ReadAll()
	if err != nil {
		log.Fatal(err)
	}
	// Strip header as we don't need it
	return records[1:]
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
