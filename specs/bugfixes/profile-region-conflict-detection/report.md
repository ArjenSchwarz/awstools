# Bugfix Report: profile-region-conflict-detection

**Date:** 2026-11-04
**Status:** Fixed
**Ticket:** T-1428

## Description of the Issue

With a naming pattern containing `{region}` (e.g. `{account_name}-{region}-{role_name}`), `awstools sso profile-generator` skipped conflict detection for every role. Roles that already had a profile were treated as new, so `--skip-existing`, `--replace-existing` and the interactive prompt were bypassed and a second profile could be created for an existing role.

**Impact:** Medium. Incorrect conflict handling and duplicate profiles for the same role; no hard failure and no automatic data loss.

## Discovered Root Cause

`ProfileConflictDetector.AnalyzeRole` called `GenerateProfileName` with an empty region, so any pattern with `{region}` failed with "missing value for placeholder". `DetectConflicts` logged that error as a warning and continued, returning an empty conflict list and nil error. The workflow then generated profiles for all roles using the template region, which succeeds.

**Defect type:** Missing input plus error swallowed as success.

## Resolution for the Issue

- `helpers/profile_conflict_detector.go` — added a `region` field and `SetRegion`; `AnalyzeRole` uses it for `{region}`. `DetectConflicts` now returns the first analysis error instead of skipping the role.
- `helpers/profile_generator.go` — `initializeConflictDetector` sets the detector region from the template profile in the loaded config, the same source generation uses.

**Alternatives considered:** Excluding failed roles from generation while continuing. Rejected: it hides the failure and still produces partial output; failing the run is simpler and safer.

## Regression Test

**Test file:** `helpers/profile_region_conflict_test.go`

- `TestDetectProfileConflicts_RegionPatternDetectsSameRole` — a region-based pattern detects an existing same-role profile and the role is not passed to non-conflicted generation.
- `TestResolveConflicts_RegionPattern` — skip keeps the existing profile; replace produces the region-based name and targets the existing profile.
- `TestDetectConflicts_AnalysisFailureIsReturned` — an analysis failure is returned as an error rather than an empty result.

All three fail on `main` (c539e60) and pass with the fix. Tests use an explicit output file and a temporary `AWS_CONFIG_FILE`, so no real AWS config is read.

**Run command:** `go test ./helpers/ -run 'RegionPattern|AnalysisFailure' -v`
