package figgo

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

func contentHash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// FontCache provides thread-safe LRU caching of parsed fonts.
// Path-based keys use the file path directly; byte-based keys use a
// "sha256:" prefixed content hash to avoid collisions.
type FontCache struct {
	mu        sync.RWMutex
	fonts     map[string]*cacheEntry
	lru       *lruList
	maxSize   int
	disk      *diskCache // nil when disk caching is not enabled
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

// CacheOption configures optional FontCache behavior.
type CacheOption func(*cacheConfig)

type cacheConfig struct {
	diskCacheCfg *DiskCacheConfig
}

// WithDiskCache enables an on-disk binary cache that persists parsed fonts
// across process restarts. The disk cache acts as an L2 behind the in-memory LRU.
func WithDiskCache(cfg DiskCacheConfig) CacheOption {
	return func(c *cacheConfig) {
		c.diskCacheCfg = &cfg
	}
}

var defaultCache = NewFontCache(100)

// NewFontCache creates a new font cache with the specified maximum number of fonts.
// A maxSize of 0 or negative means unlimited cache size.
// Optional CacheOption values configure additional behavior (e.g., disk caching).
func NewFontCache(maxSize int, opts ...CacheOption) *FontCache {
	var cfg cacheConfig
	for _, o := range opts {
		o(&cfg)
	}

	fc := &FontCache{
		fonts:   make(map[string]*cacheEntry),
		lru:     &lruList{},
		maxSize: maxSize,
	}

	if cfg.diskCacheCfg != nil {
		fc.disk = newDiskCache(*cfg.diskCacheCfg)
	}

	return fc
}

// LoadFontCached loads a font from the filesystem with caching.
// If the font has been loaded before, it returns the cached version.
// This is safe for concurrent use.
func LoadFontCached(path string) (*Font, error) {
	return defaultCache.LoadFont(path)
}

// LoadFont loads a font from the filesystem with caching.
// This method is safe for concurrent use.
func (c *FontCache) LoadFont(path string) (*Font, error) {
	if font, ok := c.lookup(path); ok {
		return font, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		c.misses.Add(1)
		return nil, err
	}

	if c.disk != nil {
		hash := contentHash(data)
		if font := c.disk.get(hash); font != nil {
			if font.Name == "" {
				font.Name = filepath.Base(path)
			}
			c.put(path, font)
			c.hits.Add(1)
			return font, nil
		}
	}

	font, err := ParseFontBytes(data)
	if err != nil {
		c.misses.Add(1)
		return nil, err
	}
	font.Name = filepath.Base(path)

	c.put(path, font)

	if c.disk != nil {
		hash := contentHash(data)
		c.disk.put(hash, font)
	}

	return font, nil
}

// ParseFontCached parses a font from byte data with caching.
// The cache key is derived from the SHA256 hash of the data.
// This function is safe for concurrent use.
func ParseFontCached(data []byte) (*Font, error) {
	return defaultCache.ParseFont(data)
}

// ParseFont parses a font from byte data with caching.
// Uses SHA256 content hash as the cache key. The "sha256:" prefix
// distinguishes content keys from path keys.
// This method is safe for concurrent use.
func (c *FontCache) ParseFont(data []byte) (*Font, error) {
	hash := contentHash(data)
	key := "sha256:" + hash

	if font, ok := c.lookup(key); ok {
		return font, nil
	}

	if c.disk != nil {
		if font := c.disk.get(hash); font != nil {
			c.put(key, font)
			c.hits.Add(1)
			return font, nil
		}
	}

	font, err := ParseFontBytes(data)
	if err != nil {
		c.misses.Add(1)
		return nil, err
	}

	c.put(key, font)

	if c.disk != nil {
		c.disk.put(hash, font)
	}

	return font, nil
}

// lookup checks the in-memory cache. Increments hits on success; callers handle misses.
func (c *FontCache) lookup(key string) (*Font, bool) {
	c.mu.RLock()
	entry, exists := c.fonts[key]
	c.mu.RUnlock()

	if !exists {
		return nil, false
	}

	c.mu.Lock()
	c.lru.moveToFront(entry.lruNode)
	c.mu.Unlock()

	c.hits.Add(1)
	return entry.font, true
}

func (c *FontCache) put(key string, font *Font) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.fonts[key]; exists {
		return
	}

	if c.maxSize > 0 && len(c.fonts) >= c.maxSize {
		c.evictLRU()
	}

	node := c.lru.pushFront(key)
	c.fonts[key] = &cacheEntry{
		key:     key,
		font:    font,
		size:    estimateFontSize(font),
		lruNode: node,
	}
}

func (c *FontCache) evictLRU() {
	if c.lru.tail == nil {
		return
	}

	key := c.lru.tail.key
	delete(c.fonts, key)
	c.lru.remove(c.lru.tail)
	c.evictions.Add(1)
}

// Clear removes all fonts from both memory and disk caches.
// This method is safe for concurrent use.
func (c *FontCache) Clear() {
	c.mu.Lock()
	c.fonts = make(map[string]*cacheEntry)
	c.lru = &lruList{}
	c.mu.Unlock()

	if c.disk != nil {
		c.disk.clear()
	}
}

// Stats returns cache statistics.
// This method is safe for concurrent use.
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

// estimateFontSize returns an approximate byte size for cache capacity decisions.
// Intentionally underestimates (ignores pointer/header overhead).
func estimateFontSize(f *Font) int64 {
	if f == nil {
		return 0
	}

	size := int64(100)
	for _, glyph := range f.glyphs {
		for _, line := range glyph {
			size += int64(len(line))
		}
		size += int64(len(glyph) * 8)
	}
	size += int64(len(f.glyphs) * 40)

	return size
}

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

	if node.prev != nil {
		node.prev.next = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	}
	if node == l.tail {
		l.tail = node.prev
	}

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

// EnableDefaultDiskCache enables on-disk caching for the default font cache.
// This replaces the default cache with one backed by a disk cache.
func EnableDefaultDiskCache(cfg DiskCacheConfig) {
	defaultCache = NewFontCache(100, WithDiskCache(cfg))
}
