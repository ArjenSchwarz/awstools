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
- `types.NatGatewayAddress.NetworkInterfaceId` is `*string` and AWS may omit it (e.g. for addresses in transitional states). Always guard for nil or use `aws.ToString` before comparing.

## Transit Gateway Route Parsing

`parseActiveRoute` and `parseBlackholeRoute` convert `types.TransitGatewayRoute`
into the internal `TransitGatewayRoute` struct. The SDK type has two optional
destination pointers — `DestinationCidrBlock` (IPv4 or IPv6 CIDR) and
`PrefixListId` — and AWS populates only one. TGW routes do not have a separate
IPv6 destination field; v6 CIDRs reuse `DestinationCidrBlock`.

The helper `tgwRouteDestination(route)` encapsulates the fallback: CIDR first,
then prefix list ID, then empty string. Never dereference the destination
pointers directly — prefix-list routes will panic.

Attachment fields `TransitGatewayAttachmentId` and `ResourceId` are also
pointers; use `aws.ToString` rather than raw deref.

## ENI Listing

`GetNetworkInterfaces` (`helpers/ec2.go`) accepts
`ec2.DescribeNetworkInterfacesAPIClient` rather than `*ec2.Client` directly.
`*ec2.Client` satisfies the interface so real callers are unaffected, and
tests can pass a mock. The helper uses `NewDescribeNetworkInterfacesPaginator`
so large accounts don't get truncated output. `GetVPCUsageOverview` reuses
this helper; there is no separate private `retrieveNetworkInterfaces`.

## ENI Matching Helpers

Pure, testable helpers live alongside the AWS-client-taking wrappers. Each scans a slice of AWS objects for an ENI or subnet match so nil-safety can be tested without mocking.

- `matchTransitGatewayAttachment(attachments, subnetID)` — T-397
- `matchNatGatewayByENI(natgateways, eniID)` — T-656 (skips addresses with nil `NetworkInterfaceId`; empty `eniID` returns nil immediately)

When adding a new `Get<Resource>FromNetworkInterface` style function, extract the matching logic into a helper following this pattern. Iterate with an index (`for i := range ...`) when returning a pointer into the slice, to avoid the loop-variable pointer reuse bug (T-456).

## Testing Pattern

`GetAllVPCRouteTables` takes `*ec2.Client` so callers don't have to change, but
the pagination logic lives in the unexported `getAllVPCRouteTables` which takes
the narrower `ec2.DescribeRouteTablesAPIClient` interface. Unit tests mock that
interface (see `helpers/vpc_routetable_pagination_test.go`) — this is the same
split used for the IAM pagination tests.

The same split is used by the per-ENI lookup helpers after T-657:
`GetVPCEndpointFromNetworkInterface`, `GetNatGatewayFromNetworkInterface`, and
`GetTransitGatewayFromNetworkInterface` forward to unexported implementations
that take `ec2.DescribeVpcEndpointsAPIClient`,
`ec2.DescribeNatGatewaysAPIClient`, and
`ec2.DescribeTransitGatewayVpcAttachmentsAPIClient` respectively. All three
walk every page via `NewDescribe*Paginator`. Tests live in
`helpers/ec2_eni_lookup_pagination_test.go`.

`GetAllVpcPeers` follows the same split: the exported function takes
`*ec2.Client` while the unexported `getAllVpcPeers` takes
`ec2.DescribeVpcPeeringConnectionsAPIClient` and walks
`ec2.NewDescribeVpcPeeringConnectionsPaginator` (T-746). Before that fix the
helper issued a single `DescribeVpcPeeringConnections` call and silently
dropped peerings on subsequent pages.

## Transit Gateway Inventory (T-669)

The TGW inventory helpers follow the same split pattern. Public wrappers take
`*ec2.Client`; private implementations take the composite
`tgwInventoryAPIClient` interface (bundles
`ec2.DescribeTransitGatewaysAPIClient`,
`ec2.DescribeTransitGatewayRouteTablesAPIClient`,
`ec2.GetTransitGatewayRouteTableAssociationsAPIClient`, and
`SearchTransitGatewayRoutes`). Mock against that composite interface — see
`helpers/tgw_pagination_test.go`.

- `GetAllTransitGateways` → `getAllTransitGateways` walks
  `NewDescribeTransitGatewaysPaginator`.
- `GetRouteTablesForTransitGateway` → `getRouteTablesForTransitGateway` walks
  `NewDescribeTransitGatewayRouteTablesPaginator`.
- `GetSourceAttachmentsForTransitGatewayRouteTable` →
  `getSourceAttachmentsForTransitGatewayRouteTable` walks
  `NewGetTransitGatewayRouteTableAssociationsPaginator`.

**`SearchTransitGatewayRoutes` is the exception** — the AWS API has no
`NextToken` for this operation. Results are capped at 1000 rows with an
`AdditionalRoutesAvailable` flag. The active-route helper
(`getActiveRoutesForTransitGatewayRouteTable`) sets `MaxResults: 1000`
explicitly and, on overflow, re-queries per route type (`propagated` and
`static`) to raise the effective ceiling to ~2000 active routes per route
table. The blackhole-route helper just logs a warning on overflow because
blackhole routes are normally few. Never rely on a single unfiltered
`SearchTransitGatewayRoutes` call in a large account.

## VPC Overview IP Analysis (T-774)

`GetVPCUsageOverview` and its `analyzeSubnetIPUsage` helper enumerate every
IPv4 address in each subnet's CIDR for per-IP analysis. **Never call the
underlying `generateIPRange` with an IPv6 CIDR** — a default /64 contains 2^64
addresses. After T-774:

- `generateIPRange` rejects IPv6 CIDRs with an explicit error.
- `calculateSubnetStats` caps totals at `math.MaxInt` for prefixes where
  `2^hostBits` would overflow a signed int (host bits >= 63).
- `analyzeSubnetIPUsage` returns empty results without error when
  `subnet.CidrBlock` is empty (IPv6-only subnets). The caller previously
  panicked.
- `GetVPCUsageOverview` uses `firstIPv6CIDR(subnet)` as a display fallback when
  no IPv4 CIDR exists, so IPv6-only subnets still appear in the overview with
  their CIDR but no per-address breakdown.

Dual-stack subnets continue to have their IPv4 portion analysed; the IPv6
block is shown separately by the caller logic only when no IPv4 CIDR exists.

Tests live in `helpers/ec2_ipv6_subnet_test.go` and use a 2 s timeout guard so
a regression that re-introduces IPv6 enumeration fails loudly rather than
hanging the test process.

## VPN Connections API

`DescribeVpnConnections` is **not** a paginated AWS API — the input/output
structs have no `NextToken` or `MaxResults` and the SDK provides no paginator.
A single call returns every VPN connection in the region. `addAllVpnNames`
(`helpers/ec2.go`) therefore uses one call, but takes
`ec2.DescribeVpnConnectionsAPIClient` instead of `*ec2.Client` so the helper
is unit-testable (T-746). The same applies to `DescribeVpnGateways`, which is
currently not used anywhere in the codebase.
