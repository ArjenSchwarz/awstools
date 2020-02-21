package cmd

import (
	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/spf13/cobra"
)

// tgwdanglingCmd represents the tgwdangling command
var tgwdanglingCmd = &cobra.Command{
	Use:   "dangling",
	Short: "Check for incomplete routes",
	Long: `Check for incomplete routes.

	An incomplete route is defined as one that goes in only a single
	direction. e.g. while VPC1 connects to VPC2, there is no returning
	connection.`,
	Run: tgwdangling,
}

func init() {
	tgwCmd.AddCommand(tgwdanglingCmd)
}

func tgwdangling(cmd *cobra.Command, args []string) {
	resultTitle := "Transit Gateway uni-directional routes"
	svc := helpers.Ec2Session()
	gateways := helpers.GetAllTransitGateways(svc)
	keys := []string{"VPC", "VPCName", "DestinationVPC", "DestinationName"}
	output := helpers.OutputArray{Keys: keys, Title: resultTitle}
	vpcs := make(map[string][]string)
	for _, gateway := range gateways {
		for _, routetable := range gateway.RouteTables {
			for _, assoc := range routetable.SourceAttachments {
				vpcs[assoc.ResourceID] = []string{}
				for _, route := range routetable.Routes {
					vpcs[assoc.ResourceID] = append(vpcs[assoc.ResourceID], route.Attachment.ResourceID)
				}
			}

		}
	}

	for vpcid, targets := range vpcs {
		for _, target := range targets {
			if !contains(vpcs[target], vpcid) {
				content := make(map[string]string)
				content["VPC"] = vpcid
				content["VPCName"] = getName(vpcid)
				content["DestinationVPC"] = target
				content["DestinationName"] = getName(target)
				holder := helpers.OutputHolder{Contents: content}
				output.AddHolder(holder)
			}
		}

	}
	// fmt.Printf("%v", vpcs)
	output.Write(*settings)
}
