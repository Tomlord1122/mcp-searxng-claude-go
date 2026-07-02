package main

import (
	"sync"
	"time"
)

type CacheEntry struct {
	Value  string
	Expiry time.Time
}

type Cache struct {
	data          map[string]CacheEntry
	mu            sync.RWMutex
	ttl           time.Duration
	maxEntries    int
	cleanupTicker *time.Ticker
	stopCleanup   chan bool
}

// NewCache creates a TTL cache. maxEntries bounds how large the cache
// may grow between cleanup cycles (retention control); pass 0 for no
// cap (not recommended for long-running processes).
func NewCache(ttlSeconds int, maxEntries int) *Cache {
	c := &Cache{
		data:        make(map[string]CacheEntry),
		ttl:         time.Duration(ttlSeconds) * time.Second,
		maxEntries:  maxEntries,
		stopCleanup: make(chan bool),
	}

	// Start cleanup goroutine - runs at 2x TTL to reduce CPU overhead
	// This means each entry is checked ~once during its lifetime
	cleanupInterval := time.Duration(ttlSeconds*2) * time.Second
	c.cleanupTicker = time.NewTicker(cleanupInterval)
	go c.cleanup()

	return c
}

func (c *Cache) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.data[key]; !exists && c.maxEntries > 0 && len(c.data) >= c.maxEntries {
		c.evictOldestLocked()
	}

	c.data[key] = CacheEntry{
		Value:  value,
		Expiry: time.Now().Add(c.ttl),
	}
}

// evictOldestLocked removes the entry with the soonest expiry (i.e. the
// oldest entry, since all entries share the same TTL). Caller must hold
// c.mu (write lock).
func (c *Cache) evictOldestLocked() {
	var oldestKey string
	var oldestExpiry time.Time
	first := true

	for key, entry := range c.data {
		if first || entry.Expiry.Before(oldestExpiry) {
			oldestKey = key
			oldestExpiry = entry.Expiry
			first = false
		}
	}

	if !first {
		delete(c.data, oldestKey)
	}
}

func (c *Cache) Get(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.data[key]
	if !exists {
		return ""
	}

	if time.Now().After(entry.Expiry) {
		// Entry expired
		return ""
	}

	return entry.Value
}

func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]CacheEntry)
}

func (c *Cache) cleanup() {
	for {
		select {
		case <-c.cleanupTicker.C:
			c.mu.Lock()
			now := time.Now()
			for key, entry := range c.data {
				if now.After(entry.Expiry) {
					delete(c.data, key)
				}
			}
			c.mu.Unlock()
		case <-c.stopCleanup:
			return
		}
	}
}

func (c *Cache) Destroy() {
	c.cleanupTicker.Stop()
	c.stopCleanup <- true
	c.Clear()
}

func (c *Cache) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"size":       len(c.data),
		"ttl":        c.ttl.Seconds(),
		"maxEntries": c.maxEntries,
	}
}
