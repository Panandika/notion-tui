package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// PageCache provides file-based caching for Notion pages with TTL support.
type PageCache struct {
	cacheDir string
	mu       sync.Mutex
	stats    CacheStats
}

// NewPageCacheInput contains the parameters for creating a new PageCache.
type NewPageCacheInput struct {
	Dir string
}

// NewPageCache creates a new PageCache instance and ensures the cache directory exists.
func NewPageCache(input NewPageCacheInput) (*PageCache, error) {
	if input.Dir == "" {
		return nil, fmt.Errorf("cache directory cannot be empty")
	}

	if err := os.MkdirAll(input.Dir, 0700); err != nil {
		return nil, fmt.Errorf("create cache directory %s: %w", input.Dir, err)
	}

	return &PageCache{
		cacheDir: input.Dir,
		stats:    CacheStats{},
	}, nil
}

// Get retrieves cached data for a given page ID.
// Returns the cached data if valid, or an error if not found or expired.
func (c *PageCache) Get(ctx context.Context, pageID string) (interface{}, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context error: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	cachePath := makeCachePath(c.cacheDir, pageID)

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			c.stats.MissCount++
			return nil, fmt.Errorf("cache miss for page %s: %w", pageID, err)
		}
		return nil, fmt.Errorf("read cache file %s: %w", cachePath, err)
	}

	var entry CacheEntry
	if err := entry.Unmarshal(data); err != nil {
		c.stats.MissCount++
		return nil, fmt.Errorf("unmarshal cache entry for page %s: %w", pageID, err)
	}

	if c.IsExpired(&entry) {
		c.stats.MissCount++
		return nil, fmt.Errorf("cache entry expired for page %s", pageID)
	}

	var result interface{}
	if err := json.Unmarshal(entry.Data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal cached data for page %s: %w", pageID, err)
	}

	c.stats.HitCount++
	return result, nil
}

// SetInput contains the parameters for caching data.
type SetInput struct {
	PageID string
	Data   interface{}
	TTL    time.Duration
}

// Set stores data in the cache with the specified TTL.
func (c *PageCache) Set(ctx context.Context, input SetInput) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context error: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	dataBytes, err := json.Marshal(input.Data)
	if err != nil {
		return fmt.Errorf("marshal data for page %s: %w", input.PageID, err)
	}

	hash := sha256.Sum256(dataBytes)
	hashStr := hex.EncodeToString(hash[:])

	entry := CacheEntry{
		PageID:    input.PageID,
		Data:      dataBytes,
		Timestamp: time.Now(),
		TTL:       input.TTL,
		Hash:      hashStr,
	}

	entryBytes, err := entry.Marshal()
	if err != nil {
		return fmt.Errorf("marshal cache entry for page %s: %w", input.PageID, err)
	}

	cachePath := makeCachePath(c.cacheDir, input.PageID)

	if err := os.WriteFile(cachePath, entryBytes, 0600); err != nil {
		return fmt.Errorf("write cache file %s: %w", cachePath, err)
	}

	c.stats.Size += int64(len(entryBytes))

	return nil
}

// Delete removes a specific cache entry.
func (c *PageCache) Delete(pageID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cachePath := makeCachePath(c.cacheDir, pageID)

	info, err := os.Stat(cachePath)
	if err == nil {
		c.stats.Size -= info.Size()
	}

	if err := os.Remove(cachePath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("delete cache file %s: %w", cachePath, err)
	}

	return nil
}

// Clear removes all cache entries from the cache directory.
func (c *PageCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entries, err := os.ReadDir(c.cacheDir)
	if err != nil {
		return fmt.Errorf("read cache directory %s: %w", c.cacheDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(c.cacheDir, entry.Name())
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove cache file %s: %w", path, err)
		}
	}

	c.stats.Size = 0

	return nil
}

// Stats returns the current cache statistics.
func (c *PageCache) Stats() CacheStats {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.stats
}

// IsExpired checks if a cache entry has exceeded its TTL.
func (c *PageCache) IsExpired(entry *CacheEntry) bool {
	if entry.TTL <= 0 {
		return false
	}
	return time.Since(entry.Timestamp) > entry.TTL
}

// makeCachePath generates the file path for a cached page.
func makeCachePath(dir, pageID string) string {
	safeID := hex.EncodeToString([]byte(pageID))
	return filepath.Join(dir, safeID+".json")
}
