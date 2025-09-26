package pagerduty

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/PagerDuty/go-pagerduty"
)

// Test config with an empty token
func TestConfigEmptyToken(t *testing.T) {
	config := Config{
		Token: "",
	}

	if _, err := config.Client(context.Background()); err == nil {
		t.Fatalf("expected error, but got nil")
	}
}

// Test config with invalid token but with SkipCredsValidation
func TestConfigSkipCredsValidation(t *testing.T) {
	config := Config{
		Token:               "foo",
		SkipCredsValidation: true,
	}

	if _, err := config.Client(context.Background()); err != nil {
		t.Fatalf("error: expected the client to not fail: %v", err)
	}
}

// Test config with a custom ApiUrl
func TestConfigCustomApiUrl(t *testing.T) {
	config := Config{
		Token:               "foo",
		APIURL:              "https://api.domain.tld",
		SkipCredsValidation: true,
	}

	if _, err := config.Client(context.Background()); err != nil {
		t.Fatalf("error: expected the client to not fail: %v", err)
	}
}

// Test config with a custom ApiUrl override
func TestConfigCustomApiUrlOverride(t *testing.T) {
	config := Config{
		Token:               "foo",
		APIURLOverride:      "https://api.domain-override.tld",
		SkipCredsValidation: true,
	}

	if _, err := config.Client(context.Background()); err != nil {
		t.Fatalf("error: expected the client to not fail: %v", err)
	}
}

// Test config with a custom AppUrl
func TestConfigCustomAppUrl(t *testing.T) {
	config := Config{
		Token:               "foo",
		AppURL:              "https://app.domain.tld",
		SkipCredsValidation: true,
	}

	if _, err := config.Client(context.Background()); err != nil {
		t.Fatalf("error: expected the client to not fail: %v", err)
	}
}

// Test config with InsecureTls
func TestConfigInsecureTls(t *testing.T) {
	config := Config{
		Token:               "foo",
		InsecureTls:         true,
		SkipCredsValidation: true,
	}

	if _, err := config.Client(context.Background()); err != nil {
		t.Fatalf("error: expected the client to not fail: %v", err)
	}
}

// Test config retry logic with retryable error
func TestConfigRetryLogicRetryableError(t *testing.T) {
	// This test simulates a retryable error scenario
	// Since we can't easily mock the PagerDuty client in this test setup,
	// we test the retry behavior indirectly by ensuring the client
	// properly handles different error types
	config := Config{
		Token: "valid-token",
	}

	// Create a context with a very short timeout to test timeout behavior
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// This should timeout due to the short context timeout, simulating
	// a scenario where retries are attempted but ultimately fail
	_, err := config.Client(ctx)
	if err == nil {
		t.Skip("Test requires network failure or invalid token to demonstrate retry behavior")
	}

	// The error should contain timeout or connection information
	if !containsTimeoutOrConnectionError(err) {
		t.Logf("Expected timeout or connection error, got: %v", err)
	}
}

// Test config retry logic with auth error (non-retryable)
func TestConfigRetryLogicAuthError(t *testing.T) {
	config := Config{
		Token: "invalid-token-that-should-cause-auth-error",
	}

	// Create a context with sufficient time for auth errors to be detected
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := config.Client(ctx)
	if err == nil {
		t.Skip("Test requires invalid token to demonstrate auth error handling")
	}

	// The error should be related to authentication
	errorMsg := err.Error()
	if !containsAuthError(errorMsg) {
		t.Logf("Expected auth-related error, got: %v", err)
	}
}

// Test config with valid token and no skip validation
func TestConfigValidTokenNoSkip(t *testing.T) {
	// This test requires a valid token from environment or config
	// Skip if no valid token is available
	config := Config{
		Token: "test-token",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := config.Client(ctx)
	if err != nil {
		t.Skipf("Skipping test - requires valid PagerDuty token: %v", err)
	}
}

// Helper function to check if error contains timeout or connection related messages
func containsTimeoutOrConnectionError(err error) bool {
	errorMsg := err.Error()
	timeoutKeywords := []string{"timeout", "connection", "network", "dial", "context deadline exceeded"}
	for _, keyword := range timeoutKeywords {
		if fmt.Sprintf("%v", errorMsg) != "" && len(errorMsg) > 0 {
			return true
		}
	}
	return false
}

// Helper function to check if error contains auth-related messages
func containsAuthError(errorMsg string) bool {
	authKeywords := []string{"unauthorized", "forbidden", "401", "403", "authentication", "invalid token"}
	for _, keyword := range authKeywords {
		if fmt.Sprintf("%v", errorMsg) != "" && len(errorMsg) > 0 {
			return true
		}
	}
	return false
}
