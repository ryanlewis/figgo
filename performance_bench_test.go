package figgo

import (
	"bytes"
	"testing"
)

// Common layout combinations for benchmarking
const (
	benchLayoutFullWidth = FitFullWidth
	benchLayoutKerning   = FitKerning
	benchLayoutSmushing  = FitSmushing | RuleEqualChar | RuleUnderscore | RuleHierarchy | RuleOppositePair | RuleBigX | RuleHardblank
)

// BenchmarkPRDTargets benchmarks the specific performance targets from the PRD
func BenchmarkPRDTargets(b *testing.B) {
	// Load the standard font for realistic testing
	font, err := LoadFont("fonts/standard.flf")
	if err != nil {
		b.Fatalf("Failed to load standard font: %v", err)
	}

	b.Run("QuickBrownFox_FullWidth", func(b *testing.B) {
		text := "The quick brown fox"
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = Render(text, font, WithLayout(benchLayoutFullWidth))
		}
	})

	b.Run("QuickBrownFox_Kerning", func(b *testing.B) {
		text := "The quick brown fox"
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = Render(text, font, WithLayout(benchLayoutKerning))
		}
	})

	b.Run("QuickBrownFox_Smushing", func(b *testing.B) {
		text := "The quick brown fox"
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = Render(text, font, WithLayout(benchLayoutSmushing))
		}
	})

	b.Run("QuickBrownFox_RenderTo", func(b *testing.B) {
		text := "The quick brown fox"
		var buf bytes.Buffer
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf.Reset()
			_ = RenderTo(&buf, text, font, WithLayout(benchLayoutKerning))
		}
	})
}

// BenchmarkAllocationTarget benchmarks allocation counts per operation
func BenchmarkAllocationTarget(b *testing.B) {
	font, err := LoadFont("fonts/standard.flf")
	if err != nil {
		b.Fatalf("Failed to load standard font: %v", err)
	}

	tests := []struct {
		name string
		text string
	}{
		{"SingleWord", "Hello"},
		{"TwoWords", "Hello World"},
		{"ShortSentence", "The quick brown fox"},
		{"LongSentence", "The quick brown fox jumps over the lazy dog"},
	}

	for _, tt := range tests {
		b.Run(tt.name+"_FullWidth", func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, _ = Render(tt.text, font, WithLayout(benchLayoutFullWidth))
			}
		})

		b.Run(tt.name+"_Kerning", func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, _ = Render(tt.text, font, WithLayout(benchLayoutKerning))
			}
		})

		b.Run(tt.name+"_Smushing", func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, _ = Render(tt.text, font, WithLayout(benchLayoutSmushing))
			}
		})
	}
}

// BenchmarkThroughputTarget measures operations per second for different scenarios
func BenchmarkThroughputTarget(b *testing.B) {
	font, err := LoadFont("fonts/standard.flf")
	if err != nil {
		b.Fatalf("Failed to load standard font: %v", err)
	}

	tests := []struct {
		name   string
		text   string
		layout Layout
	}{
		{"ShortText_FullWidth", "Hello", benchLayoutFullWidth},
		{"ShortText_Kerning", "Hello", benchLayoutKerning},
		{"ShortText_Smushing", "Hello", benchLayoutSmushing},
		{"MediumText_FullWidth", "The quick brown fox", benchLayoutFullWidth},
		{"MediumText_Kerning", "The quick brown fox", benchLayoutKerning},
		{"MediumText_Smushing", "The quick brown fox", benchLayoutSmushing},
		{"LongText_FullWidth", "The quick brown fox jumps over the lazy dog", benchLayoutFullWidth},
		{"LongText_Kerning", "The quick brown fox jumps over the lazy dog", benchLayoutKerning},
		{"LongText_Smushing", "The quick brown fox jumps over the lazy dog", benchLayoutSmushing},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, _ = Render(tt.text, font, WithLayout(tt.layout))
			}

			// Calculate and report operations per second
			ops := float64(b.N) / b.Elapsed().Seconds()
			b.ReportMetric(ops, "ops/sec")

			// Calculate character merges per second (approximation)
			charCount := len(tt.text)
			merges := float64(b.N*charCount) / b.Elapsed().Seconds()
			b.ReportMetric(merges, "merges/sec")
		})
	}
}

// BenchmarkMultipleFonts compares performance across different fonts
func BenchmarkMultipleFonts(b *testing.B) {
	fonts := []string{"standard.flf", "slant.flf", "small.flf", "big.flf"}
	text := "Hello World"

	for _, fontName := range fonts {
		font, err := LoadFont("fonts/" + fontName)
		if err != nil {
			b.Logf("Skipping font %s: %v", fontName, err)
			continue
		}

		b.Run(fontName+"_FullWidth", func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, _ = Render(text, font, WithLayout(benchLayoutFullWidth))
			}
		})

		b.Run(fontName+"_Kerning", func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, _ = Render(text, font, WithLayout(benchLayoutKerning))
			}
		})

		b.Run(fontName+"_Smushing", func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, _ = Render(text, font, WithLayout(benchLayoutSmushing))
			}
		})
	}
}

// BenchmarkConcurrentRendering tests thread-safety and concurrent performance
func BenchmarkConcurrentRendering(b *testing.B) {
	font, err := LoadFont("fonts/standard.flf")
	if err != nil {
		b.Fatalf("Failed to load standard font: %v", err)
	}

	texts := []string{
		"Hello",
		"World",
		"The quick brown fox",
		"Go",
		"FIGlet",
	}

	b.Run("Parallel_Mixed", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				text := texts[i%len(texts)]
				layout := []Layout{benchLayoutFullWidth, benchLayoutKerning, benchLayoutSmushing}[i%3]
				_, _ = Render(text, font, WithLayout(layout))
				i++
			}
		})
	})

	b.Run("Parallel_Smushing", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				text := texts[i%len(texts)]
				_, _ = Render(text, font, WithLayout(benchLayoutSmushing))
				i++
			}
		})
	})
}

// BenchmarkMemoryEfficiency measures memory usage for different scenarios
func BenchmarkMemoryEfficiency(b *testing.B) {
	font, err := LoadFont("fonts/standard.flf")
	if err != nil {
		b.Fatalf("Failed to load standard font: %v", err)
	}

	// Test memory efficiency with varying text lengths
	texts := []struct {
		name string
		text string
	}{
		{"VeryShort", "Hi"},
		{"Short", "Hello"},
		{"Medium", "The quick brown fox"},
		{"Long", "The quick brown fox jumps over the lazy dog"},
		{"VeryLong", "The quick brown fox jumps over the lazy dog every single day of the week"},
	}

	for _, tt := range texts {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			var totalBytes int64
			for i := 0; i < b.N; i++ {
				result, _ := Render(tt.text, font, WithLayout(benchLayoutSmushing))
				totalBytes += int64(len(result))
			}

			b.ReportMetric(float64(totalBytes)/float64(b.N), "result_bytes/op")
		})
	}
}

// BenchmarkEndToEnd measures complete workflow including font loading
func BenchmarkEndToEnd(b *testing.B) {
	text := "The quick brown fox"

	b.Run("LoadAndRender", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			font, _ := LoadFont("fonts/standard.flf")
			_, _ = Render(text, font, WithLayout(benchLayoutKerning))
		}
	})

	b.Run("CachedLoadAndRender", func(b *testing.B) {
		cache := NewFontCache(10)
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			font, _ := cache.LoadFont("fonts/standard.flf")
			_, _ = Render(text, font, WithLayout(benchLayoutKerning))
		}
	})
}
