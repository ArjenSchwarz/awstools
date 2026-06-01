# Bugfix Report: Profile generation cannot create a new AWS config file

**Ticket:** T-1010

## Summary

Generating AWS profiles to a config file that does not yet exist failed with
`failed to open file for locking`. This affected profile generation to a
brand-new `--output-file` path and first-time generation on machines without an
existing `~/.aws/config`.

## Root Cause

`ProfileGenerator.AppendToConfig` calls `LoadAWSConfigFile`, which correctly
treats a missing file as an empty starting state. It then calls
`AWSConfigFile.AppendProfiles`, which ends by calling `WriteToFile`.
`WriteToFile` writes with locking enabled (`writeContent(content, true)`), which
delegates to `withFileLock`.

`withFileLock` opened the target with `os.OpenFile(filePath, os.O_RDWR, 0600)`.
Without `os.O_CREATE`, this open fails when the file (or its parent directory)
does not exist, so the lock could never be acquired and the file was never
written. The actual write logic in `writeContentToFile` already creates the
directory and writes atomically via a temp file + rename, but it never ran
because acquiring the lock failed first.

## Fix

Updated `withFileLock` in `helpers/aws_config_file.go`:

- Create the parent directory with `os.MkdirAll(filepath.Dir(filePath), 0700)`
  before opening the lock file, so a brand-new config path works.
- Open the lock file with `os.O_RDWR|os.O_CREATE` so the file is created when it
  does not exist yet.

This preserves the existing file-locking behaviour while allowing first-time
creation. The atomic temp-file + rename write path in `writeContentToFile` is
unchanged.

## Testing

- Added regression test `TestAppendProfilesCreatesMissingFile` in
  `helpers/aws_config_file_test.go`. It loads a config from a non-existent path
  inside a non-existent directory, appends a profile via `AppendProfiles`,
  asserts no error, asserts the file exists, and reloads to confirm the profile
  is present.
- Verified the test FAILS before the fix with the exact ticket error
  (`failed to open file for locking: ... no such file or directory`) and PASSES
  after the fix.
- The pre-existing `TestWriteAndLoadRoundTrip` also exercised this path (it
  ignored the write error) and now passes correctly.
- `go test ./...` passes for all packages.

## Files Changed

- `helpers/aws_config_file.go` — `withFileLock` now creates the parent
  directory and opens the lock file with `O_CREATE`.
- `helpers/aws_config_file_test.go` — added `TestAppendProfilesCreatesMissingFile`
  regression test.

## Prevention

The existing roundtrip test silently ignored the error returned by
`WriteToFile`, which masked this bug. The new regression test explicitly checks
the returned error, so future regressions in the file-creation/locking path
will be caught.
