# VPC Usage Overview Design Document

## Overview

The VPC Usage Overview feature provides comprehensive visibility into VPC resource utilization, subnet sizing, IP address allocation, and usage patterns within AWS environments. This feature addresses the common need for network administrators to understand how their VPC resources are being utilized, identify available capacity, and troubleshoot network connectivity issues.

The feature will deliver a clear, hierarchical view of VPC resources, showing subnet-level details including IP address allocation, usage patterns, and classification of public vs private subnets. By providing detailed IP address tracking with proper AWS reserved IP identification, administrators can make informed decisions about subnet sizing, capacity planning, and resource optimization.

This feature will provide actionable insights for network capacity planning and resource optimization decisions.

## Architecture

The VPC Usage Overview will integrate with the existing awstools architecture following established patterns:

1. **Command Layer**: New `overview` subcommand under the existing `vpc` command structure
2. **Business Logic**: Core functionality implemented in the `helpers` package with new VPC analysis functions
3. **AWS Integration**: Leverages existing AWS SDK v2 EC2 client configuration and credential handling
4. **Output Processing**: Uses the established go-output library for consistent formatting across all supported formats

The VPC Usage Overview will integrate with the existing workflow:

1. Command initialization through the Cobra framework using existing patterns from cmd/vpc.go
2. AWS configuration and credential loading via the existing config.DefaultAwsConfig() function
3. VPC data retrieval using new helper functions that extend the existing EC2 helpers
4. Data processing and correlation to determine subnet types and IP utilization
5. Output generation using the existing format.OutputArray pattern with multiple format support

## Components and Interfaces

### VPC Overview Command Component

The command component will be implemented as a new subcommand under the existing VPC command structure.

```go
// overviewCmd represents the vpc overview command
var overviewCmd = &cobra.Command{
    Use:   "overview",
    Short: "Get VPC usage overview",
    Long:  `Get a comprehensive overview of VPC resource utilization including subnet sizing, IP address allocation, and usage patterns.`,
    Run:   vpcOverview,
}

func vpcOverview(_ *cobra.Command, _ []string) {
    awsConfig := config.DefaultAwsConfig(*settings)
    resultTitle := "VPC Overview for account " + getName(helpers.GetAccountID(awsConfig.StsClient()))
    
    overview := helpers.GetVPCUsageOverview(awsConfig.Ec2Client())
    // Output processing logic
}
```

### VPC Data Retrieval Component

New helper functions will be added to helpers/ec2.go to support VPC overview functionality.

```go
// VPCOverview represents the complete VPC usage analysis
type VPCOverview struct {
    VPCs    []VPCUsageInfo
    Summary VPCUsageSummary
}

// VPCUsageInfo contains detailed information about a single VPC
type VPCUsageInfo struct {
    ID       string
    Name     string
    CIDR     string
    Subnets  []SubnetUsageInfo
}

// SubnetUsageInfo contains detailed subnet usage information
type SubnetUsageInfo struct {
    ID               string
    Name             string
    CIDR             string
    VPCId            string
    VPCName          string
    IsPublic         bool
    TotalIPs         int
    AvailableIPs     int
    UsedIPs          int
    IPDetails        []IPAddressInfo
}

// IPAddressInfo contains information about individual IP addresses
type IPAddressInfo struct {
    IPAddress    string
    UsageType    string
    AttachmentInfo string
    PublicIP     string
}

func GetVPCUsageOverview(svc *ec2.Client) VPCOverview {
    // Implementation details
}
```

### IP Address Analysis Component

IP address analysis will determine usage patterns and AWS reserved addresses.

```go
func analyzeSubnetIPUsage(subnet types.Subnet, networkInterfaces []types.NetworkInterface, svc *ec2.Client) []IPAddressInfo {
    // Calculate IP range from CIDR
    // Identify AWS reserved IPs (first 4 and last)
    // Map network interfaces to IP addresses
    // Return detailed IP usage information
}

func isPublicSubnet(subnetId string, routeTables []types.RouteTable) bool {
    // Analyze route tables to determine if subnet is public
    // Check for routes to internet gateway
}
```

### Output Formatting Component

Output formatting will use the existing go-output framework to support all standard formats.

