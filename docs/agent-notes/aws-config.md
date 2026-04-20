# AWS Config

## Overview

`config/awsconfig.go` holds the `AWSConfig` struct and service-client factories. `DefaultAwsConfig` loads the SDK config, applies profile/region overrides, then calls `setCallerInfo` (STS GetCallerIdentity) and `setAlias` (IAM ListAccountAliases).

## Caller Identity (STS)

`setCallerInfo` populates `AccountID`, `UserID`, and `Arn` on `AWSConfig`.

**Important:** `sts.GetCallerIdentityOutput.Account`, `.Arn`, and `.UserId` are all `*string`. In some edge cases (notably SSO sessions in specific states) they can be nil. Always use the `resolveCallerIdentity` helper (or `aws.ToString`) — never dereference directly. This was the root cause of T-734.

The same pattern applies to `helpers/sts.go:GetAccountID`, which uses `accountIDFromIdentity` to safely extract the account ID.

## Account Alias

`setAlias` uses `iam.ListAccountAliases` which is account-scoped (only returns the caller's own alias). If the call fails or returns no aliases, `AccountAlias` falls back to `AccountID`. For cross-account alias lookup see `docs/agent-notes/role-discovery.md` (uses SSO `ListAccounts` instead).

## Failure Modes

- Invalid profile or missing credentials → `DefaultAwsConfig` panics (caught by CLI). Tests recover from this panic explicitly.
- STS call failure → `setCallerInfo` panics. Not graceful — consider error propagation if this ever becomes a common failure mode.
- Partial STS response (nil fields) → handled via `resolveCallerIdentity`; identity fields become empty strings, no panic.
