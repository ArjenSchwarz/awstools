# Bugfix Report: App Mesh nil nested fields

**Date:** 2026-09-26
**Status:** Investigating

## Description of the Issue

Partial App Mesh Describe payloads can panic the mesh route and backend helpers when a route action or HTTP match is missing, or when a union member is a typed-nil pointer. Zero-valued union members also produce empty service or backend names.

**Reproduction steps:** Feed `buildRoutesHolder` a route with a recognized type and nil `Action` or `Match`; feed `GetAllAppMeshPaths` or `getAppMeshVirtualNodeBackendServices2` a typed-nil member. The regression tests report nil pointer panics.

**Impact:** App Mesh commands using these helpers may crash or show meaningless empty entries.

## Investigation Summary

- **Symptoms examined:** Missing HTTP/HTTP2/gRPC/TCP actions and HTTP/HTTP2 matches panic; typed-nil provider or backend members panic.
- **Code inspected:** `helpers/appmesh.go`, callers, existing App Mesh tests, and AWS SDK v2 App Mesh union member types.
- **Hypotheses tested:** SDK union `Value` fields are structs, so they cannot be nil. The member pointer itself can be typed-nil. A zero-valued `Value` contains no service/router/node name.

## Discovered Root Cause

Top-level payloads are guarded, but `buildRoutesHolder` dereferences nested `Action` and HTTP `Match` pointers. Provider and backend type switches dereference typed-nil member pointers and append empty names from zero-valued members.

**Defect type:** Missing nested pointer validation.

## Resolution for the Issue

Pending implementation.

## Regression Test

**Test file:** `helpers/appmesh_partial_fields_test.go`

**Test names:** `TestBuildRoutesHolder_MissingNestedFields`, `TestGetAllAppMeshPaths_MissingProviderMember`, `TestGetAppMeshVirtualNodeBackendServices2_MissingMember`

**What they verify:** Partial entries do not panic or create empty destinations, and valid neighboring entries remain visible.

## Affected Files

| File | Change |
|------|--------|
| `helpers/appmesh_partial_fields_test.go` | Failing regressions |

## Verification

**Automated:** Regression tests fail before implementation on all four missing route actions, HTTP/HTTP2 missing matches, typed-nil union members, and empty names.

## Prevention

Guard nested pointers and typed-nil union members at the point of use.

## Related

- Transit T-1634
- T-346, T-345, T-1207 partial App Mesh payload fixes
