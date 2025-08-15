package renderer

import (
	"sync"
	"unicode/utf8"
)

// Default sizes for buffer allocation
const (
	defaultOutlineLimit = 10000
	defaultMaxHeight    = 20 // Most fonts are under 20 lines tall

	// Buffer retention thresholds - buffers larger than these are released
	// to prevent memory bloat in the pool from occasional large renders
	maxRetainInputBuffer  = 1024 // 4KB for rune slice
	maxRetainOutputBuffer = 8192 // 8KB for byte slice  
	maxRetainOutputLine   = 2000 // ~8KB per line for rune slice
)

// renderStatePool manages a pool of renderState objects to reduce allocations.
//
// Pool Design:
// - Pre-allocates output line slices with reasonable default capacity
// - Reuses expensive renderState objects across multiple render calls
// - Prevents allocation churn in high-throughput rendering scenarios
//
// The renderState contains large rune slice buffers (defaultOutlineLimit = 10K runes)
// that would be expensive to allocate/deallocate repeatedly. Pooling them provides
// significant performance benefits for applications that render many strings.
var renderStatePool = sync.Pool{
	New: func() interface{} {
		return &renderState{
			outputLine: make([][]rune, 0, defaultMaxHeight),
			rowLengths: make([]int, 0, defaultMaxHeight),
		}
	},
}

// tempLinePool manages temporary line buffers for RTL processing
var tempLinePool = sync.Pool{
	New: func() interface{} {
		buf := make([]rune, defaultOutlineLimit)
		return &buf
	},
}

// runeSlicePool manages rune slices for string conversions
var runeSlicePool = sync.Pool{
	New: func() interface{} {
		// Start with reasonable capacity for most glyph lines
		buf := make([]rune, 0, 64)
		return &buf
	},
}

// writeBufferPool manages write buffers for writeTo
var writeBufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 0, 256)
		return &buf
	},
}

// acquireRenderState gets a renderState from the pool and initializes it.
//
// State Initialization Strategy:
// 1. Reuse existing renderState object from pool (avoid struct allocation)
// 2. Resize internal slices only if current capacity is insufficient
// 3. Pre-allocate output line buffers to avoid per-character allocations
// 4. Zero-out existing data for clean state
//
// Buffer Management:
// - outputLine: One rune slice per font height line
// - rowLengths: Tracks actual content length per row (for trimming)
// - Both are sized to defaultOutlineLimit (10,000 runes)
//
// This pooling is essential for rendering performance, as it eliminates
// the major allocation overhead when rendering multiple strings.
func acquireRenderState(height int, hardblank rune, textLen int) *renderState {
	state := renderStatePool.Get().(*renderState)

	// Reset and initialize the state
	state.charHeight = height
	state.hardblank = hardblank
	state.outlineLenLimit = defaultOutlineLimit
	state.outlineLen = 0
	state.currentChar = nil
	state.currentCharWidth = 0
	state.previousCharWidth = 0
	state.right2left = 0
	state.smushMode = 0
	state.trimWhitespace = false

	// Reset new line breaking fields
	state.inputCount = 0
	state.lastWordBreak = -1
	state.wordbreakmode = 0

	// Reset output buffer
	state.outputBuffer = state.outputBuffer[:0]

	// Resize slices if needed (without reallocating if possible)
	if cap(state.outputLine) < height {
		state.outputLine = make([][]rune, height)
		state.rowLengths = make([]int, height)
	} else {
		state.outputLine = state.outputLine[:height]
		state.rowLengths = state.rowLengths[:height]
	}

	// Initialize inputBuffer with appropriate capacity based on text length
	// Use a minimum of 256 to avoid tiny allocations for short text
	capacity := textLen
	if capacity < 256 {
		capacity = 256
	}
	if cap(state.inputBuffer) < capacity {
		state.inputBuffer = make([]rune, 0, capacity)
	} else {
		state.inputBuffer = state.inputBuffer[:0]
	}

	// Initialize output lines with pre-allocated buffers
	for i := 0; i < height; i++ {
		if state.outputLine[i] == nil || cap(state.outputLine[i]) < defaultOutlineLimit {
			state.outputLine[i] = make([]rune, defaultOutlineLimit)
		} else {
			// Clear existing buffer
			for j := range state.outputLine[i] {
				state.outputLine[i][j] = 0
			}
		}
		state.rowLengths[i] = 0
	}

	return state
}

