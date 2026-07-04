package cmd

import (
	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	output "github.com/ArjenSchwarz/go-output/v2"
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

Use --vpc to filter results to a specific VPC.`,
	RunE: vpcOverview,
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

func vpcOverview(cmd *cobra.Command, _ []string) error {
	awsConfig := config.DefaultAwsConfig(*settings)
	accountName := getName(helpers.GetAccountID(awsConfig.StsClient()))

	ec2Client := awsConfig.Ec2Client()
	overview := helpers.GetVPCUsageOverview(ec2Client)

	// Fetch the raw route tables used for display formatting. GetAllRouteTables
	// walks every page of DescribeRouteTables so accounts with more route
	// tables than fit in a single response still render correctly (T-805).
	routeTables := helpers.GetAllRouteTables(ec2Client)

	// Filter VPCs if vpc flag is provided
	filteredVPCs := overview.VPCs
	if vpcIDFilter != "" {
		filteredVPCs = []helpers.VPCUsageInfo{}
		for _, vpc := range overview.VPCs {
			if vpc.ID == vpcIDFilter {
				filteredVPCs = append(filteredVPCs, vpc)
				break
			}
		}
	}

	// All tables accumulate on a single builder and render in one pass.
	builder := output.New()

	// Separate subnet overview tables for each VPC
	subnetKeys := []string{subnetColumn, cidrColumn, typeColumn, routeTableColumn, routesColumn, "Total IPs", "Available IPs", "Used IPs"}

	for _, vpc := range filteredVPCs {
		vpcDisplay := getResourceDisplayName(vpc.ID, vpc.Tags)
		subnetTitle := "Subnet Overview for " + vpcDisplay + " in account " + accountName

		subnetRows := []map[string]any{}
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

			subnetRows = append(subnetRows, content)
		}
		builder = builder.Table(subnetTitle, subnetRows, output.WithKeys(subnetKeys...), config.SortOption(cidrColumn))
	}

	// Individual tables for each subnet's IP details
	for _, vpc := range filteredVPCs {
		for _, subnet := range vpc.Subnets {
			if len(subnet.IPDetails) > 0 {
				ipKeys := []string{"IP Address", "Usage Type", "Attachment Info", "Public IP"}

				// Use tiered name lookup for consistent formatting
				subnetDisplay := getResourceDisplayName(subnet.ID, subnet.Tags)
				vpcDisplay := getResourceDisplayName(vpc.ID, vpc.Tags)

				ipTitle := "IP Details for subnet " + subnetDisplay + " in VPC " + vpcDisplay

				ipRows := []map[string]any{}
				for _, ipDetail := range subnet.IPDetails {
					ipContent := make(map[string]any)
					ipContent["IP Address"] = ipDetail.IPAddress
					ipContent["Usage Type"] = ipDetail.UsageType
					ipContent["Attachment Info"] = ipDetail.AttachmentInfo
					ipContent["Public IP"] = ipDetail.PublicIP

					ipRows = append(ipRows, ipContent)
				}
				builder = builder.Table(ipTitle, ipRows, output.WithKeys(ipKeys...))
			}
		}
	}

	// Third table: Summary Statistics
	summaryKeys := []string{"Metric", "Count"}

	// Calculate summary for filtered VPCs. SummarizeVPCUsage uses saturating
	// addition so VPCs with multiple IPv6-only subnets (each reporting a
	// math.MaxInt "effectively unlimited" sentinel) cannot overflow the totals
	// into negative values (T-1234).
	filteredSummary := helpers.SummarizeVPCUsage(filteredVPCs)

	// Set title based on filter
	var summaryTitle string
	if vpcIDFilter != "" {
		vpcDisplay := ""
		if len(filteredVPCs) > 0 {
			vpcDisplay = getResourceDisplayName(filteredVPCs[0].ID, filteredVPCs[0].Tags)
		}
		summaryTitle = "VPC Usage Summary for " + vpcDisplay + " in account " + accountName
	} else {
		summaryTitle = "VPC Usage Summary for account " + accountName
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

	summaryRows := []map[string]any{}
	for _, item := range summaryData {
		summaryContent := make(map[string]any)
		summaryContent["Metric"] = item.metric
		summaryContent["Count"] = item.count

		summaryRows = append(summaryRows, summaryContent)
	}
	builder = builder.Table(summaryTitle, summaryRows, output.WithKeys(summaryKeys...))

	return settings.RenderDocument(cmd.Context(), builder.Build())
}
