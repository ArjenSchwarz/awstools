package cmd

import (
	"fmt"
	"strings"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/aws/aws-sdk-go-v2/aws"
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

	// Build cache once for all ENIs to avoid per-ENI API calls (T-727).
	cache := helpers.NewENILookupCache(ec2Client, interfaces)
	attachmentFor := func(eni types.NetworkInterface) string {
		return helpers.GetAttachmentFromCache(eni, cache)
	}

	if vpceenisSplit {
		writeENIsBySubnet(interfaces, names, resultTitle, attachmentFor)
		return nil
	}

	writeENIs(interfaces, names, resultTitle, attachmentFor)
	return nil
}

// writeENIs renders the single (non-split) ENI table. It writes the populated
// OutputArray directly so both stdout and --file serialize the same ENI rows
// (T-1294). Routing through the shared package buffer would leave --file empty
// because Write() resets the buffer after the stdout pass and before the file
// pass.
func writeENIs(interfaces []types.NetworkInterface, names map[string]string, resultTitle string, attachmentFor func(types.NetworkInterface) string) {
	output := buildENIOutput(interfaces, names, resultTitle, false, attachmentFor)
	output.Write()
}

// writeENIsBySubnet renders one separate table per subnet group. Each group is
// pushed to the shared buffer for stdout fidelity (separate, per-subnet titled
// tables). The final Write() is called on a populated OutputArray that holds
// every row across all groups so the --file branch has real data to serialize
// even after Write() resets the buffer following the stdout pass (T-1294).
func writeENIsBySubnet(interfaces []types.NetworkInterface, names map[string]string, resultTitle string, attachmentFor func(types.NetworkInterface) string) {
	groups := splitBySubnet(interfaces)

	combined := format.OutputArray{Keys: splitENIKeys, Settings: settings.NewOutputSettings()}
	combined.Settings.SeparateTables = true
	combined.Settings.Title = resultTitle
	combined.Settings.SortKey = attachmentColumn

	for subnet, group := range groups {
		groupTitle := fmt.Sprintf("%s - %s: %s", resultTitle, getNameAndIDFromMap(aws.ToString(group[0].VpcId), names), getNameAndIDFromMap(subnet, names))
		groupOutput := buildENIOutput(group, names, groupTitle, true, attachmentFor)
		// Accumulate each row into the combined array so the final Write() has
		// real contents for the --file branch.
		for _, holder := range groupOutput.Contents {
			combined.AddContents(holder.Contents)
		}
		groupOutput.AddToBuffer()
	}
	combined.Write()
}

const (
	eniColumn = "ENI"
	ipsColumn = "IPs"
)

var fullENIKeys = []string{eniColumn, typeColumn, attachmentColumn, ipsColumn, vpcColumn, subnetColumn}
var splitENIKeys = []string{eniColumn, typeColumn, attachmentColumn, ipsColumn}

// buildENIOutput constructs a populated OutputArray for the supplied ENIs. The
// attachmentFor resolver decouples row construction from the AWS-backed
// ENILookupCache so the output shape is unit testable (T-1294).
func buildENIOutput(interfaces []types.NetworkInterface, names map[string]string, resultTitle string, split bool, attachmentFor func(types.NetworkInterface) string) format.OutputArray {
	keys := fullENIKeys
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = resultTitle
	output.Settings.SortKey = subnetColumn
	if split {
		// unset VPC and subnet
		output.Keys = splitENIKeys
		output.Settings.SeparateTables = true
		output.Settings.SortKey = attachmentColumn
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
		content[eniColumn] = aws.ToString(netinterface.NetworkInterfaceId)
		content[typeColumn] = netinterface.InterfaceType
		content[attachmentColumn] = getNameAndIDFromMap(attachmentFor(netinterface), names)
		content[ipsColumn] = iparray
		content[vpcColumn] = getNameAndIDFromMap(aws.ToString(netinterface.VpcId), names)
		content[subnetColumn] = getNameAndIDFromMap(aws.ToString(netinterface.SubnetId), names)
		output.AddContents(content)
	}
	return output
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
