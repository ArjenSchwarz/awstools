# Role Discovery

## Architecture

`RoleDiscovery` (`helpers/role_discovery.go`) discovers accessible roles using SSO OIDC tokens. It uses the SSO `ListAccounts` API to enumerate accounts and `ListAccountRoles` to find roles per account.

Key types:
- `RoleDiscovery` — orchestrates the discovery flow
- `DiscoveredRole` — result type with account ID, name, alias, and role info
- `SSOTokenCache` — handles cached SSO token loading

## Account Alias Lookup

Account aliases are populated from the SSO `ListAccounts` API response during role discovery. The `AccountName` field from `types.AccountInfo` is used as the alias and cached in `aliasCache`.

**Important:** IAM `ListAccountAliases` is account-scoped — it only returns the alias for the caller's own account. It cannot be used for cross-account alias lookup. This was the root cause of T-481.

`GetAccountAlias` is a cache-only lookup. If no alias is cached, it falls back to the account ID.

## Concurrency

`getRolesForAccount` is called concurrently for each account (goroutine per account in `DiscoverAccessibleRoles`). The `aliasCache` and `accountCache` are protected by `cacheMutex` (sync.RWMutex).

## Dependencies

- `ssoClient` — SSO API calls (ListAccounts, ListAccountRoles)
- `stsClient` — STS for token validation
- `tokenCache` — SSO token cache for loading cached OIDC tokens

IAM client was removed as a dependency in T-481 since the alias lookup now uses SSO data.

## Profile Generator Integration

`ProfileGenerator` (`helpers/profile_generator.go`) creates a `RoleDiscovery` instance in its constructor and uses it to discover roles, which are then converted to `GeneratedProfile` entries using a naming pattern (e.g., `{account_name}-{role_name}` or `{account_alias}-{role_name}`).
