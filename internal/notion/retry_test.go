package notion

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jomei/notionapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockNetError implements net.Error for testing.
type mockNetError struct {
	msg       string
	timeout   bool
	temporary bool
}

func (e *mockNetError) Error() string   { return e.msg }
func (e *mockNetError) Timeout() bool   { return e.timeout }
func (e *mockNetError) Temporary() bool { return e.temporary }

func TestRetryWithBackoff_Success(t *testing.T) {
	config := RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	attempts := 0
	fn := func(ctx context.Context) error {
		attempts++
		return nil // Success on first attempt
	}

	err := RetryWithBackoff(context.Background(), config, fn)

	assert.NoError(t, err)
	assert.Equal(t, 1, attempts)
}

func TestRetryWithBackoff_SuccessAfterRetries(t *testing.T) {
	config := RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	attempts := 0
	fn := func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			// Return temporary network error
			return &mockNetError{msg: "temporary error", temporary: true}
		}
		return nil // Success on 3rd attempt
	}

	start := time.Now()
	err := RetryWithBackoff(context.Background(), config, fn)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
	// Should have waited ~30ms (10ms + 20ms)
	assert.GreaterOrEqual(t, duration, 30*time.Millisecond)
}

func TestRetryWithBackoff_MaxRetriesExceeded(t *testing.T) {
	config := RetryConfig{
		MaxRetries:        2,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	attempts := 0
	fn := func(ctx context.Context) error {
		attempts++
		return &mockNetError{msg: "persistent error", temporary: true}
	}

	err := RetryWithBackoff(context.Background(), config, fn)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max retries")
	assert.Equal(t, 3, attempts) // Initial + 2 retries
}

func TestRetryWithBackoff_NonRetryableError(t *testing.T) {
	config := DefaultRetryConfig()

	tests := []struct {
		name    string
		err     error
		retries int
	}{
		{
			name:    "auth error 401",
			err:     &notionapi.Error{Status: 401},
			retries: 1, // No retries, just initial attempt
		},
		{
			name:    "auth error 403",
			err:     &notionapi.Error{Status: 403},
			retries: 1,
		},
		{
			name:    "not found 404",
			err:     &notionapi.Error{Status: 404},
			retries: 1,
		},
		{
			name:    "bad request 400",
			err:     &notionapi.Error{Status: 400},
			retries: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempts := 0
			fn := func(ctx context.Context) error {
				attempts++
				return tt.err
			}

			err := RetryWithBackoff(context.Background(), config, fn)

			assert.Error(t, err)
			assert.Equal(t, tt.retries, attempts)
		})
	}
}

func TestRetryWithBackoff_ServerErrors(t *testing.T) {
	config := RetryConfig{
		MaxRetries:        2,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	tests := []struct {
		name       string
		statusCode int
	}{
		{"500 internal server error", 500},
		{"502 bad gateway", 502},
		{"503 service unavailable", 503},
		{"504 gateway timeout", 504},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempts := 0
			fn := func(ctx context.Context) error {
				attempts++
				return &notionapi.Error{Status: tt.statusCode}
			}

			err := RetryWithBackoff(context.Background(), config, fn)

			assert.Error(t, err)
			assert.Equal(t, 3, attempts) // Initial + 2 retries
		})
	}
}

func TestRetryWithBackoff_RateLimitError(t *testing.T) {
	config := RetryConfig{
		MaxRetries:        2,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	attempts := 0
	fn := func(ctx context.Context) error {
		attempts++
		return &notionapi.Error{Status: 429}
	}

	start := time.Now()
	err := RetryWithBackoff(context.Background(), config, fn)
	duration := time.Since(start)

	assert.Error(t, err)
	assert.Equal(t, 3, attempts)
	// Should have waited for rate limit backoff (default 5 seconds per retry)
	// But our test uses milliseconds, so it should complete quickly
	assert.GreaterOrEqual(t, duration, 10*time.Second) // 2 retries * 5 seconds each
}

func TestRetryWithBackoff_ContextCancellation(t *testing.T) {
	config := RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        10 * time.Second,
		BackoffMultiplier: 2.0,
	}

	ctx, cancel := context.WithCancel(context.Background())

	attempts := 0
	fn := func(ctx context.Context) error {
		attempts++
		if attempts == 1 {
			cancel() // Cancel context after first attempt
		}
		return &mockNetError{msg: "error", temporary: true}
	}

	err := RetryWithBackoff(ctx, config, fn)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "retry cancelled")
	assert.LessOrEqual(t, attempts, 2) // Should stop after context cancellation
}

