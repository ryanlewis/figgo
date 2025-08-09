package renderer

import (
	"testing"

	"github.com/ryanlewis/figgo/internal/common"
	"github.com/ryanlewis/figgo/internal/parser"
)

const benchmarkText = "Hello, World!"

// createBenchFont creates a font with full ASCII coverage for benchmarking
func createBenchFont() *parser.Font {
	font := &parser.Font{
		Hardblank:      '$',
		Height:         3,
		Baseline:       2,
		MaxLength:      5,
		OldLayout:      -1,
		PrintDirection: 0,
		Characters:     make(map[rune][]string),
	}

	// Add ASCII printable characters (32-126)
	for r := rune(32); r <= 126; r++ {
		// Simple 3-line representation
		font.Characters[r] = []string{
			string([]byte{byte(r), byte(r), byte(r), ' ', ' '}),
			string([]byte{byte(r), ' ', byte(r), ' ', ' '}),
			string([]byte{byte(r), byte(r), byte(r), ' ', ' '}),
		}
	}

	return font
}

func BenchmarkRenderFullWidth(b *testing.B) {
	font := createBenchFont()
	text := benchmarkText
	opts := &Options{
		Layout:         common.FitFullWidth,
		PrintDirection: 0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Render(text, font, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderFullWidthLong(b *testing.B) {
	font := createBenchFont()
	text := "The quick brown fox jumps over the lazy dog. 1234567890"
	opts := &Options{
		Layout:         common.FitFullWidth,
		PrintDirection: 0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Render(text, font, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderFullWidthRTL(b *testing.B) {
	font := createBenchFont()
	text := benchmarkText
	opts := &Options{
		Layout:         common.FitFullWidth,
		PrintDirection: 1, // RTL
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Render(text, font, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderFullWidthWithHardblank(b *testing.B) {
	font := createBenchFont()
	// Add hardblank character to some glyphs
	font.Characters['H'] = []string{
		"H$$H ",
		"H$$$H",
		"H$$H ",
	}
	font.Characters['W'] = []string{
		"W$W$W",
		"W$W$W",
		"$WWW$",
	}

	text := benchmarkText
	opts := &Options{
		Layout:         common.FitFullWidth,
		PrintDirection: 0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Render(text, font, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderFullWidthNonASCII(b *testing.B) {
	font := createBenchFont()
	text := "Hello, 世界! €100 café"
	opts := &Options{
		Layout:         common.FitFullWidth,
		PrintDirection: 0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Render(text, font, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRenderFullWidthAllocs specifically measures allocations
func BenchmarkRenderFullWidthAllocs(b *testing.B) {
	font := createBenchFont()
	text := "Hello!"
	opts := &Options{
		Layout:         common.FitFullWidth,
		PrintDirection: 0,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Render(text, font, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}
