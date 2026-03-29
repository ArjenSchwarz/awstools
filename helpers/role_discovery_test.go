package helpers

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainsAny_CaseInsensitive(t *testing.T) {
	substrings := []string{"throttling", "rate limit", "too many requests"}

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Lowercase (baseline — already works)
		{"lowercase throttling", "throttling: request rate exceeded", true},
		{"lowercase rate limit", "rate limit exceeded", true},
		{"lowercase too many requests", "too many requests", true},

		// AWS-style capitalised messages (the bug: these should match but don't)
		{"capitalised Throttling", "Throttling: Rate of requests exceeds limit", true},
		{"title-case ThrottlingException", "ThrottlingException: Rate exceeded", true},
		{"capitalised Rate limit", "Rate limit exceeded for API", true},
		{"uppercase THROTTLING", "THROTTLING ERROR", true},
		{"mixed case Too Many Requests", "Too Many Requests: slow down", true},

		// Non-matching strings
		{"unrelated error", "AccessDeniedException: not authorized", false},
		{"empty string", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := containsAny(tc.input, substrings)
			assert.Equal(t, tc.expected, result, "containsAny(%q, ...) should be %v", tc.input, tc.expected)
		})
	}
}

func TestIsRetryableError_CaseInsensitiveThrottling(t *testing.T) {
	// We need a minimal RoleDiscovery to call isRetryableError.
	// The method only inspects the error value, not AWS clients.
	rd := &RoleDiscovery{logger: &defaultLogger{}}

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "lowercase throttling in API error",
			err:      NewAPIError("call failed", fmt.Errorf("throttling: rate exceeded")),
			expected: true,
		},
		{
			name:     "capitalised ThrottlingException in API error",
			err:      NewAPIError("call failed", fmt.Errorf("ThrottlingException: Rate exceeded")),
			expected: true,
		},
		{
			name:     "uppercase RATE LIMIT in API error",
			err:      NewAPIError("call failed", fmt.Errorf("RATE LIMIT exceeded")),
			expected: true,
		},
		{
			name:     "mixed-case Too Many Requests in API error",
			err:      NewAPIError("call failed", fmt.Errorf("Too Many Requests")),
			expected: true,
		},
		{
			name:     "network error is retryable",
			err:      ProfileGeneratorError{Type: ErrorTypeNetwork, Message: "timeout"},
			expected: true,
		},
		{
			name:     "non-retryable API error",
			err:      NewAPIError("call failed", fmt.Errorf("AccessDeniedException")),
			expected: false,
		},
		{
			name:     "validation error is not retryable",
			err:      NewValidationError("bad input", nil),
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := rd.isRetryableError(tc.err)
			assert.Equal(t, tc.expected, result)
		})
	}
}