func TestRetryWithBackoff_ContextDeadlineInFunction(t *testing.T) {
	config := RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	attempts := 0
	fn := func(ctx context.Context) error {
		attempts++
		return context.DeadlineExceeded // Return deadline error from function
	}

	err := RetryWithBackoff(context.Background(), config, fn)

	assert.Error(t, err)
	assert.Equal(t, 1, attempts) // Should not retry context errors
}

func TestRetryWithBackoff_ExponentialBackoff(t *testing.T) {
	config := RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        1 * time.Second,
		BackoffMultiplier: 2.0,
	}

	attempts := 0
	var timestamps []time.Time

	fn := func(ctx context.Context) error {
		attempts++
		timestamps = append(timestamps, time.Now())
		return &mockNetError{msg: "error", temporary: true}
	}

	err := RetryWithBackoff(context.Background(), config, fn)

	assert.Error(t, err)
	assert.Equal(t, 4, attempts) // Initial + 3 retries

	// Verify exponential backoff timing
	// 1st retry: ~100ms after initial
	// 2nd retry: ~200ms after 1st
	// 3rd retry: ~400ms after 2nd
	if len(timestamps) >= 2 {
		diff1 := timestamps[1].Sub(timestamps[0])
		assert.GreaterOrEqual(t, diff1, 100*time.Millisecond)
	}
	if len(timestamps) >= 3 {
		diff2 := timestamps[2].Sub(timestamps[1])
		assert.GreaterOrEqual(t, diff2, 200*time.Millisecond)
	}
	if len(timestamps) >= 4 {
		diff3 := timestamps[3].Sub(timestamps[2])
		assert.GreaterOrEqual(t, diff3, 400*time.Millisecond)
	}
}

func TestRetryWithBackoff_MaxBackoff(t *testing.T) {
	config := RetryConfig{
		MaxRetries:        5,
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        2 * time.Second, // Cap backoff at 2 seconds
		BackoffMultiplier: 2.0,
	}

	attempts := 0
	var timestamps []time.Time

	fn := func(ctx context.Context) error {
		attempts++
		timestamps = append(timestamps, time.Now())
		return &mockNetError{msg: "error", temporary: true}
	}

	err := RetryWithBackoff(context.Background(), config, fn)

	assert.Error(t, err)

	// After initial 1s and 2s delays, all subsequent should be capped at 2s
	if len(timestamps) >= 4 {
		diff := timestamps[3].Sub(timestamps[2])
		assert.LessOrEqual(t, diff, 3*time.Second) // Should be ~2s, allow some margin
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 1*time.Second, config.InitialBackoff)
	assert.Equal(t, 16*time.Second, config.MaxBackoff)
	assert.Equal(t, 2.0, config.BackoffMultiplier)
}

