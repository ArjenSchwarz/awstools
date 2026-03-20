# IAM Helpers

## Architecture

IAM helper functions live in three files:
- `helpers/iam.go` — User/group listing, policy retrieval, account info
- `helpers/iamroles.go` — Role listing, role policy retrieval, policy document caching
- `helpers/iamresources.go` — Type definitions (IAMUser, IAMGroup, IAMRole, etc.), interface definitions, access key checks

## IAMClient Interface

An `IAMClient` interface is defined in `helpers/iam.go` covering all IAM SDK methods used by the helpers package. Functions accept this interface rather than `*iam.Client` directly, which enables unit testing with mocks. The concrete `*iam.Client` satisfies the interface.

## Pagination

All IAM list operations use AWS SDK v2 built-in paginators (e.g., `iam.NewListUsersPaginator`). This was added because AWS IAM list APIs return at most 100 items per page by default.

Affected functions: `getUserList`, `GetPoliciesMap`, `GetUserPoliciesMapForUser`, `GetGroupPoliciesMapForGroup`, `GetAttachedPoliciesMapForUser`, `GetAttachedPoliciesMapForGroup`, `GetGroupNameSliceForUser`, `getAllUsersInGroup`, `GetGroupDetails`, `GetRoleDetails`, `getInlinePoliciesForRole`, `getAttachedPoliciesForRole`.

## Caching

- `cachedUsers` (package-level var) — caches the full user list from `getUserList()`. Must be reset to `nil` in tests.
- `cachedIAMPolicyDocuments` (package-level map) — caches IAM policy documents by name to avoid re-fetching.

## Testing

- `helpers/iam_pagination_test.go` — Contains `mockIAMClient` that implements the `IAMClient` interface with configurable page sizes. Tests verify pagination by setting `pageSize: 2` with 5 items (forces 3 pages).
- `helpers/iam_test.go` — Unit tests for IAMUser/IAMGroup methods, pure data tests (no AWS calls).
- `helpers/iamroles_test.go` — Tests for role type detection, policy names, caching.

## Call Sites

- `cmd/iamuserlist.go` — Calls `GetUserDetails`, `GetGroupDetails`, `HasAccessKeys`, `GetLastAccessKeyDate`
- `cmd/iamrolelist.go` — Calls `GetRolesAndPolicies`
- `cmd/names.go` — Calls `GetAccountAlias`
