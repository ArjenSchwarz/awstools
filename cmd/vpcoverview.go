package cmd

import (
	"context"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
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
	Run: vpcOverview,
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

func vpcOverview(_ *cobra.Command, _ []string) {
	awsConfig := config.DefaultAwsConfig(*settings)
	accountName := getName(helpers.GetAccountID(awsConfig.StsClient()))

	overview := helpers.GetVPCUsageOverview(awsConfig.Ec2Client())

	// Get raw route tables for route information
	ec2Client := awsConfig.Ec2Client()
	routeTablesResp, err := ec2Client.DescribeRouteTables(context.TODO(), &ec2.DescribeRouteTablesInput{})
	if err != nil {
		panic(err)
	}
	routeTables := routeTablesResp.RouteTables

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

	// Create separate subnet overview tables for each VPC
	subnetKeys := []string{"Subnet", "CIDR", "Type", "Route Table", "Routes", "Total IPs", "Available IPs", "Used IPs"}

	for _, vpc := range filteredVPCs {
		vpcDisplay := getResourceDisplayName(vpc.ID, vpc.Tags)
		subnetOutput := format.OutputArray{Keys: subnetKeys, Settings: settings.NewOutputSettings()}
		subnetOutput.Settings.Title = "Subnet Overview for " + vpcDisplay + " in account " + accountName
		subnetOutput.Settings.SortKey = "CIDR"
		subnetOutput.Settings.SeparateTables = true

		for _, subnet := range vpc.Subnets {
			// Use tiered name lookup for subnet
			subnetDisplay := getResourceDisplayName(subnet.ID, subnet.Tags)

			// Get route table information for this subnet
			routeTable := helpers.GetSubnetRouteTable(subnet.ID, routeTables)
			routeTableName, routes := helpers.FormatRouteTableInfo(routeTable)

			content := make(map[string]any)
			content["Subnet"] = subnetDisplay
			content["CIDR"] = subnet.CIDR
			if subnet.IsPublic {
				content["Type"] = "Public"
			} else {
				content["Type"] = "Private"
			}
			content["Route Table"] = routeTableName
			content["Routes"] = routes
			content["Total IPs"] = subnet.TotalIPs
			content["Available IPs"] = subnet.AvailableIPs
			content["Used IPs"] = subnet.UsedIPs

			holder := format.OutputHolder{Contents: content}
			subnetOutput.AddHolder(holder)
		}
		subnetOutput.Write()
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
				ipOutput.Write()
			}
		}
	}

	// Third table: Summary Statistics
	summaryKeys := []string{"Metric", "Count"}
	summaryOutput := format.OutputArray{Keys: summaryKeys, Settings: settings.NewOutputSettings()}
	summaryOutput.Settings.SeparateTables = true

	// Calculate summary for filtered VPCs
	var filteredSummary struct {
		totalVPCs      int
		totalSubnets   int
		totalIPs       int
		usedIPs        int
		awsReservedIPs int
		serviceIPs     int
		availableIPs   int
	}

	for _, vpc := range filteredVPCs {
		filteredSummary.totalVPCs++
		for _, subnet := range vpc.Subnets {
			filteredSummary.totalSubnets++
			filteredSummary.totalIPs += subnet.TotalIPs
			filteredSummary.usedIPs += subnet.UsedIPs
			filteredSummary.availableIPs += subnet.AvailableIPs

			// Count AWS reserved IPs and service IPs from IP details
			for _, ipDetail := range subnet.IPDetails {
				if ipDetail.UsageType == "RESERVED BY AWS" {
					filteredSummary.awsReservedIPs++
				} else {
					filteredSummary.serviceIPs++
				}
			}
		}
	}

	// Set title based on filter
	if vpcIDFilter != "" {
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
		{"Total VPCs", filteredSummary.totalVPCs},
		{"Total Subnets", filteredSummary.totalSubnets},
		{"Total IP Addresses", filteredSummary.totalIPs},
		{"Used IP Addresses", filteredSummary.usedIPs},
		{"  - AWS Reserved IPs", filteredSummary.awsReservedIPs},
		{"  - Service IPs", filteredSummary.serviceIPs},
		{"Available IP Addresses", filteredSummary.availableIPs},
	}

	for _, item := range summaryData {
		summaryContent := make(map[string]any)
		summaryContent["Metric"] = item.metric
		summaryContent["Count"] = item.count

		summaryHolder := format.OutputHolder{Contents: summaryContent}
		summaryOutput.AddHolder(summaryHolder)
	}
	summaryOutput.Write()
}