```go
func formatVPCOverviewOutput(overview VPCOverview, settings *format.OutputSettings) {
    keys := []string{"VPC ID", "VPC Name", "Subnet ID", "Subnet Name", "CIDR", "Type", "Total IPs", "Available IPs", "Used IPs"}
    output := format.OutputArray{Keys: keys, Settings: settings}
    
    // Format output data according to requirements
}
```

## Data Models

The feature will introduce several new data structures to represent VPC usage information:

```go
type VPCOverview struct {
    VPCs    []VPCUsageInfo `json:"vpcs"`
    Summary VPCUsageSummary `json:"summary"`
}

type VPCUsageInfo struct {
    ID       string            `json:"id"`
    Name     string            `json:"name"`
    CIDR     string            `json:"cidr"`
    Subnets  []SubnetUsageInfo `json:"subnets"`
}

type SubnetUsageInfo struct {
    ID               string          `json:"id"`
    Name             string          `json:"name"`
    CIDR             string          `json:"cidr"`
    VPCId            string          `json:"vpc_id"`
    VPCName          string          `json:"vpc_name"`
    IsPublic         bool            `json:"is_public"`
    TotalIPs         int             `json:"total_ips"`
    AvailableIPs     int             `json:"available_ips"`
    UsedIPs          int             `json:"used_ips"`
    IPDetails        []IPAddressInfo `json:"ip_details,omitempty"`
}

type IPAddressInfo struct {
    IPAddress      string `json:"ip_address"`
    UsageType      string `json:"usage_type"`
    AttachmentInfo string `json:"attachment_info"`
    PublicIP       string `json:"public_ip,omitempty"`
}

type VPCUsageSummary struct {
    TotalVPCs     int `json:"total_vpcs"`
    TotalSubnets  int `json:"total_subnets"`
    TotalIPs      int `json:"total_ips"`
    UsedIPs       int `json:"used_ips"`
    AvailableIPs  int `json:"available_ips"`
}
```

## Error Handling

Error handling will follow existing awstools patterns:

1. **AWS API Errors**: Use panic() for AWS SDK errors consistent with existing helper functions
2. **Configuration Errors**: Leverage existing config validation and error reporting
3. **Data Processing Errors**: Handle CIDR parsing and IP calculation errors gracefully
4. **Missing Permissions**: Provide clear error messages when AWS permissions are insufficient

The implementation will maintain consistency with existing error handling patterns found in helpers/ec2.go, using panic() for AWS API errors and letting the application's global error handling manage user-facing error messages.

## Testing Strategy

We'll implement the following tests:

1. **Unit Tests**: Test individual helper functions for IP address calculation, subnet classification, and data processing
2. **Integration Tests**: Test AWS API interactions using mock EC2 clients
3. **Output Format Tests**: Verify all supported output formats (JSON, CSV, table, HTML, etc.) render correctly
4. **Edge Case Tests**: Test with empty VPCs, subnets with no available IPs, and various subnet sizes

Test cases will include:

- Subnet with all AWS reserved IPs and no ENIs
- Subnet with maximum IP utilization
- Mixed public and private subnets in the same VPC
- VPCs with no subnets
- Network interfaces with and without public IP addresses
- Route table analysis for public/private subnet determination
- CIDR block parsing and IP range calculations
- Multiple output format validation

## Implementation Plan

1. **Create new data structures** in helpers/ec2.go for VPC usage information
2. **Implement core AWS API functions** for retrieving VPC, subnet, network interface, and route table data
3. **Add IP address analysis logic** including AWS reserved IP identification and utilization calculations
4. **Implement subnet classification logic** using route table analysis to determine public/private status
5. **Create the vpc overview command** in cmd/vpc.go following existing command patterns
6. **Add output formatting** using the go-output library with support for all existing formats
7. **Add comprehensive unit tests** for all new helper functions and data processing logic
8. **Integration testing** with the existing command structure and AWS configuration
9. **Documentation updates** if required for the new command functionality

## Conclusion

This design for VPC Usage Overview provides comprehensive visibility into VPC resource utilization and detailed IP address allocation analysis while maintaining full compatibility with existing awstools architecture and patterns. The implementation approach prioritizes consistency with existing code patterns and ensures seamless integration with the established CLI structure, AWS configuration handling, and output formatting systems.

The design leverages existing AWS SDK integration and helper function patterns while introducing focused new functionality for VPC usage analysis. This approach minimizes code duplication and maintains the reliability and consistency that users expect from awstools commands.