func TestRetryableOperation(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		op := NewDefaultRetryableOperation()
		require.NotNil(t, op)

		attempts := 0
		err := op.Execute(context.Background(), func(ctx context.Context) error {
			attempts++
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, attempts)
	})

	t.Run("custom config", func(t *testing.T) {
		config := RetryConfig{
			MaxRetries:        1,
			InitialBackoff:    10 * time.Millisecond,
			MaxBackoff:        100 * time.Millisecond,
			BackoffMultiplier: 2.0,
		}
		op := NewRetryableOperation(config)

		attempts := 0
		err := op.Execute(context.Background(), func(ctx context.Context) error {
			attempts++
			return &mockNetError{msg: "error", temporary: true}
		})

		assert.Error(t, err)
		assert.Equal(t, 2, attempts) // Initial + 1 retry
	})

	t.Run("with builders", func(t *testing.T) {
		op := NewDefaultRetryableOperation().
			WithMaxRetries(2).
			WithInitialBackoff(10 * time.Millisecond).
			WithMaxBackoff(100 * time.Millisecond)

		attempts := 0
		err := op.Execute(context.Background(), func(ctx context.Context) error {
			attempts++
			if attempts < 3 {
				return &mockNetError{msg: "error", temporary: true}
			}
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 3, attempts)
	})
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{"nil error", nil, false},
		{"network timeout", &mockNetError{msg: "timeout", timeout: true}, true},
		{"network temporary", &mockNetError{msg: "temp", temporary: true}, true},
		{"401 unauthorized", &notionapi.Error{Status: 401}, false},
		{"404 not found", &notionapi.Error{Status: 404}, false},
		{"500 server error", &notionapi.Error{Status: 500}, true},
		{"429 rate limit", &notionapi.Error{Status: 429}, true},
		{"context canceled", context.Canceled, false},
		{"generic error", errors.New("unknown"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			assert.Equal(t, tt.retryable, result)
		})
	}
}

func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		isNetwork bool
	}{
		{"nil error", nil, false},
		{"net.Error timeout", &mockNetError{msg: "network error", timeout: true}, true},
		{"net.Error temporary", &mockNetError{msg: "network error", temporary: true}, true},
		{"connection refused", errors.New("connection refused"), true},
		{"no such host", errors.New("no such host"), true},
		{"timeout", errors.New("request timeout"), true},
		{"deadline exceeded", errors.New("context deadline exceeded"), true},
		{"generic error", errors.New("something else"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNetworkError(tt.err)
			assert.Equal(t, tt.isNetwork, result)
		})
	}
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		isAuth bool
	}{
		{"nil error", nil, false},
		{"401 unauthorized", &notionapi.Error{Status: 401}, true},
		{"403 forbidden", &notionapi.Error{Status: 403}, true},
		{"404 not found", &notionapi.Error{Status: 404}, false},
		{"string 401", errors.New("401 unauthorized"), true},
		{"string forbidden", errors.New("access forbidden"), true},
		{"generic error", errors.New("something else"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAuthError(tt.err)
			assert.Equal(t, tt.isAuth, result)
		})
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		isNotFound bool
	}{
		{"nil error", nil, false},
		{"404 not found", &notionapi.Error{Status: 404}, true},
		{"401 unauthorized", &notionapi.Error{Status: 401}, false},
		{"string 404", errors.New("404 page not found"), true},
		{"string not found", errors.New("resource not found"), true},
		{"generic error", errors.New("something else"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotFoundError(tt.err)
			assert.Equal(t, tt.isNotFound, result)
		})
	}
}

func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		isRateLimit bool
	}{
		{"nil error", nil, false},
		{"429 rate limit", &notionapi.Error{Status: 429}, true},
		{"404 not found", &notionapi.Error{Status: 404}, false},
		{"string 429", errors.New("429 too many requests"), true},
		{"string rate limit", errors.New("rate limit exceeded"), true},
		{"generic error", errors.New("something else"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRateLimitError(tt.err)
			assert.Equal(t, tt.isRateLimit, result)
		})
	}
}

func TestIsServerError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		isServer bool
	}{
		{"nil error", nil, false},
		{"500 internal server error", &notionapi.Error{Status: 500}, true},
		{"502 bad gateway", &notionapi.Error{Status: 502}, true},
		{"503 service unavailable", &notionapi.Error{Status: 503}, true},
		{"504 gateway timeout", &notionapi.Error{Status: 504}, true},
		{"404 not found", &notionapi.Error{Status: 404}, false},
		{"string 500", errors.New("500 internal server error"), true},
		{"string 503", errors.New("503 service unavailable"), true},
		{"generic error", errors.New("something else"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsServerError(tt.err)
			assert.Equal(t, tt.isServer, result)
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"hello world", "world", true},
		{"hello world", "WORLD", true}, // Case insensitive
		{"hello world", "xyz", false},
		{"Hello World", "hello", true},
		{"", "", true},
		{"test", "", true},
		{"", "test", false},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}
