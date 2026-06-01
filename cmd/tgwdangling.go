package cmd

import (
	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	format "github.com/ArjenSchwarz/go-output"
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

func tgwdangling(_ *cobra.Command, _ []string) {
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "Transit Gateway uni-directional routes"
	gateways := helpers.GetAllTransitGateways(awsConfig.Ec2Client())
	keys := []string{vpcColumn, "VPCName", "DestinationVPC", "DestinationName"}
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = resultTitle
	vpcs := danglingRouteTargets(gateways)

	for vpcid, targets := range vpcs {
		for _, target := range targets {
			if !contains(vpcs[target], vpcid) {
				content := make(map[string]any)
				content[vpcColumn] = vpcid
				content["VPCName"] = getName(vpcid)
				content["DestinationVPC"] = target
				content["DestinationName"] = getName(target)
				holder := format.OutputHolder{Contents: content}
				output.AddHolder(holder)
			}
		}

	}
	// fmt.Printf("%v", vpcs)
	output.Write()
}

// danglingRouteTargets builds, for each source attachment, the list of resource
// IDs it routes to. Blackhole routes are parsed with an empty attachment
// ResourceID (see helpers.parseBlackholeRoute); they have no destination, so
// they are skipped to avoid emitting dangling rows with an empty destination.
func danglingRouteTargets(gateways []helpers.TransitGateway) map[string][]string {
	vpcs := make(map[string][]string)
	for _, gateway := range gateways {
		for _, routetable := range gateway.RouteTables {
			for _, assoc := range routetable.SourceAttachments {
				vpcs[assoc.ResourceID] = []string{}
				for _, route := range routetable.Routes {
					if route.Attachment.ResourceID == "" {
						continue
					}
					vpcs[assoc.ResourceID] = append(vpcs[assoc.ResourceID], route.Attachment.ResourceID)
				}
			}

		}
	}
	return vpcs
}
