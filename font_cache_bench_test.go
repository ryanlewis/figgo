package figgo

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkFontCache benchmarks the font caching mechanisms
func BenchmarkFontCache(b *testing.B) {
	// Create a temporary font file for testing
	tmpDir := b.TempDir()
	fontPath := filepath.Join(tmpDir, "test.flf")

	fontData := []byte(`flf2a$ 8 6 14 0 3
Test font for benchmarking
Created for cache testing
No commercial use

@@
@@
@@
@@
@@
@@
@@
@@
@@
@@
@@
@@
@@
@@
@@
@@`)

	err := os.WriteFile(fontPath, fontData, 0644)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("LoadWithoutCache", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := LoadFont(fontPath)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("LoadWithCache", func(b *testing.B) {
		cache := NewFontCache(10)
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := cache.LoadFont(fontPath)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ParseBytesWithoutCache", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := ParseFontBytes(fontData)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ParseBytesWithCache", func(b *testing.B) {
		cache := NewFontCache(10)
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := cache.ParseFont(fontData)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkCacheHitRate benchmarks cache performance with different hit rates
func BenchmarkCacheHitRate(b *testing.B) {
	cache := NewFontCache(5)

	// Create test font data
	fonts := make([][]byte, 10)
	for i := 0; i < 10; i++ {
		fonts[i] = []byte(fmt.Sprintf(`flf2a$ 8 6 14 0 0
Font %d
        @@
        @@
        @@
        @@
        @@
        @@
        @@
        @@`, i))
	}

	b.Run("HighHitRate", func(b *testing.B) {
		b.ReportAllocs()

		// Pre-warm cache with first 3 fonts
		for i := 0; i < 3; i++ {
			cache.ParseFont(fonts[i])
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Access mostly cached fonts (90% hit rate)
			idx := i % 10
			if idx >= 7 {
				idx = idx % 3
			}
			cache.ParseFont(fonts[idx])
		}

		stats := cache.Stats()
		b.ReportMetric(stats.HitRate(), "hit_rate_%")
	})

	b.Run("LowHitRate", func(b *testing.B) {
		cache.Clear()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Access all 10 fonts cyclically (cache size is 5)
			idx := i % 10
			cache.ParseFont(fonts[idx])
		}

		stats := cache.Stats()
		b.ReportMetric(stats.HitRate(), "hit_rate_%")
	})
}

// BenchmarkCacheConcurrent benchmarks concurrent cache access
func BenchmarkCacheConcurrent(b *testing.B) {
	cache := NewFontCache(20)

	// Create test fonts
	fonts := make([][]byte, 5)
	for i := 0; i < 5; i++ {
		fonts[i] = []byte(fmt.Sprintf(`flf2a$ 8 6 14 0 0
Font %d
        @@
        @@
        @@
        @@
        @@
        @@
        @@
        @@`, i))
	}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			idx := i % 5
			cache.ParseFont(fonts[idx])
			i++
		}
	})

	stats := cache.Stats()
	b.ReportMetric(stats.HitRate(), "hit_rate_%")
	b.ReportMetric(float64(stats.Evictions), "evictions")
}

// BenchmarkLRUEviction benchmarks the LRU eviction performance
func BenchmarkLRUEviction(b *testing.B) {
	cache := NewFontCache(3) // Small cache to trigger evictions

	// Create test fonts
	fonts := make([][]byte, 10)
	for i := 0; i < 10; i++ {
		fonts[i] = []byte(fmt.Sprintf(`flf2a$ 8 6 14 0 0
Font %d
        @@
        @@
        @@
        @@
        @@
        @@
        @@
        @@`, i))
	}

	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Access fonts in a pattern that triggers evictions
		idx := i % 10
		cache.ParseFont(fonts[idx])
	}

	stats := cache.Stats()
	b.ReportMetric(float64(stats.Evictions), "evictions")
}
