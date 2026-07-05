package cmd

import (
	"fmt"
	"os"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/spf13/cobra"
)

// ipFinderCmd represents the ip-finder command
var ipFinderCmd = &cobra.Command{
	Use:   "ip-finder [IP_ADDRESS]",
	Short: "Find ENI and resource details for an IP address",
	Long: `Search for an IP address across ENIs and return detailed information about the associated resources.
	
	This command will search for the specified IP address across all Network Interfaces (ENIs) in your AWS account
	and return comprehensive information about the resource associated with that IP address.
	
	The search includes both primary and secondary IP addresses on ENIs.

	The search is limited to a single region. Use the --region flag to target a
	specific region. Searching across all regions is not yet supported.

	Examples:
	  awstools vpc ip-finder 10.0.1.100
	  awstools vpc ip-finder 10.0.1.100 --output json
	  awstools vpc ip-finder 10.0.1.100 --region eu-west-1`,
	Args: cobra.ExactArgs(1),
	RunE: findIPAddress,
}

var (
	searchAllRegions bool
)

func init() {
	vpcCmd.AddCommand(ipFinderCmd)
	ipFinderCmd.Flags().BoolVar(&searchAllRegions, "search-all-regions", false, "Search across all regions (not yet supported)")
}

// validateIPFinderFlags validates the command arguments and flags before any
// AWS calls are made. It is kept separate from findIPAddress so the flag logic
// can be unit tested without AWS configuration or credentials.
//
// The --search-all-regions flag is rejected because multi-region search is not
// implemented. Previously the flag was accepted and silently ignored, so the
// single-region search would run anyway and a user could receive a false
// "not found" result that looked like an all-region search had been performed
// (T-1222).
func validateIPFinderFlags(ipAddress string, allRegions bool) error {
	if !helpers.IsValidIPAddress(ipAddress) {
		return fmt.Errorf("invalid IP address format: %s\n\nPlease provide a valid IPv4 or IPv6 address.\nExamples:\n  - IPv4: 192.168.1.1\n  - IPv6: 2001:db8::1", ipAddress)
	}
	if allRegions {
		return fmt.Errorf("--search-all-regions is not yet supported\n\nThe search is limited to a single region. Use the --region flag to target a specific region instead")
	}
	return nil
}

func findIPAddress(cmd *cobra.Command, args []string) error {
	ipAddress := args[0]

	// Validate arguments and flags before any AWS calls. Invalid user input is
	// an expected CLI error, so it is returned as a normal error (handled by
	// Cobra/Execute) rather than triggering a panic (T-1370).
	if err := validateIPFinderFlags(ipAddress, searchAllRegions); err != nil {
		return err
	}

	// Load AWS configuration
	awsConfig := config.DefaultAwsConfig(*settings)

	// Call helper function. A single private IP can match more than one ENI
	// (e.g. duplicate RFC1918 ranges across unrelated VPCs), so every match is
	// returned and rendered.
	results := helpers.FindIPAddressDetails(awsConfig.Ec2Client(), ipAddress)

	// Format and output results
	return formatIPFinderOutput(cmd, results)
}

func formatIPFinderOutput(cmd *cobra.Command, results []helpers.IPFinderResult) error {
	// The helper always returns at least one result. When nothing matched it is
	// a single result with Found=false.
	if len(results) == 0 || !results[0].Found {
		ipAddress := ""
		if len(results) > 0 {
			ipAddress = results[0].IPAddress
		}
		fmt.Fprintf(os.Stderr, "IP address %s not found in any ENI in the current region\n", ipAddress)
		fmt.Fprintf(os.Stderr, "\nTroubleshooting suggestions:\n")
		fmt.Fprintf(os.Stderr, "  - Verify the IP address is correct\n")
		fmt.Fprintf(os.Stderr, "  - Check if the IP is in a different AWS region using --region flag\n")
		fmt.Fprintf(os.Stderr, "  - Ensure you have the necessary permissions to describe network interfaces\n")
		fmt.Fprintf(os.Stderr, "  - Consider that the IP might be associated with a different AWS account\n")
		return nil
	}

	if len(results) > 1 {
		fmt.Fprintf(os.Stderr, "Note: IP address %s matched %d ENIs (duplicate private IPs across VPCs). Showing all matches.\n",
			results[0].IPAddress, len(results))
	}

	// Each match gets its own titled table; all tables accumulate on a single
	// builder and render in one pass.
	builder := output.New()
	for _, result := range results {
		title, rows := singleIPFinderResultTable(result)
		builder = builder.Table(title, rows, output.WithKeys(fieldColumn, valueColumn))
	}
	return settings.RenderDocument(cmd.Context(), builder.Build())
}

// singleIPFinderResultTable builds the title and field/value rows for a single
// ENI match.
func singleIPFinderResultTable(result helpers.IPFinderResult) (string, []map[string]any) {
	title := fmt.Sprintf("IP Address Details: %s", result.IPAddress)

	// Build output data with proper handling of missing names
	var resourceName string
	if result.ResourceName != "" {
		resourceName = result.ResourceName
	} else {
		resourceName = "No Name Tag"
	}

	var vpcDisplay string
	if result.VPC.Name != "" {
		vpcDisplay = fmt.Sprintf("%s (%s)", result.VPC.Name, result.VPC.ID)
	} else {
		vpcDisplay = result.VPC.ID
	}

	var subnetDisplay string
	if result.Subnet.Name != "" {
		subnetDisplay = fmt.Sprintf("%s (%s)", result.Subnet.Name, result.Subnet.ID)
	} else {
		subnetDisplay = result.Subnet.ID
	}

	var eniID string
	if result.ENI != nil {
		eniID = aws.ToString(result.ENI.NetworkInterfaceId)
	}

	outputData := []map[string]any{
		{fieldColumn: "IP Address", valueColumn: result.IPAddress},
		{fieldColumn: "ENI ID", valueColumn: eniID},
		{fieldColumn: "Resource Type", valueColumn: result.ResourceType},
		{fieldColumn: "Resource Name", valueColumn: resourceName},
		{fieldColumn: "Resource ID", valueColumn: result.ResourceID},
		{fieldColumn: vpcColumn, valueColumn: vpcDisplay},
		{fieldColumn: subnetColumn, valueColumn: subnetDisplay},
		{fieldColumn: "Is Secondary IP", valueColumn: result.IsSecondaryIP},
	}

	// Add security groups if present
	if len(result.SecurityGroups) > 0 {
		var sgList []string
		for _, sg := range result.SecurityGroups {
			if sg.Name != "" && sg.Name != sg.ID {
				sgList = append(sgList, fmt.Sprintf("%s (%s)", sg.Name, sg.ID))
			} else {
				sgList = append(sgList, sg.ID)
			}
		}
		outputData = append(outputData, map[string]any{
			fieldColumn: "Security Groups",
			valueColumn: sgList,
		})
	}

	// Add route table information if present
	if result.RouteTable.ID != "" {
		outputData = append(outputData, map[string]any{
			fieldColumn: routeTableColumn,
			valueColumn: result.RouteTable.Name,
		})

		// Add routes if present
		if len(result.RouteTable.Routes) > 0 {
			outputData = append(outputData, map[string]any{
				fieldColumn: routesColumn,
				valueColumn: result.RouteTable.Routes,
			})
		}
	}

	return title, outputData
}
