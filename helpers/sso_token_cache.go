package helpers

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"
)

// tokenCacheNotFoundMessage is the error message used when an SSO token cache
// file does not exist. It is shared so callers can detect the not-found case.
const tokenCacheNotFoundMessage = "SSO token cache not found"

// SSOTokenCache handles access to AWS SSO token cache
type SSOTokenCache struct {
	cacheDir string
	logger   Logger
}

// CachedToken represents a cached SSO token
type CachedToken struct {
	AccessToken           string    `json:"accessToken"`
	ExpiresAt             time.Time `json:"expiresAt"`
	Region                string    `json:"region"`
	StartURL              string    `json:"startUrl"`
	ClientID              string    `json:"clientId,omitempty"`
	ClientSecret          string    `json:"clientSecret,omitempty"`
	RegistrationExpiresAt time.Time `json:"registrationExpiresAt"`
	RefreshToken          string    `json:"refreshToken,omitempty"`
}

// NewSSOTokenCache creates a new SSO token cache
func NewSSOTokenCache() (*SSOTokenCache, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return nil, NewFileSystemError("failed to get home directory", err)
	}

	cacheDir := filepath.Join(homeDir, ".aws", "sso", "cache")

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return nil, NewFileSystemError("failed to create SSO cache directory", err).
			WithContext("cache_dir", cacheDir)
	}

	return &SSOTokenCache{
		cacheDir: cacheDir,
		logger:   &defaultLogger{},
	}, nil
}

// SetLogger sets a custom logger
func (stc *SSOTokenCache) SetLogger(logger Logger) {
	stc.logger = logger
}

// LoadTokenForTemplateProfile loads the cached SSO token for a template profile.
//
// Modern profiles (sso_session) and legacy profiles (sso_start_url only) store
// their tokens under different cache filenames. For modern profiles the AWS CLI
// hashes the session name; for legacy profiles it hashes the start URL. This
// method tries the session-name cache first when a session is present and falls
// back to the start-url cache so that caches written by older tooling still
// resolve.
func (stc *SSOTokenCache) LoadTokenForTemplateProfile(profile *TemplateProfile) (*CachedToken, error) {
	// Modern profiles cache the token under the SHA1 of the session name.
	if profile.SSOSession != "" {
		token, err := stc.LoadTokenForKey(profile.SSOSession, profile.SSORegion)
		if err == nil {
			return token, nil
		}
		// Fall back to the legacy start-url cache (e.g. caches written by an
		// older AWS CLI) only when the session-name cache is absent.
		if !isTokenCacheNotFound(err) {
			return nil, err
		}
	}

	return stc.LoadTokenForKey(profile.SSOStartURL, profile.SSORegion)
}

// LoadTokenForProfile loads cached token for a specific SSO profile keyed by its
// start URL. Retained for legacy callers; prefer LoadTokenForTemplateProfile so
// modern sso_session profiles resolve correctly.
func (stc *SSOTokenCache) LoadTokenForProfile(startURL, region string) (*CachedToken, error) {
	return stc.LoadTokenForKey(startURL, region)
}

// LoadTokenForKey loads and validates the cached token whose filename is derived
// from the SHA1 hash of cacheKey (either an SSO session name or an SSO start URL).
func (stc *SSOTokenCache) LoadTokenForKey(cacheKey, region string) (*CachedToken, error) {
	tokenFile := stc.getTokenFilePath(cacheKey)

	if _, err := os.Stat(tokenFile); os.IsNotExist(err) {
		return nil, NewAuthError(tokenCacheNotFoundMessage, err).
			WithContext("cache_key", cacheKey).
			WithContext("region", region).
			WithContext("suggestion", "Run 'aws sso login' to authenticate")
	}

	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, NewFileSystemError("failed to read SSO token cache", err).
			WithContext("token_file", tokenFile)
	}

	var token CachedToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, NewFileSystemError("failed to parse SSO token cache", err).
			WithContext("token_file", tokenFile)
	}

	// Check if token is expired
	if time.Now().After(token.ExpiresAt) {
		return nil, NewAuthError("SSO token has expired", nil).
			WithContext("expired_at", token.ExpiresAt).
			WithContext("suggestion", "Run 'aws sso login' to refresh authentication")
	}

	// Check if token is expiring soon (within 5 minutes)
	if time.Now().Add(5 * time.Minute).After(token.ExpiresAt) {
		stc.logger.Printf("Warning: SSO token expires soon at %s", token.ExpiresAt.Format(time.RFC3339))
	}

	return &token, nil
}

