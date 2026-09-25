# Bugfix Report: IAM Access Key API Errors

**Date:** 2026-09-25  
**Status:** Investigating

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

Pending implementation.

## Regression Test

**Test file:** `helpers/iam_access_key_errors_test.go`  
**Test name:** `TestIAMUser_AccessKeyAPIErrorsDoNotPanic_T1549`

**What it verifies:** Failures from each access-key API call do not panic.

**Run command:** `go test ./helpers -run TestIAMUser_AccessKeyAPIErrorsDoNotPanic_T1549`

## Affected Files

| File | Change |
|------|--------|
| `helpers/iam_access_key_errors_test.go` | Regression coverage for three API failure paths |
| `helpers/iamresources.go` | Planned error return from access-key helpers |
| `cmd/iamuserlist.go` | Planned command error propagation |

## Verification

**Automated:**
- [ ] Regression test passes
- [ ] Full test suite passes
- [ ] Linters/validators pass

## Prevention

Return ordinary AWS API failures as errors, and test each API failure path with a mock client.

## Related

- Transit T-1549
