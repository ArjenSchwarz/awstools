# Bugfix Report: App Mesh nil nested fields

**Date:** 2026-09-26
**Status:** Fixed locally

## Description of the Issue

Partial App Mesh Describe payloads could panic the mesh route and backend helpers when a route action or HTTP match was missing, or when a union member was a typed-nil pointer. Zero-valued union members also produced empty service or backend names.

**Reproduction steps:** Feed `buildRoutesHolder` a route with a recognized type and nil `Action` or `Match`; feed `GetAllAppMeshPaths` or `getAppMeshVirtualNodeBackendServices2` a typed-nil member. Before the fix, the regression tests reported nil pointer panics.

**Impact:** App Mesh commands using these helpers could crash or show meaningless empty entries.

## Investigation Summary

- **Symptoms examined:** Missing HTTP/HTTP2/gRPC/TCP actions and HTTP/HTTP2 matches panicked; typed-nil provider or backend members panicked.
- **Code inspected:** `helpers/appmesh.go`, callers, existing App Mesh tests, and AWS SDK v2 App Mesh union member types.
- **Hypotheses tested:** SDK union `Value` fields are structs, so they cannot be nil. The member pointer itself can be typed-nil. A zero-valued `Value` contains no service/router/node name.

## Discovered Root Cause

Top-level payloads were guarded, but `buildRoutesHolder` dereferenced nested `Action` and HTTP `Match` pointers. Provider and backend type switches dereferenced typed-nil member pointers and appended empty names from zero-valued members.

**Defect type:** Missing nested pointer validation.

## Resolution for the Issue

- `helpers/appmesh.go` skips routes with no action. HTTP/HTTP2 routes with targets and no match retain the destination with an empty path, matching the existing gRPC behavior. TCP routes remain supported.
- Provider and backend type switches skip typed-nil members and members with no router, node, or service name. Valid neighboring entries remain in the result.
- `docs/agent-notes/appmesh-helpers.md` and `rules/go.md` record the project behavior and Go typed-nil interface pitfall.

**Approach rationale:** The guards sit beside the dereferences and preserve the helper's existing partial-payload behavior. They avoid adding new error signatures to the command path.

**Alternatives considered:** Returning errors for each malformed Describe payload would require changing the partial-result contract used by callers.

## Regression Test

**Test file:** `helpers/appmesh_partial_fields_test.go`

**Test names:** `TestBuildRoutesHolder_MissingNestedFields`, `TestGetAllAppMeshPaths_MissingProviderMember`, `TestGetAppMeshVirtualNodeBackendServices2_MissingMember`

**What they verify:** Partial entries do not panic or create empty destinations, and valid neighboring entries remain visible across all route types and relevant union member types.

## Affected Files

| File | Change |
|------|--------|
| `helpers/appmesh.go` | Guard route action/match and provider/backend members |
| `helpers/appmesh_partial_fields_test.go` | Regression tests |
| `docs/agent-notes/appmesh-helpers.md` | App Mesh partial-payload behavior |
| `rules/go.md` | Typed-nil interface rule |
| `CHANGELOG.md` | User-visible fix |

## Verification

**Automated:** The regression tests failed before implementation for all four missing route actions, HTTP/HTTP2 missing matches, typed-nil union members, and empty names. `make fmt`, `make vet`, `make lint`, and `make test` pass with writable caches.

**Manual verification:** No live App Mesh account was used; mock Describe payloads cover the failure path.

## Prevention

Validate nested pointer fields and typed-nil union members before dereferencing; include partial AWS responses in tests.

## Related

- Transit T-1634
- T-346, T-345, T-1207 partial App Mesh payload fixes