// releaseRenderState returns a renderState to the pool.
// It shrinks oversized buffers to prevent memory bloat from occasional large renders.
func releaseRenderState(state *renderState) {
	if state == nil {
		return
	}

	// Clear references to help GC
	state.currentChar = nil

	// Shrink oversized buffers to prevent memory bloat
	// These will be reallocated at appropriate size when needed
	if cap(state.inputBuffer) > maxRetainInputBuffer {
		state.inputBuffer = nil
	}

	if cap(state.outputBuffer) > maxRetainOutputBuffer {
		state.outputBuffer = nil
	}

	// Check each output line
	for i := range state.outputLine {
		if state.outputLine[i] != nil && cap(state.outputLine[i]) > maxRetainOutputLine {
			state.outputLine[i] = nil
		}
	}

	// Return to pool
	renderStatePool.Put(state)
}

// acquireTempLine gets a temporary line buffer from the pool
func acquireTempLine() []rune {
	bufPtr := tempLinePool.Get().(*[]rune)
	buf := *bufPtr

	// Clear the buffer
	for i := range buf {
		buf[i] = 0
	}

	return buf
}

// releaseTempLine returns a temporary line buffer to the pool
func releaseTempLine(buf []rune) {
	if buf == nil || cap(buf) < defaultOutlineLimit {
		return // Don't pool small or nil buffers
	}
	tempLinePool.Put(&buf)
}

// acquireRuneSlice gets a rune slice from the pool
func acquireRuneSlice() []rune {
	bufPtr := runeSlicePool.Get().(*[]rune)
	buf := *bufPtr
	return buf[:0] // Reset length but keep capacity
}

// releaseRuneSlice returns a rune slice to the pool
func releaseRuneSlice(buf []rune) {
	if buf == nil || cap(buf) < 32 {
		return // Don't pool small buffers
	}
	// Clear references to help GC
	for i := range buf {
		buf[i] = 0
	}
	runeSlicePool.Put(&buf)
}

// acquireWriteBuffer gets a write buffer from the pool
func acquireWriteBuffer() []byte {
	bufPtr := writeBufferPool.Get().(*[]byte)
	buf := *bufPtr
	return buf[:0] // Reset length but keep capacity
}

// releaseWriteBuffer returns a write buffer to the pool
func releaseWriteBuffer(buf []byte) {
	if buf == nil || cap(buf) < 128 {
		return // Don't pool small buffers
	}
	writeBufferPool.Put(&buf)
}

// Lookup tables for smushing rules to avoid repeated string searches
var (
	// Characters that can smush with underscore
	underscoreBorders = map[rune]bool{
		'|': true, '/': true, '\\': true,
		'[': true, ']': true, '{': true, '}': true,
		'(': true, ')': true, '<': true, '>': true,
	}

	// Hierarchy level 1: can be replaced by |
	hierarchyLevel1 = map[rune]bool{
		'/': true, '\\': true,
		'[': true, ']': true, '{': true, '}': true,
		'(': true, ')': true, '<': true, '>': true,
	}

	// Hierarchy level 2: can be replaced by /\
	hierarchyLevel2 = map[rune]bool{
		'[': true, ']': true, '{': true, '}': true,
		'(': true, ')': true, '<': true, '>': true,
	}

	// Hierarchy level 3: can be replaced by []
	hierarchyLevel3 = map[rune]bool{
		'{': true, '}': true,
		'(': true, ')': true, '<': true, '>': true,
	}

	// Hierarchy level 4: can be replaced by {}
	hierarchyLevel4 = map[rune]bool{
		'(': true, ')': true, '<': true, '>': true,
	}

	// Hierarchy level 5: can be replaced by ()
	hierarchyLevel5 = map[rune]bool{
		'<': true, '>': true,
	}
)

