# Bugfix Report: IAM Listing Pagination

**Date:** 2026-03-20
**Status:** Fixed

## Description of the Issue

IAM helper functions in `helpers/iam.go` and `helpers/iamroles.go` made single-page API calls without processing pagination markers. AWS IAM list APIs return at most 100 items per page by default. Accounts exceeding this limit would silently receive incomplete data.

**Reproduction steps:**
1. Have an AWS account with more than 100 IAM users, groups, roles, or policies
2. Run `awstools iam userlist` or `awstools iam rolelist`
3. Observe that only the first 100 items are returned

**Impact:** Any account with more than 100 IAM resources of a given type would get incomplete results from the userlist and rolelist commands. This affects all downstream operations that depend on complete IAM data.

## Investigation Summary

Examined all IAM API call sites in `helpers/iam.go` and `helpers/iamroles.go` and identified that none of the list/get-group operations handled pagination.

- **Symptoms examined:** Single API call per list operation, no pagination loop
- **Code inspected:** `helpers/iam.go`, `helpers/iamroles.go`, `helpers/iamresources.go`
- **Hypotheses tested:** Confirmed that AWS SDK v2 provides built-in paginators for all affected APIs

## Discovered Root Cause

Every IAM list helper made exactly one API call and used only the items from that single response page.

**Defect type:** Missing pagination handling

**Why it occurred:** The original implementation was likely written against small test accounts where results fit in a single page, so the truncation was never observed.

**Contributing factors:** AWS IAM APIs default to 100 items per page. The SDK does not warn when responses are truncated.

## Resolution for the Issue

**Changes made:**
- `helpers/iam.go` - Introduced `IAMClient` interface; replaced all single-call list operations with SDK paginator loops
- `helpers/iamroles.go` - Replaced single-call list operations with SDK paginator loops; updated function signatures to use `IAMClient` interface
- `helpers/iamresources.go` - Updated `HasAccessKeys` and `GetLastAccessKeyDate` to accept `IAMClient` interface

**Approach rationale:** AWS SDK v2 provides built-in paginators (`NewListUsersPaginator`, `NewListGroupsPaginator`, etc.) that handle the `IsTruncated`/`Marker` protocol correctly. Using these is the idiomatic approach and avoids manual marker management.

**Alternatives considered:**
- Manual `IsTruncated`/`Marker` loop - Rejected because the SDK paginators already encapsulate this logic correctly and are less error-prone
- Increasing `MaxItems` parameter - Rejected because this only raises the limit, it does not eliminate it

## Regression Test

**Test file:** `helpers/iam_pagination_test.go`
**Test names:** `TestGetUserList_Pagination`, `TestGetGroupNameSliceForUser_Pagination`, `TestGetAllUsersInGroup_Pagination`, `TestGetUserPoliciesMapForUser_Pagination`, `TestGetGroupPoliciesMapForGroup_Pagination`, `TestGetAttachedPoliciesMapForUser_Pagination`, `TestGetAttachedPoliciesMapForGroup_Pagination`, `TestGetPoliciesMap_Pagination`

**What it verifies:** Each test creates a mock IAM client with 5 items and a page size of 2, forcing 3 pages. The test asserts that all 5 items are returned, which would fail if only the first page was read.

**Run command:** `go test ./helpers/ -run Pagination -v`

## Affected Files

| File | Change |
|------|--------|
| `helpers/iam.go` | Added `IAMClient` interface; replaced single API calls with paginator loops for 8 functions |
| `helpers/iamroles.go` | Replaced single API calls with paginator loops for 3 functions; updated signatures to `IAMClient` |
| `helpers/iamresources.go` | Updated 2 method signatures from `*iam.Client` to `IAMClient` |
| `helpers/iam_pagination_test.go` | New file with mock IAMClient and 8 pagination regression tests |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes
- [x] go vet passes
- [x] make test passes

## Prevention

**Recommendations to avoid similar bugs:**
- When calling AWS list APIs, always use the SDK paginator type if available
- Add pagination tests for any new AWS list API integrations
- The `IAMClient` interface enables unit testing of IAM helpers without live AWS access

## Related

- Transit ticket: T-498
