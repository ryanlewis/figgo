package figgo

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"sync/atomic"
)

// FontCache provides thread-safe caching of parsed fonts for long-running applications.
// The cache uses a simple LRU eviction policy when the maximum size is reached.
type FontCache struct {
	mu        sync.RWMutex
	fonts     map[string]*cacheEntry
	lru       *lruList
	maxSize   int
	hits      atomic.Uint64
	misses    atomic.Uint64
	evictions atomic.Uint64
}

type cacheEntry struct {
	key     string
	font    *Font
	size    int64 // Approximate memory size in bytes
	lruNode *lruNode
}

type lruNode struct {
	key  string
	prev *lruNode
	next *lruNode
}

type lruList struct {
	head *lruNode
	tail *lruNode
	size int
}

// Global default cache for convenience
var defaultCache = NewFontCache(100)

// NewFontCache creates a new font cache with the specified maximum number of fonts.
// A maxSize of 0 or negative means unlimited cache size.
func NewFontCache(maxSize int) *FontCache {
	return &FontCache{
		fonts:   make(map[string]*cacheEntry),
		lru:     &lruList{},
		maxSize: maxSize,
	}
}

// LoadFontCached loads a font from the filesystem with caching.
// If the font has been loaded before, it returns the cached version.
// This is safe for concurrent use.
func LoadFontCached(path string) (*Font, error) {
	return defaultCache.LoadFont(path)
}

// LoadFont loads a font from the filesystem with caching.
func (c *FontCache) LoadFont(path string) (*Font, error) {
	// Check cache first
	if font := c.get(path); font != nil {
		return font, nil
	}

	// Load the font
	font, err := LoadFont(path)
	if err != nil {
		c.misses.Add(1)
		return nil, err
	}

	// Cache it
	c.put(path, font)
	return font, nil
}

// ParseFontCached parses a font from byte data with caching.
// The cache key is derived from the SHA256 hash of the data.
func ParseFontCached(data []byte) (*Font, error) {
	return defaultCache.ParseFont(data)
}

// ParseFont parses a font from byte data with caching.
func (c *FontCache) ParseFont(data []byte) (*Font, error) {
	// Generate cache key from content hash
	hash := sha256.Sum256(data)
	key := "sha256:" + hex.EncodeToString(hash[:])

	// Check cache first
	if font := c.get(key); font != nil {
		return font, nil
	}

	// Parse the font
	font, err := ParseFontBytes(data)
	if err != nil {
		c.misses.Add(1)
		return nil, err
	}

	// Cache it
	c.put(key, font)
	return font, nil
}

// get retrieves a font from the cache
func (c *FontCache) get(key string) *Font {
	c.mu.RLock()
	entry, exists := c.fonts[key]
	c.mu.RUnlock()

	if !exists {
		c.misses.Add(1)
		return nil
	}

	// Update LRU position
	c.mu.Lock()
	c.lru.moveToFront(entry.lruNode)
	c.mu.Unlock()

	c.hits.Add(1)
	return entry.font
}

// put adds a font to the cache
func (c *FontCache) put(key string, font *Font) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if already exists
	if _, exists := c.fonts[key]; exists {
		return
	}

	// Evict if necessary
	if c.maxSize > 0 && len(c.fonts) >= c.maxSize {
		c.evictLRU()
	}

	// Add to cache
	node := c.lru.pushFront(key)
	c.fonts[key] = &cacheEntry{
		key:     key,
		font:    font,
		size:    estimateFontSize(font),
		lruNode: node,
	}
}

// evictLRU removes the least recently used font from the cache
func (c *FontCache) evictLRU() {
	if c.lru.tail == nil {
		return
	}

	key := c.lru.tail.key
	delete(c.fonts, key)
	c.lru.remove(c.lru.tail)
	c.evictions.Add(1)
}

// Clear removes all fonts from the cache
func (c *FontCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.fonts = make(map[string]*cacheEntry)
	c.lru = &lruList{}
}

// Stats returns cache statistics
func (c *FontCache) Stats() CacheStats {
	c.mu.RLock()
	size := len(c.fonts)
	c.mu.RUnlock()

	return CacheStats{
		Size:      size,
		MaxSize:   c.maxSize,
		Hits:      c.hits.Load(),
		Misses:    c.misses.Load(),
		Evictions: c.evictions.Load(),
	}
}

// CacheStats contains cache performance statistics
type CacheStats struct {
	Size      int    // Current number of cached fonts
	MaxSize   int    // Maximum cache size
	Hits      uint64 // Number of cache hits
	Misses    uint64 // Number of cache misses
	Evictions uint64 // Number of evictions
}

// HitRate returns the cache hit rate as a percentage (0-100)
func (s CacheStats) HitRate() float64 {
	total := s.Hits + s.Misses
	if total == 0 {
		return 0
	}
	return float64(s.Hits) * 100 / float64(total)
}

// estimateFontSize estimates the memory size of a font in bytes
func estimateFontSize(f *Font) int64 {
	if f == nil {
		return 0
	}

	// Base struct size
	size := int64(100) // Approximate base struct overhead

	// Add glyph data size
	for _, glyph := range f.glyphs {
		for _, line := range glyph {
			size += int64(len(line))
		}
		size += int64(len(glyph) * 8) // Slice overhead
	}

	// Add map overhead (rough estimate)
	size += int64(len(f.glyphs) * 40)

	return size
}

// LRU list operations
func (l *lruList) pushFront(key string) *lruNode {
	node := &lruNode{key: key}

	if l.head == nil {
		l.head = node
		l.tail = node
	} else {
		node.next = l.head
		l.head.prev = node
		l.head = node
	}

	l.size++
	return node
}

func (l *lruList) moveToFront(node *lruNode) {
	if node == l.head {
		return
	}

	// Remove from current position
	if node.prev != nil {
		node.prev.next = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	}
	if node == l.tail {
		l.tail = node.prev
	}

	// Move to front
	node.prev = nil
	node.next = l.head
	l.head.prev = node
	l.head = node
}

func (l *lruList) remove(node *lruNode) {
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		l.head = node.next
	}

	if node.next != nil {
		node.next.prev = node.prev
	} else {
		l.tail = node.prev
	}

	l.size--
}

// SetDefaultCacheSize sets the maximum size of the default cache.
// This should be called once at application startup.
func SetDefaultCacheSize(maxSize int) {
	defaultCache = NewFontCache(maxSize)
}

// ClearDefaultCache clears the default font cache.
func ClearDefaultCache() {
	defaultCache.Clear()
}

// DefaultCacheStats returns statistics for the default cache.
func DefaultCacheStats() CacheStats {
	return defaultCache.Stats()
}

