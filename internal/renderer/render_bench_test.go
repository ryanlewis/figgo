package renderer

import (
	"bytes"
	"io"
	"testing"

	"github.com/ryanlewis/figgo/internal/parser"
)

// createBenchmarkFont creates a font for benchmarking
func createBenchmarkFont() *parser.Font {
	return &parser.Font{
		Height:         8,
		Hardblank:      '$',
		OldLayout:      0, // Kerning by default
		PrintDirection: 0, // LTR
		Characters: map[rune][]string{
			' ': {"    ", "    ", "    ", "    ", "    ", "    ", "    ", "    "},
			'A': {
				"  ___  ",
				" / _ \\ ",
				"/ /_\\ \\",
				"|  _  |",
				"| | | |",
				"\\_| |_/",
				"       ",
				"       ",
			},
			'B': {
				"______ ",
				"| ___ \\",
				"| |_/ /",
				"| ___ \\",
				"| |_/ /",
				"\\____/ ",
				"       ",
				"       ",
			},
			'C': {
				" _____ ",
				"/  __ \\",
				"| /  \\/",
				"| |    ",
				"| \\__/\\",
				" \\____/",
				"       ",
				"       ",
			},
			'D': {
				"______ ",
				"|  _  \\",
				"| | | |",
				"| | | |",
				"| |/ / ",
				"|___/  ",
				"       ",
				"       ",
			},
			'E': {
				" _____ ",
				"|  ___|",
				"| |__  ",
				"|  __| ",
				"| |___ ",
				"\\____/ ",
				"       ",
				"       ",
			},
			'F': {
				"______ ",
				"|  ___|",
				"| |_   ",
				"|  _|  ",
				"| |    ",
				"\\_|    ",
				"       ",
				"       ",
			},
			'G': {
				" _____ ",
				"|  __ \\",
				"| |  \\/",
				"| | __ ",
				"| |_\\ \\",
				" \\____/",
				"       ",
				"       ",
			},
			'H': {
				" _   _ ",
				"| | | |",
				"| |_| |",
				"|  _  |",
				"| | | |",
				"\\_| |_/",
				"       ",
				"       ",
			},
			'I': {
				" _____ ",
				"|_   _|",
				"  | |  ",
				"  | |  ",
				" _| |_ ",
				" \\___/ ",
				"       ",
				"       ",
			},
			'J': {
				"   ___ ",
				"  |_  |",
				"    | |",
				"    | |",
				"/\\__/ /",
				"\\____/ ",
				"       ",
				"       ",
			},
			'K': {
				" _   __",
				"| | / /",
				"| |/ / ",
				"|    \\ ",
				"| |\\  \\",
				"\\_| \\_/",
				"       ",
				"       ",
			},
			'L': {
				" _     ",
				"| |    ",
				"| |    ",
				"| |    ",
				"| |____",
				"\\_____/",
				"       ",
				"       ",
			},
			'M': {
				" __  __ ",
				"|  \\/  |",
				"| .  . |",
				"| |\\/| |",
				"| |  | |",
				"\\_|  |_/",
				"        ",
				"        ",
			},
			'N': {
				" _   _ ",
				"| \\ | |",
				"|  \\| |",
				"| . ` |",
				"| |\\  |",
				"\\_| \\_/",
				"       ",
				"       ",
			},
			'O': {
				" _____ ",
				"|  _  |",
				"| | | |",
				"| | | |",
				"\\ \\_/ /",
				" \\___/ ",
				"       ",
				"       ",
			},
			'P': {
				"______ ",
				"| ___ \\",
				"| |_/ /",
				"|  __/ ",
				"| |    ",
				"\\_|    ",
				"       ",
				"       ",
			},
			'Q': {
				" _____ ",
				"|  _  |",
				"| | | |",
				"| | | |",
				"\\ \\/' /",
				" \\_/\\_\\",
				"       ",
				"       ",
			},
			'R': {
				"______ ",
				"| ___ \\",
				"| |_/ /",
				"|    / ",
				"| |\\ \\ ",
				"\\_| \\_|",
				"       ",
				"       ",
			},
			'S': {
				" _____ ",
				"/  ___|",
				"\\ `--. ",
				" `--. \\",
				"/\\__/ /",
				"\\____/ ",
				"       ",
				"       ",
			},
			'T': {
				"_____ ",
				"|_   _|",
				"  | |  ",
				"  | |  ",
				"  | |  ",
				"  \\_/  ",
				"       ",
				"       ",
			},
			'U': {
				" _   _ ",
				"| | | |",
				"| | | |",
				"| | | |",
				"| |_| |",
				" \\___/ ",
				"       ",
				"       ",
			},
			'V': {
				" _   _ ",
				"| | | |",
				"| | | |",
				"| | | |",
				"\\ \\_/ /",
				" \\___/ ",
				"       ",
				"       ",
			},
			'W': {
				" _    _ ",
				"| |  | |",
				"| |  | |",
				"| |/\\| |",
				"\\  /\\  /",
				" \\/  \\/ ",
				"        ",
				"        ",
			},
			'X': {
				"__   __",
				"\\ \\ / /",
				" \\ V / ",
				" /   \\ ",
				"/ /^\\ \\",
				"\\/   \\/",
				"       ",
				"       ",
			},
			'Y': {
				"__   __",
				"\\ \\ / /",
				" \\ V / ",
				"  \\ /  ",
				"  | |  ",
				"  \\_/  ",
				"       ",
				"       ",
			},
			'Z': {
				" ______",
				"|___  /",
				"   / / ",
				"  / /  ",
				"./ /___",
				"\\_____/",
				"       ",
				"       ",
			},
		},
	}
}

