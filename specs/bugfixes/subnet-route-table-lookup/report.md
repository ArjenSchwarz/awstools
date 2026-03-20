# Bugfix Report: subnet-route-table-lookup

**Date:** 2026-03-20
**Status:** Fixed
**Transit:** T-510

## Description of the Issue

`GetSubnetRouteTable` returns the wrong route table for subnets that have no explicit route table association. When falling back to a main route table, it picks the first main route table found across all VPCs rather than constraining the lookup to the subnet's own VPC. This causes incorrect public/private classification and wrong route table output in VPC overview and IP finder commands.

**Reproduction steps:**
1. Have two or more VPCs in the same region
2. Have a subnet in VPC B with no explicit route table association (uses the VPC's default main route table)
3. VPC A's main route table appears first in the DescribeRouteTables response
4. Run `awstools vpc overview` or `awstools vpc ipfinder`
5. The subnet in VPC B is assigned VPC A's main route table

**Impact:** Any account with multiple VPCs where subnets rely on the default main route table will see incorrect route table assignments. This can misclassify private subnets as public (or vice versa) and display wrong routes in the output.

## Investigation Summary

- **Symptoms examined:** `GetSubnetRouteTable` fallback path iterates all route tables and returns the first one with `Main == true`, ignoring VPC boundaries
- **Code inspected:** `helpers/ec2.go` (GetSubnetRouteTable, isPublicSubnet, getRouteTableInfo), `cmd/vpcoverview.go` (caller that has VPC context available)
- **Hypotheses tested:** Confirmed that the `types.RouteTable` struct from the AWS SDK includes a `VpcId` field, so the information needed for correct filtering is already present in the data

## Discovered Root Cause

**Defect type:** Missing filter constraint

The fallback branch in `GetSubnetRouteTable` (lines 808-815) iterates all route tables looking for any main route table without checking `routeTable.VpcId`. Each VPC has exactly one main route table, so the function should match only the main route table whose VPC ID matches the subnet's VPC.

**Why it occurred:** The original implementation assumed the route tables list would only contain route tables from a single VPC, or that the first main route table encountered would always be the correct one.

**Contributing factors:**
- The function signature only accepted `subnetID` and had no way to know which VPC the subnet belongs to
- No unit tests existed for this function to catch cross-VPC lookup errors
- In single-VPC accounts, the bug is invisible

## Resolution for the Issue

**Changes made:**
- `helpers/ec2.go` — Added `vpcID` parameter to `GetSubnetRouteTable` and `isPublicSubnet`; the main route table fallback now checks `routeTable.VpcId` matches the provided VPC ID
- `helpers/ec2.go` — Updated `getRouteTableInfo` to accept and pass `vpcID`
- `helpers/ec2.go` — Updated `isPublicSubnet` call in `GetVPCUsageOverview` to pass the VPC ID
- `helpers/ec2.go` — Updated `getRouteTableInfo` call in `FindIPAddressDetails` to pass the VPC ID
- `cmd/vpcoverview.go` — Updated `GetSubnetRouteTable` call to pass `subnet.VPCId`

**Approach rationale:** Adding a `vpcID` parameter is the minimal change that fixes the root cause. The VPC ID is already available at every call site (from the VPC iteration context, from subnet structs, or from ENI data). No new API calls are needed.

**Alternatives considered:**
- Pre-filtering route tables per VPC before passing to the function — rejected because it would require refactoring all callers and adds unnecessary allocation
- Building a subnet-to-VPC lookup map — rejected because the VPC ID is already known at each call site

## Regression Test

**Test file:** `helpers/ec2_test.go`
**Test names:**
- `TestGetSubnetRouteTable_ExplicitAssociation` — verifies explicit subnet association still works
- `TestGetSubnetRouteTable_MainRouteTableMatchesVPC` — verifies main RT fallback selects the correct VPC's table
- `TestGetSubnetRouteTable_NoMatchReturnsNil` — verifies nil is returned when no RT exists for the VPC
- `TestIsPublicSubnet_UsesCorrectVPCMainRouteTable` — verifies public/private classification uses the correct VPC's main RT

**What it verifies:** That the main route table fallback constrains results to the subnet's VPC, preventing cross-VPC misassignment.

**Run command:** `go test ./helpers/ -run "TestGetSubnetRouteTable|TestIsPublicSubnet"`

## Affected Files

| File | Change |
|------|--------|
| `helpers/ec2.go` | Add `vpcID` parameter to `GetSubnetRouteTable`, `isPublicSubnet`, and `getRouteTableInfo`; filter main RT by VPC |
| `helpers/ec2_test.go` | Add 4 regression tests |
| `cmd/vpcoverview.go` | Pass VPC ID to `GetSubnetRouteTable` |

## Verification

**Automated:**
- [ ] Regression test passes
- [ ] Full test suite passes
- [ ] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- When looking up VPC-scoped resources, always constrain queries by VPC ID
- Add unit tests for functions that operate on collections of cross-VPC resources
- Consider using typed VPC-scoped wrapper types to make VPC boundary violations a compile-time error

## Related

- Transit ticket: T-510
