# Bugfix Report: IPv6 Public Subnet Detection

**Date:** 2026-03-29
**Status:** Fixed
**Transit:** T-573

## Description of the Issue

The subnet classification logic in `helpers/ec2.go` only checks for IPv4 default routes (`0.0.0.0/0`) via internet gateways when determining if a subnet is public. Subnets that have an IPv6 default route (`::/0`) via an internet gateway — such as IPv6-only or dual-stack public subnets — are incorrectly classified as private.

**Reproduction steps:**
1. Create a VPC with an IPv6 CIDR block
2. Create a subnet with only an IPv6 default route (`::/0`) pointing to an internet gateway
3. Run `awstools vpc usage` or any command that classifies subnets
4. Observe: the subnet is incorrectly reported as private

**Impact:** IPv6-only and dual-stack public subnets are misclassified, leading to inaccurate VPC usage reports. This affects AWS practitioners analyzing their infrastructure who use IPv6 networking.

## Investigation Summary

- **Symptoms examined:** `hasInternetGatewayRoute()` returns `false` for route tables containing only IPv6 default routes via IGW
- **Code inspected:** `helpers/ec2.go:881-895` — the `hasInternetGatewayRoute` function
- **Hypotheses tested:** The function only inspects `route.DestinationCidrBlock` (IPv4) and never checks `route.DestinationIpv6CidrBlock`

## Discovered Root Cause

The `hasInternetGatewayRoute` function at `helpers/ec2.go:881` only examines `route.DestinationCidrBlock` (the IPv4 CIDR field on the AWS Route type). It does not examine `route.DestinationIpv6CidrBlock`, which is the separate field the AWS SDK uses for IPv6 destinations.

**Defect type:** Missing condition — incomplete protocol coverage

**Why it occurred:** The original implementation only considered IPv4 routing, which was the common case when the code was written. IPv6 support in AWS VPCs has become more prevalent, but the subnet classification logic was never updated.

**Contributing factors:** The AWS SDK uses separate fields for IPv4 and IPv6 CIDR destinations on the `Route` type, making it easy to overlook one when only testing with IPv4.

## Resolution for the Issue

**Changes made:**
- `helpers/ec2.go:881-895` — Extended `hasInternetGatewayRoute` to also check `route.DestinationIpv6CidrBlock` for `::/0` when the route targets an internet gateway (`igw-` prefix)

**Approach rationale:** Minimal change that mirrors the existing IPv4 check pattern, keeping the function simple and readable.

**Alternatives considered:**
- Combining both checks into a single helper — unnecessary complexity for a two-field check
- Checking for any broad IPv6 range (not just `::/0`) — `::/0` is the standard IPv6 default route; no need to over-generalise

## Regression Test

**Test file:** `helpers/ec2_test.go`
**Test names:**
- `TestHasInternetGatewayRoute_IPv6DefaultRoute` — IPv6-only route table with IGW
- `TestHasInternetGatewayRoute_DualStack` — dual-stack route table with IGW
- `TestHasInternetGatewayRoute_IPv6EgressOnly` — eigw should NOT count as public
- `TestIsPublicSubnet_IPv6OnlyPublic` — end-to-end IPv6 subnet classification

**What it verifies:** Subnets with `::/0` via `igw-*` are classified as public; subnets with `::/0` via `eigw-*` are not.

**Run command:** `go test ./helpers/ -run "TestHasInternetGatewayRoute_IPv6|TestIsPublicSubnet_IPv6|TestHasInternetGatewayRoute_DualStack|TestHasInternetGatewayRoute_IPv6EgressOnly" -v`

## Affected Files

| File | Change |
|------|--------|
| `helpers/ec2.go` | Add IPv6 `::/0` check in `hasInternetGatewayRoute` |
| `helpers/ec2_test.go` | Add 4 regression tests for IPv6 subnet classification |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- When handling AWS route properties, always consider both IPv4 (`DestinationCidrBlock`) and IPv6 (`DestinationIpv6CidrBlock`) fields
- Add IPv6 test cases alongside IPv4 tests for any routing-related functionality
- Consider adding a linter or code review checklist item for dual-stack coverage

## Related

- Transit: T-573
