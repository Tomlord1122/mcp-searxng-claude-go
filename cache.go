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
	cleanupTicker *time.Ticker
	stopCleanup   chan bool
}

func NewCache(ttlSeconds int) *Cache {
	c := &Cache{
		data:        make(map[string]CacheEntry),
		ttl:         time.Duration(ttlSeconds) * time.Second,
		stopCleanup: make(chan bool),
	}

	// Start cleanup goroutine
	c.cleanupTicker = time.NewTicker(30 * time.Second)
	go c.cleanup()

	return c
}

func (c *Cache) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = CacheEntry{
		Value:  value,
		Expiry: time.Now().Add(c.ttl),
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
		"size": len(c.data),
		"ttl":  c.ttl.Seconds(),
	}
}
