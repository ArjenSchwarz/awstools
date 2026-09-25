# Bugfix Report: S3 bucket enumeration error

**Date:** 2026-09-25
**Status:** Investigating

## Description of the Issue

`awstools s3 list` panics when `ListBuckets` returns an expected AWS error, such as access denied, expired credentials, throttling, or an endpoint failure.

**Reproduction steps:** Run the helper with an `S3API` mock whose `ListBuckets` returns an error. The regression test observes a panic.

**Impact:** The CLI crashes instead of reporting the failure through its normal command error path.

## Investigation Summary

- **Symptoms examined:** `ListBuckets` error causes an immediate panic.
- **Code inspected:** `helpers/s3.go`, `cmd/s3list.go`, S3 mocks and tests, and Cobra root command handling.
- **Hypotheses tested:** The first page fails before bucket details or rendering. A later page should also fail the entire enumeration rather than yield partial results.

## Discovered Root Cause

`GetAllBuckets` calls `panic(err)` for `ListBuckets` errors. Its return type has no error, so `GetBucketDetails` and `s3List` cannot propagate the failure to Cobra.

**Defect type:** Error handling.

**Why it occurred:** The helper contract provided buckets and owner only; the command used Cobra `Run`, which has no error return path.

## Resolution for the Issue

Pending implementation.

## Regression Test

**Test file:** `helpers/s3_test.go`
**Test name:** `TestGetAllBuckets_ListErrorDoesNotPanic`

**What it verifies:** An ordinary enumeration error does not panic.

**Run command:** `GOCACHE=/private/tmp/awstools-go-build go test ./helpers -run TestGetAllBuckets_ListErrorDoesNotPanic -count=1`

## Affected Files

| File | Change |
|------|--------|
| `helpers/s3_test.go` | Failing regression |

## Verification

**Automated:** Regression test fails as expected before the fix, reporting `GetAllBuckets panicked on ListBuckets error: access denied`.

## Prevention

Return AWS API errors through helper and command signatures rather than panicking for expected failures.

## Related

- Transit T-1643
