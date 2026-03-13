# Bugfix Report: non-http-appmesh-routes

**Date:** 2026-03-13
**Status:** Fixed

## Description of the Issue

`GetAllAppMeshPaths` in `helpers/appmesh.go` unconditionally dereferences `route.Spec.HttpRoute` when iterating over routes. AWS App Mesh supports four route types: HTTP, HTTP/2, gRPC, and TCP. When a route uses gRPC, TCP, or HTTP/2 (via `Http2Route`), the `HttpRoute` field is nil, causing a nil pointer dereference panic.

**Reproduction steps:**
1. Configure an App Mesh with a virtual router that has gRPC or TCP routes
2. Run any awstools command that calls `GetAllAppMeshPaths` (e.g., appmesh connections/paths)
3. Observe panic: `runtime error: invalid memory address or nil pointer dereference`

**Impact:** Any mesh containing non-HTTP routes causes a crash, making the tool unusable for mixed-protocol meshes.

## Investigation Summary

- **Symptoms examined:** Nil pointer dereference at `route.Spec.HttpRoute.Action.WeightedTargets`
- **Code inspected:** `helpers/appmesh.go` lines 108-130, AWS SDK types for `RouteSpec`
- **Hypotheses tested:** Confirmed that `RouteSpec` has four optional route fields (`HttpRoute`, `Http2Route`, `GrpcRoute`, `TcpRoute`), all of which contain `WeightedTargets` in their action

## Discovered Root Cause

**Defect type:** Missing nil check / incomplete type handling

**Why it occurred:** The original implementation only considered HTTP routes. When App Mesh added gRPC, TCP, and HTTP/2 route types, the code was not updated to handle them.

**Contributing factors:** All route types share similar structure (Action with WeightedTargets) but are separate fields on `RouteSpec`, making it easy to assume only one exists.

## Resolution for the Issue

**Changes made:**
- `helpers/appmesh.go` - Extracted route processing into `buildRoutesHolder` function that branches on route type (HttpRoute, Http2Route, GrpcRoute, TcpRoute), with nil checks for each. Routes with no recognized type are skipped.

**Approach rationale:** Handle all four route types since they all have WeightedTargets. For the Path field: HTTP/HTTP2 use Match.Prefix, gRPC uses ServiceName/MethodName, TCP uses empty string (no path concept).

**Alternatives considered:**
- Skip non-HTTP routes entirely — rejected because it loses useful routing information
- Use a single interface-based approach — rejected because SDK types don't share a common interface

## Regression Test

**Test file:** `helpers/appmesh_test.go`
**Test names:** `TestBuildRoutesHolder_GrpcRoute_NoPanic`, `TestBuildRoutesHolder_TcpRoute_NoPanic`, `TestBuildRoutesHolder_Http2Route_NoPanic`, `TestBuildRoutesHolder_MixedRouteTypes`, `TestBuildRoutesHolder_NilRouteSpec_NoPanic`

**What it verifies:** Non-HTTP route types are processed without panic and produce correct routing entries.

**Run command:** `go test ./helpers/ -run TestBuildRoutesHolder -v`

## Affected Files

| File | Change |
|------|--------|
| `helpers/appmesh.go` | Extract `buildRoutesHolder`, handle all route types with nil checks |
| `helpers/appmesh_test.go` | Add regression tests for all route types |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- When handling AWS SDK union-like structs with multiple optional fields, always check which field is populated
- Add tests for each variant when processing polymorphic AWS resource types
- Consider adding nil-safety helper functions for common AWS type patterns

## Related

- Transit ticket: T-345
