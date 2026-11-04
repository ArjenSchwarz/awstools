package helpers

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// hostRecorder is an HTTP client that records request hosts and answers every
// SSO portal call with an empty account list.
type hostRecorder struct {
	mu    sync.Mutex
	hosts []string
}

func (h *hostRecorder) Do(req *http.Request) (*http.Response, error) {
	h.mu.Lock()
	h.hosts = append(h.hosts, req.URL.Host)
	h.mu.Unlock()
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"accountList":[]}`)),
		Request:    req,
	}, nil
}

// TestDiscoverRolesUsesTemplateSSORegion is a regression test for T-1468: role
// discovery must call the SSO portal in the template profile's sso_region,
// not the ambient AWS config region.
func TestDiscoverRolesUsesTemplateSSORegion(t *testing.T) {
	const startURL = "https://example.awsapps.com/start"
	const wantHost = "portal.sso.ap-southeast-2.amazonaws.com"

	tests := map[string]struct {
		ambientRegion string
	}{
		"ambient region differs from sso_region": {ambientRegion: "us-east-1"},
		"no ambient region":                      {ambientRegion: ""},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Setenv("HOME", tempDir)
			t.Setenv("AWS_CONFIG_FILE", filepath.Join(tempDir, "unused-config"))

			configFile := filepath.Join(tempDir, "config")
			configContent := `[profile template]
sso_start_url = ` + startURL + `
sso_region = ap-southeast-2
sso_account_id = 123456789012
sso_role_name = ReadOnly
region = eu-west-1
`
			if err := os.WriteFile(configFile, []byte(configContent), 0600); err != nil {
				t.Fatal(err)
			}

			recorder := &hostRecorder{}
			awsConfig := aws.Config{Region: tc.ambientRegion, HTTPClient: recorder}

			pg, err := NewProfileGenerator("template", "", true, configFile, ConflictSkip, awsConfig)
			if err != nil {
				t.Fatalf("NewProfileGenerator() error = %v", err)
			}

			// Keep the token cache inside the test directory.
			cacheDir := filepath.Join(tempDir, "sso-cache")
			if err := os.MkdirAll(cacheDir, 0700); err != nil {
				t.Fatal(err)
			}
			pg.roleDiscovery.tokenCache.cacheDir = cacheDir
			writeTestToken(t, cacheDir, startURL, "ap-southeast-2")

			templateProfile, err := pg.ValidateTemplateProfile()
			if err != nil {
				t.Fatalf("ValidateTemplateProfile() error = %v", err)
			}

			// The fake portal returns no accounts, so an error is expected; the
			// assertion is on where the request went.
			_, _ = pg.DiscoverRoles(templateProfile)

			if len(recorder.hosts) == 0 {
				t.Fatal("no SSO portal request was made")
			}
			for _, got := range recorder.hosts {
				if got != wantHost {
					t.Errorf("SSO request host = %q, want %q", got, wantHost)
				}
			}
		})
	}
}

func writeTestToken(t *testing.T, cacheDir, startURL, region string) {
	t.Helper()
	data, err := json.Marshal(CachedToken{
		AccessToken: "test-token",
		ExpiresAt:   time.Now().Add(time.Hour),
		Region:      region,
		StartURL:    startURL,
	})
	if err != nil {
		t.Fatal(err)
	}
	hash := sha1.Sum([]byte(startURL))
	path := filepath.Join(cacheDir, hex.EncodeToString(hash[:])+".json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}
