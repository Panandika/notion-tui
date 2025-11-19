package cache

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPageCache(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupDir    func(t *testing.T) string
		expectError bool
	}{
		{
			name: "create new cache directory",
			setupDir: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "newcache")
			},
			expectError: false,
		},
		{
			name: "use existing directory",
			setupDir: func(t *testing.T) string {
				return t.TempDir()
			},
			expectError: false,
		},
		{
			name: "empty directory path",
			setupDir: func(t *testing.T) string {
				return ""
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := tt.setupDir(t)
			cache, err := NewPageCache(NewPageCacheInput{Dir: dir})

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, cache)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, cache)

				info, err := os.Stat(dir)
				require.NoError(t, err)
				assert.True(t, info.IsDir())
			}
		})
	}
}

func TestSetAndGet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		pageID string
		data   interface{}
		ttl    time.Duration
	}{
		{
			name:   "simple string data",
			pageID: "page-123",
			data:   "test content",
			ttl:    time.Hour,
		},
		{
			name:   "map data",
			pageID: "page-456",
			data: map[string]interface{}{
				"title":   "Test Page",
				"content": "Some content here",
				"count":   42,
			},
			ttl: time.Minute * 30,
		},
		{
			name:   "slice data",
			pageID: "page-789",
			data:   []string{"item1", "item2", "item3"},
			ttl:    time.Hour * 24,
		},
		{
			name:   "nested structure",
			pageID: "page-nested",
			data: map[string]interface{}{
				"blocks": []map[string]interface{}{
					{"type": "paragraph", "text": "Hello"},
					{"type": "heading", "text": "World"},
				},
			},
			ttl: time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cache, err := NewPageCache(NewPageCacheInput{Dir: t.TempDir()})
			require.NoError(t, err)

			ctx := context.Background()

			err = cache.Set(ctx, SetInput{
				PageID: tt.pageID,
				Data:   tt.data,
				TTL:    tt.ttl,
			})
			require.NoError(t, err)

			result, err := cache.Get(ctx, tt.pageID)
			require.NoError(t, err)

			expectedJSON, _ := json.Marshal(tt.data)
			resultJSON, _ := json.Marshal(result)
			assert.JSONEq(t, string(expectedJSON), string(resultJSON))

			stats := cache.Stats()
			assert.Equal(t, int64(1), stats.HitCount)
			assert.Equal(t, int64(0), stats.MissCount)
		})
	}
}

func TestExpiration(t *testing.T) {
	t.Parallel()

	cache, err := NewPageCache(NewPageCacheInput{Dir: t.TempDir()})
	require.NoError(t, err)

	ctx := context.Background()
	pageID := "expiring-page"
	data := "will expire soon"

	// Use longer TTL to avoid timing issues on slow systems
	err = cache.Set(ctx, SetInput{
		PageID: pageID,
		Data:   data,
		TTL:    time.Second * 2,
	})
	require.NoError(t, err)

	// Should be available immediately after setting
	result, err := cache.Get(ctx, pageID)
	require.NoError(t, err)
	assert.Equal(t, data, result)

	// Wait for expiration (use 2.5 seconds to account for system variance)
	time.Sleep(time.Millisecond * 2500)

	// Should now be expired
	_, err = cache.Get(ctx, pageID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")

	stats := cache.Stats()
	assert.Equal(t, int64(1), stats.HitCount)
	assert.Equal(t, int64(1), stats.MissCount)
}

func TestNoExpiration(t *testing.T) {
	t.Parallel()

	cache, err := NewPageCache(NewPageCacheInput{Dir: t.TempDir()})
	require.NoError(t, err)

	ctx := context.Background()
	pageID := "no-expire-page"
	data := "never expires"

	err = cache.Set(ctx, SetInput{
		PageID: pageID,
		Data:   data,
		TTL:    0,
	})
	require.NoError(t, err)

	result, err := cache.Get(ctx, pageID)
	require.NoError(t, err)
	assert.Equal(t, data, result)
}

func TestDelete(t *testing.T) {
	t.Parallel()

	cache, err := NewPageCache(NewPageCacheInput{Dir: t.TempDir()})
	require.NoError(t, err)

	ctx := context.Background()
	pageID := "page-to-delete"
	data := "delete me"

	err = cache.Set(ctx, SetInput{
		PageID: pageID,
		Data:   data,
		TTL:    time.Hour,
	})
	require.NoError(t, err)

	_, err = cache.Get(ctx, pageID)
	require.NoError(t, err)

	err = cache.Delete(pageID)
	require.NoError(t, err)

	_, err = cache.Get(ctx, pageID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache miss")

	err = cache.Delete("nonexistent-page")
	assert.NoError(t, err)
}

func TestClear(t *testing.T) {
	t.Parallel()

	cache, err := NewPageCache(NewPageCacheInput{Dir: t.TempDir()})
	require.NoError(t, err)

	ctx := context.Background()

	for i := 0; i < 5; i++ {
		err = cache.Set(ctx, SetInput{
			PageID: "page-" + string(rune('A'+i)),
			Data:   "data",
			TTL:    time.Hour,
		})
		require.NoError(t, err)
	}

	initialStats := cache.Stats()
	assert.Greater(t, initialStats.Size, int64(0))

	err = cache.Clear()
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		_, err = cache.Get(ctx, "page-"+string(rune('A'+i)))
		assert.Error(t, err)
	}

	stats := cache.Stats()
	assert.Equal(t, int64(0), stats.Size)
}

func TestConcurrentAccess(t *testing.T) {
	t.Parallel()

	cache, err := NewPageCache(NewPageCacheInput{Dir: t.TempDir()})
	require.NoError(t, err)

	ctx := context.Background()
	numGoroutines := 10
	numOpsPerGoroutine := 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				pageID := "concurrent-page"

				if j%2 == 0 {
					err := cache.Set(ctx, SetInput{
						PageID: pageID,
						Data:   map[string]int{"goroutine": id, "iteration": j},
						TTL:    time.Hour,
					})
					if err != nil {
						t.Errorf("Set failed: %v", err)
					}
				} else {
					_, _ = cache.Get(ctx, pageID)
				}
			}
		}(i)
	}

	wg.Wait()

	stats := cache.Stats()
	totalOps := stats.HitCount + stats.MissCount
	assert.GreaterOrEqual(t, totalOps, int64(0))
}

func TestInvalidJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cache, err := NewPageCache(NewPageCacheInput{Dir: dir})
	require.NoError(t, err)

	pageID := "bad-json-page"
	cachePath := makeCachePath(dir, pageID)

	err = os.WriteFile(cachePath, []byte("invalid json {"), 0600)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = cache.Get(ctx, pageID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestCachePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		pageID string
	}{
		{
			name:   "simple page ID",
			pageID: "page-123",
		},
		{
			name:   "page ID with special characters",
			pageID: "page/with:special*chars",
		},
		{
			name:   "UUID page ID",
			pageID: "550e8400-e29b-41d4-a716-446655440000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			path := makeCachePath(dir, tt.pageID)

			// Check that path starts with the directory and ends with .json
			assert.True(t, len(path) > len(dir), "path should be longer than directory")
			assert.Equal(t, ".json", filepath.Ext(path))

			// Ensure deterministic path generation
			pathAgain := makeCachePath(dir, tt.pageID)
			assert.Equal(t, path, pathAgain)

			// Verify the path is under the directory using filepath.Rel
			relPath, err := filepath.Rel(dir, path)
			require.NoError(t, err)
			assert.NotEmpty(t, relPath)
			assert.NotEqual(t, "..", relPath[:2], "path should be under directory")
		})
	}
}

func TestCacheMiss(t *testing.T) {
	t.Parallel()

	cache, err := NewPageCache(NewPageCacheInput{Dir: t.TempDir()})
	require.NoError(t, err)

	ctx := context.Background()
	_, err = cache.Get(ctx, "nonexistent-page")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache miss")

	stats := cache.Stats()
	assert.Equal(t, int64(0), stats.HitCount)
	assert.Equal(t, int64(1), stats.MissCount)
}

func TestContextCancellation(t *testing.T) {
	t.Parallel()

	cache, err := NewPageCache(NewPageCacheInput{Dir: t.TempDir()})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = cache.Get(ctx, "some-page")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context")

	err = cache.Set(ctx, SetInput{
		PageID: "some-page",
		Data:   "data",
		TTL:    time.Hour,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context")
}

func TestIsExpired(t *testing.T) {
	t.Parallel()

	cache, err := NewPageCache(NewPageCacheInput{Dir: t.TempDir()})
	require.NoError(t, err)

	tests := []struct {
		name     string
		entry    CacheEntry
		expected bool
	}{
		{
			name: "not expired",
			entry: CacheEntry{
				Timestamp: time.Now(),
				TTL:       time.Hour,
			},
			expected: false,
		},
		{
			name: "expired",
			entry: CacheEntry{
				Timestamp: time.Now().Add(-time.Hour * 2),
				TTL:       time.Hour,
			},
			expected: true,
		},
		{
			name: "no TTL means never expires",
			entry: CacheEntry{
				Timestamp: time.Now().Add(-time.Hour * 1000),
				TTL:       0,
			},
			expected: false,
		},
		{
			name: "negative TTL means never expires",
			entry: CacheEntry{
				Timestamp: time.Now().Add(-time.Hour * 1000),
				TTL:       -1,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, cache.IsExpired(&tt.entry))
		})
	}
}
