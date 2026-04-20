# EC2 Helpers

## Route Table Lookup

`GetSubnetRouteTable(subnetID, vpcID, routeTables)` in `helpers/ec2.go` resolves the route table for a subnet. It first checks for an explicit subnet association, then falls back to the VPC's main route table. The `vpcID` parameter is required to correctly scope the main route table fallback when multiple VPCs share the route table list.

Callers:
- `isPublicSubnet` (internal, used by `GetVPCUsageOverview`)
- `getRouteTableInfo` (internal, used by `FindIPAddressDetails` / IP finder)
- `cmd/vpcoverview.go` (VPC overview command)

All callers must pass the subnet's VPC ID. The VPC ID is available from:
- `*subnet.VpcId` (AWS SDK subnet type)
- `subnet.VPCId` (SubnetUsageInfo struct)
- `eni.VpcId` (ENI struct)

## Key Types

- `SubnetUsageInfo` — used by VPC overview, includes `VPCId` field
- `IPFinderResult` — used by IP finder, includes VPC/Subnet/RouteTable info
- `RouteTableInfo` — simplified route table data for output

## AWS SDK Notes

- `types.RouteTable` has a `VpcId` field — always use it when filtering by VPC
- `DescribeRouteTables` without filters returns route tables across all VPCs
- `types.NatGatewayAddress.NetworkInterfaceId` is `*string` and AWS may omit it (e.g. for addresses in transitional states). Always guard for nil or use `aws.ToString` before comparing.

## ENI Matching Helpers

Pure, testable helpers live alongside the AWS-client-taking wrappers. Each scans a slice of AWS objects for an ENI or subnet match so nil-safety can be tested without mocking.

- `matchTransitGatewayAttachment(attachments, subnetID)` — T-397
- `matchNatGatewayByENI(natgateways, eniID)` — T-656 (skips addresses with nil `NetworkInterfaceId`; empty `eniID` returns nil immediately)

When adding a new `Get<Resource>FromNetworkInterface` style function, extract the matching logic into a helper following this pattern. Iterate with an index (`for i := range ...`) when returning a pointer into the slice, to avoid the loop-variable pointer reuse bug (T-456).
