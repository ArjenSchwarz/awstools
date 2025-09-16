package cmd

import (
	"fmt"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/spf13/cobra"
)

// peeringsCmd represents the peerings command
var enisCmd = &cobra.Command{
	Use:   "enis",
	Short: "Get ENIs overview",
	Long: `Get an overview of ENIs in a VPC. For a graphical option consider using
	the dot or drawio output formats.

	awstools vpc peerings -o dot | dot -Tpng  -o peerings.png
	awstools vpc peerings -o drawio | pbcopy`,
	Run: enis,
}

var vpceenisSplit bool

func init() {
	vpcCmd.AddCommand(enisCmd)
	enisCmd.Flags().BoolVar(&vpceenisSplit, "split", false, "Split the result by subnet")
}

func enis(_ *cobra.Command, _ []string) {
	awsConfig := config.DefaultAwsConfig(*settings)
	names := helpers.GetAllEC2ResourceNames(awsConfig.Ec2Client())
	resultTitle := "VPC ENIs for account " + getName(helpers.GetAccountID(awsConfig.StsClient()))
	interfaces := helpers.GetNetworkInterfaces(awsConfig.Ec2Client())
	output := format.OutputArray{Settings: settings.NewOutputSettings()}
	if vpceenisSplit {
		output.Settings.SeparateTables = true
		groups := splitBySubnet(interfaces)
		for subnet, group := range groups {
			printENIs(group, names, fmt.Sprintf("%s - %s: %s", resultTitle, getNameAndIDFromMap(*group[0].VpcId, names), getNameAndIDFromMap(subnet, names)), true)
		}
	} else {
		printENIs(interfaces, names, resultTitle, false)
	}
	output.Write()
}

func printENIs(interfaces []types.NetworkInterface, names map[string]string, resultTitle string, split bool) {
	keys := []string{"ENI", "Type", "Attachment", "IPs", "VPC", "Subnet"}
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = resultTitle
	output.Settings.SortKey = "Subnet"
	if split {
		// unset VPC and subnet
		output.Keys = []string{"ENI", "Type", "Attachment", "IPs"}
		output.Settings.SeparateTables = true
		output.Settings.SortKey = "Attachment"
	}

	for _, netinterface := range interfaces {
		content := make(map[string]any)
		iparray := make([]string, 0)
		if netinterface.Association != nil {
			iparray = append(iparray, *netinterface.Association.PublicIp)
		}
		for _, ips := range netinterface.PrivateIpAddresses {
			iparray = append(iparray, *ips.PrivateIpAddress)
		}
		content["ENI"] = *netinterface.NetworkInterfaceId
		content["Type"] = netinterface.InterfaceType
		content["Attachment"] = getNameAndIDFromMap(getAttachment(netinterface), names)
		content["IPs"] = iparray
		content["VPC"] = getNameAndIDFromMap(*netinterface.VpcId, names)
		content["Subnet"] = getNameAndIDFromMap(*netinterface.SubnetId, names)
		output.AddContents(content)
	}
	output.AddToBuffer()
}

func splitBySubnet(interfaces []types.NetworkInterface) map[string][]types.NetworkInterface {
	result := make(map[string][]types.NetworkInterface)
	for _, netinterface := range interfaces {
		result[*netinterface.SubnetId] = append(result[*netinterface.SubnetId], netinterface)
	}
	return result
}

func getAttachment(netinterface types.NetworkInterface) string {
	awsConfig := config.DefaultAwsConfig(*settings)
	if netinterface.Attachment != nil && netinterface.Attachment.InstanceId != nil {
		return *netinterface.Attachment.InstanceId
	}
	if netinterface.InterfaceType == types.NetworkInterfaceTypeTransitGateway {
		return helpers.GetTransitGatewayFromNetworkInterface(netinterface, awsConfig.Ec2Client())
	}
	if netinterface.InterfaceType == types.NetworkInterfaceTypeNatGateway || netinterface.InterfaceType == "nat_gateway" {
		natgw := helpers.GetNatGatewayFromNetworkInterface(netinterface, awsConfig.Ec2Client())
		if natgw != nil {
			return *natgw.NatGatewayId
		}
		return ""
	}
	if netinterface.InterfaceType == types.NetworkInterfaceTypeVpcEndpoint {
		endpoint := helpers.GetVPCEndpointFromNetworkInterface(netinterface, awsConfig.Ec2Client())
		if endpoint != nil {
			return fmt.Sprintf("%s (%s)", *endpoint.ServiceName, *endpoint.VpcEndpointId)
		}
		return ""
	}
	return ""
}

func getNameAndIDFromMap(id string, names map[string]string) string {
	if names[id] != "" {
		if id == names[id] {
			return id
		}
		return fmt.Sprintf("%v (%v)", names[id], id)
	}
	return id
}
