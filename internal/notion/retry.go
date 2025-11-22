package notion

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/jomei/notionapi"
)

// RetryConfig holds configuration for retry behavior.
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts (default: 3).
	MaxRetries int
	// InitialBackoff is the initial backoff duration (default: 1 second).
	InitialBackoff time.Duration
	// MaxBackoff is the maximum backoff duration (default: 16 seconds).
	MaxBackoff time.Duration
	// BackoffMultiplier is the multiplier for exponential backoff (default: 2.0).
	BackoffMultiplier float64
}

// DefaultRetryConfig returns the default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        16 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// RetryableFunc is a function that can be retried.
type RetryableFunc func(ctx context.Context) error

// RetryWithBackoff executes a function with exponential backoff retry logic.
// It handles different error types appropriately:
// - Rate limit (429): Uses Retry-After header if available
// - Temporary network errors: Retries with exponential backoff
// - Auth errors (401, 403): Does not retry
// - Not found (404): Does not retry
// - Server errors (5xx): Retries up to MaxRetries times
func RetryWithBackoff(ctx context.Context, config RetryConfig, fn RetryableFunc) error {
	var lastErr error
	backoff := config.InitialBackoff

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Execute the function
		err := fn(ctx)
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if we should retry
		shouldRetry, retryAfter := shouldRetryError(err)
		if !shouldRetry {
			return err // Don't retry
		}

		// Check if this is the last attempt
		if attempt >= config.MaxRetries {
			return fmt.Errorf("max retries (%d) exceeded: %w", config.MaxRetries, lastErr)
		}

		// Calculate backoff duration
		var waitDuration time.Duration
		if retryAfter > 0 {
			// Use server-specified retry-after
			waitDuration = retryAfter
		} else {
			// Use exponential backoff
			waitDuration = backoff
			if waitDuration > config.MaxBackoff {
				waitDuration = config.MaxBackoff
			}
		}

		// Wait before retrying
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		case <-time.After(waitDuration):
			// Continue to next attempt
		}

		// Increase backoff for next iteration
		backoff = time.Duration(float64(backoff) * config.BackoffMultiplier)
		if backoff > config.MaxBackoff {
			backoff = config.MaxBackoff
		}
	}

	return fmt.Errorf("retry failed after %d attempts: %w", config.MaxRetries+1, lastErr)
}

// shouldRetryError determines if an error is retryable and returns the retry-after duration.
func shouldRetryError(err error) (shouldRetry bool, retryAfter time.Duration) {
	if err == nil {
		return false, 0
	}

	// Check for context errors first (don't retry)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false, 0
	}

	// Check for Notion API errors
	var notionErr *notionapi.Error
	if errors.As(err, &notionErr) {
		return shouldRetryHTTPError(notionErr)
	}

	// Check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		// Retry on temporary network errors
		if netErr.Temporary() || netErr.Timeout() {
			return true, 0
		}
	}

	// Default: retry unknown errors
	return true, 0
}

// shouldRetryHTTPError determines retry behavior for HTTP errors.
func shouldRetryHTTPError(httpErr *notionapi.Error) (shouldRetry bool, retryAfter time.Duration) {
	switch httpErr.Status {
	case http.StatusUnauthorized, http.StatusForbidden:
		// Auth errors - don't retry
		return false, 0

	case http.StatusNotFound:
		// Not found - don't retry
		return false, 0

	case http.StatusBadRequest:
		// Validation errors - don't retry
		return false, 0

	case http.StatusTooManyRequests:
		// Rate limit - retry with server-specified backoff
		retryAfter := parseRetryAfter(httpErr)
		return true, retryAfter

	case http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		// Server errors - retry
		return true, 0

	case http.StatusRequestTimeout:
		// Timeout - retry
		return true, 0

	default:
		// Unknown HTTP error - retry
		return true, 0
	}
}

// parseRetryAfter attempts to parse the Retry-After header from a Notion API error.
// It returns 0 if the header is not present or cannot be parsed.
func parseRetryAfter(httpErr *notionapi.Error) time.Duration {
	// The notionapi.Error struct doesn't expose headers directly,
	// so we'll use a default retry duration for rate limits
	// In a production system, we might want to extend the error type to include headers

	// Default to 5 seconds for rate limit errors
	return 5 * time.Second
}

