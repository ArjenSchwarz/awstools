# Bugfix Report: appmesh-nil-deref

**Date:** 2026-03-13
**Status:** Fixed
**Ticket:** T-346

## Description of the Issue

When AWS App Mesh API calls return errors (e.g., AccessDenied, NotFound), the response object is nil. The code printed the error but continued to dereference fields on the nil response (e.g., `output.Route`, `nodetails.VirtualNode`), causing a nil pointer panic.

**Reproduction steps:**
1. Call any App Mesh command (e.g., `appmesh routes`) with insufficient IAM permissions or against a non-existent mesh
2. AWS SDK returns an error with a nil response object
3. Code prints the error but dereferences the nil response → panic

**Impact:** Any AWS error from App Mesh Describe/List calls causes the entire CLI to crash with a nil pointer dereference.

## Investigation Summary

- **Symptoms examined:** All 8 AWS API call sites in `helpers/appmesh.go` that print errors but fall through to dereference the response
- **Code inspected:** `helpers/appmesh.go` — all functions that call AWS App Mesh APIs
- **Hypotheses tested:** The error handling pattern of `fmt.Print(err)` without return/continue is the sole cause

## Discovered Root Cause

All AWS API call sites in `helpers/appmesh.go` use the pattern:
```go
output, err := svc.SomeAPI(...)
if err != nil {
    fmt.Print(err)
}
// output is nil here when err != nil → panic on dereference
```

**Defect type:** Missing error handling (early return/continue)

**Why it occurred:** The original code assumed API calls would always succeed, or the error print was intended as a temporary placeholder that was never replaced with proper handling.

**Contributing factors:** No interface abstraction meant these paths couldn't be unit-tested without real AWS credentials.

## Resolution for the Issue

**Changes made:**
- `helpers/appmesh.go` — Added `return nil` after errors in List* calls at function scope, and `continue` after errors in Describe* calls within loops
- `helpers/appmesh.go` — Introduced `AppMeshAPI` interface to enable mock-based testing; changed all function signatures from `*appmesh.Client` to `AppMeshAPI`
- `helpers/appmesh_test.go` — Added 9 regression tests covering every nil dereference site, including a partial-error test

**Approach rationale:** Using `continue` in loops allows the function to skip failed items and return partial results, which is the most useful behaviour for CLI users. The interface change enables proper unit testing without AWS credentials.

**Alternatives considered:**
- Returning errors from every function — would require significant refactoring of callers; deferred for a separate task
- Using `log.Fatal` — too aggressive; crashing on a single failed Describe is worse than skipping it

## Regression Test

**Test file:** `helpers/appmesh_test.go`
**Test names:** `TestGetAllAppMeshRoutes_ListVirtualRoutersError_NoPanic`, `TestGetAllAppMeshRoutes_ListRoutesError_NoPanic`, `TestGetAppMeshRouteDescriptions_DescribeRouteError_NoPanic`, `TestGetAllAppMeshNodes_ListVirtualNodesError_NoPanic`, `TestGetAllAppMeshNodes_DescribeVirtualNodeError_NoPanic`, `TestGetAllAppMeshVirtualServices_ListError_NoPanic`, `TestGetAllAppMeshVirtualServices_DescribeError_NoPanic`, `TestGetAppMeshVirtualNodeBackendServices2_Error_NoPanic`, `TestGetAppMeshRouteDescriptions_PartialError_SkipsFailed`

**What it verifies:** Each test injects an AWS API error via the mock client and verifies the function returns gracefully (nil or empty slice) without panicking.

**Run command:** `go test -v -run 'NoPanic|PartialError' ./helpers/`

## Affected Files

| File | Change |
|------|--------|
| `helpers/appmesh.go` | Added `AppMeshAPI` interface; changed function signatures to use interface; added `return nil` / `continue` after all error checks |
| `helpers/appmesh_test.go` | Added mock client and 9 regression tests for error handling |

## Verification

**Automated:**
- [x] Regression tests pass (9 new tests)
- [x] Full test suite passes (`go test ./...`)
- [x] Code formatted (`go fmt ./...`)

## Prevention

**Recommendations to avoid similar bugs:**
- Always add `return` or `continue` after error handling in AWS SDK call sites
- Use interfaces for AWS clients to enable unit testing of error paths
- Consider a project-wide lint rule or code review checklist item for "error printed but not returned"

## Related

- T-346: Avoid nil deref on App Mesh Describe* errors
