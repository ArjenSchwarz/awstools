package cmd

import (
	"fmt"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/spf13/cobra"
)

// routesCmd represents the routes command
var routesCmd = &cobra.Command{
	Use:   "routes",
	Short: "Get VPC Routes",
	Long:  `Get an overview of the routes of all VPCs in the account.`,
	Run:   routes,
}

func init() {
	vpcCmd.AddCommand(routesCmd)
}

func routes(cmd *cobra.Command, args []string) {
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "VPC Routes for account " + getName(helpers.GetAccountID(awsConfig.StsClient()))
	routes := helpers.GetAllVPCRouteTables(awsConfig.Ec2Client())
	keys := []string{"AccountID", "Account Name", "ID", "Name", "VPC", "VPC Name", "Subnets", "Routes"}
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = resultTitle
	for _, routetable := range routes {
		content := make(map[string]interface{})
		content["ID"] = routetable.ID
		content["Name"] = getName(routetable.ID)
		content["VPC"] = routetable.Vpc.ID
		content["VPC Name"] = getName(routetable.Vpc.ID)
		var subnets []string
		for _, subnet := range routetable.Subnets {
			subnets = append(subnets, fmt.Sprintf("%v (%v)", getName(subnet), subnet))
		}
		content["Subnets"] = subnets
		content["AccountID"] = routetable.Vpc.AccountID
		content["Account Name"] = getName(routetable.Vpc.AccountID)
		var routelist []string
		for _, route := range routetable.Routes {
			routelist = append(routelist, fmt.Sprintf("%v: %v", route.DestinationCIDR, route.DestinationTarget))
		}
		content["Routes"] = routelist
		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	// if settings.IsDrawIO() {
	// 	keys = append(keys, "Image")
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
	// 		accountid := row[headers["AccountID"]]
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
	// 	content["Name"] = getName(id)
	// 	if len(entry) > 0 {
	// 		content["AccountID"] = vpcs[id].AccountID
	// 		content["PeeringIDs"] = peeringIDs
	// 		if settings.IsDrawIO() {
	// 			content["Image"] = drawio.ShapeAWSVPC
	// 		}
	// 	} else {
	// 		if settings.IsDrawIO() {
	// 			content["Image"] = drawio.ShapeAWSVPCPeering
	// 		}
	// 	}
	// }
	output.Write()
}
