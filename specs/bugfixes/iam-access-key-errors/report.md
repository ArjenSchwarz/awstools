# Bugfix Report: IAM Access Key API Errors

**Date:** 2026-09-25  
**Status:** Fixed

## Description of the Issue

`awstools iam userlist` panics when IAM returns an error while checking a user's access keys or their last use date.

**Reproduction steps:**
1. Run `iam userlist` with a user whose `ListAccessKeys` or `GetAccessKeyLastUsed` call fails.
2. Observe a panic and stack trace instead of a command error.

**Impact:** One expected IAM failure aborts the command without an actionable error.

## Investigation Summary

- **Symptoms examined:** The three API error branches in `IAMUser.HasAccessKeys` and `IAMUser.GetLastAccessKeyDate` call `panic(err)`.
- **Code inspected:** `helpers/iamresources.go`, its only caller in `cmd/iamuserlist.go`, IAM client test mocks, and Cobra's `RunE` handling.
- **Hypotheses tested:** This is independent of nil optional SDK fields (T-1240). Errors returned by the client reach explicit panic statements.

## Discovered Root Cause

**Defect type:** Error handling.

**Why it occurred:** IAM calls can fail for routine reasons; these helpers treat every failure as unrecoverable. Their caller uses Cobra `Run`, which cannot return an error. The command therefore has no error path for access-key enrichment.

**Contributing factors:** The IAM helper tests did not exercise API failures for access-key calls.

## Resolution for the Issue

**Changes made:**
- `helpers/iamresources.go` returns wrapped errors from both access-key helpers.
- `cmd/iamuserlist.go` uses Cobra `RunE` to return those errors with the affected IAM username.

**Approach rationale:** The existing output remains unchanged when IAM succeeds. On an API failure, Cobra reports the error and the command exits without a stack trace.

**Alternatives considered:**
- Render an unknown value for a failed user and continue — this could disguise a permission failure as complete output.


## Regression Test

**Test file:** `helpers/iam_access_key_errors_test.go`  
**Test name:** `TestIAMUser_AccessKeyAPIErrorsDoNotPanic_T1549`

**What it verifies:** Failures from each access-key API call return the original error with the affected username and do not panic.

**Run command:** `go test ./helpers -run TestIAMUser_AccessKeyAPIErrorsDoNotPanic_T1549`

## Affected Files

| File | Change |
|------|--------|
| `helpers/iam_access_key_errors_test.go` | Regression coverage for three API failure paths |
| `helpers/iamresources.go` | Return contextual errors from access-key helpers |
| `cmd/iamuserlist.go` | Propagate access-key errors through Cobra |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`make test`)
- [x] Linters/validators pass (`make fmt`, `make vet`, `make lint`)

**Manual verification:** The regression test failed with the original panic before the fix and passed after it.

## Prevention

Return ordinary AWS API failures as errors, and test each API failure path with a mock client.

## Related

- Transit T-1549