// BenchmarkRenderOptimized tests the optimized render performance
func BenchmarkRenderOptimized(b *testing.B) {
	font := createBenchmarkFont()
	opts := &Options{Layout: (1 << 6)} // Kerning mode
	text := "HELLO WORLD"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = Render(text, font, opts)
	}
}

// BenchmarkRenderLongText tests rendering with longer text
func BenchmarkRenderLongText(b *testing.B) {
	font := createBenchmarkFont()
	opts := &Options{Layout: (1 << 6)} // Kerning mode
	text := "THE QUICK BROWN FOX JUMPS OVER THE LAZY DOG"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = Render(text, font, opts)
	}
}

// BenchmarkRenderRTL tests RTL rendering performance
func BenchmarkRenderRTL(b *testing.B) {
	font := createBenchmarkFont()
	font.PrintDirection = 1
	text := "HELLO WORLD"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = Render(text, font, nil)
	}
}

// BenchmarkRenderWithSmushing tests smushing performance
func BenchmarkRenderWithSmushing(b *testing.B) {
	font := createBenchmarkFont()
	opts := &Options{Layout: (1 << 7) | 63} // All smushing rules
	text := "HELLO WORLD"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = Render(text, font, opts)
	}
}

// BenchmarkRenderParallel tests concurrent rendering
func BenchmarkRenderParallel(b *testing.B) {
	font := createBenchmarkFont()
	opts := &Options{Layout: (1 << 6)} // Kerning mode
	text := "HELLO WORLD"

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = Render(text, font, opts)
		}
	})
}

// BenchmarkRenderTo tests direct writer performance
func BenchmarkRenderTo(b *testing.B) {
	font := createBenchmarkFont()
	opts := &Options{Layout: (1 << 6)} // Kerning mode
	text := "HELLO WORLD"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = RenderTo(io.Discard, text, font, opts)
	}
}

// BenchmarkRenderToBuffer tests rendering to a reused buffer
func BenchmarkRenderToBuffer(b *testing.B) {
	font := createBenchmarkFont()
	opts := &Options{Layout: (1 << 6)} // Kerning mode
	text := "HELLO WORLD"
	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = RenderTo(&buf, text, font, opts)
	}
}

// BenchmarkRenderVsRenderTo compares both methods with same output
func BenchmarkRenderVsRenderTo(b *testing.B) {
	font := createBenchmarkFont()
	opts := &Options{Layout: (1 << 6)} // Kerning mode
	text := "THE QUICK BROWN FOX JUMPS OVER THE LAZY DOG"

	b.Run("Render", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = Render(text, font, opts)
		}
	})

	b.Run("RenderTo", func(b *testing.B) {
		var buf bytes.Buffer
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			_ = RenderTo(&buf, text, font, opts)
		}
	})

	b.Run("RenderToDiscard", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = RenderTo(io.Discard, text, font, opts)
		}
	})
}

// BenchmarkRenderToParallel tests concurrent RenderTo
func BenchmarkRenderToParallel(b *testing.B) {
	font := createBenchmarkFont()
	opts := &Options{Layout: (1 << 6)} // Kerning mode
	text := "HELLO WORLD"

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		var buf bytes.Buffer
		for pb.Next() {
			buf.Reset()
			_ = RenderTo(&buf, text, font, opts)
		}
	})
}

// BenchmarkSimpleText shows realistic performance for common use cases
func BenchmarkSimpleText(b *testing.B) {
	font := createBenchmarkFont()

	tests := []struct {
		name string
		text string
	}{
		{"Single Word", "HELLO"},
		{"Two Words", "HELLO WORLD"},
		{"Short Sentence", "WELCOME TO FIGGO"},
		{"Long Sentence", "THE QUICK BROWN FOX JUMPS OVER THE LAZY DOG"},
		{"Single Char", "A"},
		{"Numbers", "12345"},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = Render(tt.text, font, nil)
			}
		})
	}
}

// BenchmarkSimpleTextRenderTo shows RenderTo performance for common use cases
func BenchmarkSimpleTextRenderTo(b *testing.B) {
	font := createBenchmarkFont()

	tests := []struct {
		name string
		text string
	}{
		{"Single Word", "HELLO"},
		{"Two Words", "HELLO WORLD"},
		{"Short Sentence", "WELCOME TO FIGGO"},
		{"Long Sentence", "THE QUICK BROWN FOX JUMPS OVER THE LAZY DOG"},
		{"Single Char", "A"},
		{"Numbers", "12345"},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			var buf bytes.Buffer
			for i := 0; i < b.N; i++ {
				buf.Reset()
				_ = RenderTo(&buf, tt.text, font, nil)
			}
		})
	}
}

// BenchmarkThroughput measures operations per second for different text sizes
func BenchmarkThroughput(b *testing.B) {
	font := createBenchmarkFont()

	tests := []struct {
		name string
		text string
	}{
		{"Tiny", "Hi"},
		{"Small", "Hello"},
		{"Medium", "Hello World"},
		{"Large", "The Quick Brown Fox Jumps"},
		{"XLarge", "The Quick Brown Fox Jumps Over The Lazy Dog Every Day"},
	}

	for _, tt := range tests {
		b.Run("Render_"+tt.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = Render(tt.text, font, nil)
			}
			ops := float64(b.N) / b.Elapsed().Seconds()
			b.ReportMetric(ops, "ops/sec")
		})

		b.Run("RenderTo_"+tt.name, func(b *testing.B) {
			b.ReportAllocs()
			var buf bytes.Buffer
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				buf.Reset()
				_ = RenderTo(&buf, tt.text, font, nil)
			}
			ops := float64(b.N) / b.Elapsed().Seconds()
			b.ReportMetric(ops, "ops/sec")
		})
	}
}
