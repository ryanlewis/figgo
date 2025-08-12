package parser

import (
	"bufio"
	"io"
	"sync"
)

const (
	// Buffer pool sizes
	scannerBufferSize    = 64 * 1024       // 64KB per scanner
	maxScannerBufferSize = 4 * 1024 * 1024 // 4MB max
)

// scannerPool manages a pool of scanner buffers to reduce allocations
var scannerPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, scannerBufferSize)
	},
}

// glyphSlicePool manages string slices for glyphs
var glyphSlicePool = sync.Pool{
	New: func() interface{} {
		// Most fonts have height < 20
		s := make([]string, 0, 20)
		return &s
	},
}

// glyphTrimPool manages GlyphTrim slices
var glyphTrimPool = sync.Pool{
	New: func() interface{} {
		// Most fonts have height < 20
		s := make([]GlyphTrim, 0, 20)
		return &s
	},
}

// stringSlicePool for comment lines
var stringSlicePool = sync.Pool{
	New: func() interface{} {
		// Most fonts have < 10 comment lines
		s := make([]string, 0, 10)
		return &s
	},
}

// warningsPool for warning messages
var warningsPool = sync.Pool{
	New: func() interface{} {
		s := make([]string, 0, 5)
		return &s
	},
}

// acquireScannerBuffer gets a buffer for the scanner from the pool
func acquireScannerBuffer() []byte {
	buf := scannerPool.Get().([]byte)
	return buf[:0] // Reset length but keep capacity
}

// releaseScannerBuffer returns a scanner buffer to the pool
func releaseScannerBuffer(buf []byte) {
	if buf == nil || cap(buf) < scannerBufferSize/2 {
		return // Don't pool small buffers
	}
	// Only pool if not too large (prevent memory bloat)
	if cap(buf) <= maxScannerBufferSize {
		scannerPool.Put(buf[:0])
	}
}

// createPooledScanner creates a scanner with a pooled buffer
func createPooledScanner(r io.Reader) (*bufio.Scanner, []byte) {
	scanner := bufio.NewScanner(r)
	buf := acquireScannerBuffer()

	// Set the buffer for the scanner
	scanner.Buffer(buf, maxScannerBufferSize)

	return scanner, buf
}

// acquireGlyphSlice gets a string slice for glyph data from the pool
func acquireGlyphSlice(capacity int) []string {
	slicePtr := glyphSlicePool.Get().(*[]string)
	slice := *slicePtr

	// Ensure capacity
	if cap(slice) < capacity {
		slice = make([]string, 0, capacity)
	} else {
		slice = slice[:0]
	}

	return slice
}

// releaseGlyphSlice returns a glyph slice to the pool
func releaseGlyphSlice(slice []string) {
	if slice == nil || cap(slice) < 10 {
		return // Don't pool small slices
	}

	// Clear references to help GC
	for i := range slice {
		slice[i] = ""
	}

	// Reset and return to pool
	slice = slice[:0]
	glyphSlicePool.Put(&slice)
}

// acquireGlyphTrimSlice gets a GlyphTrim slice from the pool
func acquireGlyphTrimSlice(capacity int) []GlyphTrim {
	slicePtr := glyphTrimPool.Get().(*[]GlyphTrim)
	slice := *slicePtr

	// Ensure capacity
	if cap(slice) < capacity {
		slice = make([]GlyphTrim, 0, capacity)
	} else {
		slice = slice[:0]
	}

	return slice
}

// releaseGlyphTrimSlice returns a GlyphTrim slice to the pool
func releaseGlyphTrimSlice(slice []GlyphTrim) {
	if slice == nil || cap(slice) < 10 {
		return // Don't pool small slices
	}

	// Reset and return to pool
	slice = slice[:0]
	glyphTrimPool.Put(&slice)
}

// acquireStringSlice gets a string slice from the pool
func acquireStringSlice(capacity int) []string {
	slicePtr := stringSlicePool.Get().(*[]string)
	slice := *slicePtr

	// Ensure capacity
	if cap(slice) < capacity {
		slice = make([]string, 0, capacity)
	} else {
		slice = slice[:0]
	}

	return slice
}

// releaseStringSlice returns a string slice to the pool
func releaseStringSlice(slice []string) {
	if slice == nil || cap(slice) < 5 {
		return // Don't pool small slices
	}

	// Clear references to help GC
	for i := range slice {
		slice[i] = ""
	}

	// Reset and return to pool
	slice = slice[:0]
	stringSlicePool.Put(&slice)
}

// acquireWarnings gets a warnings slice from the pool
func acquireWarnings() []string {
	slicePtr := warningsPool.Get().(*[]string)
	slice := *slicePtr
	return slice[:0]
}

// releaseWarnings returns a warnings slice to the pool
func releaseWarnings(slice []string) {
	if slice == nil {
		return
	}

	// Clear references
	for i := range slice {
		slice[i] = ""
	}

	// Reset and return to pool
	slice = slice[:0]
	warningsPool.Put(&slice)
}

