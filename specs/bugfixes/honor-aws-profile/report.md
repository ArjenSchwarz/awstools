# Bugfix Report: honor-aws-profile

**Date:** 2025-07-15
**Status:** Fixed
**Ticket:** T-405

## Description of the Issue

When setting `aws.profile` via the `--profile` CLI flag or the `aws.profile` config key, the AWS SDK was always loading the default profile instead of the specified one.

**Reproduction steps:**
1. Set `aws.profile` to a named profile (e.g., `--profile myprofile` or in `.awstools.yaml`)
2. Run any awstools command
3. Observe that the default AWS profile credentials are used instead of `myprofile`

**Impact:** All users relying on `--profile` or `aws.profile` config to target non-default AWS profiles were silently using the wrong credentials, potentially operating against the wrong AWS account.

## Investigation Summary

- **Symptoms examined:** Profile flag accepted but AWS operations used default credentials
- **Code inspected:** `config/awsconfig.go` — `DefaultAwsConfig` function
- **Hypotheses tested:** Profile key mismatch between check and usage

## Discovered Root Cause

In `DefaultAwsConfig`, line 33 correctly checks `config.GetLCString("aws.profile")` and line 34 correctly stores the profile name, but line 35 passed `config.GetLCString("profile")` (missing the `aws.` prefix) to `WithSharedConfigProfile`. Since the `profile` key is never set, `GetLCString("profile")` always returns an empty string, making `WithSharedConfigProfile("")` a no-op.

**Defect type:** Key name typo / inconsistent config key usage

**Why it occurred:** The `aws.profile` key was partially referenced — the guard clause and assignment used the correct key, but the SDK call used a shorter key that doesn't exist in the config hierarchy.

**Contributing factors:** No unit test verified that the profile value was actually passed through to the AWS config loader.

## Resolution for the Issue

**Changes made:**
- `config/awsconfig.go:31-36` — Extracted `resolveProfile()` helper that reads `aws.profile` once, then used the returned value consistently for both the `ProfileName` assignment and `WithSharedConfigProfile` call.

**Approach rationale:** Extracting `resolveProfile()` ensures the config key is read in exactly one place, eliminating the class of bug where different call sites use different keys. It also makes the profile resolution independently testable.

**Alternatives considered:**
- Inline fix (just change `"profile"` to `"aws.profile"` on line 35) — simpler but leaves the same key duplicated in three places, risking the same class of bug recurring.

## Regression Test

**Test file:** `config/awsconfig_test.go`
**Test names:** `TestDefaultAwsConfig_ProfileKeyConsistency`, `TestDefaultAwsConfig_ProfilePassedToLoader`, `TestDefaultAwsConfig_NoProfileReturnsEmpty`

**What it verifies:**
- `GetLCString("aws.profile")` returns the configured value while `GetLCString("profile")` returns empty (proving the old code was broken)
- `resolveProfile()` returns the correct profile value when `aws.profile` is set
- `resolveProfile()` returns empty when no profile is configured

**Run command:** `go test ./config/ -run "TestDefaultAwsConfig_Profile" -v`

## Affected Files

| File | Change |
|------|--------|
| `config/awsconfig.go` | Extract `resolveProfile()`, fix profile key from `"profile"` to `"aws.profile"` |
| `config/awsconfig_test.go` | Add three regression tests for profile key consistency |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`go test ./...`)
- [x] Code formatted (`go fmt`)

## Prevention

**Recommendations to avoid similar bugs:**
- Use a single helper function when a config key is needed in multiple places (as done in this fix)
- Add unit tests that verify config values are correctly propagated, not just that code paths are exercised

## Related

- Transit ticket: T-405
