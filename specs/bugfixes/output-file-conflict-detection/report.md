# Bugfix Report: Profile Generator Ignores Output File for Conflict Detection

**Date:** 2026-03-20
**Status:** Fixed

## Description of the Issue

When `--output-file` is specified for the profile generator, three methods (`ValidateTemplateProfile`, `GenerateProfiles`, and `initializeConflictDetector`) still read the default AWS config file (via `AWS_CONFIG_FILE` env var or `~/.aws/config`) instead of the specified output file. This causes conflict detection to run against the wrong file, potentially missing conflicts that exist in the actual output file or reporting false conflicts from the default config.

**Reproduction steps:**
1. Create an output file at a custom path with existing profiles
2. Run `awstools sso profile-generator --template my-profile --output-file /path/to/custom-config`
3. Observe that conflict detection checks the default AWS config file instead of the custom output file

**Impact:** Users writing to a custom output file may silently overwrite existing profiles in that file, or receive incorrect conflict reports based on profiles in their default config that are irrelevant to the output file.

## Investigation Summary

- **Symptoms examined:** `LoadAWSConfigFile("")` called with empty string in three methods despite `pg.outputFile` being available
- **Code inspected:** `helpers/profile_generator.go` — `initializeConflictDetector()` (line 195), `ValidateTemplateProfile()` (line 215), `GenerateProfiles()` (line 280)
- **Hypotheses tested:** Confirmed that `AppendToConfig()` already correctly uses `pg.outputFile`, establishing the intended pattern

## Discovered Root Cause

**Defect type:** Hardcoded argument error

**Why it occurred:** The three methods were written to call `LoadAWSConfigFile("")` (empty string), which falls back to the default config file resolution. The `outputFile` field was added to `ProfileGenerator` but these call sites were not updated to use it.

**Contributing factors:** The `AppendToConfig` method was correctly implemented with `pg.outputFile`, but this pattern was not applied consistently to the other three methods that also need to read the same file being written to.

## Resolution for the Issue

**Changes made:**
- `helpers/profile_generator.go:195` - Changed `LoadAWSConfigFile("")` to `LoadAWSConfigFile(pg.outputFile)` in `initializeConflictDetector()`
- `helpers/profile_generator.go:215` - Changed `LoadAWSConfigFile("")` to `LoadAWSConfigFile(pg.outputFile)` in `ValidateTemplateProfile()`
- `helpers/profile_generator.go:280` - Changed `LoadAWSConfigFile("")` to `LoadAWSConfigFile(pg.outputFile)` in `GenerateProfiles()`

**Approach rationale:** When `pg.outputFile` is empty, `LoadAWSConfigFile("")` falls back to the default config file anyway, so this change is fully backward-compatible. When `pg.outputFile` is set, all methods now consistently read from the same file that `AppendToConfig` writes to.

**Alternatives considered:**
- Adding a separate source file parameter for template validation — rejected because the output file should be self-consistent (contain the template if it contains profiles)
- Caching the loaded config across methods — out of scope for this bugfix; the current approach matches the existing pattern

## Regression Test

**Test file:** `helpers/profile_generator_test.go`
**Test names:** `TestOutputFileUsedForConflictDetection`, `TestOutputFileUsedForValidateTemplateProfile`, `TestOutputFileUsedForGenerateProfiles`

**What it verifies:** Each test creates a default config and a separate output file with different content, then verifies the respective method reads from the output file (not the default config).

**Run command:** `go test ./helpers/ -run "TestOutputFileUsedFor" -v`

## Affected Files

| File | Change |
|------|--------|
| `helpers/profile_generator.go` | Pass `pg.outputFile` instead of `""` to `LoadAWSConfigFile` in three methods |
| `helpers/profile_generator_test.go` | Add three regression tests verifying output file is used |

## Verification

**Automated:**
- [ ] Regression test passes
- [ ] Full test suite passes
- [ ] Linters/validators pass

**Manual verification:**
- Confirmed `AppendToConfig` already uses `pg.outputFile` correctly, establishing the intended pattern

## Prevention

**Recommendations to avoid similar bugs:**
- When adding a field to a struct that affects file path resolution, audit all call sites that resolve that path
- Consider adding a helper method on `ProfileGenerator` that encapsulates config file loading (e.g., `pg.loadConfigFile()`) to centralize the file path logic

## Related

- Transit ticket: T-538