// isTokenCacheNotFound reports whether err indicates the token cache file was
// absent (as opposed to being present but expired or unreadable).
func isTokenCacheNotFound(err error) bool {
	var pgErr ProfileGeneratorError
	if errors.As(err, &pgErr) {
		return pgErr.Type == ErrorTypeAuth && pgErr.Message == tokenCacheNotFoundMessage
	}
	return false
}

// getTokenFilePath generates the cache file path for a given cache key (an SSO
// session name for modern profiles or an SSO start URL for legacy profiles).
func (stc *SSOTokenCache) getTokenFilePath(cacheKey string) string {
	// NOTE: SHA1 is used here for compatibility with AWS CLI cache file naming conventions.
	// This is NOT for security purposes. SHA1 is cryptographically weak and should not be used
	// for integrity or authentication. AWS CLI uses a SHA1 hash of the session name (modern
	// sso_session profiles) or the start URL (legacy profiles) to generate cache file names,
	// and we follow the same convention to maintain compatibility.
	hash := sha1.Sum([]byte(cacheKey))
	hashStr := hex.EncodeToString(hash[:])

	return filepath.Join(stc.cacheDir, hashStr+".json")
}

// IsTokenValid checks if a token is valid and not expired
func (stc *SSOTokenCache) IsTokenValid(token *CachedToken) bool {
	if token == nil {
		return false
	}

	if token.AccessToken == "" {
		return false
	}

	return time.Now().Before(token.ExpiresAt)
}

// GetAllCachedTokens returns all cached SSO tokens
func (stc *SSOTokenCache) GetAllCachedTokens() ([]*CachedToken, error) {
	files, err := os.ReadDir(stc.cacheDir)
	if err != nil {
		return nil, NewFileSystemError("failed to read SSO cache directory", err).
			WithContext("cache_dir", stc.cacheDir)
	}

	var tokens []*CachedToken
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		tokenFile := filepath.Join(stc.cacheDir, file.Name())
		data, err := os.ReadFile(tokenFile)
		if err != nil {
			stc.logger.Printf("Warning: failed to read token file %s: %v", tokenFile, err)
			continue
		}

		var token CachedToken
		if err := json.Unmarshal(data, &token); err != nil {
			stc.logger.Printf("Warning: failed to parse token file %s: %v", tokenFile, err)
			continue
		}

		tokens = append(tokens, &token)
	}

	return tokens, nil
}

// ValidateProfileToken validates that a cached token exists for the given profile
func (stc *SSOTokenCache) ValidateProfileToken(profile *TemplateProfile) error {
	if !profile.IsSSO {
		return NewValidationError("profile is not an SSO profile", nil)
	}

	token, err := stc.LoadTokenForTemplateProfile(profile)
	if err != nil {
		return err
	}

	if !stc.IsTokenValid(token) {
		return NewAuthError("SSO token is invalid or expired", nil).
			WithContext("start_url", profile.SSOStartURL).
			WithContext("suggestion", "Run 'aws sso login' to refresh authentication")
	}

	return nil
}

// GetAuthGuideMessage returns a helpful message for authentication issues
func (stc *SSOTokenCache) GetAuthGuideMessage(startURL, region string) string {
	return fmt.Sprintf(`
SSO authentication required. Please run:

  aws sso login --profile <profile-name>

Or if you know the SSO start URL:

  aws sso login --sso-start-url %s --sso-region %s

After successful authentication, retry this command.
`, startURL, region)
}

// CleanExpiredTokens removes expired tokens from the cache
func (stc *SSOTokenCache) CleanExpiredTokens() (int, error) {
	tokens, err := stc.GetAllCachedTokens()
	if err != nil {
		return 0, err
	}

	expiredCount := 0
	for _, token := range tokens {
		if !stc.IsTokenValid(token) {
			tokenFile := stc.getTokenFilePath(token.StartURL)
			if err := os.Remove(tokenFile); err != nil {
				stc.logger.Printf("Warning: failed to remove expired token file %s: %v", tokenFile, err)
			} else {
				expiredCount++
			}
		}
	}

	return expiredCount, nil
}

// GetCacheInfo returns information about the SSO token cache
func (stc *SSOTokenCache) GetCacheInfo() (map[string]any, error) {
	info := make(map[string]any)
	info["cache_dir"] = stc.cacheDir

	tokens, err := stc.GetAllCachedTokens()
	if err != nil {
		return info, err
	}

	info["total_tokens"] = len(tokens)

	validTokens := 0
	expiredTokens := 0

	for _, token := range tokens {
		if stc.IsTokenValid(token) {
			validTokens++
		} else {
			expiredTokens++
		}
	}

	info["valid_tokens"] = validTokens
	info["expired_tokens"] = expiredTokens

	return info, nil
}
