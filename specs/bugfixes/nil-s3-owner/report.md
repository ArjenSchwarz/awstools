# Bugfix Report: nil-s3-owner

**Date:** 2026-03-13
**Status:** Fixed
**Ticket:** T-354

## Description of the Issue

`GetAllBuckets` in `helpers/s3.go` dereferences `*resp.Owner.DisplayName` without nil checks. The AWS `ListBuckets` API can omit `Owner` or `DisplayName` depending on the caller's permissions, bucket ownership settings (e.g., S3 Object Ownership set to BucketOwnerEnforced), or the region endpoint used. When either field is nil, the dereference causes a nil pointer panic, crashing the application.

**Reproduction steps:**
1. Configure AWS credentials with limited S3 permissions (or use an account where `ListBuckets` omits Owner)
2. Run any awstools command that calls `GetAllBuckets` (e.g., `awstools s3`)
3. Observe nil pointer dereference panic

**Impact:** Application crash (panic) when listing S3 buckets under certain AWS permission configurations.

## Investigation Summary

- **Symptoms examined:** Nil pointer dereference on `resp.Owner.DisplayName`
- **Code inspected:** `helpers/s3.go:GetAllBuckets` (line 46), callers in `GetBucketDetails`
- **Hypotheses tested:** Confirmed that the AWS SDK types `ListBucketsOutput.Owner` (`*types.Owner`) and `Owner.DisplayName` (`*string`) are both pointer types that can be nil

## Discovered Root Cause

The `GetAllBuckets` function directly dereferences `*resp.Owner.DisplayName` without checking whether `resp.Owner` or `resp.Owner.DisplayName` is nil.

**Defect type:** Missing nil validation

**Why it occurred:** The original code assumed `ListBuckets` always returns a populated `Owner` with a `DisplayName`, which is not guaranteed by the AWS API.

**Contributing factors:** AWS SDK v2 uses pointer types for optional fields, requiring explicit nil checks that were omitted here.

## Resolution for the Issue

**Changes made:**
- `helpers/s3.go` — Extracted `resolveOwnerName(*types.Owner) string` helper that safely handles nil `Owner`, nil `DisplayName`, and empty `DisplayName` (falling back to Owner ID)
- `helpers/s3.go` — Updated `GetAllBuckets` to use `resolveOwnerName` instead of raw dereference
- `helpers/s3_test.go` — Added `TestResolveOwnerName` with 6 test cases covering all nil/empty combinations

**Approach rationale:** Extracted a small testable function using the same `aws.ToString()` pattern already used throughout the codebase. Falls back to Owner ID when DisplayName is unavailable, providing a useful identifier rather than an empty string when possible.

**Alternatives considered:**
- Inline nil checks without helper — less testable, harder to read
- Always return empty string on nil — discards useful Owner ID information

## Regression Test

**Test file:** `helpers/s3_test.go`
**Test name:** `TestResolveOwnerName`

**What it verifies:** All nil/empty combinations for Owner and DisplayName fields return safe values without panicking.

**Run command:** `go test ./helpers/ -run TestResolveOwnerName -v`

## Affected Files

| File | Change |
|------|--------|
| `helpers/s3.go` | Added `resolveOwnerName` helper, updated `GetAllBuckets` to use it |
| `helpers/s3_test.go` | Added `TestResolveOwnerName` with 6 test cases |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Code formatted with `go fmt`

## Prevention

**Recommendations to avoid similar bugs:**
- Always use `aws.ToString()` for AWS SDK `*string` fields instead of raw dereference
- Check parent structs for nil before accessing nested pointer fields
- Add nil-safety linting rules for AWS SDK pointer types

## Related

- T-354: Handle nil S3 owner in GetAllBuckets
