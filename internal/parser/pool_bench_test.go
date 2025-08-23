package parser

import (
	"bufio"
	"fmt"
	"strings"
	"testing"
)

// createTestFont creates a test font for benchmarking
func createTestFont(glyphCount int) string {
	var sb strings.Builder

	// Header
	sb.WriteString("flf2a$ 8 6 14 0 3\n")
	sb.WriteString("Test font for benchmarking\n")
	sb.WriteString("Created for performance testing\n")
	sb.WriteString("No commercial use\n")

	// Generate glyphs
	for i := 0; i < glyphCount; i++ {
		for j := 0; j < 8; j++ {
			sb.WriteString("  ###  @@\n")
		}
	}

	return sb.String()
}

// BenchmarkParseWithPooling benchmarks parsing with the new pooling optimizations
func BenchmarkParseWithPooling(b *testing.B) {
	testCases := []struct {
		name       string
		glyphCount int
	}{
		{"Small_10", 10},
		{"Medium_50", 50},
		{"Large_100", 100},
		{"VeryLarge_200", 200},
	}

	for _, tc := range testCases {
		fontData := createTestFont(tc.glyphCount)

		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				r := strings.NewReader(fontData)
				_, _ = Parse(r)
			}
		})
	}
}

// BenchmarkParseParallel benchmarks parallel parsing performance
func BenchmarkParseParallel(b *testing.B) {
	fontData := createTestFont(100)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r := strings.NewReader(fontData)
			_, _ = Parse(r)
		}
	})
}

// BenchmarkLazyTrimComputation benchmarks the lazy trim computation
func BenchmarkLazyTrimComputation(b *testing.B) {
	fontData := createTestFont(100)
	r := strings.NewReader(fontData)
	font, err := Parse(r)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("FirstAccess", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Access a character that hasn't been computed yet
			// This simulates worst-case scenario
			testRune := rune(33 + (i % 94))
			_, _ = font.GetCharacterTrims(testRune)
		}
	})

	// Pre-compute all trims
	for r := rune(32); r <= 126; r++ {
		font.GetCharacterTrims(r)
	}

	b.Run("CachedAccess", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Access already computed trims
			testRune := rune(33 + (i % 94))
			_, _ = font.GetCharacterTrims(testRune)
		}
	})
}

// BenchmarkMemoryUsage compares memory usage with different optimization levels
func BenchmarkMemoryUsage(b *testing.B) {
	sizes := []int{10, 50, 100, 200}

	for _, size := range sizes {
		fontData := createTestFont(size)

		b.Run(fmt.Sprintf("Glyphs_%d", size), func(b *testing.B) {
			b.ReportAllocs()

			var totalMem int64
			for i := 0; i < b.N; i++ {
				r := strings.NewReader(fontData)
				font, _ := Parse(r)

				// Estimate memory usage
				if font != nil {
					totalMem += estimateFontMemory(font)
				}
			}

			b.ReportMetric(float64(totalMem)/float64(b.N), "bytes/op")
		})
	}
}

// estimateFontMemory estimates the memory usage of a Font
func estimateFontMemory(f *Font) int64 {
	if f == nil {
		return 0
	}

	var size int64

	// Base struct size
	size += 200 // Approximate

	// Characters map
	for _, glyph := range f.Characters {
		for _, line := range glyph {
			size += int64(len(line))
		}
		size += int64(len(glyph) * 24) // Slice overhead
	}

	// CharacterTrims map (if computed)
	size += int64(len(f.CharacterTrims) * 16) // GlyphTrim size

	// Map overhead
	size += int64(len(f.Characters) * 40)

	return size
}

// BenchmarkScannerBufferReuse benchmarks the scanner buffer pooling
func BenchmarkScannerBufferReuse(b *testing.B) {
	fontData := createTestFont(100)

	b.Run("WithPooling", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			r := strings.NewReader(fontData)
			scanner, buf := createPooledScanner(r)

			// Simulate scanning
			for scanner.Scan() {
				_ = scanner.Text()
			}

			releaseScannerBuffer(buf)
		}
	})

	b.Run("WithoutPooling", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			r := strings.NewReader(fontData)
			scanner := bufio.NewScanner(r)
			scanner.Buffer(make([]byte, 0, defaultBufferSize), maxBufferSize)

			// Simulate scanning
			for scanner.Scan() {
				_ = scanner.Text()
			}
		}
	})
}

// BenchmarkGlyphSlicePooling benchmarks the glyph slice pooling
func BenchmarkGlyphSlicePooling(b *testing.B) {
	b.Run("WithPooling", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			slice := acquireGlyphSlice(20)

			// Simulate usage
			for j := 0; j < 20; j++ {
				slice = append(slice, "test line")
			}

			releaseGlyphSlice(slice)
		}
	})

	b.Run("WithoutPooling", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			slice := make([]string, 0, 20)

			// Simulate usage
			for j := 0; j < 20; j++ {
				slice = append(slice, "test line")
			}
		}
	})
}

// BenchmarkRealWorldFont benchmarks parsing a realistic font
func BenchmarkRealWorldFont(b *testing.B) {
	// Create a more realistic font with varied glyphs
	var sb strings.Builder
	sb.WriteString("flf2a$ 8 6 14 -1 12\n")
	sb.WriteString("Standard by Glenn Chappell & Ian Chai\n")
	sb.WriteString("figlet release 2.1 -- 12 Aug 1994\n")
	sb.WriteString("Permission is hereby given to modify\n")

	// Space character
	for i := 0; i < 8; i++ {
		sb.WriteString("        @@\n")
	}

	// ASCII printable characters with varied widths
	for r := 33; r <= 126; r++ {
		width := 5 + (r % 10) // Vary width between 5-14
		for line := 0; line < 8; line++ {
			// Create varied patterns
			for w := 0; w < width; w++ {
				if (w+line)%3 == 0 {
					sb.WriteByte('#')
				} else {
					sb.WriteByte(' ')
				}
			}
			sb.WriteString("@@\n")
		}
	}

	fontData := sb.String()

	b.Run("Parse", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			r := strings.NewReader(fontData)
			_, _ = Parse(r)
		}
	})

	b.Run("ParseHeader", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			r := strings.NewReader(fontData)
			_, _ = ParseHeader(r)
		}
	})
}
