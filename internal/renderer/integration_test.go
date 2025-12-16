package renderer

import (
	"strings"
	"testing"

	"github.com/ryanlewis/figgo/internal/parser"
)

// Helper function to create a test font
func createTestFont() *parser.Font {
	return &parser.Font{
		Height:         3,
		Hardblank:      '$',
		OldLayout:      0, // Kerning by default
		PrintDirection: 0, // LTR
		Characters: map[rune][]string{
			' ': {"   ", "   ", "   "},
			'H': {
				"H  H",
				"HHHH",
				"H  H",
			},
			'E': {
				"EEEE",
				"EE  ",
				"EEEE",
			},
			'L': {
				"L   ",
				"L   ",
				"LLLL",
			},
			'O': {
				" OOO ",
				"O   O",
				" OOO ",
			},
			'W': {
				"W   W",
				"W W W",
				" W W ",
			},
			'R': {
				"RRRR ",
				"RR  R",
				"R  R ",
			},
			'D': {
				"DDDD ",
				"D   D",
				"DDDD ",
			},
			'!': {
				"!",
				"!",
				"!",
			},
			'|': {
				"|",
				"|",
				"|",
			},
			'/': {
				"  /",
				" / ",
				"/  ",
			},
			'\\': {
				"\\  ",
				" \\ ",
				"  \\",
			},
			'_': {
				"   ",
				"   ",
				"___",
			},
			'[': {
				"[",
				"[",
				"[",
			},
			']': {
				"]",
				"]",
				"]",
			},
		},
	}
}

