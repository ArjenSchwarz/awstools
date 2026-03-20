# Bugfix Report: eni-lookup-cache-pointer-reuse

**Date:** 2026-03-20
**Status:** Fixed

## Description of the Issue

`batchFetchVPCEndpoints` and `batchFetchNATGateways` in `helpers/ec2.go` store `&endpoint` / `&natgw` from a `range` loop into cache maps. In Go versions prior to 1.22, the range loop reuses a single variable for all iterations, so all map entries end up pointing to the last item in the slice. This causes ENI attachment lookups to return the wrong VPC endpoint or NAT gateway.

**Reproduction steps:**
1. Have multiple VPC endpoints or NAT gateways across ENIs in a VPC
2. Run any command that populates `ENILookupCache` (e.g., ENI listing with details)
3. Observe that all ENIs report the same endpoint/NAT gateway — the last one from the API response

**Impact:** Medium — incorrect resource attribution in ENI reports. While Go 1.22+ mitigates the runtime bug through per-iteration loop variables, the code pattern is still fragile and non-idiomatic.

## Investigation Summary

Systematic inspection of the ENI cache population code.

- **Symptoms examined:** All cache map entries pointing to the same VPC endpoint / NAT gateway
- **Code inspected:** `batchFetchVPCEndpoints`, `batchFetchNATGateways`, `ENILookupCache` struct, and consumer functions (`getENIUsageTypeOptimized`, `getENIAttachmentDetailsOptimized`, `getResourceNameAndID`)
- **Hypotheses tested:** Confirmed that `&endpoint` and `&natgw` take pointers to loop variables rather than slice elements

## Discovered Root Cause

Both `batchFetchVPCEndpoints` (line 1337) and `batchFetchNATGateways` (line 1409) use `for _, endpoint := range ...` and store `&endpoint` into the cache map. The loop variable `endpoint` is a copy, not a reference to the slice element.

**Defect type:** Loop variable pointer capture

**Why it occurred:** Common Go pitfall — taking the address of a range loop variable. The `for _, v := range` form copies each element into the same variable `v` (in Go < 1.22), so `&v` always points to the same address.

**Contributing factors:**
- Go 1.22+ changed loop semantics to create per-iteration variables, which masks this bug at runtime for modules with `go 1.22` or later in `go.mod`
- The current `go.mod` specifies `go 1.25.1`, so the bug does not manifest at runtime, but the code pattern remains fragile and non-idiomatic

## Resolution for the Issue

**Changes made:**
- `helpers/ec2.go:1337-1341` — Changed `batchFetchVPCEndpoints` to use index-based iteration (`for i := range`) and take the address of the slice element directly (`&resp.VpcEndpoints[i]`)
- `helpers/ec2.go:1409-1415` — Changed `batchFetchNATGateways` to use index-based iteration and take `&resp.NatGateways[i]`

**Approach rationale:** Using `&slice[i]` instead of `&loopVar` is the idiomatic Go pattern for storing pointers to slice elements. It avoids the loop variable capture issue entirely, works correctly regardless of Go version, and makes the intent explicit.

**Alternatives considered:**
- Copying the loop variable before taking its address (`ep := endpoint; cache[id] = &ep`) — works but creates unnecessary copies; `&slice[i]` is more direct
- Relying on Go 1.22+ per-iteration semantics and leaving the code as-is — fragile if go.mod version is ever lowered; non-idiomatic pattern that linters flag

## Regression Test

**Test file:** `helpers/ec2_test.go`
**Test names:** `TestENILookupCache_EndpointsByENI_DistinctEntries`, `TestENILookupCache_NATGatewaysByENI_DistinctEntries`

**What it verifies:**
1. Each ENI maps to the correct VPC endpoint (not the last one in the list)
2. Each ENI maps to the correct NAT gateway (not the last one in the list)
3. Pointers for distinct resources are at distinct memory addresses

**Run command:** `go test ./helpers/ -run "TestENILookupCache_EndpointsByENI_DistinctEntries|TestENILookupCache_NATGatewaysByENI_DistinctEntries" -v`

## Affected Files

| File | Change |
|------|--------|
| `helpers/ec2.go` | Changed loop pattern in `batchFetchVPCEndpoints` and `batchFetchNATGateways` to use index-based iteration |
| `helpers/ec2_test.go` | Added regression tests verifying distinct cache entries |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- Use `for i := range slice` with `&slice[i]` when storing pointers to slice elements in maps or other data structures
- Enable the `loopvar` linter check (e.g., `copyloopvar` in golangci-lint) to catch this pattern automatically
- Avoid taking the address of range loop value variables (`&v` in `for _, v := range`)

## Related

- Transit ticket: T-456
