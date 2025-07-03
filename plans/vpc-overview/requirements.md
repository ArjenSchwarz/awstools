# VPC Usage Overview Requirements

## Project Overview

This document specifies the requirements for implementing a new VPC usage overview functionality in awstools. The feature will provide comprehensive visibility into VPC resource utilization, subnet sizing, IP address allocation, and usage patterns.

## Functional Requirements

### FR-1: Subnet Overview Table
**Requirement**: The awstools vpc overview command shall display a primary table containing subnet information.

**Sub-requirements**:
- FR-1.1: The system shall display subnet ID and name (if available) for each subnet
- FR-1.2: The system shall show the CIDR block and total IP count for each subnet
- FR-1.3: The system shall indicate whether each subnet is public or private
- FR-1.4: The system shall calculate and display available IP addresses for each subnet
- FR-1.5: The system shall show the VPC ID and name (if available) for context

### FR-2: IP Address Detail Tables
**Requirement**: When displaying subnet details, the system shall provide per-subnet IP address usage tables.

**Sub-requirements**:
- FR-2.1: The system shall display IP addresses sorted in ascending numerical order
- FR-2.2: The system shall show the usage type for each IP address (ENI attachment details or AWS reserved)
- FR-2.3: When an IP is reserved by AWS, the system shall display "RESERVED BY AWS" as the usage type
- FR-2.4: When an IP is associated with an ENI, the system shall display attachment information without showing ENI ID
- FR-2.5: The system shall include public IP addresses where applicable

### FR-3: AWS Reserved IP Handling
**Requirement**: The system shall identify and properly categorize AWS reserved IP addresses.

**Sub-requirements**:
- FR-3.1: The system shall mark the first IP in each subnet as "RESERVED BY AWS" (network address)
- FR-3.2: The system shall mark the second IP in each subnet as "RESERVED BY AWS" (VPC router)
- FR-3.3: The system shall mark the third IP in each subnet as "RESERVED BY AWS" (DNS server)
- FR-3.4: The system shall mark the fourth IP in each subnet as "RESERVED BY AWS" (reserved for future use)
- FR-3.5: The system shall mark the last IP in each subnet as "RESERVED BY AWS" (broadcast address)

### FR-4: Integration with Existing Architecture
**Requirement**: The new functionality shall integrate with existing awstools command structure and patterns.

**Sub-requirements**:  
- FR-4.1: The system shall implement the command as `awstools vpc overview`
- FR-4.2: The system shall support existing output formats (JSON, CSV, table, HTML)
- FR-4.3: The system shall use existing AWS configuration and credential handling
- FR-4.4: The system shall support existing naming file integration for resource name resolution
- FR-4.5: The system shall follow existing error handling patterns

## Non-Functional Requirements

### NFR-1: Performance
**Requirement**: The system shall retrieve and process VPC data efficiently.

**Sub-requirements**:
- NFR-1.1: The system shall minimize API calls to AWS services
- NFR-1.2: When possible, the system shall reuse existing helper functions for AWS API interactions

### NFR-2: Usability
**Requirement**: The command output shall be clear and actionable for network administrators.

**Sub-requirements**:
- NFR-2.1: The system shall present information in a logical, hierarchical format
- NFR-2.2: The system shall use consistent formatting with existing awstools commands
- NFR-2.3: The system shall provide meaningful column headers and labels

### NFR-3: Maintainability
**Requirement**: The implementation shall follow existing code patterns and conventions.

**Sub-requirements**:
- NFR-3.1: The system shall use the existing Cobra framework for command structure
- NFR-3.2: The system shall implement business logic in the helpers package
- NFR-3.3: The system shall follow existing Go coding standards and formatting
- NFR-3.4: The system shall include appropriate error handling and logging

## Technical Constraints

### TC-1: AWS API Dependencies
**Constraint**: The system shall depend on the following AWS EC2 API operations:
- DescribeVpcs
- DescribeSubnets  
- DescribeNetworkInterfaces
- DescribeRouteTables (for public/private determination)

### TC-2: Backward Compatibility
**Constraint**: The implementation shall not break existing vpc subcommands or functionality.

## Data Requirements

### DR-1: Data Sources
**Requirement**: The system shall retrieve data from AWS EC2 service APIs.

**Sub-requirements**:
- DR-1.1: The system shall use existing AWS SDK v2 integration
- DR-1.2: The system shall respect AWS credential and region configuration
- DR-1.3: The system shall handle multi-VPC scenarios within a single region

### DR-2: Data Processing
**Requirement**: The system shall process and correlate data from multiple AWS resources.

**Sub-requirements**:
- DR-2.1: The system shall correlate subnets with their associated route tables
- DR-2.2: The system shall determine public/private subnet classification based on route table analysis
- DR-2.3: The system shall map network interfaces to their IP addresses and attachments
- DR-2.4: The system shall calculate IP address utilization and availability

## Success Criteria

### SC-1: Functional Completeness
- All subnet information is accurately displayed in the overview table
- IP address details are correctly shown for each subnet
- AWS reserved IPs are properly identified and labeled
- Public/private subnet classification is accurate

### SC-2: Integration Success  
- Command integrates seamlessly with existing awstools CLI structure
- All standard output formats work correctly
- Existing AWS configuration and naming integration functions properly

### SC-3: Performance Acceptance
- Command completes execution within reasonable time for typical VPC configurations
- API call efficiency is maintained through appropriate batching and reuse of data

## Assumptions and Dependencies

### Assumptions
- A-1: Users have appropriate AWS permissions to describe VPC, subnet, and network interface resources
- A-2: The target AWS regions contain standard VPC configurations
- A-3: Network interfaces follow standard AWS attachment patterns

### Dependencies
- D-1: AWS SDK Go v2 for EC2 service interactions
- D-2: Existing awstools configuration and helper infrastructure
- D-3: Cobra framework for command line interface
- D-4: go-output library for formatting support