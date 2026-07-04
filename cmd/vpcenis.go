package cmd

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	output "github.com/ArjenSchwarz/go-output/v2"
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
// diagram, so dot and drawio are rejected with a clear error instead of falling
// through to the generic unsupported-format handling. The comparison is
// case-insensitive to match the format normalisation in config.
func enisGraphFormatError(format string) error {
	switch strings.ToLower(format) {
	case "dot", "drawio":
		return fmt.Errorf("the %s output format is not supported by 'vpc enis'; supported formats are json, csv, table, and html", strings.ToLower(format))
	default:
		return nil
	}
}

func enis(cmd *cobra.Command, _ []string) error {
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

	var doc *output.Document
	if vpceenisSplit {
		doc = buildENIsBySubnetDocument(interfaces, names, resultTitle, attachmentFor)
	} else {
		doc = buildENIsDocument(interfaces, names, resultTitle, attachmentFor)
	}
	return settings.RenderDocument(cmd.Context(), doc)
}

const (
	eniColumn = "ENI"
	ipsColumn = "IPs"
)

var fullENIKeys = []string{eniColumn, typeColumn, attachmentColumn, ipsColumn, vpcColumn, subnetColumn}
var splitENIKeys = []string{eniColumn, typeColumn, attachmentColumn, ipsColumn}

// buildENIsDocument constructs the single-table (non-split) ENI document. One
// document serves both stdout and --file, so both destinations always carry
// the same rows (T-1294). The attachmentFor resolver decouples row
// construction from the AWS-backed ENILookupCache so the output shape is unit
// testable.
func buildENIsDocument(interfaces []types.NetworkInterface, names map[string]string, resultTitle string, attachmentFor func(types.NetworkInterface) string) *output.Document {
	return output.New().
		Table(resultTitle, eniRows(interfaces, names, false, attachmentFor),
			output.WithKeys(fullENIKeys...), config.SortOption(subnetColumn)).
		Build()
}

// buildENIsBySubnetDocument constructs the --split document: one table per
// subnet group, each with the v1 group title and the reduced split key set.
// All groups live in a single document so one render serves both stdout and
// --file with every subnet table (R7.2, T-1294). Subnet keys are sorted so
// table order is deterministic between runs (R2.8).
func buildENIsBySubnetDocument(interfaces []types.NetworkInterface, names map[string]string, resultTitle string, attachmentFor func(types.NetworkInterface) string) *output.Document {
	groups := splitBySubnet(interfaces)
	builder := output.New()
	for _, subnet := range slices.Sorted(maps.Keys(groups)) {
		group := groups[subnet]
		groupTitle := fmt.Sprintf("%s - %s: %s", resultTitle, getNameAndIDFromMap(aws.ToString(group[0].VpcId), names), getNameAndIDFromMap(subnet, names))
		builder = builder.Table(groupTitle, eniRows(group, names, true, attachmentFor),
			output.WithKeys(splitENIKeys...), config.SortOption(attachmentColumn))
	}
	return builder.Build()
}

// eniRows builds the table rows for the supplied ENIs. Split rows omit the VPC
// and subnet columns because the split table's title already carries them.
func eniRows(interfaces []types.NetworkInterface, names map[string]string, split bool, attachmentFor func(types.NetworkInterface) string) []map[string]any {
	rows := make([]map[string]any, 0, len(interfaces))
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
		if !split {
			content[vpcColumn] = getNameAndIDFromMap(aws.ToString(netinterface.VpcId), names)
			content[subnetColumn] = getNameAndIDFromMap(aws.ToString(netinterface.SubnetId), names)
		}
		rows = append(rows, content)
	}
	return rows
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
