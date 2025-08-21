package helpers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/sso/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// RoleDiscovery handles discovery of accessible roles using OIDC tokens
type RoleDiscovery struct {
	ssoClient    *sso.Client
	stsClient    *sts.Client
	iamClient    *iam.Client
	tokenCache   *SSOTokenCache
	logger       Logger
	accountCache map[string]string
	aliasCache   map[string]string
	cacheMutex   sync.RWMutex
}

// Logger interface for logging operations
type Logger interface {
	Printf(format string, args ...interface{})
}

// defaultLogger provides a default logger implementation
type defaultLogger struct{}

func (dl *defaultLogger) Printf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

// NewRoleDiscovery creates a new role discovery instance using OIDC tokens
func NewRoleDiscovery(ssoClient *sso.Client, stsClient *sts.Client, iamClient *iam.Client) (*RoleDiscovery, error) {
	if ssoClient == nil {
		return nil, NewValidationError("SSO client cannot be nil", nil)
	}
	if stsClient == nil {
		return nil, NewValidationError("STS client cannot be nil", nil)
	}
	if iamClient == nil {
		return nil, NewValidationError("IAM client cannot be nil", nil)
	}

	tokenCache, err := NewSSOTokenCache()
	if err != nil {
		return nil, err
	}

	rd := &RoleDiscovery{
		ssoClient:    ssoClient,
		stsClient:    stsClient,
		iamClient:    iamClient,
		tokenCache:   tokenCache,
		logger:       &defaultLogger{},
		accountCache: make(map[string]string),
		aliasCache:   make(map[string]string),
	}

	return rd, nil
}

// SetLogger sets a custom logger
func (rd *RoleDiscovery) SetLogger(logger Logger) {
	rd.logger = logger
	rd.tokenCache.SetLogger(logger)
}

// DiscoverAccessibleRoles discovers all accessible roles using OIDC token approach
func (rd *RoleDiscovery) DiscoverAccessibleRoles(templateProfile *TemplateProfile) ([]DiscoveredRole, error) {
	ctx := context.TODO()

	// Load cached token for the template profile
	cachedToken, err := rd.tokenCache.LoadTokenForProfile(templateProfile.SSOStartURL, templateProfile.SSORegion)
	if err != nil {
		return nil, err
	}

	// Get accounts accessible with this token
	accounts, err := rd.getAccountsFromToken(ctx, cachedToken)
	if err != nil {
		return nil, err
	}

	if len(accounts) == 0 {
		return []DiscoveredRole{}, nil
	}

	// Discover roles for each account
	var allRoles []DiscoveredRole
	var rolesMutex sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(accounts))

	// Process accounts concurrently
	for _, account := range accounts {
		wg.Add(1)
		go func(acc types.AccountInfo) {
			defer wg.Done()

			roles, err := rd.getRolesForAccount(ctx, cachedToken, acc)
			if err != nil {
				errChan <- err
				return
			}

			rolesMutex.Lock()
			allRoles = append(allRoles, roles...)
			rolesMutex.Unlock()
		}(account)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	rd.logger.Printf("Discovered %d accessible roles across %d accounts", len(allRoles), len(accounts))
	return allRoles, nil
}

// getAccountsFromToken retrieves accounts accessible with the given token
func (rd *RoleDiscovery) getAccountsFromToken(ctx context.Context, token *CachedToken) ([]types.AccountInfo, error) {
	var accounts []types.AccountInfo
	maxResults := int32(100)
	var nextToken *string

	for {
		input := &sso.ListAccountsInput{
			AccessToken: &token.AccessToken,
			MaxResults:  &maxResults,
			NextToken:   nextToken,
		}

		result, err := rd.ssoClient.ListAccounts(ctx, input)
		if err != nil {
			return nil, NewAPIError("failed to list accounts", err).
				WithContext("start_url", token.StartURL).
				WithContext("suggestion", "Run 'aws sso login' to refresh authentication")
		}

		accounts = append(accounts, result.AccountList...)

		nextToken = result.NextToken
		if nextToken == nil {
			break
		}
	}

	return accounts, nil
}

