package helpers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin/types"
)

// RoleDiscovery handles discovery of accessible roles in AWS SSO
type RoleDiscovery struct {
	ssoClient       *ssoadmin.Client
	orgsClient      *organizations.Client
	instanceArn     string
	identityStoreID string
	logger          Logger
	accountCache    map[string]string
	cacheMutex      sync.RWMutex
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

// NewRoleDiscovery creates a new role discovery instance
func NewRoleDiscovery(ssoClient *ssoadmin.Client, orgsClient *organizations.Client) (*RoleDiscovery, error) {
	if ssoClient == nil {
		return nil, NewValidationError("SSO client cannot be nil", nil)
	}

	rd := &RoleDiscovery{
		ssoClient:    ssoClient,
		orgsClient:   orgsClient,
		logger:       &defaultLogger{},
		accountCache: make(map[string]string),
	}

	// Get SSO instance information
	if err := rd.initializeSSO(); err != nil {
		return nil, err
	}

	return rd, nil
}

// SetLogger sets a custom logger
func (rd *RoleDiscovery) SetLogger(logger Logger) {
	rd.logger = logger
}

// initializeSSO initializes the SSO instance information
func (rd *RoleDiscovery) initializeSSO() error {
	ctx := context.TODO()

	instances, err := rd.ssoClient.ListInstances(ctx, &ssoadmin.ListInstancesInput{})
	if err != nil {
		return NewAPIError("failed to list SSO instances", err)
	}

	if len(instances.Instances) == 0 {
		return NewAPIError("no SSO instances found", nil)
	}

	if len(instances.Instances) > 1 {
		return NewAPIError("multiple SSO instances found, cannot determine which to use", nil)
	}

	instance := instances.Instances[0]
	rd.instanceArn = *instance.InstanceArn
	rd.identityStoreID = *instance.IdentityStoreId

	return nil
}

// DiscoverAccessibleRoles discovers all accessible roles in the SSO instance
func (rd *RoleDiscovery) DiscoverAccessibleRoles() ([]DiscoveredRole, error) {
	ctx := context.TODO()

	// Get all permission sets
	permissionSets, err := rd.getPermissionSets(ctx)
	if err != nil {
		return nil, err
	}

	if len(permissionSets) == 0 {
		return []DiscoveredRole{}, nil
	}

	// Discover roles for each permission set
	var allRoles []DiscoveredRole
	var rolesMutex sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(permissionSets))

	// Process permission sets concurrently
	for _, permissionSet := range permissionSets {
		wg.Add(1)
		go func(ps types.PermissionSet) {
			defer wg.Done()

			roles, err := rd.discoverRolesForPermissionSet(ctx, ps)
			if err != nil {
				errChan <- err
				return
			}

			rolesMutex.Lock()
			allRoles = append(allRoles, roles...)
			rolesMutex.Unlock()
		}(permissionSet)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	rd.logger.Printf("Discovered %d accessible roles across %d permission sets", len(allRoles), len(permissionSets))
	return allRoles, nil
}

// getPermissionSets retrieves all permission sets from the SSO instance
func (rd *RoleDiscovery) getPermissionSets(ctx context.Context) ([]types.PermissionSet, error) {
	var permissionSets []types.PermissionSet
	maxResults := int32(100)
	var nextToken *string

	for {
		input := &ssoadmin.ListPermissionSetsInput{
			InstanceArn: &rd.instanceArn,
			MaxResults:  &maxResults,
			NextToken:   nextToken,
		}

		result, err := rd.ssoClient.ListPermissionSets(ctx, input)
		if err != nil {
			return nil, NewAPIError("failed to list permission sets", err)
		}

		// Get details for each permission set
		for _, permissionSetArn := range result.PermissionSets {
			psDetails, err := rd.ssoClient.DescribePermissionSet(ctx, &ssoadmin.DescribePermissionSetInput{
				InstanceArn:      &rd.instanceArn,
				PermissionSetArn: &permissionSetArn,
			})
			if err != nil {
				return nil, NewAPIError("failed to describe permission set", err).
					WithContext("permission_set_arn", permissionSetArn)
			}

			permissionSets = append(permissionSets, *psDetails.PermissionSet)
		}

		nextToken = result.NextToken
		if nextToken == nil {
			break
		}
	}

	return permissionSets, nil
}

// discoverRolesForPermissionSet discovers roles for a specific permission set
func (rd *RoleDiscovery) discoverRolesForPermissionSet(ctx context.Context, permissionSet types.PermissionSet) ([]DiscoveredRole, error) {
	var roles []DiscoveredRole
	maxResults := int32(100)
	var nextToken *string

	for {
		input := &ssoadmin.ListAccountsForProvisionedPermissionSetInput{
			InstanceArn:      &rd.instanceArn,
			PermissionSetArn: permissionSet.PermissionSetArn,
			MaxResults:       &maxResults,
			NextToken:        nextToken,
		}

		result, err := rd.ssoClient.ListAccountsForProvisionedPermissionSet(ctx, input)
		if err != nil {
			return nil, NewAPIError("failed to list accounts for permission set", err).
				WithContext("permission_set_arn", *permissionSet.PermissionSetArn).
				WithContext("permission_set_name", *permissionSet.Name)
		}

		// Create discovered role for each account
		for _, accountID := range result.AccountIds {
			accountName, err := rd.GetAccountInfo(accountID)
			if err != nil {
				// Log warning but continue - account name is not critical
				rd.logger.Printf("Warning: failed to get account name for %s: %v", accountID, err)
				accountName = accountID // Use account ID as fallback
			}

			role := DiscoveredRole{
				AccountID:         accountID,
				AccountName:       accountName,
				PermissionSetName: *permissionSet.Name,
				PermissionSetArn:  *permissionSet.PermissionSetArn,
			}

			if err := role.Validate(); err != nil {
				return nil, NewValidationError("invalid discovered role", err).
					WithContext("account_id", accountID).
					WithContext("permission_set_name", *permissionSet.Name)
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

	// Get account info from Organizations API if available
	if rd.orgsClient != nil {
		accountName, err := rd.getAccountNameFromOrgs(accountID)
		if err == nil {
			// Cache the result
			rd.cacheMutex.Lock()
			rd.accountCache[accountID] = accountName
			rd.cacheMutex.Unlock()
			return accountName, nil
		}
		// Log warning but continue - we'll use account ID as fallback
		rd.logger.Printf("Warning: failed to get account name from Organizations API for %s: %v", accountID, err)
	}

	// Try to get account info from SSO (less reliable but worth trying)
	accountName, err := rd.getAccountNameFromSSO(accountID)
	if err == nil {
		// Cache the result
		rd.cacheMutex.Lock()
		rd.accountCache[accountID] = accountName
		rd.cacheMutex.Unlock()
		return accountName, nil
	}

	// Return account ID as fallback
	rd.cacheMutex.Lock()
	rd.accountCache[accountID] = accountID
	rd.cacheMutex.Unlock()

	return accountID, nil
}

// getAccountNameFromOrgs gets account name from Organizations API
func (rd *RoleDiscovery) getAccountNameFromOrgs(accountID string) (string, error) {
	ctx := context.TODO()

	input := &organizations.DescribeAccountInput{
		AccountId: &accountID,
	}

	result, err := rd.orgsClient.DescribeAccount(ctx, input)
	if err != nil {
		return "", NewAPIError("failed to describe account", err).
			WithContext("account_id", accountID)
	}

	if result.Account == nil || result.Account.Name == nil {
		return "", NewAPIError("account name not found", nil).
			WithContext("account_id", accountID)
	}

	return *result.Account.Name, nil
}

// getAccountNameFromSSO attempts to get account name from SSO (less reliable)
func (rd *RoleDiscovery) getAccountNameFromSSO(accountID string) (string, error) {
	// This is a fallback method - SSO doesn't directly provide account names
	// We could potentially implement account name resolution through other means
	// For now, we'll return an error to indicate this method is not implemented
	return "", NewAPIError("account name resolution from SSO not implemented", nil).
		WithContext("account_id", accountID)
}

// DiscoverRolesWithRetry discovers roles with exponential backoff retry
func (rd *RoleDiscovery) DiscoverRolesWithRetry(maxRetries int) ([]DiscoveredRole, error) {
	var lastErr error
	baseDelay := 1 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		roles, err := rd.DiscoverAccessibleRoles()
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

// GetInstanceArn returns the SSO instance ARN
func (rd *RoleDiscovery) GetInstanceArn() string {
	return rd.instanceArn
}

// GetIdentityStoreID returns the identity store ID
func (rd *RoleDiscovery) GetIdentityStoreID() string {
	return rd.identityStoreID
}

// ClearAccountCache clears the account name cache
func (rd *RoleDiscovery) ClearAccountCache() {
	rd.cacheMutex.Lock()
	defer rd.cacheMutex.Unlock()
	rd.accountCache = make(map[string]string)
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
