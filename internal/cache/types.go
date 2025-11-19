package cache

import (
	"encoding/json"
	"time"
)

// CacheEntry represents a single cached item with metadata.
type CacheEntry struct {
	PageID    string          `json:"page_id"`
	Data      json.RawMessage `json:"data"`
	Timestamp time.Time       `json:"timestamp"`
	TTL       time.Duration   `json:"ttl"`
	Hash      string          `json:"hash"`
}

// CacheStats provides statistics about cache performance.
type CacheStats struct {
	HitCount  int64 `json:"hit_count"`
	MissCount int64 `json:"miss_count"`
	Size      int64 `json:"size"`
}

// Marshal serializes a CacheEntry to JSON bytes.
func (e *CacheEntry) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// Unmarshal deserializes JSON bytes into a CacheEntry.
func (e *CacheEntry) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

// MarshalJSON implements custom JSON marshaling for CacheEntry to handle time.Duration.
func (e CacheEntry) MarshalJSON() ([]byte, error) {
	type Alias CacheEntry
	return json.Marshal(&struct {
		TTL int64 `json:"ttl"`
		*Alias
	}{
		TTL:   int64(e.TTL),
		Alias: (*Alias)(&e),
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for CacheEntry to handle time.Duration.
func (e *CacheEntry) UnmarshalJSON(data []byte) error {
	type Alias CacheEntry
	aux := &struct {
		TTL int64 `json:"ttl"`
		*Alias
	}{
		Alias: (*Alias)(e),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	e.TTL = time.Duration(aux.TTL)
	return nil
}
