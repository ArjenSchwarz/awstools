package helpers

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// newTestTokenCache creates an SSOTokenCache backed by a temporary directory so
// tests do not touch the real ~/.aws/sso/cache.
func newTestTokenCache(t *testing.T) *SSOTokenCache {
	t.Helper()
	dir := t.TempDir()
	return &SSOTokenCache{
		cacheDir: dir,
		logger:   &defaultLogger{},
	}
}

// writeCacheFile writes a CachedToken to a cache file keyed by the SHA1 hash of key.
func writeCacheFile(t *testing.T, cache *SSOTokenCache, key string, token CachedToken) {
	t.Helper()
	hash := sha1.Sum([]byte(key))
	name := hex.EncodeToString(hash[:]) + ".json"
	data, err := json.Marshal(token)
	if err != nil {
		t.Fatalf("failed to marshal token: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cache.cacheDir, name), data, 0600); err != nil {
		t.Fatalf("failed to write cache file: %v", err)
	}
}

func validToken(startURL, region string) CachedToken {
	return CachedToken{
		AccessToken: "test-access-token",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
		Region:      region,
		StartURL:    startURL,
	}
}

// TestLoadTokenForProfile_LegacyStartURLCache verifies legacy profiles (no
// sso_session) still resolve via the start-url hash.
func TestLoadTokenForProfile_LegacyStartURLCache(t *testing.T) {
	cache := newTestTokenCache(t)
	startURL := "https://legacy-org.awsapps.com/start"
	region := "us-east-1"
	writeCacheFile(t, cache, startURL, validToken(startURL, region))

	profile := &TemplateProfile{
		Name:        "legacy",
		SSOStartURL: startURL,
		SSORegion:   region,
		IsSSO:       true,
	}

	token, err := cache.LoadTokenForTemplateProfile(profile)
	if err != nil {
		t.Fatalf("expected to load legacy token, got error: %v", err)
	}
	if token.AccessToken != "test-access-token" {
		t.Fatalf("unexpected access token: %q", token.AccessToken)
	}
}

// TestLoadTokenForProfile_ModernSessionCache reproduces T-1204: modern
// sso_session profiles store the token under the SHA1 of the session name, not
// the start URL. Before the fix this fails with "SSO token cache not found".
func TestLoadTokenForProfile_ModernSessionCache(t *testing.T) {
	cache := newTestTokenCache(t)
	startURL := "https://modern-org.awsapps.com/start"
	region := "us-east-1"
	sessionName := "my-sso-session"

	// AWS CLI writes the modern token keyed by the session name only.
	writeCacheFile(t, cache, sessionName, validToken(startURL, region))

	profile := &TemplateProfile{
		Name:        "modern",
		SSOStartURL: startURL,
		SSORegion:   region,
		SSOSession:  sessionName,
		IsSSO:       true,
	}

	token, err := cache.LoadTokenForTemplateProfile(profile)
	if err != nil {
		t.Fatalf("expected to load modern session token, got error: %v", err)
	}
	if token.AccessToken != "test-access-token" {
		t.Fatalf("unexpected access token: %q", token.AccessToken)
	}
}

// TestLoadTokenForProfile_ModernFallsBackToStartURL verifies that when a modern
// profile's session-name cache file is absent but a start-url cache file exists
// (e.g. cache written by an older CLI), lookup still succeeds via fallback.
func TestLoadTokenForProfile_ModernFallsBackToStartURL(t *testing.T) {
	cache := newTestTokenCache(t)
	startURL := "https://fallback-org.awsapps.com/start"
	region := "us-east-1"
	sessionName := "fallback-session"

	// Only the legacy start-url file exists.
	writeCacheFile(t, cache, startURL, validToken(startURL, region))

	profile := &TemplateProfile{
		Name:        "modern-fallback",
		SSOStartURL: startURL,
		SSORegion:   region,
		SSOSession:  sessionName,
		IsSSO:       true,
	}

	token, err := cache.LoadTokenForTemplateProfile(profile)
	if err != nil {
		t.Fatalf("expected fallback to start-url token, got error: %v", err)
	}
	if token.AccessToken != "test-access-token" {
		t.Fatalf("unexpected access token: %q", token.AccessToken)
	}
}

// TestLoadTokenForProfile_NotFound verifies a missing token returns an error.
func TestLoadTokenForProfile_NotFound(t *testing.T) {
	cache := newTestTokenCache(t)
	profile := &TemplateProfile{
		Name:        "missing",
		SSOStartURL: "https://missing-org.awsapps.com/start",
		SSORegion:   "us-east-1",
		SSOSession:  "missing-session",
		IsSSO:       true,
	}

	if _, err := cache.LoadTokenForTemplateProfile(profile); err == nil {
		t.Fatal("expected error for missing token cache, got nil")
	}
}
