package cmd

import (
	"fmt"

	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/helpers"
	format "github.com/ArjenSchwarz/go-output"
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
	
	Examples:
	  awstools vpc ip-finder 10.0.1.100
	  awstools vpc ip-finder 10.0.1.100 --output json`,
	Args: cobra.ExactArgs(1),
	Run:  findIPAddress,
}

var (
	searchAllRegions bool
)

func init() {
	vpcCmd.AddCommand(ipFinderCmd)
	ipFinderCmd.Flags().BoolVar(&searchAllRegions, "search-all-regions", false, "Search across all regions (future enhancement)")
}

func findIPAddress(_ *cobra.Command, args []string) {
	ipAddress := args[0]

	// Validate IP address format with helpful error message
	if !helpers.IsValidIPAddress(ipAddress) {
		panic(fmt.Errorf("invalid IP address format: %s\n\nPlease provide a valid IPv4 or IPv6 address.\nExamples:\n  - IPv4: 192.168.1.1\n  - IPv6: 2001:db8::1", ipAddress))
	}

	// Load AWS configuration
	awsConfig := config.DefaultAwsConfig(*settings)

	// Call helper function
	result := helpers.FindIPAddressDetails(awsConfig.Ec2Client(), ipAddress)

	// Format and output results
	formatIPFinderOutput(result)
}

func formatIPFinderOutput(result helpers.IPFinderResult) {
	if !result.Found {
		fmt.Printf("IP address %s not found in any ENI in the current region\n", result.IPAddress)
		fmt.Printf("\nTroubleshooting suggestions:\n")
		fmt.Printf("  - Verify the IP address is correct\n")
		fmt.Printf("  - Check if the IP is in a different AWS region using --region flag\n")
		fmt.Printf("  - Ensure you have the necessary permissions to describe network interfaces\n")
		fmt.Printf("  - Consider that the IP might be associated with a different AWS account\n")
		return
	}

	keys := []string{"Field", "Value"}
	output := format.OutputArray{
		Keys:     keys,
		Settings: settings.NewOutputSettings(),
	}

	output.Settings.Title = fmt.Sprintf("IP Address Details: %s", result.IPAddress)

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

	outputData := []map[string]any{
		{"Field": "IP Address", "Value": result.IPAddress},
		{"Field": "ENI ID", "Value": *result.ENI.NetworkInterfaceId},
		{"Field": "Resource Type", "Value": result.ResourceType},
		{"Field": "Resource Name", "Value": resourceName},
		{"Field": "Resource ID", "Value": result.ResourceID},
		{"Field": "VPC", "Value": vpcDisplay},
		{"Field": "Subnet", "Value": subnetDisplay},
		{"Field": "Is Secondary IP", "Value": result.IsSecondaryIP},
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
			"Field": "Security Groups",
			"Value": sgList,
		})
	}

	// Add route table information if present
	if result.RouteTable.ID != "" {
		outputData = append(outputData, map[string]any{
			"Field": "Route Table",
			"Value": result.RouteTable.Name,
		})

		// Add routes if present
		if len(result.RouteTable.Routes) > 0 {
			outputData = append(outputData, map[string]any{
				"Field": "Routes",
				"Value": result.RouteTable.Routes,
			})
		}
	}

	for _, data := range outputData {
		output.AddContents(data)
	}

	output.Write()
}
