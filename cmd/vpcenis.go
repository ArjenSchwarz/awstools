package cmd

import (
	"fmt"
	"strings"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/spf13/cobra"
)

// enisCmd represents the enis command
var enisCmd = &cobra.Command{
	Use:   "enis",
	Short: "Get ENIs overview",
	Long: `Get an overview of ENIs in a VPC.

	Supported output formats are json, csv, table, and html. The graph formats
	(dot and drawio) are not supported for this command because ENIs are leaf
	resources without a meaningful from/to relationship to diagram.`,
	RunE: enis,
}

var vpceenisSplit bool

func init() {
	vpcCmd.AddCommand(enisCmd)
	enisCmd.Flags().BoolVar(&vpceenisSplit, "split", false, "Split the result by subnet")
}

// enisGraphFormatError reports whether the requested output format is a graph
// format the enis command cannot produce. ENIs have no from/to relationship to
// diagram, so dot and drawio are rejected with a clear error instead of letting
// go-output log.Fatal (non-split) or silently emit nothing (split). The
// comparison is case-insensitive to match the format normalisation in config.
func enisGraphFormatError(format string) error {
	switch strings.ToLower(format) {
	case "dot", "drawio":
		return fmt.Errorf("the %s output format is not supported by 'vpc enis'; supported formats are json, csv, table, and html", strings.ToLower(format))
	default:
		return nil
	}
}

func enis(_ *cobra.Command, _ []string) error {
	if err := enisGraphFormatError(settings.GetOutputFormat()); err != nil {
		return err
	}
	awsConfig := config.DefaultAwsConfig(*settings)
	ec2Client := awsConfig.Ec2Client()
	names := helpers.GetAllEC2ResourceNames(ec2Client, awsConfig.DirectConnectClient())
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
	return nil
}

func printENIs(interfaces []types.NetworkInterface, names map[string]string, resultTitle string, split bool, svc *ec2.Client) {
	keys := []string{"ENI", typeColumn, attachmentColumn, "IPs", vpcColumn, subnetColumn}
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = resultTitle
	output.Settings.SortKey = subnetColumn
	if split {
		// unset VPC and subnet
		output.Keys = []string{"ENI", typeColumn, attachmentColumn, "IPs"}
		output.Settings.SeparateTables = true
		output.Settings.SortKey = attachmentColumn
	}

	// Build cache once for all ENIs to avoid per-ENI API calls (T-727).
	cache := helpers.NewENILookupCache(svc, interfaces)

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
		content[typeColumn] = netinterface.InterfaceType
		content[attachmentColumn] = getNameAndIDFromMap(helpers.GetAttachmentFromCache(netinterface, cache), names)
		content["IPs"] = iparray
		content[vpcColumn] = getNameAndIDFromMap(aws.ToString(netinterface.VpcId), names)
		content[subnetColumn] = getNameAndIDFromMap(aws.ToString(netinterface.SubnetId), names)
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

func getNameAndIDFromMap(id string, names map[string]string) string {
	if names[id] != "" {
		if id == names[id] {
			return id
		}
		return fmt.Sprintf("%v (%v)", names[id], id)
	}
	return id
}
