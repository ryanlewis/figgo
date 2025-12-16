package figgo

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkPRDTarget_QuickBrownFox validates the PRD p50 latency target.
// Target: < 50µs for "The quick brown fox" with standard.flf
func BenchmarkPRDTarget_QuickBrownFox(b *testing.B) {
	fontPath := filepath.Join("fonts", "standard.flf")
	f, err := os.Open(fontPath)
	if err != nil {
		b.Fatalf("failed to open font: %v", err)
	}
	defer f.Close()

	font, err := ParseFont(f)
	if err != nil {
		b.Fatalf("failed to parse font: %v", err)
	}

	text := "The quick brown fox"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := Render(text, font)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPRDTarget_Allocations validates the allocation target.
// Target: < 4 allocs/op with pooling enabled
func BenchmarkPRDTarget_Allocations(b *testing.B) {
	fontPath := filepath.Join("fonts", "standard.flf")
	f, err := os.Open(fontPath)
	if err != nil {
		b.Fatalf("failed to open font: %v", err)
	}
	defer f.Close()

	font, err := ParseFont(f)
	if err != nil {
		b.Fatalf("failed to parse font: %v", err)
	}

	text := "The quick brown fox"
	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		err := RenderTo(&buf, text, font)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPRDTarget_MergeThruput measures glyph merge throughput.
// Stretch target: ~1M glyph merges/sec with smushing enabled
func BenchmarkPRDTarget_MergeThruput(b *testing.B) {
	fontPath := filepath.Join("fonts", "standard.flf")
	f, err := os.Open(fontPath)
	if err != nil {
		b.Fatalf("failed to open font: %v", err)
	}
	defer f.Close()

	font, err := ParseFont(f)
	if err != nil {
		b.Fatalf("failed to parse font: %v", err)
	}

	// Use text with many characters to measure merge throughput
	text := "The quick brown fox jumps over the lazy dog"
	charCount := len(text) - 1 // merges = characters - 1

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := Render(text, font)
		if err != nil {
			b.Fatal(err)
		}
	}

	// Report merges per second
	mergesPerOp := float64(charCount)
	opsPerSec := float64(b.N) / b.Elapsed().Seconds()
	mergesPerSec := mergesPerOp * opsPerSec
	b.ReportMetric(mergesPerSec, "merges/sec")
}

// BenchmarkPRDTarget_AllFonts tests PRD target across all bundled fonts
func BenchmarkPRDTarget_AllFonts(b *testing.B) {
	fonts := []string{"standard.flf", "slant.flf", "small.flf", "big.flf"}
	text := "The quick brown fox"

	for _, fontName := range fonts {
		b.Run(fontName, func(b *testing.B) {
			fontPath := filepath.Join("fonts", fontName)
			f, err := os.Open(fontPath)
			if err != nil {
				b.Skipf("font not found: %s", fontPath)
				return
			}
			defer f.Close()

			font, err := ParseFont(f)
			if err != nil {
				b.Fatalf("failed to parse font: %v", err)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, _ = Render(text, font)
			}
		})
	}
}

// BenchmarkE2E_ParseAndRender measures the full pipeline without caching
func BenchmarkE2E_ParseAndRender(b *testing.B) {
	fontPath := filepath.Join("fonts", "standard.flf")
	text := "Hello World"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		f, err := os.Open(fontPath)
		if err != nil {
			b.Fatal(err)
		}

		font, err := ParseFont(f)
		f.Close()
		if err != nil {
			b.Fatal(err)
		}

		_, err = Render(text, font)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkE2E_CachedRender measures rendering with pre-parsed font
func BenchmarkE2E_CachedRender(b *testing.B) {
	fontPath := filepath.Join("fonts", "standard.flf")
	f, err := os.Open(fontPath)
	if err != nil {
		b.Fatalf("failed to open font: %v", err)
	}

	font, err := ParseFont(f)
	f.Close()
	if err != nil {
		b.Fatalf("failed to parse font: %v", err)
	}

	text := "Hello World"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err = Render(text, font)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkE2E_FontCacheHit measures performance with font cache
func BenchmarkE2E_FontCacheHit(b *testing.B) {
	cache := NewFontCache(10)
	fontPath := filepath.Join("fonts", "standard.flf")

	// Pre-warm the cache
	_, err := cache.LoadFont(fontPath)
	if err != nil {
		b.Fatalf("failed to load font: %v", err)
	}

	text := "Hello World"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		font, err := cache.LoadFont(fontPath)
		if err != nil {
			b.Fatal(err)
		}

		_, err = Render(text, font)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkE2E_LayoutModes compares performance across layout modes
func BenchmarkE2E_LayoutModes(b *testing.B) {
	fontPath := filepath.Join("fonts", "standard.flf")
	f, err := os.Open(fontPath)
	if err != nil {
		b.Fatalf("failed to open font: %v", err)
	}

	font, err := ParseFont(f)
	f.Close()
	if err != nil {
		b.Fatalf("failed to parse font: %v", err)
	}

	text := "The quick brown fox"

	b.Run("FullWidth", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = Render(text, font, WithLayout(FitFullWidth))
		}
	})

	b.Run("Kerning", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = Render(text, font, WithLayout(FitKerning))
		}
	})

	b.Run("Smushing", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = Render(text, font, WithLayout(FitSmushing))
		}
	})
}
