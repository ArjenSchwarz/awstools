package cmd

import (
	"fmt"
	"os"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/spf13/cobra"
)

// overviewCmd represents the vpc overview command
var overviewCmd = &cobra.Command{
	Use:   "overview",
	Short: "Get VPC usage overview",
	Long: `Get a comprehensive overview of VPC resource utilization including subnet sizing, IP address allocation, and usage patterns.

The command shows separate tables for each VPC, displaying:
- Subnet overview with route table information
- Detailed IP address usage per subnet
- Summary statistics

Use --vpc to filter results to a specific VPC. When --file is used, all of
these tables are saved to the file, not only the summary.`,
	RunE: vpcOverview,
	// Execute reports returned errors; avoid duplicate errors and usage from Cobra.
	SilenceErrors: true,
	SilenceUsage:  true,
}

var vpcIDFilter string

func init() {
	vpcCmd.AddCommand(overviewCmd)
	overviewCmd.Flags().StringVar(&vpcIDFilter, "vpc", "", "Filter by VPC ID (e.g., vpc-12345678)")
}

// getResourceDisplayName provides tiered name lookup for AWS resources using the centralized helper
func getResourceDisplayName(resourceID string, tags []types.Tag) string {
	return helpers.GetResourceDisplayNameWithGlobalLookup(resourceID, tags, getName)
}

func vpcOverview(_ *cobra.Command, _ []string) error {
	awsConfig := config.DefaultAwsConfig(*settings)
	accountName := getName(helpers.GetAccountID(awsConfig.StsClient()))

	ec2Client := awsConfig.Ec2Client()
	overview := helpers.GetVPCUsageOverview(ec2Client)

	// Fetch the raw route tables used for display formatting. GetAllRouteTables
	// walks every page of DescribeRouteTables so accounts with more route
	// tables than fit in a single response still render correctly (T-805).
	routeTables := helpers.GetAllRouteTables(ec2Client)

	tables := buildVPCOverviewTables(overview.VPCs, routeTables, accountName, vpcIDFilter)
	if err := writeOutputTables(tables); err != nil {
		return fmt.Errorf("writing VPC overview: %w", err)
	}
	return nil
}

// buildVPCOverviewTables returns every table of the VPC overview report in
// display order: one subnet overview per VPC, one IP details table per subnet
// with IP details, and the usage summary.
func buildVPCOverviewTables(vpcs []helpers.VPCUsageInfo, routeTables []types.RouteTable, accountName string, vpcFilter string) []format.OutputArray {
	// Filter VPCs if vpc flag is provided
	filteredVPCs := vpcs
	if vpcFilter != "" {
		filteredVPCs = []helpers.VPCUsageInfo{}
		for _, vpc := range vpcs {
			if vpc.ID == vpcFilter {
				filteredVPCs = append(filteredVPCs, vpc)
				break
			}
		}
	}

	var tables []format.OutputArray

	// Create separate subnet overview tables for each VPC
	subnetKeys := []string{subnetColumn, cidrColumn, typeColumn, routeTableColumn, routesColumn, "Total IPs", "Available IPs", "Used IPs"}

	for _, vpc := range filteredVPCs {
		vpcDisplay := getResourceDisplayName(vpc.ID, vpc.Tags)
		subnetOutput := format.OutputArray{Keys: subnetKeys, Settings: settings.NewOutputSettings()}
		subnetOutput.Settings.Title = "Subnet Overview for " + vpcDisplay + " in account " + accountName
		subnetOutput.Settings.SortKey = cidrColumn
		subnetOutput.Settings.SeparateTables = true

		for _, subnet := range vpc.Subnets {
			// Use tiered name lookup for subnet
			subnetDisplay := getResourceDisplayName(subnet.ID, subnet.Tags)

			// Get route table information for this subnet
			routeTable := helpers.GetSubnetRouteTable(subnet.ID, subnet.VPCId, routeTables)
			routeTableName, routes := helpers.FormatRouteTableInfo(routeTable)

			content := make(map[string]any)
			content[subnetColumn] = subnetDisplay
			content[cidrColumn] = subnet.CIDR
			if subnet.IsPublic {
				content[typeColumn] = "Public"
			} else {
				content[typeColumn] = "Private"
			}
			content[routeTableColumn] = routeTableName
			content[routesColumn] = routes
			content["Total IPs"] = subnet.TotalIPs
			content["Available IPs"] = subnet.AvailableIPs
			content["Used IPs"] = subnet.UsedIPs

			holder := format.OutputHolder{Contents: content}
			subnetOutput.AddHolder(holder)
		}
		tables = append(tables, subnetOutput)
	}

	// Individual tables for each subnet's IP details
	for _, vpc := range filteredVPCs {
		for _, subnet := range vpc.Subnets {
			if len(subnet.IPDetails) > 0 {
				ipKeys := []string{"IP Address", "Usage Type", "Attachment Info", "Public IP"}
				ipOutput := format.OutputArray{Keys: ipKeys, Settings: settings.NewOutputSettings()}
				ipOutput.Settings.SeparateTables = true

				// Use tiered name lookup for consistent formatting
				subnetDisplay := getResourceDisplayName(subnet.ID, subnet.Tags)
				vpcDisplay := getResourceDisplayName(vpc.ID, vpc.Tags)

				ipOutput.Settings.Title = "IP Details for subnet " + subnetDisplay + " in VPC " + vpcDisplay

				for _, ipDetail := range subnet.IPDetails {
					ipContent := make(map[string]any)
					ipContent["IP Address"] = ipDetail.IPAddress
					ipContent["Usage Type"] = ipDetail.UsageType
					ipContent["Attachment Info"] = ipDetail.AttachmentInfo
					ipContent["Public IP"] = ipDetail.PublicIP

					ipHolder := format.OutputHolder{Contents: ipContent}
					ipOutput.AddHolder(ipHolder)
				}
				tables = append(tables, ipOutput)
			}
		}
	}

	// Third table: Summary Statistics
	summaryKeys := []string{"Metric", "Count"}
	summaryOutput := format.OutputArray{Keys: summaryKeys, Settings: settings.NewOutputSettings()}
	summaryOutput.Settings.SeparateTables = true

	// Calculate summary for filtered VPCs. SummarizeVPCUsage uses saturating
	// addition so VPCs with multiple IPv6-only subnets (each reporting a
	// math.MaxInt "effectively unlimited" sentinel) cannot overflow the totals
	// into negative values (T-1234).
	filteredSummary := helpers.SummarizeVPCUsage(filteredVPCs)

	// Set title based on filter
	if vpcFilter != "" {
		vpcDisplay := ""
		if len(filteredVPCs) > 0 {
			vpcDisplay = getResourceDisplayName(filteredVPCs[0].ID, filteredVPCs[0].Tags)
		}
		summaryOutput.Settings.Title = "VPC Usage Summary for " + vpcDisplay + " in account " + accountName
	} else {
		summaryOutput.Settings.Title = "VPC Usage Summary for account " + accountName
	}

	summaryData := []struct {
		metric string
		count  int
	}{
		{"Total VPCs", filteredSummary.TotalVPCs},
		{"Total Subnets", filteredSummary.TotalSubnets},
		{"Total IP Addresses", filteredSummary.TotalIPs},
		{"Used IP Addresses", filteredSummary.UsedIPs},
		{"  - AWS Reserved IPs", filteredSummary.AWSReservedIPs},
		{"  - Service IPs", filteredSummary.ServiceIPs},
		{"Available IP Addresses", filteredSummary.AvailableIPs},
	}

	for _, item := range summaryData {
		summaryContent := make(map[string]any)
		summaryContent["Metric"] = item.metric
		summaryContent["Count"] = item.count

		summaryHolder := format.OutputHolder{Contents: summaryContent}
		summaryOutput.AddHolder(summaryHolder)
	}
	tables = append(tables, summaryOutput)

	return tables
}

