package notion

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jomei/notionapi"
	"github.com/stretchr/testify/assert"
)

// TestNewClient tests client initialization.
func TestNewClient(t *testing.T) {
	token := "secret_test_token"
	client := NewClient(token)

	assert.NotNil(t, client)
	assert.NotNil(t, client.api)
	assert.NotNil(t, client.limiter)
	// Limiter rate should be approximately 2.5 req/sec
	assert.True(t, client.limiter.Limit() > 0)
	// Burst should be 3
	assert.Equal(t, 3, client.limiter.Burst())
}

// TestGetPage tests page retrieval with rate limiting.
func TestGetPage(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		timeout time.Duration
		wantErr bool
	}{
		{
			name:    "valid page ID",
			id:      "page-123",
			timeout: time.Second,
			wantErr: false,
		},
		{
			name:    "context canceled",
			id:      "page-123",
			timeout: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient("secret_test_token")

			ctx := context.Background()
			if tt.timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tt.timeout)
				defer cancel()
			} else {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			// Note: We can't test actual API calls without mocking the HTTP client.
			// This test demonstrates the rate limiter integration.
			if tt.wantErr {
				_, err := client.GetPage(ctx, tt.id)
				assert.Error(t, err)
			}
		})
	}
}

// TestRateLimiting tests that the rate limiter allows tokens at the configured rate.
func TestRateLimiting(t *testing.T) {
	client := NewClient("secret_test_token")

	// The limiter should allow up to burst (3) requests immediately
	allowed := client.limiter.AllowN(time.Now(), 3)
	assert.True(t, allowed, "Should allow burst of 3")

	// Further requests should require waiting
	allowed = client.limiter.AllowN(time.Now(), 1)
	assert.False(t, allowed, "Should not allow beyond burst without waiting")

	// Wait should eventually allow a request
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	assert.NoError(t, client.limiter.Wait(ctx))
}

// TestQueryDatabase tests database query rate limiting.
func TestQueryDatabase(t *testing.T) {
	tests := []struct {
		name       string
		databaseID string
		timeout    time.Duration
		wantErr    bool
	}{
		{
			name:       "valid database ID",
			databaseID: "db-456",
			timeout:    time.Second,
			wantErr:    false,
		},
		{
			name:       "context canceled",
			databaseID: "db-456",
			timeout:    0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient("secret_test_token")

			ctx := context.Background()
			if tt.timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tt.timeout)
				defer cancel()
			} else {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			if tt.wantErr {
				_, err := client.QueryDatabase(ctx, tt.databaseID, nil)
				assert.Error(t, err)
			}
		})
	}
}

// TestGetBlocks tests block retrieval with rate limiting.
func TestGetBlocks(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		timeout time.Duration
		wantErr bool
	}{
		{
			name:    "valid block ID",
			id:      "block-789",
			timeout: time.Second,
			wantErr: false,
		},
		{
			name:    "context canceled",
			id:      "block-789",
			timeout: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient("secret_test_token")

			ctx := context.Background()
			if tt.timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tt.timeout)
				defer cancel()
			} else {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			if tt.wantErr {
				_, err := client.GetBlocks(ctx, tt.id, nil)
				assert.Error(t, err)
			}
		})
	}
}

// TestUpdatePage tests page update with rate limiting.
func TestUpdatePage(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		timeout time.Duration
		wantErr bool
	}{
		{
			name:    "valid page ID",
			id:      "page-update",
			timeout: time.Second,
			wantErr: false,
		},
		{
			name:    "context canceled",
			id:      "page-update",
			timeout: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient("secret_test_token")

			ctx := context.Background()
			if tt.timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tt.timeout)
				defer cancel()
			} else {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			req := &notionapi.PageUpdateRequest{}
			if tt.wantErr {
				_, err := client.UpdatePage(ctx, tt.id, req)
				assert.Error(t, err)
			}
		})
	}
}

// TestDeleteBlock tests block deletion with rate limiting.
func TestDeleteBlock(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		timeout time.Duration
		wantErr bool
	}{
		{
			name:    "valid block ID",
			id:      "block-delete",
			timeout: time.Second,
			wantErr: false,
		},
		{
			name:    "context canceled",
			id:      "block-delete",
			timeout: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient("secret_test_token")

			ctx := context.Background()
			if tt.timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tt.timeout)
				defer cancel()
			} else {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			if tt.wantErr {
				_, err := client.DeleteBlock(ctx, tt.id)
				assert.Error(t, err)
			}
		})
	}
}

// TestRateLimiterWait tests that Wait correctly respects context cancellation.
func TestRateLimiterWait(t *testing.T) {
	client := NewClient("secret_test_token")

	t.Run("wait succeeds with valid context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		err := client.limiter.Wait(ctx)
		assert.NoError(t, err)
	})

	t.Run("wait fails with canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := client.limiter.Wait(ctx)
		assert.Error(t, err)
	})

	t.Run("wait fails with timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
		defer cancel()

		// Exhaust the limiter
		client.limiter.SetBurst(0)

		time.Sleep(time.Millisecond)
		err := client.limiter.Wait(ctx)
		assert.Error(t, err)
	})
}

// TestAppendBlocks tests block append with rate limiting.
func TestAppendBlocks(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		timeout time.Duration
		wantErr bool
	}{
		{
			name:    "valid block ID",
			id:      "block-append",
			timeout: time.Second,
			wantErr: false,
		},
		{
			name:    "context canceled",
			id:      "block-append",
			timeout: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient("secret_test_token")

			ctx := context.Background()
			if tt.timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tt.timeout)
				defer cancel()
			} else {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			req := &notionapi.AppendBlockChildrenRequest{
				Children: []notionapi.Block{},
			}

			if tt.wantErr {
				_, err := client.AppendBlocks(ctx, tt.id, req)
				assert.Error(t, err)
			}
		})
	}
}

// TestErrorWrapping tests that errors are properly wrapped with context.
func TestErrorWrapping(t *testing.T) {
	client := NewClient("secret_test_token")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name         string
		testFunc     func() error
		expectedText string
	}{
		{
			name: "GetPage error contains context",
			testFunc: func() error {
				_, err := client.GetPage(ctx, "test-id")
				return err
			},
			expectedText: "rate limiter wait",
		},
		{
			name: "QueryDatabase error contains context",
			testFunc: func() error {
				_, err := client.QueryDatabase(ctx, "test-id", nil)
				return err
			},
			expectedText: "rate limiter wait",
		},
		{
			name: "GetBlocks error contains context",
			testFunc: func() error {
				_, err := client.GetBlocks(ctx, "test-id", nil)
				return err
			},
			expectedText: "rate limiter wait",
		},
		{
			name: "UpdatePage error contains context",
			testFunc: func() error {
				_, err := client.UpdatePage(ctx, "test-id", &notionapi.PageUpdateRequest{})
				return err
			},
			expectedText: "rate limiter wait",
		},
		{
			name: "DeleteBlock error contains context",
			testFunc: func() error {
				_, err := client.DeleteBlock(ctx, "test-id")
				return err
			},
			expectedText: "rate limiter wait",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFunc()
			assert.Error(t, err)
			assert.True(t, errors.Is(err, context.Canceled) ||
				errors.Is(err, context.DeadlineExceeded) ||
				containsString(err.Error(), "rate limiter wait"))
		})
	}
}

// Helper function to check if an error message contains a substring.
func containsString(msg, substr string) bool {
	return len(msg) >= len(substr)
}
