# Bugfix Report: vpc ip-finder omits Transit Gateway resource IDs

**Date:** 2026-06-01
**Status:** Fixed
**Ticket:** T-1147

## Description of the Issue

`awstools vpc ip-finder` against an IP that belongs to a Transit Gateway (TGW)
ENI rendered `Resource Type = Transit Gateway` with a useful Resource Name /
attachment detail, but the `Resource ID` column stayed empty.

**Reproduction steps:**
1. Run `awstools vpc ip-finder` for an IP on a TGW ENI.
2. Inspect the output table (`cmd/vpcipfinder.go`).
3. `Resource Type` shows `Transit Gateway`, `Resource Name` shows the attachment
   detail, but `Resource ID` is blank.

**Impact:** Low severity (data completeness). TGW ENIs were inconsistent with
EC2 instances, VPC endpoints, and NAT gateways, which all populate the ID.

## Investigation Summary

- **Symptoms examined:** `IPFinderResult.ResourceID` empty for TGW ENIs while
  `ResourceName`/`ResourceType` were correct.
- **Code inspected:** `helpers/ec2.go` (`getResourceNameAndID`,
  `getENIAttachmentDetailsOptimized`, `GetAttachmentFromCache`,
  `ENILookupCache`), `cmd/vpcipfinder.go`.
- **Hypotheses tested:** Confirmed the attachment ID is available in
  `ENILookupCache.TransitGateways` (`map[string]string`, VPC ID -> TGW
  attachment ID), populated by `batchFetchTransitGateways`. The name path
  already used it; only the ID path was missing.

## Discovered Root Cause

`getResourceNameAndID` resolved the resource ID for EC2 instances
(`eni.Attachment.InstanceId`), VPC endpoints (`cache.EndpointsByENI`), and NAT
gateways (`cache.NATGatewaysByENI`), but had no branch for Transit Gateway
ENIs. TGW ENIs therefore fell through to `return attachmentDetails, ""`.

**Defect type:** Missing case / incomplete branch coverage.

**Why it occurred:** TGW attachment IDs are keyed by VPC ID rather than by ENI
ID (unlike endpoints and NAT gateways), so the existing ENI-keyed lookups did
not cover them, and no VPC-keyed lookup was added for the ID path.

## Resolution for the Issue

**Changes made:**
- `helpers/ec2.go` (`getResourceNameAndID`) - Added a Transit Gateway branch:
  when `eni.InterfaceType == types.NetworkInterfaceTypeTransitGateway` and
  `eni.VpcId != nil`, look up `cache.TransitGateways[*eni.VpcId]` and return the
  attachment ID alongside the existing attachment detail string.

**Approach rationale:** Mirrors how `getENIAttachmentDetailsOptimized` and
`GetAttachmentFromCache` already resolve the TGW attachment for the name,
reusing the same cache and nil guards. Minimal, consistent, no new API calls.

**Alternatives considered:**
- Refactor `getResourceNameAndID` into a `switch` on `InterfaceType` like
  `GetAttachmentFromCache` - Rejected to keep the change minimal and preserve
  the existing EC2/endpoint/NAT ordering.

## Regression Test

**Test file:** `helpers/ec2_test.go`
**Test name:** `TestGetResourceNameAndID_TransitGateway`

**What it verifies:**
- TGW ENI with a matching cache entry returns the attachment ID.
- TGW ENI with a nil `VpcId` returns an empty ID (no panic).
- TGW ENI with no matching cache entry returns an empty ID.

Verified red/green: the first subtest fails before the fix
(`resource ID = "", want "tgw-attach-..."`) and passes after.

**Run command:** `go test ./helpers/ -run TestGetResourceNameAndID_TransitGateway -count=1`

## Affected Files

| File | Change |
|------|--------|
| `helpers/ec2.go` | Added Transit Gateway branch to `getResourceNameAndID` |
| `helpers/ec2_test.go` | Added `TestGetResourceNameAndID_TransitGateway` |

## Verification

**Automated:**
- [x] Regression test passes (and fails without the fix)
- [x] Full test suite passes (`go test ./...`)
- [x] `go vet ./...` passes, `go fmt ./...` clean

## Prevention

- When adding a resource type to ENI usage/attachment helpers, wire it into all
  three resolvers (`getENIUsageTypeOptimized`, `getENIAttachmentDetailsOptimized`,
  and `getResourceNameAndID`) so type, name, and ID stay in sync.

## Related

- PR #94
