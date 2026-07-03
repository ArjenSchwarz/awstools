package cmd

import (
	"fmt"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/cobra"
)

// routesCmd represents the routes command
var routesCmd = &cobra.Command{
	Use:   "routes",
	Short: "Get VPC Routes",
	Long:  `Get an overview of the routes of all VPCs in the account.`,
	RunE:  routes,
}

func init() {
	vpcCmd.AddCommand(routesCmd)
}

func routes(cmd *cobra.Command, _ []string) error {
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "VPC Routes for account " + getName(helpers.GetAccountID(awsConfig.StsClient()))
	routes := helpers.GetAllVPCRouteTables(awsConfig.Ec2Client())
	keys := []string{accountIDColumn, "Account Name", "ID", nameColumn, vpcColumn, "VPC Name", "Default", "Subnets", routesColumn}
	rows := make([]map[string]any, 0, len(routes))
	for _, routetable := range routes {
		content := make(map[string]any)
		content["ID"] = routetable.ID
		content[nameColumn] = getName(routetable.ID)
		content[vpcColumn] = routetable.Vpc.ID
		content["VPC Name"] = getName(routetable.Vpc.ID)
		content["Default"] = routetable.Default
		var subnets []string
		for _, subnet := range routetable.Subnets {
			subnets = append(subnets, fmt.Sprintf("%v (%v)", getName(subnet), subnet))
		}
		// The main/default route table applies to every subnet that has no
		// explicit association. EC2 reports no SubnetId for it, so make that
		// state visible instead of leaving the column blank (T-1270).
		if routetable.Default && len(subnets) == 0 {
			subnets = append(subnets, "main (all unassociated subnets)")
		}
		content["Subnets"] = subnets
		content[accountIDColumn] = routetable.Vpc.AccountID
		content["Account Name"] = getName(routetable.Vpc.AccountID)
		var routelist []string
		for _, route := range routetable.Routes {
			routelist = append(routelist, fmt.Sprintf("%v: %v", route.DestinationCIDR, route.DestinationTarget))
		}
		content[routesColumn] = routelist
		rows = append(rows, content)
	}
	// if settings.IsDrawIO() {
	// 	keys = append(keys, imageColumn)
	// }
	//
	// switch settings.GetOutputFormat() {
	// case "drawio":
	// 	output.DrawIOHeader = createVpcPeeringsDrawIOHeader()
	// case "dot":
	// 	dotcolumns := config.DotColumns{
	// 		From: "ID",
	// 		To:   "PeeringIDs",
	// 	}
	// 	settings.DotColumns = &dotcolumns
	// }
	// vpcs := make(map[string]helpers.VPCHolder)
	// sorted := make(map[string][]string)
	// if settings.ShouldCombineAndAppend() {
	// 	headers, previousResults := drawio.GetHeaderAndContentsFromFile(*settings.OutputFile)
	// 	for _, row := range previousResults {
	// 		id := row[headers["ID"]]
	// 		accountid := row[headers[accountIDColumn]]
	// 		peeringids := row[headers["PeeringIDs"]]
	// 		if peeringids != "" {
	// 			sorted[id] = strings.Split(peeringids, ",")
	// 			vpcHolder := helpers.VPCHolder{
	// 				ID:        id,
	// 				AccountID: accountid,
	// 			}
	// 			vpcs[id] = vpcHolder
	// 		} else {
	// 			sorted[id] = []string{}
	// 		}
	// 	}
	// }

	// for _, peering := range peerings {
	// 	if _, ok := sorted[peering.PeeringID]; !ok {
	// 		sorted[peering.PeeringID] = []string{}
	// 	}
	// 	if _, ok := sorted[peering.AccepterVpc.ID]; !ok {
	// 		sorted[peering.AccepterVpc.ID] = []string{peering.PeeringID}
	// 		vpcs[peering.AccepterVpc.ID] = peering.AccepterVpc
	// 	} else {
	// 		sorted[peering.AccepterVpc.ID] = append(sorted[peering.AccepterVpc.ID], peering.PeeringID)
	// 	}
	// 	if _, ok := sorted[peering.RequesterVpc.ID]; !ok {
	// 		sorted[peering.RequesterVpc.ID] = []string{peering.PeeringID}
	// 		vpcs[peering.RequesterVpc.ID] = peering.RequesterVpc
	// 	} else {
	// 		sorted[peering.RequesterVpc.ID] = append(sorted[peering.RequesterVpc.ID], peering.PeeringID)
	// 	}
	// }
	// for id, entry := range sorted {
	// 	peeringIDs := unique(entry)
	// 	content := make(map[string]interface{})
	// 	content["ID"] = id
	// 	content[nameColumn] = getName(id)
	// 	if len(entry) > 0 {
	// 		content[accountIDColumn] = vpcs[id].AccountID
	// 		content["PeeringIDs"] = peeringIDs
	// 		if settings.IsDrawIO() {
	// 			content[imageColumn] = drawio.ShapeAWSVPC
	// 		}
	// 	} else {
	// 		if settings.IsDrawIO() {
	// 			content[imageColumn] = drawio.ShapeAWSVPCPeering
	// 		}
	// 	}
	// }
	doc := output.New().
		Table(resultTitle, rows, output.WithKeys(keys...)).
		Build()
	return settings.RenderDocument(cmd.Context(), doc)
}
