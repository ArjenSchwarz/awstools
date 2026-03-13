# Bugfix Report: tgw-route-attachment-indexing

**Date:** 2025-07-15
**Status:** Fixed
**Ticket:** T-398

## Description of the Issue

`GetActiveRoutesForTransitGatewayRouteTable` in `helpers/ec2.go` panicked with an index-out-of-range error when processing Transit Gateway routes that have no attachments (e.g., local or propagated routes).

**Reproduction steps:**
1. Have a Transit Gateway with local or propagated routes that lack attachments
2. Run any command that calls `GetActiveRoutesForTransitGatewayRouteTable` (e.g., `tgw routes`)
3. Observe panic: `runtime error: index out of range [0] with length 0`

**Impact:** Any AWS account with local/propagated TGW routes would crash the tool.

## Investigation Summary

- **Symptoms examined:** Panic on `route.TransitGatewayAttachments[0]` access
- **Code inspected:** `helpers/ec2.go` lines 445-460 (the active route parsing loop)
- **Hypotheses tested:** Confirmed that AWS SDK `TransitGatewayRoute.TransitGatewayAttachments` can be an empty slice for local/propagated route types

## Discovered Root Cause

**Defect type:** Missing bounds check

**Why it occurred:** The code unconditionally accessed `route.TransitGatewayAttachments[0]` on three separate lines without verifying the slice had any elements. The original author likely only tested with VPC-attached or VPN routes, which always have attachments.

**Contributing factors:** The AWS SDK type uses a slice (not a pointer), so a nil check alone wouldn't catch empty slices.

## Resolution for the Issue

**Changes made:**
- `helpers/ec2.go` — Extracted inline route-parsing into `parseActiveRoute()` function with a `len(route.TransitGatewayAttachments) > 0` guard around all attachment access

**Approach rationale:** Extracting a function makes the logic independently testable and keeps the guard in one place. Routes without attachments get zero-value attachment fields, which is safe for downstream consumers.

**Alternatives considered:**
- Skipping routes with no attachments entirely — rejected because those routes are still valid and should appear in output

## Regression Test

**Test file:** `helpers/ec2_test.go`
**Test names:** `TestParseActiveRoute_NoAttachments`, `TestParseActiveRoute_NilAttachments`, `TestParseActiveRoute_WithAttachment`, `TestParseActiveRoute_VPNStripsPublicIP`

**What it verifies:** Routes with empty, nil, normal, and VPN attachment slices are all handled correctly without panicking.

**Run command:** `go test ./helpers/ -run TestParseActiveRoute -v`

## Affected Files

| File | Change |
|------|--------|
| `helpers/ec2.go` | Extracted `parseActiveRoute()` with attachment length guard |
| `helpers/ec2_test.go` | Added 4 regression tests for `parseActiveRoute` |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes
- [x] `go fmt` passes

## Prevention

**Recommendations to avoid similar bugs:**
- Always check slice length before indexing into AWS SDK response slices
- Extract parsing logic into testable functions rather than inlining in API-calling functions