// RetryableOperation wraps a Notion API operation with retry logic.
type RetryableOperation struct {
	config RetryConfig
}

// NewRetryableOperation creates a new retryable operation with the given config.
func NewRetryableOperation(config RetryConfig) *RetryableOperation {
	return &RetryableOperation{
		config: config,
	}
}

// NewDefaultRetryableOperation creates a new retryable operation with default config.
func NewDefaultRetryableOperation() *RetryableOperation {
	return &RetryableOperation{
		config: DefaultRetryConfig(),
	}
}

// Execute runs the given function with retry logic.
func (ro *RetryableOperation) Execute(ctx context.Context, fn RetryableFunc) error {
	return RetryWithBackoff(ctx, ro.config, fn)
}

// WithMaxRetries returns a new RetryableOperation with the specified max retries.
func (ro *RetryableOperation) WithMaxRetries(maxRetries int) *RetryableOperation {
	newConfig := ro.config
	newConfig.MaxRetries = maxRetries
	return &RetryableOperation{config: newConfig}
}

// WithInitialBackoff returns a new RetryableOperation with the specified initial backoff.
func (ro *RetryableOperation) WithInitialBackoff(backoff time.Duration) *RetryableOperation {
	newConfig := ro.config
	newConfig.InitialBackoff = backoff
	return &RetryableOperation{config: newConfig}
}

// WithMaxBackoff returns a new RetryableOperation with the specified max backoff.
func (ro *RetryableOperation) WithMaxBackoff(backoff time.Duration) *RetryableOperation {
	newConfig := ro.config
	newConfig.MaxBackoff = backoff
	return &RetryableOperation{config: newConfig}
}

// IsRetryableError returns whether the given error is retryable.
func IsRetryableError(err error) bool {
	shouldRetry, _ := shouldRetryError(err)
	return shouldRetry
}

// IsNetworkError returns whether the given error is a network error.
func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}

	// Check for net.Error
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	// Check for common network error strings
	errStr := err.Error()
	networkKeywords := []string{
		"connection refused",
		"no such host",
		"network is unreachable",
		"timeout",
		"deadline exceeded",
		"connection reset",
		"broken pipe",
	}

	for _, keyword := range networkKeywords {
		if contains(errStr, keyword) {
			return true
		}
	}

	return false
}

// IsAuthError returns whether the given error is an authentication error.
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}

	var notionErr *notionapi.Error
	if errors.As(err, &notionErr) {
		return notionErr.Status == http.StatusUnauthorized ||
			notionErr.Status == http.StatusForbidden
	}

	errStr := err.Error()
	return contains(errStr, "401") || contains(errStr, "unauthorized") ||
		contains(errStr, "403") || contains(errStr, "forbidden")
}

// IsNotFoundError returns whether the given error is a not found error.
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var notionErr *notionapi.Error
	if errors.As(err, &notionErr) {
		return notionErr.Status == http.StatusNotFound
	}

	errStr := err.Error()
	return contains(errStr, "404") || contains(errStr, "not found")
}

// IsRateLimitError returns whether the given error is a rate limit error.
func IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	var notionErr *notionapi.Error
	if errors.As(err, &notionErr) {
		return notionErr.Status == http.StatusTooManyRequests
	}

	errStr := err.Error()
	return contains(errStr, "429") || contains(errStr, "rate limit")
}

// IsServerError returns whether the given error is a server error (5xx).
func IsServerError(err error) bool {
	if err == nil {
		return false
	}

	var notionErr *notionapi.Error
	if errors.As(err, &notionErr) {
		return notionErr.Status >= 500 && notionErr.Status < 600
	}

	errStr := err.Error()
	for code := 500; code < 600; code++ {
		if contains(errStr, strconv.Itoa(code)) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	// Simple case-insensitive contains
	return len(s) >= len(substr) && stringContains(s, substr)
}

// stringContains is a helper for case-insensitive substring search.
func stringContains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}

	// Convert to lowercase for comparison
	sLower := toLower(s)
	substrLower := toLower(substr)

	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
			return true
		}
	}
	return false
}

// toLower converts a string to lowercase (simple ASCII version).
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}
