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
- `cachedIAMPolicyDocuments` (package-level map) — caches role policy documents. Inline entries use `<role>/<policy>`; attached entries use the policy ARN, because attached policy names are not unique (T-1656). `IAMRole.AttachedPolicies` and the attached entries in `GetRolesAndPolicies` output are also keyed by ARN — use `IAMPolicyDocument.Name` for display.

## Testing

- `helpers/iam_pagination_test.go` — Contains `mockIAMClient` that implements the `IAMClient` interface with configurable page sizes. Tests verify pagination by setting `pageSize: 2` with 5 items (forces 3 pages).
- `helpers/iam_test.go` — Unit tests for IAMUser/IAMGroup methods, pure data tests (no AWS calls).
- `helpers/iamroles_test.go` — Tests for role type detection, policy names, caching.

## Call Sites

- `cmd/iamuserlist.go` — Calls `GetUserDetails`, `GetGroupDetails`, `HasAccessKeys`, `GetLastAccessKeyDate`
- `cmd/iamrolelist.go` — Calls `GetRolesAndPolicies`
- `cmd/names.go` — Calls `GetAccountAlias`

## Trust Policy Action Parsing (T-818)

`IAMPolicyDocumentStatement.Action` is typed `any` because AWS policy
documents allow either a string or an array. After JSON unmarshal the runtime
value is `string` or `[]any`. `CanBeAssumedFrom` delegates to two helpers in
`helpers/iamresources.go`:

- `normalizeActions(action any) []string` — converts `string`, `[]string`, or
  `[]any` into a flat string slice (any unknown type yields `nil`).
- `statementAllowsAssumeRole(statement) bool` — requires
  `Effect == "Allow"` (case-insensitive) and matches normalised actions
  case-insensitively against `assumeRoleActions`
  (`sts:AssumeRole`, `sts:AssumeRoleWithSAML`).

When adding new trust actions (e.g. `sts:AssumeRoleWithWebIdentity`) extend
`assumeRoleActions`. Deny statements are always excluded from the returned
allowances.

## rolelist graph output (T-1656)

`cmd/iamrolelist.go:writeIamrolelist` links roles to policies by policy ID (ARN for attached, `<name> (inline for <role>)` for inline) in graph formats. go-output v1.4.0 quirks it works around:

- DOT/Mermaid use the From column (`Name`) as node identity and label, so attached policy nodes are labelled with the ARN. Draw.io connects `Policies` to `DrawioID`, so labels stay friendly.
- A row missing the To column renders as `%!s(<nil>)` and creates an edge to a missing node (panics in Mermaid), so policy rows set `Policies` to `""`.
- Mermaid joins `[]string` with `", "` but splits on `","`, so the role link column is passed pre-joined with `","`.
- Not worked around: Mermaid `AddBasicNode` reuses node IDs (`len(map)+1`) when a source node has several edges, so roles with multiple policies can share an ID. Tests use one policy per role for Mermaid.