func TestIntegrationFullWidth(t *testing.T) {
	font := createTestFont()

	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "single character",
			text: "H",
			want: "H  H\nHHHH\nH  H",
		},
		{
			name: "word HELLO",
			text: "HELLO",
			want: strings.Join([]string{
				"H  HEEEEL    L    OOO ",
				"HHHHHEE  L    L   O   O",
				"H  HEEEEL   LLLL  OOO ",
			}, "\n"),
		},
		{
			name: "with spaces",
			text: "H E",
			want: strings.Join([]string{
				"H  H   EEEE",
				"HHHH   EE  ",
				"H  H   EEEE",
			}, "\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &Options{Layout: 0} // Full width
			got, err := Render(tt.text, font, opts)
			if err != nil {
				t.Errorf("Render() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Render() got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestIntegrationKerning(t *testing.T) {
	font := createTestFont()
	font.OldLayout = 0 // Kerning mode

	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "HE with kerning",
			text: "HE",
			want: strings.Join([]string{
				"H  HEEEE",
				"HHHHHEE  ",
				"H  HEEEE",
			}, "\n"),
		},
		{
			name: "WORLD with kerning",
			text: "WORLD",
			want: strings.Join([]string{
				"W   W OOO RRRR L   DDDD ",
				"W W WO   ORR  RL   D   D",
				" W W  OOO R  R L   DDDD ",
			}, "\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Render(tt.text, font, nil)
			if err != nil {
				t.Errorf("Render() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Render() got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestIntegrationSmushing(t *testing.T) {
	font := createTestFont()

	tests := []struct {
		name   string
		text   string
		layout int
		want   string
	}{
		{
			name:   "smushing with equal rule",
			text:   "||",
			layout: (1 << 7) | SMEqual, // Smushing with equal rule
			want:   "|\n|\n|",
		},
		{
			name:   "smushing with underscore rule",
			text:   "_|",
			layout: (1 << 7) | SMLowline, // Smushing with underscore rule
			want:   " |\n |\n_|",
		},
		{
			name:   "smushing with hierarchy rule",
			text:   "|/",
			layout: (1 << 7) | SMHierarchy, // Smushing with hierarchy rule
			want:   " /\n/ \n/ ",
		},
		{
			name:   "smushing with pair rule",
			text:   "[]",
			layout: (1 << 7) | SMPair, // Smushing with pair rule
			want:   "|\n|\n|",
		},
		{
			name:   "smushing with big X rule",
			text:   "/\\",
			layout: (1 << 7) | SMBigX, // Smushing with big X rule
			want:   " |\n | \n| ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &Options{Layout: tt.layout}
			got, err := Render(tt.text, font, opts)
			if err != nil {
				t.Errorf("Render() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Render() got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestIntegrationRTL(t *testing.T) {
	font := createTestFont()
	font.PrintDirection = 1 // RTL

	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "HELLO in RTL",
			text: "HELLO",
			// Note: Row 2 has 8 consecutive L's because the 'L' glyph row 2 is "LLLL"
			// (no trailing spaces). In kerning mode, smushAmount is the MINIMUM across
			// all rows, and row 2 constrains it to 0, so L's appear adjacent.
			want: strings.Join([]string{
				" OOO L   L   EEEEH  H",
				"O   OL   L   EE  HHHH",
				" OOO LLLLLLLLEEEEH  H",
			}, "\n"),
		},
		{
			name: "HI in RTL",
			text: "H!",
			want: strings.Join([]string{
				"!H  H",
				"!HHHH",
				"!H  H",
			}, "\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Render(tt.text, font, nil)
			if err != nil {
				t.Errorf("Render() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Render() got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestIntegrationComplexScenarios(t *testing.T) {
	font := createTestFont()

	tests := []struct {
		name string
		text string
		opts *Options
		want string
	}{
		{
			name: "trim whitespace",
			text: "H",
			opts: &Options{
				Layout:         0,
				TrimWhitespace: true,
			},
			want: "H  H\nHHHH\nH  H",
		},
		{
			name: "unknown rune with fallback",
			text: "HX",
			opts: &Options{
				Layout:      0,
				UnknownRune: func() *rune { r := '!'; return &r }(),
			},
			want: strings.Join([]string{
				"H  H!",
				"HHHH!",
				"H  H!",
			}, "\n"),
		},
		{
			name: "empty text",
			text: "",
			opts: &Options{Layout: 0},
			want: "\n\n",
		},
		{
			name: "hardblank in text",
			text: "H$E",
			opts: &Options{Layout: 0},
			want: strings.Join([]string{
				"H  H EEEE",
				"HHHH EE  ",
				"H  H EEEE",
			}, "\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Render(tt.text, font, tt.opts)
			if err != nil {
				t.Errorf("Render() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Render() got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestIntegrationLargeText(t *testing.T) {
	font := createTestFont()

	// Test with a longer string
	text := "HELLO WORLD"
	opts := &Options{Layout: 0} // Full width

	got, err := Render(text, font, opts)
	if err != nil {
		t.Errorf("Render() error = %v", err)
		return
	}

	// Check that output has correct number of lines
	lines := strings.Split(got, "\n")
	if len(lines) != font.Height {
		t.Errorf("Expected %d lines, got %d", font.Height, len(lines))
	}

	// Check that all lines have same length
	firstLen := len(lines[0])
	for i, line := range lines {
		if len(line) != firstLen {
			t.Errorf("Line %d has length %d, expected %d", i, len(line), firstLen)
		}
	}
}

func BenchmarkRenderFullWidth(b *testing.B) {
	font := createTestFont()
	opts := &Options{Layout: 0}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Render("HELLO WORLD", font, opts)
	}
}

func BenchmarkRenderKerning(b *testing.B) {
	font := createTestFont()
	font.OldLayout = 0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Render("HELLO WORLD", font, nil)
	}
}

func BenchmarkRenderSmushing(b *testing.B) {
	font := createTestFont()
	opts := &Options{Layout: (1 << 7) | 63} // All smushing rules

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Render("HELLO WORLD", font, opts)
	}
}

func BenchmarkRenderRTLDirection(b *testing.B) {
	font := createTestFont()
	font.PrintDirection = 1

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Render("HELLO WORLD", font, nil)
	}
}