// Pre-calculated capacity for strings.Builder based on typical output
func calculateBuilderCapacity(height int, maxWidth int) int {
	// Estimate: height * (maxWidth + 1 for newline)
	// Add some buffer for safety
	capacity := height * (maxWidth + 1)
	if capacity < 1024 {
		capacity = 1024 // Minimum reasonable size
	}
	return capacity
}

// optimizedHardblankReplace replaces hardblanks with spaces more efficiently
func optimizedHardblankReplace(s string, hardblank rune) string {
	if hardblank == ' ' {
		return s // No replacement needed
	}

	// Check if hardblank exists first
	found := false
	for _, r := range s {
		if r == hardblank {
			found = true
			break
		}
	}

	if !found {
		return s // No replacement needed
	}

	// Only allocate if we need to replace
	runes := []rune(s)
	for i, r := range runes {
		if r == hardblank {
			runes[i] = ' '
		}
	}
	return string(runes)
}

// cachedRuneCount caches the rune count for frequently converted strings
type runeCache struct {
	str   string
	runes []rune
	count int
}

// Small cache for current character rows (usually just a few rows)
var runeConversionCache = make([]runeCache, 0, defaultMaxHeight)

// getCachedRunesInto converts string to runes using a provided buffer.
// Returns the number of runes written. Caller must ensure buffer is large enough.
func getCachedRunesInto(s string, buf []rune) int {
	// Check cache first
	for i := range runeConversionCache {
		if runeConversionCache[i].str == s {
			cached := runeConversionCache[i].runes
			copy(buf, cached)
			return len(cached)
		}
	}

	// Convert to runes into buffer
	count := 0
	for _, r := range s {
		if count < len(buf) {
			buf[count] = r
			count++
		} else {
			break // Buffer full
		}
	}

	// Cache the result if it's reasonable size
	if count < 128 && len(runeConversionCache) < cap(runeConversionCache) {
		// Make a copy for cache
		cachedRunes := make([]rune, count)
		copy(cachedRunes, buf[:count])

		runeConversionCache = append(runeConversionCache, runeCache{
			str:   s,
			runes: cachedRunes,
			count: count,
		})
	}

	return count
}

// getCachedRunes converts string to runes, using pool for allocation
func getCachedRunes(s string) []rune {
	// Check cache first
	for i := range runeConversionCache {
		if runeConversionCache[i].str == s {
			return runeConversionCache[i].runes
		}
	}

	// Use pooled buffer for conversion
	buf := acquireRuneSlice()
	defer releaseRuneSlice(buf)

	// Ensure buffer is large enough
	needed := len(s) // Worst case: all ASCII
	if cap(buf) < needed {
		buf = make([]rune, 0, needed)
	}

	// Convert to runes
	for _, r := range s {
		buf = append(buf, r)
	}

	// Make a copy to return (since we'll release the buffer)
	result := make([]rune, len(buf))
	copy(result, buf)

	// Cache if reasonable size
	if len(result) < 128 && len(runeConversionCache) < cap(runeConversionCache) {
		runeConversionCache = append(runeConversionCache, runeCache{
			str:   s,
			runes: result,
			count: len(result),
		})
	}

	return result
}

// getCachedRuneCount returns the rune count for a string, using cache if possible.
//
// Caching Strategy:
// Frequently used glyph rows (like the same characters appearing multiple times)
// benefit from caching their rune counts. This micro-optimization reduces the
// overhead of UTF-8 scanning for repeated strings.
//
// Cache Characteristics:
// - Small cache size (defaultMaxHeight = 20 entries)
// - Only caches strings < 128 runes (avoid caching large strings)
// - Simple linear search (fast for small cache)
// - Falls back to utf8.RuneCountInString for cache misses
//
// This provides measurable performance benefits when rendering the same
// character multiple times in a string.
func getCachedRuneCount(s string) int {
	// Check cache
	for i := range runeConversionCache {
		if runeConversionCache[i].str == s {
			return runeConversionCache[i].count
		}
	}

	// Fall back to standard count
	return utf8.RuneCountInString(s)
}
