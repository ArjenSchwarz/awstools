# Bugfix Report: tgw-attachment-lookup

**Date:** 2026-03-13
**Status:** Fixed

## Description of the Issue

`GetTransitGatewayFromNetworkInterface` only inspected the first element of `resp.TransitGatewayVpcAttachments`. When a VPC had multiple TGW attachments and the network interface's subnet belonged to a later attachment, the function returned an empty string instead of the correct attachment ID.

**Reproduction steps:**
1. Have a VPC with multiple Transit Gateway attachments using different subnets
2. Query an ENI whose subnet is on the second (or later) TGW attachment
3. Observe that `GetTransitGatewayFromNetworkInterface` returns empty string

**Impact:** Any ENI on a subnet served by a non-first TGW attachment would show no Transit Gateway association in VPC ENI reports, causing incomplete or misleading output.

## Investigation Summary

- **Symptoms examined:** Function returns empty string for ENIs on valid TGW-attached subnets
- **Code inspected:** `helpers/ec2.go:GetTransitGatewayFromNetworkInterface` (lines 556-576)
- **Hypotheses tested:** Off-by-one or missing iteration — confirmed by reading the code

## Discovered Root Cause

The function checked only `resp.TransitGatewayVpcAttachments[0]` instead of iterating through all returned attachments.

**Defect type:** Logic error — incomplete iteration

**Why it occurred:** The original implementation assumed at most one TGW attachment per VPC, which is incorrect — a VPC can have multiple TGW attachments with different subnet associations.

**Contributing factors:** The neighbouring function `GetVPCEndpointFromNetworkInterface` correctly iterates all results, suggesting this was an oversight rather than a deliberate design choice.

## Resolution for the Issue

**Changes made:**
- `helpers/ec2.go:557-583` — Extracted matching logic into `matchTransitGatewayAttachment` which iterates all attachments and returns the first whose `SubnetIds` contain the target subnet.

**Approach rationale:** Extracting the pure matching function makes the logic testable without AWS API calls while keeping the change minimal and consistent with neighbouring functions.

**Alternatives considered:**
- Inline loop without extraction — simpler but untestable without mocking the EC2 client

## Regression Test

**Test file:** `helpers/ec2_test.go`
**Test name:** `TestMatchTransitGatewayAttachment`

**What it verifies:** That the correct attachment ID is returned when the target subnet appears in the first, second, third, or no attachment. The key regression case is "subnet on second attachment" which would have returned empty before the fix.

**Run command:** `go test ./helpers/ -run TestMatchTransitGatewayAttachment -v`

## Affected Files

| File | Change |
|------|--------|
| `helpers/ec2.go` | Extract `matchTransitGatewayAttachment`; iterate all attachments |
| `helpers/ec2_test.go` | Add `TestMatchTransitGatewayAttachment` with 5 cases |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] `go fmt` clean

## Prevention

**Recommendations to avoid similar bugs:**
- When processing AWS API responses that return slices, always iterate the full result set
- Extract matching logic into pure functions for testability
- Look at neighbouring functions for correct patterns (e.g., `GetVPCEndpointFromNetworkInterface`)

## Related

- Transit ticket: T-397