// getRolesForAccount retrieves roles for a specific account
func (rd *RoleDiscovery) getRolesForAccount(ctx context.Context, token *CachedToken, account types.AccountInfo) ([]DiscoveredRole, error) {
	var roles []DiscoveredRole
	maxResults := int32(100)
	var nextToken *string

	for {
		input := &sso.ListAccountRolesInput{
			AccessToken: &token.AccessToken,
			AccountId:   account.AccountId,
			MaxResults:  &maxResults,
			NextToken:   nextToken,
		}

		result, err := rd.ssoClient.ListAccountRoles(ctx, input)
		if err != nil {
			return nil, NewAPIError("failed to list account roles", err).
				WithContext("account_id", *account.AccountId).
				WithContext("account_name", *account.AccountName).
				WithContext("suggestion", "Run 'aws sso login' to refresh authentication")
		}

		// Create discovered roles from the response
		for _, roleInfo := range result.RoleList {
			accountName := *account.AccountName
			if accountName == "" {
				accountName = *account.AccountId // Fallback to account ID
			}

			// Get account alias with fallback to account ID
			accountAlias, err := rd.GetAccountAlias(*account.AccountId)
			if err != nil {
				rd.logger.Printf("Warning: failed to get account alias for %s: %v", *account.AccountId, err)
				accountAlias = *account.AccountId // Fallback to account ID
			}

			role := DiscoveredRole{
				AccountID:         *account.AccountId,
				AccountName:       accountName,
				AccountAlias:      accountAlias,
				PermissionSetName: *roleInfo.RoleName,
				RoleName:          *roleInfo.RoleName,
			}

			if err := role.Validate(); err != nil {
				return nil, NewValidationError("invalid discovered role", err).
					WithContext("account_id", *account.AccountId).
					WithContext("role_name", *roleInfo.RoleName)
			}

			roles = append(roles, role)
		}

		nextToken = result.NextToken
		if nextToken == nil {
			break
		}
	}

	return roles, nil
}

// GetAccountInfo retrieves account information, using cache when available
func (rd *RoleDiscovery) GetAccountInfo(accountID string) (string, error) {
	// Check cache first
	rd.cacheMutex.RLock()
	if accountName, exists := rd.accountCache[accountID]; exists {
		rd.cacheMutex.RUnlock()
		return accountName, nil
	}
	rd.cacheMutex.RUnlock()

	// For OIDC token approach, we get account names from the SSO API response
	// This method is kept for compatibility but may not be as useful
	rd.cacheMutex.Lock()
	rd.accountCache[accountID] = accountID // Use account ID as fallback
	rd.cacheMutex.Unlock()

	return accountID, nil
}

// GetAccountAlias retrieves the account alias for a given account ID
func (rd *RoleDiscovery) GetAccountAlias(accountID string) (string, error) {
	// Check cache first
	rd.cacheMutex.RLock()
	if alias, exists := rd.aliasCache[accountID]; exists {
		rd.cacheMutex.RUnlock()
		return alias, nil
	}
	rd.cacheMutex.RUnlock()

	ctx := context.TODO()

	// Get account aliases from IAM
	input := &iam.ListAccountAliasesInput{}
	result, err := rd.iamClient.ListAccountAliases(ctx, input)
	if err != nil {
		// Cache the failure and return account ID as fallback
		rd.cacheMutex.Lock()
		rd.aliasCache[accountID] = accountID
		rd.cacheMutex.Unlock()

		return accountID, NewAPIError("failed to retrieve account alias", err).
			WithContext("account_id", accountID).
			WithContext("suggestion", "Ensure IAM permissions for ListAccountAliases")
	}

	// AWS accounts can have at most one alias
	var alias string
	if len(result.AccountAliases) > 0 {
		alias = result.AccountAliases[0]
	} else {
		// No alias configured, use account ID
		alias = accountID
	}

	// Cache the result
	rd.cacheMutex.Lock()
	rd.aliasCache[accountID] = alias
	rd.cacheMutex.Unlock()

	return alias, nil
}

// ValidateTokenAccess validates that we can access SSO with the given profile
func (rd *RoleDiscovery) ValidateTokenAccess(templateProfile *TemplateProfile) error {
	return rd.tokenCache.ValidateProfileToken(templateProfile)
}

