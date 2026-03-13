# Bugfix Report: output-file-path-lowercase

**Date:** 2026-03-13
**Status:** Fixed

## Description of the Issue

`NewOutputSettings` in `config/config.go` used `GetLCString("output.file")` to read the output file path, which lowercases the entire string. On case-sensitive filesystems (Linux, some macOS configurations), this causes output to be written to the wrong path or fail entirely when the intended path contains uppercase characters.

**Reproduction steps:**
1. Set `output.file` to a path with uppercase characters (e.g., `/tmp/MyProject/Output.json`)
2. Run any awstools command
3. Observe that the output file is written to `/tmp/myproject/output.json` instead

**Impact:** Medium — any user on a case-sensitive filesystem whose output path contains uppercase characters will get incorrect behaviour.

## Investigation Summary

The bug is a straightforward misuse of `GetLCString` where `GetString` should have been used.

- **Symptoms examined:** Output file path being lowercased
- **Code inspected:** `config/config.go` — `NewOutputSettings()`, `GetLCString()`, `GetString()`
- **Hypotheses tested:** Only one hypothesis needed — `GetLCString` lowercases its return value by design (confirmed at line 17: `strings.ToLower`)

## Discovered Root Cause

`NewOutputSettings()` at line 91 calls `config.GetLCString("output.file")` which applies `strings.ToLower()` to the file path. `GetLCString` is appropriate for settings like output format (where "JSON" and "json" should be equivalent) but not for file paths which are opaque strings that must preserve case.

**Defect type:** Incorrect API usage

**Why it occurred:** `GetLCString` was likely used for consistency with the adjacent `output.format` call, without considering that file paths are case-sensitive.

**Contributing factors:** The existing test used an all-lowercase path (`/tmp/output.json`), so the lowercasing had no visible effect.

## Resolution for the Issue

**Changes made:**
- `config/config.go:91` — Changed `config.GetLCString("output.file")` to `config.GetString("output.file")`

**Approach rationale:** Direct fix — use the case-preserving getter for file paths.

**Alternatives considered:**
- None needed — the fix is unambiguous.

## Regression Test

**Test file:** `config/config_test.go`
**Test name:** `TestConfig_NewOutputSettings/preserves_case_of_output_file_path`

**What it verifies:** Setting `output.file` to a mixed-case path and asserting that `NewOutputSettings().OutputFile` preserves the original case.

**Run command:** `go test ./config/ -run "TestConfig_NewOutputSettings/preserves_case"`

## Affected Files

| File | Change |
|------|--------|
| `config/config.go` | Use `GetString` instead of `GetLCString` for output file path |
| `config/config_test.go` | Add regression test with mixed-case file path |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- Reserve `GetLCString` for enum-like settings (formats, modes) — never for paths, names, or opaque strings.
- Use mixed-case values in test fixtures to catch unintended normalisation.

## Related

- Transit ticket: T-406
