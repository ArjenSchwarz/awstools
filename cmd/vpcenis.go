package cmd

import (
	"fmt"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/spf13/cobra"
)

// eniAttachmentLookupClient is the minimum EC2 API surface required by
// getAttachment to resolve ENI attachment metadata. Combining the three
// Describe* API client interfaces here keeps production callers passing a
// single *ec2.Client while letting tests supply a mock that paginates.
// Pagination for each underlying Describe call lives in helpers/ec2.go
// (T-657); this interface just pins the command-side path to those
// paginated helpers.
type eniAttachmentLookupClient interface {
	ec2.DescribeVpcEndpointsAPIClient
	ec2.DescribeNatGatewaysAPIClient
	ec2.DescribeTransitGatewayVpcAttachmentsAPIClient
}

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
	ec2Client := awsConfig.Ec2Client()
	names := helpers.GetAllEC2ResourceNames(ec2Client)
	resultTitle := "VPC ENIs for account " + getName(helpers.GetAccountID(awsConfig.StsClient()))
	interfaces := helpers.GetNetworkInterfaces(ec2Client)
	output := format.OutputArray{Settings: settings.NewOutputSettings()}
	if vpceenisSplit {
		output.Settings.SeparateTables = true
		groups := splitBySubnet(interfaces)
		for subnet, group := range groups {
			printENIs(group, names, fmt.Sprintf("%s - %s: %s", resultTitle, getNameAndIDFromMap(aws.ToString(group[0].VpcId), names), getNameAndIDFromMap(subnet, names)), true, ec2Client)
		}
	} else {
		printENIs(interfaces, names, resultTitle, false, ec2Client)
	}
	output.Write()
}

func printENIs(interfaces []types.NetworkInterface, names map[string]string, resultTitle string, split bool, svc eniAttachmentLookupClient) {
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
		if netinterface.Association != nil && netinterface.Association.PublicIp != nil {
			iparray = append(iparray, *netinterface.Association.PublicIp)
		}
		for _, ips := range netinterface.PrivateIpAddresses {
			if ips.PrivateIpAddress != nil {
				iparray = append(iparray, *ips.PrivateIpAddress)
			}
		}
		content["ENI"] = aws.ToString(netinterface.NetworkInterfaceId)
		content["Type"] = netinterface.InterfaceType
		content["Attachment"] = getNameAndIDFromMap(getAttachment(netinterface, svc), names)
		content["IPs"] = iparray
		content["VPC"] = getNameAndIDFromMap(aws.ToString(netinterface.VpcId), names)
		content["Subnet"] = getNameAndIDFromMap(aws.ToString(netinterface.SubnetId), names)
		output.AddContents(content)
	}
	output.AddToBuffer()
}

func splitBySubnet(interfaces []types.NetworkInterface) map[string][]types.NetworkInterface {
	result := make(map[string][]types.NetworkInterface)
	for _, netinterface := range interfaces {
		subnetID := aws.ToString(netinterface.SubnetId)
		result[subnetID] = append(result[subnetID], netinterface)
	}
	return result
}

// getAttachment resolves the attachment label for a given ENI. For instance
// ENIs it returns the instance ID directly; for TGW/NAT/VPC-endpoint ENIs it
// dispatches to the matching paginated helper (T-657 fixed those helpers to
// walk every page). The svc parameter is the composite client interface so
// tests can supply a paginating mock without a real *ec2.Client.
func getAttachment(netinterface types.NetworkInterface, svc eniAttachmentLookupClient) string {
	if netinterface.Attachment != nil && netinterface.Attachment.InstanceId != nil {
		return *netinterface.Attachment.InstanceId
	}
	if netinterface.InterfaceType == types.NetworkInterfaceTypeTransitGateway {
		return helpers.GetTransitGatewayFromNetworkInterface(netinterface, svc)
	}
	if netinterface.InterfaceType == types.NetworkInterfaceTypeNatGateway || netinterface.InterfaceType == "nat_gateway" {
		natgw := helpers.GetNatGatewayFromNetworkInterface(netinterface, svc)
		if natgw != nil {
			return aws.ToString(natgw.NatGatewayId)
		}
		return ""
	}
	if netinterface.InterfaceType == types.NetworkInterfaceTypeVpcEndpoint {
		endpoint := helpers.GetVPCEndpointFromNetworkInterface(netinterface, svc)
		if endpoint != nil {
			return fmt.Sprintf("%s (%s)", aws.ToString(endpoint.ServiceName), aws.ToString(endpoint.VpcEndpointId))
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
