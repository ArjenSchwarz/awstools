# Bugfix Report: org-helper-panic

**Date:** 2026-03-13
**Status:** Fixed
**Transit:** T-418

## Description of the Issue

The organizations helper functions in `helpers/organizations.go` would panic with nil pointer dereferences when AWS API calls (ListChildren, DescribeOrganizationalUnit, DescribeAccount) returned errors. Errors were printed to stdout but execution continued, causing the code to dereference fields on nil response objects.

**Reproduction steps:**
1. Run any `awstools organizations` subcommand (structure or names)
2. Encounter an AWS API error (e.g., access denied, throttling, expired credentials)
3. Application panics with nil pointer dereference instead of displaying an error

**Impact:** Any AWS API error during organization traversal crashes the application with an unrecoverable panic. This affects all users of the `organizations structure` and `organizations names` commands when they have insufficient permissions or experience transient API failures.

## Investigation Summary

- **Symptoms examined:** `findChildren` prints ListChildren errors but continues to range over a potentially nil response; `formatChild` prints Describe* errors but dereferences fields on nil response objects
- **Code inspected:** `helpers/organizations.go` (getOrganizationRoot, findChildren, formatChild), `cmd/organizationsstructure.go`, `cmd/organizationsnames.go`
- **Hypotheses tested:** Confirmed that all four functions had the same pattern: error logged/ignored, nil response dereferenced

## Discovered Root Cause

**Defect type:** Missing error propagation

All four functions (`getOrganizationRoot`, `GetFullOrganization`, `findChildren`, `formatChild`) either printed errors with `fmt.Print`/`fmt.Println` and continued execution, or silently discarded errors with `_`. When AWS API calls returned errors, the response objects were nil, leading to nil pointer dereferences on the next line.

**Why it occurred:** The original code assumed AWS API calls would always succeed. Error handling was limited to printing diagnostics without stopping execution.

**Contributing factors:**
- Functions accepted concrete `*organizations.Client` types, making them impossible to unit test with mocks
- No regression tests existed for error paths (skipped integration tests noted the need for an interface)

## Resolution for the Issue

**Changes made:**
- `helpers/organizations.go` — Introduced `OrganizationsAPI` interface; changed all four functions to return `error` alongside their results; replaced `fmt.Print`/ignored errors with proper error wrapping using `fmt.Errorf` with `%w`
- `cmd/organizationsstructure.go` — Handle error from `GetFullOrganization` with `log.Fatal`
- `cmd/organizationsnames.go` — Handle error from `GetFullOrganization` with `log.Fatal`
- `helpers/organizations_test.go` — Added mock client and 9 regression tests covering all error paths

**Approach rationale:** Returning errors follows the project's own guidelines (CLAUDE.md: "Never use panic()", "Always return errors"). The `OrganizationsAPI` interface enables unit testing without AWS credentials.

**Alternatives considered:**
- Using `panic(err)` to match other helpers — rejected per project guidelines
- Returning sentinel values on error — rejected; proper error propagation is cleaner and more idiomatic Go

## Regression Test

**Test file:** `helpers/organizations_test.go`
**Test names:**
- `TestGetOrganizationRoot_ListRootsError_ReturnsError`
- `TestGetOrganizationRoot_EmptyRoots_ReturnsError`
- `TestFindChildren_ListOUChildrenError_ReturnsError`
- `TestFindChildren_ListAccountChildrenError_ReturnsError`
- `TestFindChildren_DescribeOUErrorDuringTraversal_ReturnsError`
- `TestFormatChild_DescribeOUError_ReturnsError`
- `TestFormatChild_DescribeAccountError_ReturnsError`
- `TestGetFullOrganization_ListRootsError_ReturnsError`
- `TestGetFullOrganization_Success`

**What it verifies:** Every AWS API call in the organizations helper returns a meaningful error when the underlying SDK call fails, instead of panicking with a nil pointer dereference.

**Run command:** `go test ./helpers/ -run "TestGetOrganizationRoot|TestFindChildren|TestFormatChild|TestGetFullOrganization" -v`

## Affected Files

| File | Change |
|------|--------|
| `helpers/organizations.go` | Added `OrganizationsAPI` interface; all functions now return errors properly |
| `helpers/organizations_test.go` | Added mock client and 9 regression tests for error paths |
| `cmd/organizationsstructure.go` | Handle error from `GetFullOrganization` |
| `cmd/organizationsnames.go` | Handle error from `GetFullOrganization` |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- Introduce interfaces for all AWS service clients to enable unit testing of error paths
- Always return errors from functions that make AWS API calls; never log-and-continue
- Use `%w` error wrapping to preserve the original error for callers
- Consider a linter rule or code review checklist item for unchecked AWS SDK errors

## Related

- Transit ticket: T-418