// writeOutputTables writes several independent tables to stdout and, when
// --file is set, saves all of them to that file (T-1536).
//
// go-output v1.4.0 truncates the output file on every Write() call, so writing
// each table with --file set would leave only the last table in the file. The
// tables are therefore first written to stdout without a file, then rendered
// again in the file format and saved with a single PrintByteSlice call. The
// file holds the same content as the terminal output, in the file format.
func writeOutputTables(tables []format.OutputArray) error {
	if len(tables) == 0 {
		return nil
	}
	fileSettings := *tables[0].Settings
	for _, table := range tables {
		withoutFile(table).Write()
	}
	if fileSettings.OutputFile == "" {
		return nil
	}

	fileFormat := fileSettings.OutputFileFormat
	if fileFormat == "" {
		fileFormat = fileSettings.OutputFormat
	}
	contents, err := captureStdout(func() {
		for _, table := range tables {
			table = withoutFile(table)
			table.Settings.OutputFormat = fileFormat
			table.Write()
		}
	})
	if err != nil {
		return err
	}
	return format.PrintByteSlice(contents, fileSettings.OutputFile, fileSettings.S3Bucket)
}

// withoutFile returns a copy of the table whose settings render only to
// stdout. Append is cleared as well because go-output reads the output file
// when appending HTML, which would fail without a file.
func withoutFile(table format.OutputArray) format.OutputArray {
	tableSettings := *table.Settings
	tableSettings.OutputFile = ""
	tableSettings.OutputFileFormat = ""
	tableSettings.ShouldAppend = false
	table.Settings = &tableSettings
	return table
}

// captureStdout returns everything render writes to os.Stdout. go-output
// v1.4.0 only exposes rendered tables through stdout or a file it truncates on
// every call, so this is how several tables can be combined into one file.
// A temporary file is used instead of a pipe so large reports cannot block.
func captureStdout(render func()) ([]byte, error) {
	tmp, err := os.CreateTemp("", "awstools-output-*")
	if err != nil {
		return nil, fmt.Errorf("creating temporary output file: %w", err)
	}
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}()

	original := os.Stdout
	os.Stdout = tmp
	defer func() { os.Stdout = original }()
	render()
	os.Stdout = original

	contents, err := os.ReadFile(tmp.Name())
	if err != nil {
		return nil, fmt.Errorf("reading temporary output file: %w", err)
	}
	return contents, nil
}
