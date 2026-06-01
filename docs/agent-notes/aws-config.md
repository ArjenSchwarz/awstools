# AWS Config

## Overview

`config/awsconfig.go` holds the `AWSConfig` struct and service-client factories. `DefaultAwsConfig` loads the SDK config, applies profile/region overrides, then calls `setCallerInfo` (STS GetCallerIdentity) and `setAlias` (IAM ListAccountAliases).

## Caller Identity (STS)

`setCallerInfo` populates `AccountID`, `UserID`, and `Arn` on `AWSConfig`.

**Important:** `sts.GetCallerIdentityOutput.Account`, `.Arn`, and `.UserId` are all `*string`. In some edge cases (notably SSO sessions in specific states) they can be nil. Always use the `resolveCallerIdentity` helper (or `aws.ToString`) — never dereference directly. This was the root cause of T-734.

The same pattern applies to `helpers/sts.go:GetAccountID`, which uses `accountIDFromIdentity` to safely extract the account ID.

## Account Alias

`setAlias` uses `iam.ListAccountAliases` which is account-scoped (only returns the caller's own alias). If the call fails or returns no aliases, `AccountAlias` falls back to `AccountID`. For cross-account alias lookup see `docs/agent-notes/role-discovery.md` (uses SSO `ListAccounts` instead).

## Profile Name Case Sensitivity

AWS profile names are case-sensitive — they are matched verbatim against `[profile ...]` section headers in `~/.aws/config`. `resolveProfile` must read `aws.profile` via `Config.GetString`, never `Config.GetLCString`. `GetLCString` lowercases the value, which silently breaks mixed-case profile names (T-848). Same rule applies to any other case-sensitive identifier stored in viper (ARNs, resource IDs, file paths).

## Failure Modes

- Invalid profile or missing credentials → `DefaultAwsConfig` panics (caught by CLI). Tests recover from this panic explicitly.
- STS call failure → `setCallerInfo` panics. Not graceful — consider error propagation if this ever becomes a common failure mode.
- Partial STS response (nil fields) → handled via `resolveCallerIdentity`; identity fields become empty strings, no panic.

## AWS Config File Parser

`helpers/aws_config_file.go:parseConfigFileWithRecovery` reads `~/.aws/config` with `bufio.Scanner`. The default token size (64 KiB) is too small for configs with long `credential_process` commands or other oversized custom properties, so the parser calls `scanner.Buffer` with a 1 MiB cap (`maxConfigLineSize`). Without this, a single long line causes `bufio.ErrTooLong` and aborts the whole parse, bypassing the partial-recovery logic. This was the fix for T-867.

## Appending Profiles (AppendToFile)

`AppendToFile` opens the config with `os.O_APPEND|os.O_CREATE|os.O_RDWR` and writes each generated profile's `ToConfigString()`. Each `ToConfigString()` block already ends in `\n\n`, so successive appended profiles separate correctly — but the boundary between pre-existing content and the first appended profile is not guaranteed. Before the append loop it now `Stat`s the file and, if non-empty with a final byte that is not `\n`, writes one separator newline. Without this, a user config ending on a property line with no trailing newline would have the next `[profile ...]` header concatenated onto it, corrupting both entries (T-1314). The fd is `O_RDWR` (not `O_WRONLY`) specifically so the trailing byte can be read via `ReadAt`; `O_APPEND` still forces all writes to the end.
