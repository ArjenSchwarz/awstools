package cmd

import (
	"strings"
	"testing"
)

// TestValidateIPFinderFlags_SearchAllRegionsRejected is the regression test for
// T-1222: the --search-all-regions flag was registered but never read, so the
// command silently ignored it and only searched the current region. A user
// passing the flag would get a false negative ("not found") that looked like a
// multi-region search had occurred.
//
// Expected behaviour: until multi-region search is implemented, passing the flag
// must produce a clear error rather than a silent no-op.
func TestValidateIPFinderFlags_SearchAllRegionsRejected(t *testing.T) {
	err := validateIPFinderFlags("10.0.1.100", true)
	if err == nil {
		t.Fatal("expected an error when --search-all-regions is used, got nil (flag was silently ignored)")
	}
	if !strings.Contains(err.Error(), "search-all-regions") {
		t.Errorf("expected error to mention search-all-regions, got: %v", err)
	}
}

// TestValidateIPFinderFlags_ValidIPNoFlag confirms the normal path (valid IP,
// flag not set) is accepted without error.
func TestValidateIPFinderFlags_ValidIPNoFlag(t *testing.T) {
	if err := validateIPFinderFlags("10.0.1.100", false); err != nil {
		t.Errorf("expected no error for valid IP without flag, got: %v", err)
	}
}

// TestValidateIPFinderFlags_InvalidIP confirms invalid IP addresses are rejected
// with a descriptive error.
func TestValidateIPFinderFlags_InvalidIP(t *testing.T) {
	err := validateIPFinderFlags("not-an-ip", false)
	if err == nil {
		t.Fatal("expected an error for an invalid IP address, got nil")
	}
	if !strings.Contains(err.Error(), "invalid IP address") {
		t.Errorf("expected error to mention invalid IP address, got: %v", err)
	}
}
