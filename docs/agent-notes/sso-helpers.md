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
- **Pagination mix**: The four `List*` calls inside permission-set traversal use manual `for { ... NextToken ... break }` loops for backward compatibility with T-479. `ListInstances` uses the SDK's `ssoadmin.NewListInstancesPaginator` (added in T-891) — the SSO SDK's `ListInstancesAPIClient` interface is satisfied by `SSOAdminAPI` because it only requires a single `ListInstances` method with the standard signature.

## Pagination

Five listing APIs require pagination:
1. `ListInstances` — in `getSSOInstance()`, via `ssoadmin.NewListInstancesPaginator`
2. `ListPermissionSets` — in `getPermissionSets()`
3. `ListAccountsForProvisionedPermissionSet` — in `addAccountInfo()`
4. `ListAccountAssignments` — in `addAccountInfo()` (nested per account)
5. `ListManagedPoliciesInPermissionSet` — in `getPermissionSetDetails()`

Calls 2-5 use `MaxResults: 100` (AWS default max page size) and manual NextToken loops. `ListInstances` relies on the SDK paginator's defaults because the call has no tunable parameters.

## Testing

Tests use a `mockSSOAdminClient` struct with function fields (same pattern as `mockOrganizationsClient` in `organizations_test.go`). The `newBasicMock()` helper returns a mock with sensible defaults — override individual function fields to test specific scenarios.

## Callers

Four cmd files call `GetSSOAccountInstance()`:
- `cmd/ssolistpermissionsets.go`
- `cmd/ssooverviewaccount.go`
- `cmd/ssooverviewpermissionset.go`
- `cmd/ssodangling.go`
