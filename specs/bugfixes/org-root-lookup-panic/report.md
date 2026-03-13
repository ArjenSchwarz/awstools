# Bugfix Report: org-root-lookup-panic

**Date:** 2025-07-15
**Status:** Fixed

## Description of the Issue

`getOrganizationRoot` in `helpers/organizations.go` panics when the AWS Organizations `ListRoots` API call fails or returns an empty result. The error from `ListRoots` was printed to stdout but execution continued, causing a nil pointer dereference or index-out-of-range panic on `root.Roots[0]`.

**Reproduction steps:**
1. Call any `awstools organizations` subcommand (`structure` or `names`)
2. Have the `ListRoots` API call fail (e.g., insufficient permissions, no organization)
3. Observe runtime panic instead of a graceful error message

**Impact:** Any user without Organizations access or with a misconfigured account experiences an unhandled panic with a stack trace instead of a useful error message.

## Investigation Summary

- **Symptoms examined:** `getOrganizationRoot` prints error from `ListRoots` but continues to access `root.Roots[0]` unconditionally
- **Code inspected:** `helpers/organizations.go`, `cmd/organizationsnames.go`, `cmd/organizationsstructure.go`
- **Hypotheses tested:** Confirmed that (1) API error leaves `root` as nil → nil pointer panic, and (2) successful call with empty `Roots` slice → index out of range panic

## Discovered Root Cause

**Defect type:** Missing error handling / missing nil/empty guard

**Why it occurred:** The original code used `fmt.Print(err)` as a side-effect log but did not return early or propagate the error. The function signature `OrganizationEntry` (no error return) made it impossible for callers to handle failures gracefully.

**Contributing factors:** The function predates the project's newer error-handling patterns (returning `(Type, error)` tuples). Older helpers in the codebase use `panic(err)` which, while not ideal, at least prevents silent continuation past errors.

## Resolution for the Issue

**Changes made:**
- `helpers/organizations.go` — Added `organizationsListRootsAPI` interface for testability. Changed `getOrganizationRoot` to return `(OrganizationEntry, error)` with proper error checking for both API failure and empty results. Changed `GetFullOrganization` to return `(OrganizationEntry, error)` and propagate the error.
- `cmd/organizationsnames.go` — Handle error from `GetFullOrganization` with `log.Fatal`.
- `cmd/organizationsstructure.go` — Handle error from `GetFullOrganization` with `log.Fatal`; added `log` import.

**Approach rationale:** Follows the project's newer convention of returning `(Type, error)` tuples from helper functions, letting callers decide how to handle failures. The narrow `organizationsListRootsAPI` interface enables unit testing without requiring a real AWS client.

**Alternatives considered:**
- Using `panic(err)` like older helpers (s3.go, iam.go) — rejected because panic is an anti-pattern for expected errors per project guidelines.
- Broader `OrganizationsAPI` interface covering all methods — rejected as unnecessarily wide for this fix; only `ListRoots` needed mocking.

## Regression Test

**Test file:** `helpers/organizations_test.go`
**Test names:**
- `TestGetOrganizationRoot_ReturnsErrorOnAPIFailure`
- `TestGetOrganizationRoot_ReturnsErrorOnEmptyRoots`
- `TestGetOrganizationRoot_Success`

**What it verifies:** That `getOrganizationRoot` returns a descriptive error (not a panic) when `ListRoots` fails or returns no roots, and correctly parses a valid root.

**Run command:** `go test -v ./helpers/ -run TestGetOrganizationRoot`

## Affected Files

| File | Change |
|------|--------|
| `helpers/organizations.go` | Added interface, error returns, and nil/empty guards |
| `helpers/organizations_test.go` | Added mock and 3 regression tests |
| `cmd/organizationsnames.go` | Handle error from `GetFullOrganization` |
| `cmd/organizationsstructure.go` | Handle error from `GetFullOrganization` |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] `go fmt` clean

## Prevention

**Recommendations to avoid similar bugs:**
- Always return errors from functions that call AWS APIs; never ignore or print-and-continue
- Use narrow interfaces for AWS client dependencies to enable unit testing
- Consider a codebase-wide audit of remaining `fmt.Print(err)` and `panic(err)` patterns in helpers/
