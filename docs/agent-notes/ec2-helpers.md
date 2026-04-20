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
- `DescribeRouteTables` is paginated (default page size 100). Always walk
  `ec2.NewDescribeRouteTablesPaginator`; a single `DescribeRouteTables` call
  truncates results in large accounts. `GetAllVPCRouteTables`, `retrieveRouteTables`,
  and `addAllRouteTableNames` all follow the paginator pattern.

## Testing Pattern

`GetAllVPCRouteTables` takes `*ec2.Client` so callers don't have to change, but
the pagination logic lives in the unexported `getAllVPCRouteTables` which takes
the narrower `ec2.DescribeRouteTablesAPIClient` interface. Unit tests mock that
interface (see `helpers/vpc_routetable_pagination_test.go`) — this is the same
split used for the IAM pagination tests.
