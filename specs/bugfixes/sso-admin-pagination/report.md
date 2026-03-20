# Bugfix Report: SSO Admin Pagination

**Date:** 2026-03-20
**Status:** Fixed

## Description of the Issue

The SSO Admin helper functions in `helpers/sso.go` made single API calls to four listing endpoints without handling the `NextToken` pagination field. When any of these APIs returned more than 100 results (the maximum page size), the additional items were silently dropped.

**Reproduction steps:**
1. Have an AWS SSO instance with more than 100 permission sets, or a permission set provisioned to more than 100 accounts, or more than 100 account assignments, or more than 100 managed policies on a permission set.
2. Run any SSO command (e.g., `awstools sso list-permission-sets`).
3. Observe that only the first 100 items are returned for the affected listing.

**Impact:** Data truncation. Users with large SSO deployments would see incomplete results without any warning. This affects all four SSO commands: `list-permission-sets`, `by-account`, `by-permission-set`, and `dangling`.

## Investigation Summary

- **Symptoms examined:** API calls with `MaxResults: 100` and no `NextToken` handling.
- **Code inspected:** `helpers/sso.go` — all four listing functions (`getPermissionSets`, `getPermissionSetDetails`, `addAccountInfo`).
- **Hypotheses tested:** Confirmed that all four AWS SSO Admin listing APIs (`ListPermissionSets`, `ListAccountsForProvisionedPermissionSet`, `ListAccountAssignments`, `ListManagedPoliciesInPermissionSet`) return `NextToken` fields for pagination.

## Discovered Root Cause

All four SSO Admin listing calls made a single API request and used only the first page of results. The `NextToken` field in the response was never checked.

**Defect type:** Missing pagination loop.

**Why it occurred:** The original implementation assumed the results would fit in a single page. No pagination was implemented for any of the SSO Admin listing APIs.

**Contributing factors:** The functions used `panic()` for error handling, which made them untestable with mock clients (no interface was defined), so this bug was never caught by tests.

## Resolution for the Issue

**Changes made:**
- `helpers/sso.go` — Introduced `SSOAdminAPI` interface. Added pagination loops (NextToken handling) to all four listing calls. Converted all `panic()` calls to proper error returns.
- `cmd/ssolistpermissionsets.go` — Updated to handle error return from `GetSSOAccountInstance`.
- `cmd/ssooverviewaccount.go` — Updated to handle error return from `GetSSOAccountInstance`.
- `cmd/ssooverviewpermissionset.go` — Updated to handle error return from `GetSSOAccountInstance`.
- `cmd/ssodangling.go` — Updated to handle error return from `GetSSOAccountInstance`.
- `helpers/sso_test.go` — Added mock client, pagination regression tests for all four listing APIs, error handling tests, and a full end-to-end pagination test.

**Approach rationale:** Follows the existing pagination pattern used in `helpers/role_discovery.go` and the interface/mock pattern used in `helpers/organizations.go`.

**Alternatives considered:**
- Using AWS SDK built-in paginators — rejected because the existing codebase uses manual pagination loops consistently, and the manual approach keeps the code uniform.

## Regression Test

**Test file:** `helpers/sso_test.go`
**Test names:** `TestListPermissionSets_Pagination`, `TestListAccountsForProvisionedPermissionSet_Pagination`, `TestListAccountAssignments_Pagination`, `TestListManagedPoliciesInPermissionSet_Pagination`, `TestGetSSOAccountInstance_FullPagination`

**What it verifies:** Each test configures a mock that returns results across two pages (with a NextToken on the first response). The tests assert that all items from both pages are collected and that the API is called the expected number of times.

**Run command:** `go test ./helpers/ -run "TestList.*Pagination|TestGetSSOAccountInstance_Full"`

## Affected Files

| File | Change |
|------|--------|
| `helpers/sso.go` | Added `SSOAdminAPI` interface, pagination loops, error returns |
| `helpers/sso_test.go` | Added mock client and pagination/error regression tests |
| `cmd/ssolistpermissionsets.go` | Handle error return from `GetSSOAccountInstance` |
| `cmd/ssooverviewaccount.go` | Handle error return from `GetSSOAccountInstance` |
| `cmd/ssooverviewpermissionset.go` | Handle error return from `GetSSOAccountInstance` |
| `cmd/ssodangling.go` | Handle error return from `GetSSOAccountInstance` |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass (`make test`)

## Prevention

**Recommendations to avoid similar bugs:**
- When using AWS SDK listing APIs, always implement pagination loops. Check for `NextToken` in every listing response.
- Define interfaces for AWS service clients so helper functions can be tested with mocks.
- Return errors instead of using `panic()` — this enables proper testing and graceful error handling.

## Related

- Transit ticket: T-479
