package renderer

import (
	"strings"
	"testing"

	"github.com/ryanlewis/figgo/internal/parser"
)

// createMinimalFont creates a simple test font with a few characters
func createMinimalFont() *parser.Font {
	return &parser.Font{
		Hardblank:      '$',
		Height:         3,
		Baseline:       2,
		MaxLength:      5,
		OldLayout:      -1,
		PrintDirection: 0,
		Characters: map[rune][]string{
			' ': {
				"   ",
				"   ",
				"   ",
			},
			'H': {
				"H  H ",
				"HHHH ",
				"H  H ",
			},
			'I': {
				" III ",
				"  I  ",
				" III ",
			},
			'e': {
				" eee ",
				"e e e",
				" ee e",
			},
			'l': {
				"l    ",
				"l    ",
				"llll ",
			},
			'o': {
				" ooo ",
				"o   o",
				" ooo ",
			},
			'$': { // Character with hardblank
				"  $  ",
				" $$$ ",
				"$$$$",
			},
		},
	}
}

// createFontWithHardblank creates a font where hardblank appears in glyphs
func createFontWithHardblank() *parser.Font {
	return &parser.Font{
		Hardblank:      '#',
		Height:         3,
		Baseline:       2,
		MaxLength:      5,
		OldLayout:      -1,
		PrintDirection: 0,
		Characters: map[rune][]string{
			'A': {
				"#AA#",
				"A##A",
				"A##A",
			},
			'B': {
				"BBB#",
				"B##B",
				"BBB#",
			},
		},
	}
}

func TestRenderFullWidth(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		font    *parser.Font
		opts    *Options
		want    string
		wantErr bool
		errMsg  string
	}{
		{
			name: "simple ASCII string",
			text: "HI",
			font: createMinimalFont(),
			opts: &Options{
				Layout:         0x00000040, // FitFullWidth
				PrintDirection: 0,
			},
			want: "H  H  III \nHHHH   I  \nH  H  III ",
		},
		{
			name: "single character",
			text: "H",
			font: createMinimalFont(),
			opts: &Options{
				Layout:         0x00000040, // FitFullWidth
				PrintDirection: 0,
			},
			want: "H  H \nHHHH \nH  H ",
		},
		{
			name: "Hello text",
			text: "Hello",
			font: createMinimalFont(),
			opts: &Options{
				Layout:         0x00000040, // FitFullWidth
				PrintDirection: 0,
			},
			want: "H  H  eee l    l     ooo \nHHHH e e el    l    o   o\nH  H  ee ellll llll  ooo ",
		},
		{
			name: "hardblank replacement after composition",
			text: "AB",
			font: createFontWithHardblank(),
			opts: &Options{
				Layout:         0x00000040, // FitFullWidth
				PrintDirection: 0,
			},
			want: " AA BBB \nA  AB  B\nA  ABBB ",
		},
		{
			name: "unsupported rune error",
			text: "~",
			font: createMinimalFont(),
			opts: &Options{
				Layout:         0x00000040, // FitFullWidth
				PrintDirection: 0,
			},
			wantErr: true,
			errMsg:  "unsupported rune",
		},
		{
			name: "RTL print direction",
			text: "HI",
			font: createMinimalFont(),
			opts: &Options{
				Layout:         0x00000040, // FitFullWidth
				PrintDirection: 1,          // RTL
			},
			want: " III  H  H\n  I   HHHH\n III  H  H",
		},
		{
			name: "nil font error",
			text: "test",
			font: nil,
			opts: &Options{
				Layout:         0x00000040, // FitFullWidth
				PrintDirection: 0,
			},
			wantErr: true,
			errMsg:  "font",
		},
		{
			name: "empty text",
			text: "",
			font: createMinimalFont(),
			opts: &Options{
				Layout:         0x00000040, // FitFullWidth
				PrintDirection: 0,
			},
			want: "\n\n", // Font height minus 1 newlines
		},
		{
			name: "space character",
			text: "H I",
			font: createMinimalFont(),
			opts: &Options{
				Layout:         0x00000040, // FitFullWidth
				PrintDirection: 0,
			},
			want: "H  H     III \nHHHH      I  \nH  H     III ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Render(tt.text, tt.font, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Render() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}
			if got != tt.want {
				t.Errorf("Render() output mismatch:\ngot:\n%q\nwant:\n%q", got, tt.want)
				// Print visual comparison
				t.Logf("Visual comparison:\nGot:\n%s\n\nWant:\n%s", got, tt.want)
			}
		})
	}
}

func TestRenderFullWidth_GlyphHeightValidation(t *testing.T) {
	font := &parser.Font{
		Hardblank:      '$',
		Height:         3,
		Baseline:       2,
		MaxLength:      5,
		OldLayout:      -1,
		PrintDirection: 0,
		Characters: map[rune][]string{
			'X': { // Invalid: only 2 lines instead of 3
				"XXX",
				"XXX",
			},
		},
	}

	_, err := Render("X", font, &Options{
		Layout:         0x00000040, // FitFullWidth
		PrintDirection: 0,
	})

	if err == nil {
		t.Error("Expected error for mismatched glyph height")
	}
}

func TestRenderFullWidth_InvalidFontHeight(t *testing.T) {
	tests := []struct {
		name   string
		height int
	}{
		{"zero height", 0},
		{"negative height", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			font := &parser.Font{
				Hardblank:      '$',
				Height:         tt.height,
				Baseline:       2,
				MaxLength:      5,
				OldLayout:      -1,
				PrintDirection: 0,
				Characters:     map[rune][]string{},
			}

			_, err := Render("test", font, &Options{
				Layout:         0x00000040, // FitFullWidth
				PrintDirection: 0,
			})

			if err == nil {
				t.Error("Expected error for invalid font height")
			}
		})
	}
}