// DiscoverRolesWithRetry discovers roles with exponential backoff retry
func (rd *RoleDiscovery) DiscoverRolesWithRetry(templateProfile *TemplateProfile, maxRetries int) ([]DiscoveredRole, error) {
	var lastErr error
	baseDelay := 1 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		roles, err := rd.DiscoverAccessibleRoles(templateProfile)
		if err == nil {
			return roles, nil
		}

		lastErr = err

		// Check if this is a retryable error
		if !rd.isRetryableError(err) {
			break
		}

		if attempt < maxRetries-1 {
			delay := baseDelay * time.Duration(1<<attempt) // Exponential backoff
			rd.logger.Printf("Attempt %d failed, retrying in %v: %v", attempt+1, delay, err)
			time.Sleep(delay)
		}
	}

	return nil, NewAPIError("failed to discover roles after retries", lastErr).
		WithContext("max_retries", maxRetries)
}

// isRetryableError determines if an error is retryable
func (rd *RoleDiscovery) isRetryableError(err error) bool {
	// Check for specific error types that are retryable
	if apiErr, ok := err.(ProfileGeneratorError); ok {
		switch apiErr.Type {
		case ErrorTypeNetwork:
			return true
		case ErrorTypeAPI:
			// Check if it's a throttling error (this is a simplified check)
			if apiErr.Cause != nil {
				errStr := apiErr.Cause.Error()
				return containsAny(errStr, []string{"throttling", "rate limit", "too many requests"})
			}
			return false
		}
	}
	return false
}

// containsAny checks if a string contains any of the provided substrings
func containsAny(s string, substrings []string) bool {
	for _, substring := range substrings {
		if len(s) >= len(substring) {
			for i := 0; i <= len(s)-len(substring); i++ {
				if s[i:i+len(substring)] == substring {
					return true
				}
			}
		}
	}
	return false
}

// GetCachedAccountNames returns a copy of the cached account names
func (rd *RoleDiscovery) GetCachedAccountNames() map[string]string {
	rd.cacheMutex.RLock()
	defer rd.cacheMutex.RUnlock()

	cache := make(map[string]string)
	for k, v := range rd.accountCache {
		cache[k] = v
	}
	return cache
}

// ClearAccountCache clears the account name cache
func (rd *RoleDiscovery) ClearAccountCache() {
	rd.cacheMutex.Lock()
	defer rd.cacheMutex.Unlock()
	rd.accountCache = make(map[string]string)
}

// ClearAliasCache clears the account alias cache
func (rd *RoleDiscovery) ClearAliasCache() {
	rd.cacheMutex.Lock()
	defer rd.cacheMutex.Unlock()
	rd.aliasCache = make(map[string]string)
}

// GetCachedAccountAliases returns a copy of the cached account aliases
func (rd *RoleDiscovery) GetCachedAccountAliases() map[string]string {
	rd.cacheMutex.RLock()
	defer rd.cacheMutex.RUnlock()

	cache := make(map[string]string)
	for k, v := range rd.aliasCache {
		cache[k] = v
	}
	return cache
}

// GetTokenCacheInfo returns information about the SSO token cache
func (rd *RoleDiscovery) GetTokenCacheInfo() (map[string]interface{}, error) {
	return rd.tokenCache.GetCacheInfo()
}

// CleanExpiredTokens removes expired tokens from the cache
func (rd *RoleDiscovery) CleanExpiredTokens() (int, error) {
	return rd.tokenCache.CleanExpiredTokens()
}

// GetAuthGuideMessage returns a helpful message for authentication issues
func (rd *RoleDiscovery) GetAuthGuideMessage(startURL, region string) string {
	return rd.tokenCache.GetAuthGuideMessage(startURL, region)
}

// TestConnection tests the connection to SSO with the given profile
func (rd *RoleDiscovery) TestConnection(templateProfile *TemplateProfile) error {
	ctx := context.TODO()

	// Load cached token
	cachedToken, err := rd.tokenCache.LoadTokenForProfile(templateProfile.SSOStartURL, templateProfile.SSORegion)
	if err != nil {
		return err
	}

	// Try to list accounts as a connection test
	input := &sso.ListAccountsInput{
		AccessToken: &cachedToken.AccessToken,
		MaxResults:  aws.Int32(1), // Only need one account to test
	}

	_, err = rd.ssoClient.ListAccounts(ctx, input)
	if err != nil {
		return NewAPIError("failed to connect to SSO", err).
			WithContext("start_url", templateProfile.SSOStartURL).
			WithContext("suggestion", "Run 'aws sso login' to refresh authentication")
	}

	return nil
}
