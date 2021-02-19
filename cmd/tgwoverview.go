package cmd

import (
	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	"github.com/spf13/cobra"
)

// tgwoverviewCmd represents the tgwoverview command
var tgwoverviewCmd = &cobra.Command{
	Use:   "overview",
	Short: "A basic overview of the Transit Gateway",
	Long: `Provides an overview of all the route tables and routes in the Transit Gateway.
	This can be improved on, but offers a simple text based overview with all relevant information
	`,
	Run: tgwoverview,
}

func init() {
	tgwCmd.AddCommand(tgwoverviewCmd)
}

func tgwoverview(cmd *cobra.Command, args []string) {
	awsConfig := config.DefaultAwsConfig(*settings)
	resultTitle := "Transit Gateway Routes in account " + getName(helpers.GetAccountID(awsConfig.StsClient()))
	gateways := helpers.GetAllTransitGateways(awsConfig.Ec2Client())
	keys := []string{"Transit Gateway Account", "Transit Gateway ID", "Route Table ID", "Route Table Name", "CIDR", "Target VPC"}
	output := helpers.OutputArray{Keys: keys, Title: resultTitle}
	for _, gateway := range gateways {
		for _, routetable := range gateway.RouteTables {
			for _, route := range routetable.Routes {
				content := make(map[string]string)
				content["Transit Gateway Account"] = gateway.AccountID
				content["Transit Gateway ID"] = gateway.ID
				content["Route Table ID"] = routetable.ID
				content["Route Table Name"] = routetable.Name
				content["CIDR"] = route.CIDR
				content["Target VPC"] = getName(route.Attachment.ResourceID)
				holder := helpers.OutputHolder{Contents: content}
				output.AddHolder(holder)
			}
		}

	}
	output.Write(*settings)
}
