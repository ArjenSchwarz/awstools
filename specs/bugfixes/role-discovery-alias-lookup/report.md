# Bugfix Report: role-discovery-alias-lookup

**Date:** 2026-03-20
**Status:** Fixed

## Description of the Issue

`RoleDiscovery.GetAccountAlias` uses IAM `ListAccountAliases` with the template profile's credentials, which always returns the alias for the currently authenticated account. When discovering roles across multiple accounts via SSO, every account receives the same alias (the template profile's account alias), mislabeling all other accounts.

**Reproduction steps:**
1. Configure an SSO template profile that authenticates against account A (alias: "shared-services")
2. Run profile generation that discovers roles across accounts A, B, and C
3. All three accounts get `AccountAlias: "shared-services"` instead of their own aliases

**Impact:** High — profile names using the `{account_alias}` naming pattern are incorrect for all non-template accounts, making generated profiles confusing and potentially causing users to assume roles in the wrong account.

## Investigation Summary

- **Symptoms examined:** All discovered accounts receive the same alias regardless of account ID
- **Code inspected:** `GetAccountAlias`, `getRolesForAccount`, `NewRoleDiscovery`
- **Root cause identified via:** Code inspection of IAM `ListAccountAliases` API semantics

## Discovered Root Cause

`GetAccountAlias(accountID string)` accepts an account ID parameter but ignores it when calling `rd.iamClient.ListAccountAliases()`. The IAM API always returns the alias for the account that the IAM client is authenticated against (the template profile's account), not the account specified by `accountID`. The result is then cached under the requested account ID, so every account ends up with the same alias.

**Defect type:** Logic error — wrong API used for cross-account data

**Why it occurred:** IAM `ListAccountAliases` is an account-scoped API that only works for the caller's own account. It cannot retrieve aliases for other accounts. The method signature suggests per-account lookup via the `accountID` parameter, but the IAM call ignores that parameter entirely.

**Contributing factors:**
- IAM `ListAccountAliases` does not accept an account ID parameter (it always operates on the caller's account)
- The SSO `ListAccounts` API already provides `AccountName` for each account, making the IAM call unnecessary

## Resolution for the Issue

**Changes made:**
- `helpers/role_discovery.go` — Removed IAM dependency from `RoleDiscovery`. `GetAccountAlias` now reads from the alias cache only, falling back to account ID. `getRolesForAccount` populates the alias cache from the SSO-provided `AccountName` for each account.
- `helpers/profile_generator.go` — Removed IAM client creation and field, since `RoleDiscovery` no longer needs it.
- `helpers/profile_generator_test.go` — Removed unused `MockIAMClient` type and IAM import.

**Approach rationale:** The SSO `ListAccounts` API already provides per-account names during role discovery. Using this data as the alias is correct, avoids cross-account IAM calls, and requires no additional API permissions.

**Alternatives considered:**
- Assuming a role in each target account and calling IAM `ListAccountAliases` per-account — correct but adds latency, complexity, and requires assume-role permissions
- Using Organizations `DescribeAccount` — requires Organizations API access which not all users have

## Regression Test

**Test file:** `helpers/profile_generator_test.go`
**Test names:** `TestGetAccountAlias_UsesSSO_NotIAM`, `TestGetAccountAlias_CrossAccountNotMislabeled`

**What it verifies:**
1. `GetAccountAlias` returns distinct per-account aliases from the SSO-populated cache
2. Uncached accounts fall back to account ID instead of calling IAM
3. Multiple accounts do not share the same alias

**Run command:** `go test ./helpers/ -run "TestGetAccountAlias_UsesSSO_NotIAM|TestGetAccountAlias_CrossAccountNotMislabeled" -v`

## Affected Files

| File | Change |
|------|--------|
| `helpers/role_discovery.go` | Removed IAM client dependency; `GetAccountAlias` uses cache only; `getRolesForAccount` populates alias cache from SSO data |
| `helpers/profile_generator.go` | Removed IAM client creation and struct field |
| `helpers/profile_generator_test.go` | Added 2 regression tests; removed unused `MockIAMClient` |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- When using AWS APIs, verify whether the API is account-scoped or cross-account capable
- Prefer data already available from the discovery flow (SSO `AccountInfo`) over additional API calls

## Related

- Transit ticket: T-481
