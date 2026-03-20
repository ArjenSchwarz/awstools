# SSO Helpers

## Architecture

The SSO helpers (`helpers/sso.go`) interact with AWS SSO Admin APIs to build a nested data model:

```
SSOInstance
  -> []SSOPermissionSet
       -> []SSOAccount (per permission set)
            -> []SSOAccountAssignment
       -> []SSOPolicy (managed policies)
       -> InlinePolicy (string)
  -> map[string]SSOAccount (aggregated across all permission sets)
```

## Key Design Decisions

- **SSOAdminAPI interface**: Introduced in T-479 to enable mock-based testing. All SSO helper functions accept this interface rather than `*ssoadmin.Client`.
- **Error returns (not panics)**: The helpers return errors. The cmd-layer callers currently use `panic(err)` to handle these, matching the pattern elsewhere in the cmd package.
- **Manual pagination**: Uses `for { ... NextToken ... break }` loops, consistent with `helpers/role_discovery.go`.

## Pagination

Four listing APIs require pagination:
1. `ListPermissionSets` — in `getPermissionSets()`
2. `ListAccountsForProvisionedPermissionSet` — in `addAccountInfo()`
3. `ListAccountAssignments` — in `addAccountInfo()` (nested per account)
4. `ListManagedPoliciesInPermissionSet` — in `getPermissionSetDetails()`

All use `MaxResults: 100` (AWS default max page size) and follow the NextToken pattern.

## Testing

Tests use a `mockSSOAdminClient` struct with function fields (same pattern as `mockOrganizationsClient` in `organizations_test.go`). The `newBasicMock()` helper returns a mock with sensible defaults — override individual function fields to test specific scenarios.

## Callers

Four cmd files call `GetSSOAccountInstance()`:
- `cmd/ssolistpermissionsets.go`
- `cmd/ssooverviewaccount.go`
- `cmd/ssooverviewpermissionset.go`
- `cmd/ssodangling.go`
