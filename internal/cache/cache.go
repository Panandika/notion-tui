package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// Load reads what was stored under a key into out, however old it is. age
// says how long ago it was stored, and fresh whether that is still within
// its time to live. Reading works without a connection, so an entry that
// is no longer fresh is what there is to show when Notion cannot be
// reached.
func (c *PageCache) Load(ctx context.Context, key string, out any) (age time.Duration, fresh bool, err error) {
	if err := ctx.Err(); err != nil {
		return 0, false, fmt.Errorf("context error: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	cachePath := makeCachePath(c.cacheDir, key)
	data, err := os.ReadFile(cachePath)
	if err != nil {
		c.stats.MissCount++
		return 0, false, fmt.Errorf("read cache entry %s: %w", key, err)
	}
	var entry CacheEntry
	if err := entry.Unmarshal(data); err != nil {
		c.stats.MissCount++
		return 0, false, fmt.Errorf("unmarshal cache entry %s: %w", key, err)
	}
	if err := json.Unmarshal(entry.Data, out); err != nil {
		c.stats.MissCount++
		return 0, false, fmt.Errorf("unmarshal cached data %s: %w", key, err)
	}

	c.stats.HitCount++
	return time.Since(entry.Timestamp), !c.IsExpired(&entry), nil
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

	if err := writeFileAtomic(cachePath, entryBytes); err != nil {
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

// writeFileAtomic writes a file under a temporary name and renames it into
// place, so a reader (or a crash) never sees half an entry.
func writeFileAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), tempPrefix+"*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	_, err = tmp.Write(data)
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmpName, path)
	}
	if err != nil {
		_ = os.Remove(tmpName)
	}
	return err
}

// tempPrefix starts the name of a file that is still being written.
const tempPrefix = ".tmp-"

// Prune deletes entries that were last written more than maxAge ago, and
// temporary files left behind by a write that never finished. It returns
// how many files it removed. Only the cache's own files are touched:
// "<hex>.json" entries and its temporary files.
func (c *PageCache) Prune(maxAge time.Duration) (removed int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entries, err := os.ReadDir(c.cacheDir)
	if err != nil {
		return 0, fmt.Errorf("read cache directory %s: %w", c.cacheDir, err)
	}
	cutoff := time.Now().Add(-maxAge)
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !isCacheFile(entry.Name()) {
			continue
		}
		info, statErr := entry.Info()
		if statErr != nil || info.ModTime().After(cutoff) {
			continue
		}
		if os.Remove(filepath.Join(c.cacheDir, entry.Name())) == nil {
			removed++
		}
	}
	return removed, nil
}

// isCacheFile reports whether a file name is one this cache writes.
func isCacheFile(name string) bool {
	if strings.HasPrefix(name, tempPrefix) {
		return true
	}
	stem, isJSON := strings.CutSuffix(name, ".json")
	if !isJSON || stem == "" {
		return false
	}
	_, err := hex.DecodeString(stem)
	return err == nil
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
