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

// TestValidateIPFinderFlags_InvalidInput_T1370 verifies that invalid IP input is
// reported as a normal error rather than triggering a panic.
//
// Bug (T-1370): cmd/vpcipfinder.go validated the positional IP but on failure
// called panic(fmt.Errorf(...)). Invalid user input is an expected CLI error, so
// it must be returned as a controlled error (handled by Cobra/Execute) instead
// of aborting with a stack trace. Validation now lives in validateIPFinderFlags.
func TestValidateIPFinderFlags_InvalidInput_T1370(t *testing.T) {
	invalidInputs := []string{
		"not-an-ip",
		"",
		"999.999.999.999", // out-of-range IPv4
		"192.168.1",       // truncated IPv4
		"2001:db8::g",     // invalid IPv6 (bad hex digit)
		"::ffff:999.1.1.1",
	}

	for _, input := range invalidInputs {
		t.Run(input, func(t *testing.T) {
			// validateIPFinderFlags must never panic on invalid input.
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("validateIPFinderFlags panicked on %q: %v", input, r)
				}
			}()

			err := validateIPFinderFlags(input, false)
			if err == nil {
				t.Fatalf("expected an error for invalid IP %q, got nil", input)
			}
			if !strings.Contains(err.Error(), "invalid IP address") {
				t.Errorf("expected error to mention invalid IP address, got: %v", err)
			}
		})
	}
}

// TestValidateIPFinderFlags_ValidInput_T1370 verifies that valid IPv4 and IPv6
// addresses pass validation without error.
func TestValidateIPFinderFlags_ValidInput_T1370(t *testing.T) {
	validInputs := []string{
		"192.168.1.1",
		"10.0.1.100",
		"2001:db8::1",
		"::1",
	}

	for _, input := range validInputs {
		t.Run(input, func(t *testing.T) {
			if err := validateIPFinderFlags(input, false); err != nil {
				t.Errorf("expected no error for valid IP %q, got: %v", input, err)
			}
		})
	}
}
