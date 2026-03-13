# Bugfix Report: profile-replace-duplicate-sections

**Date:** 2025-07-14
**Status:** Fixed

## Description of the Issue

When using the profile generator with conflict strategy "replace" (or auto-approve with existing conflicts), the tool appends new profile sections to `~/.aws/config` without removing the old ones. This results in duplicate `[profile ...]` blocks in the config file, which causes unpredictable behavior when AWS CLI or SDKs parse the file.

**Reproduction steps:**
1. Have an existing `~/.aws/config` with profile `[profile my-account-AdminAccess]`
2. Run profile generation with `--yes` (auto-approve) that generates a profile with the same name
3. Observe two `[profile my-account-AdminAccess]` sections in the config file

**Impact:** High — duplicate profile sections in AWS config can cause AWS CLI/SDK to use stale credentials or incorrect settings, and confuse users inspecting their config.

## Investigation Summary

Systematic inspection of the profile generation → config file write pipeline.

- **Symptoms examined:** Duplicate profile blocks in `~/.aws/config` after replace operations
- **Code inspected:** `AppendToConfig`, `AppendProfiles`, `AppendToFile`, `WriteToFile`, `ReplaceProfile`, `AddProfile`
- **Hypotheses tested:** Confirmed that `AppendToFile` uses `O_APPEND` flag and never removes existing sections

## Discovered Root Cause

`AWSConfigFile.AppendProfiles()` calls `AddProfile()` (which correctly overwrites profiles in the in-memory map) but then calls `AppendToFile()` which opens the file with `os.O_APPEND` and blindly writes new profile text at the end, leaving old sections intact.

**Defect type:** Logic error — wrong file write method used

**Why it occurred:** `AppendProfiles` was originally designed for adding new profiles only. When conflict resolution (replace strategy) was added, the same append path was reused without accounting for the need to remove existing sections.

**Contributing factors:**
- `WriteToFile()` (which truncates and rewrites) and `ReplaceProfile()` exist but are never called in the replace workflow
- The in-memory state is correct after `AddProfile` (map overwrites), masking the bug in unit tests that only check in-memory state

## Resolution for the Issue

**Changes made:**
- `helpers/aws_config_file.go:700` — Changed `AppendProfiles` to call `WriteToFile()` instead of `AppendToFile()`, since `AddProfile()` already produces the correct in-memory state and `WriteToFile` serializes the full in-memory state (truncate + rewrite).

**Approach rationale:** `WriteToFile` already handles backup creation, file locking, and writing both SSO sessions and profiles from the in-memory map. Since `AddProfile` correctly overwrites existing profiles in the map, `WriteToFile` produces a correct file with no duplicates for both new and replaced profiles.

**Alternatives considered:**
- Surgically removing individual profile sections from the file before appending — more complex and error-prone
- Adding a separate code path in `AppendToConfig` for conflicts vs new profiles — unnecessary complexity since `WriteToFile` handles both correctly

## Regression Test

**Test file:** `helpers/aws_config_file_test.go`
**Test names:** `TestAppendProfiles_ReplaceDoesNotDuplicate`, `TestAppendToConfig_ReplaceConflictsNoDuplicates`

**What it verifies:**
1. `AppendProfiles` with a profile name matching an existing one produces exactly one section in the file
2. End-to-end `AppendToConfig` with auto-approve and conflicting profiles produces no duplicates
3. Non-conflicting profiles are preserved in both cases

**Run command:** `go test ./helpers/ -run "TestAppendProfiles_ReplaceDoesNotDuplicate|TestAppendToConfig_ReplaceConflictsNoDuplicates" -v`

## Affected Files

| File | Change |
|------|--------|
| `helpers/aws_config_file.go` | Changed `AppendProfiles` to use `WriteToFile` instead of `AppendToFile` |
| `helpers/aws_config_file_test.go` | Added two regression tests for duplicate profile detection |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed duplicate profile sections in test output before fix
- Confirmed single profile section in test output after fix

## Prevention

**Recommendations to avoid similar bugs:**
- Prefer full-rewrite (WriteToFile) over append (AppendToFile) when the in-memory model is authoritative, to avoid state drift between memory and disk
- Integration tests for file I/O operations should always verify the resulting file content, not just in-memory state
- Consider deprecating `AppendToFile` since `WriteToFile` is safer for all current use cases

## Related

- Transit ticket: T-444
