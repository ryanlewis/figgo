package renderer

import (
	"strings"
	"testing"

	"github.com/ryanlewis/figgo/internal/common"
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
			'?': { // Replacement character for non-ASCII
				" ??? ",
				"  ?  ",
				"  ?  ",
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
				Layout:         0, // FitFullWidth
				PrintDirection: 0,
			},
			want: "H  H  III \nHHHH   I  \nH  H  III ",
		},
		{
			name: "single character",
			text: "H",
			font: createMinimalFont(),
			opts: &Options{
				Layout:         0, // FitFullWidth
				PrintDirection: 0,
			},
			want: "H  H \nHHHH \nH  H ",
		},
		{
			name: "Hello text",
			text: "Hello",
			font: createMinimalFont(),
			opts: &Options{
				Layout:         0, // FitFullWidth
				PrintDirection: 0,
			},
			want: "H  H  eee l    l     ooo \nHHHH e e el    l    o   o\nH  H  ee ellll llll  ooo ",
		},
		{
			name: "hardblank replacement after composition",
			text: "AB",
			font: createFontWithHardblank(),
			opts: &Options{
				Layout:         0, // FitFullWidth
				PrintDirection: 0,
			},
			want: " AA BBB \nA  AB  B\nA  ABBB ",
		},
		{
			name: "unsupported rune error - missing question mark",
			text: "?",
			font: &parser.Font{
				Hardblank:      '$',
				Height:         3,
				Baseline:       2,
				MaxLength:      5,
				OldLayout:      -1,
				PrintDirection: 0,
				Characters: map[rune][]string{
					// No '?' glyph - should error
					'H': {
						"H  H ",
						"HHHH ",
						"H  H ",
					},
				},
			},
			opts: &Options{
				Layout:         0, // FitFullWidth
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
				Layout:         0, // FitFullWidth
				PrintDirection: 1, // RTL
			},
			want: " III  H  H\n  I   HHHH\n III  H  H",
		},
		{
			name: "nil font error",
			text: "test",
			font: nil,
			opts: &Options{
				Layout:         0, // FitFullWidth
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
				Layout:         0, // FitFullWidth
				PrintDirection: 0,
			},
			want: "\n\n", // Font height minus 1 newlines
		},
		{
			name: "space character",
			text: "H I",
			font: createMinimalFont(),
			opts: &Options{
				Layout:         0, // FitFullWidth
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
		Layout:         0, // FitFullWidth
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
				Layout:         0, // FitFullWidth
				PrintDirection: 0,
			})

			if err == nil {
				t.Error("Expected error for invalid font height")
			}
		})
	}
}

func TestFallbackGlyphAvailable(t *testing.T) {
	// Test that our fonts have either '?' or ' ' for fallback
	font := createMinimalFont()

	// Should have '?' glyph
	_, hasQuestionMark := font.Characters['?']
	if !hasQuestionMark {
		// Should at least have space as fallback
		_, hasSpace := font.Characters[' ']
		if !hasSpace {
			t.Error("Font should have either '?' or ' ' glyph for fallback")
		}
	}
}

func TestRenderNonASCIIFiltering(t *testing.T) {
	font := createMinimalFont()

	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "unicode replaced with question mark",
			text: "H€I",
			want: "H  H  ???  III \nHHHH   ?    I  \nH  H   ?   III ",
		},
		{
			name: "control characters replaced",
			text: "H\x01I",
			want: "H  H  ???  III \nHHHH   ?    I  \nH  H   ?   III ",
		},
		{
			name: "high ASCII replaced",
			text: "H\x80I",
			want: "H  H  ???  III \nHHHH   ?    I  \nH  H   ?   III ",
		},
		{
			name: "tab replaced",
			text: "H\tI",
			want: "H  H  ???  III \nHHHH   ?    I  \nH  H   ?   III ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Render(tt.text, font, &Options{
				Layout:         0, // FitFullWidth
				PrintDirection: 0,
			})
			if err != nil {
				t.Errorf("Render() unexpected error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Render() output mismatch:\ngot:\n%q\nwant:\n%q", got, tt.want)
			}
		})
	}
}

func TestRenderRulesWithoutSmushing(t *testing.T) {
	// Test that rule bits are ignored when FitSmushing is not set
	font := createMinimalFont()

	// Layout with rules but no FitSmushing (should behave as FitFullWidth)
	optsWithRules := &Options{
		Layout:         common.RuleEqualChar | common.RuleUnderscore | common.RuleHierarchy,
		PrintDirection: 0,
	}

	// Layout with just FitFullWidth
	optsFullWidth := &Options{
		Layout:         0,
		PrintDirection: 0,
	}

	text := "HI"

	gotWithRules, err1 := Render(text, font, optsWithRules)
	if err1 != nil {
		t.Fatalf("Render() with rules error = %v", err1)
	}

	gotFullWidth, err2 := Render(text, font, optsFullWidth)
	if err2 != nil {
		t.Fatalf("Render() full width error = %v", err2)
	}

	if gotWithRules != gotFullWidth {
		t.Errorf("Rule bits without FitSmushing should be ignored\ngot:\n%q\nwant:\n%q",
			gotWithRules, gotFullWidth)
	}
}

func TestRenderNilOptionsWithFullLayout(t *testing.T) {
	// Test that nil options uses font's FullLayout when set
	font := &parser.Font{
		Hardblank:      '$',
		Height:         3,
		Baseline:       2,
		MaxLength:      5,
		OldLayout:      0, // Would be kerning
		FullLayout:     0, // Full width
		FullLayoutSet:  true,
		PrintDirection: 0,
		Characters: map[rune][]string{
			'A': {
				"AAA",
				"A A",
				"A A",
			},
		},
	}

	// With nil options, should use FullLayout (full width)
	got, err := Render("A", font, nil)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	want := "AAA\nA A\nA A"
	if got != want {
		t.Errorf("Render() with nil opts and FullLayout:\ngot:\n%q\nwant:\n%q", got, want)
	}
}

func TestRenderPrintDirectionValidation(t *testing.T) {
	font := createMinimalFont()

	tests := []struct {
		name     string
		printDir int
		want     string // Expected output (should be LTR for invalid values)
	}{
		{
			name:     "valid LTR (0)",
			printDir: 0,
			want:     "H  H \nHHHH \nH  H ",
		},
		{
			name:     "valid RTL (1)",
			printDir: 1,
			want:     " H  H\n HHHH\n H  H",
		},
		{
			name:     "invalid negative (-1) defaults to LTR",
			printDir: -1,
			want:     "H  H \nHHHH \nH  H ",
		},
		{
			name:     "invalid high value (2) defaults to LTR",
			printDir: 2,
			want:     "H  H \nHHHH \nH  H ",
		},
		{
			name:     "invalid high value (99) defaults to LTR",
			printDir: 99,
			want:     "H  H \nHHHH \nH  H ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Render("H", font, &Options{
				Layout:         0,
				PrintDirection: tt.printDir,
			})
			if err != nil {
				t.Errorf("Render() unexpected error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Render() with printDir=%d:\ngot:\n%q\nwant:\n%q",
					tt.printDir, got, tt.want)
			}
		})
	}
}

func TestPickLayoutPrecedence(t *testing.T) {
	tests := []struct {
		name       string
		font       *parser.Font
		opts       *Options
		wantLayout int
		wantErr    bool
	}{
		{
			name: "opts takes precedence over font",
			font: &parser.Font{
				OldLayout:     0, // kerning
				FullLayout:    common.FitSmushing,
				FullLayoutSet: true,
			},
			opts: &Options{
				Layout: common.FitFullWidth,
			},
			wantLayout: common.FitFullWidth,
		},
		{
			name: "FullLayout takes precedence over OldLayout when set",
			font: &parser.Font{
				OldLayout:     0, // kerning
				FullLayout:    common.FitSmushing,
				FullLayoutSet: true,
			},
			opts:       nil,
			wantLayout: common.FitSmushing,
		},
		{
			name: "OldLayout used when FullLayout not set",
			font: &parser.Font{
				OldLayout:     0, // kerning
				FullLayout:    common.FitSmushing,
				FullLayoutSet: false,
			},
			opts:       nil,
			wantLayout: common.FitKerning,
		},
		{
			name: "FullLayout 0 with FullLayoutSet true defaults to full width",
			font: &parser.Font{
				OldLayout:     1, // would be smushing
				FullLayout:    0,
				FullLayoutSet: true,
			},
			opts:       nil,
			wantLayout: common.FitFullWidth,
		},
		{
			name: "conflicting layout bits returns error",
			font: &parser.Font{
				OldLayout: -1,
			},
			opts: &Options{
				Layout: common.FitKerning | common.FitSmushing,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout, err := pickLayout(tt.font, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("pickLayout() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && layout != tt.wantLayout {
				t.Errorf("pickLayout() = %v, want %v", layout, tt.wantLayout)
			}
		})
	}
}

// createKerningTestFont creates a font designed to test kerning behavior
func createKerningTestFont() *parser.Font {
	return &parser.Font{
		Hardblank:      '$',
		Height:         3,
		Baseline:       2,
		MaxLength:      6,
		OldLayout:      0, // kerning
		PrintDirection: 0,
		Characters: map[rune][]string{
			'A': {
				"  A   ",
				" A A  ",
				"A   A ",
			},
			'V': {
				"V   V ",
				" V V  ",
				"  V   ",
			},
			'W': {
				"W   W ",
				"W W W ",
				" W W  ",
			},
			'|': {
				"  |   ",
				"  |   ",
				"  |   ",
			},
			' ': {
				"      ",
				"      ",
				"      ",
			},
		},
	}
}

func TestRenderKerning(t *testing.T) {
	font := createKerningTestFont()

	tests := []struct {
		name string
		text string
		opts *Options
		want string
	}{
		{
			name: "kerning between A and V - should tighten",
			text: "AV",
			opts: &Options{
				Layout:         common.FitKerning,
				PrintDirection: 0,
			},
			// A has trailing spaces, V has no leading spaces - kerning should tighten
			// Row 0: "  A   " + "V   V " -> A at pos 2, need 1 space, V starts at 0
			// Row 1: " A A  " + " V V  " -> last A at pos 3, V starts at 1
			// Row 2: "A   A " + "  V   " -> last A at pos 4, V starts at 2
			// Min gap is determined by row needing most space
			want: "  A V   V \n A A  V V  \nA   A   V   ",
		},
		{
			name: "kerning with vertical bar - no tightening",
			text: "A|",
			opts: &Options{
				Layout:         common.FitKerning,
				PrintDirection: 0,
			},
			// | starts at column 2, A ends at column 2, need at least 1 space
			want: "  A   |   \n A A   |   \nA   A   |   ",
		},
		{
			name: "kerning between W and A",
			text: "WA",
			opts: &Options{
				Layout:         common.FitKerning,
				PrintDirection: 0,
			},
			// W ends with visible at col 4, A starts with visible at col 2
			// Row 2: W ends at 3, A starts at 1 - tightest constraint
			want: "W   W   A   \nW W W  A A  \n W W A   A ",
		},
		{
			name: "kerning with spaces preserved",
			text: "A A",
			opts: &Options{
				Layout:         common.FitKerning,
				PrintDirection: 0,
			},
			// Space character is 6 spaces wide, should be preserved as-is
			// A (6 wide) + space (6 wide) + A (6 wide, no kerning after space)
			want: "  A           A   \n A A         A A  \nA   A       A   A ",
		},
		{
			name: "kerning with RTL direction",
			text: "AV",
			opts: &Options{
				Layout:         common.FitKerning,
				PrintDirection: 1, // RTL
			},
			// RTL composes glyphs in reverse order (V then A), not mirrored
			// V first, then A with kerning
			want: "V   V   A   \n V V  A A  \n  V A   A ",
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
				t.Errorf("Render() kerning mismatch:\ngot:\n%q\nwant:\n%q", got, tt.want)
				t.Logf("Visual comparison:\nGot:\n%s\n\nWant:\n%s", got, tt.want)
			}
		})
	}
}

func TestRenderKerningWithHardblanks(t *testing.T) {
	// Font with hardblanks that should prevent over-tightening
	font := &parser.Font{
		Hardblank:      '#',
		Height:         3,
		Baseline:       2,
		MaxLength:      5,
		OldLayout:      0, // kerning
		PrintDirection: 0,
		Characters: map[rune][]string{
			'X': {
				"X###X",
				"#X#X#",
				"X###X",
			},
			'Y': {
				"Y###Y",
				"#Y#Y#",
				"##Y##",
			},
		},
	}

	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "hardblanks prevent over-tightening",
			text: "XY",
			// Hardblanks should be treated as visible for collision detection
			// But replaced with spaces in final output
			// All hardblanks count as visible, preventing any kerning
			want: "X   XY   Y\n X X  Y Y \nX   X  Y  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Render(tt.text, font, &Options{
				Layout:         common.FitKerning,
				PrintDirection: 0,
			})
			if err != nil {
				t.Errorf("Render() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Render() with hardblanks:\ngot:\n%q\nwant:\n%q", got, tt.want)
				t.Logf("Visual:\nGot:\n%s\n\nWant:\n%s", got, tt.want)
			}
		})
	}
}

func TestRenderKerningZeroGap(t *testing.T) {
	// Test that glyphs can touch (zero gap) when they don't collide
	font := &parser.Font{
		Hardblank:      '$',
		Height:         3,
		Baseline:       2,
		MaxLength:      4,
		OldLayout:      0, // kerning
		PrintDirection: 0,
		Characters: map[rune][]string{
			'L': {
				"L   ",
				"L   ",
				"LLL ",
			},
			'J': {
				"  J ",
				"  J ",
				"JJJ ",
			},
			'T': {
				"TTT ",
				" T  ",
				" T  ",
			},
			'I': {
				" I  ",
				" I  ",
				" I  ",
			},
		},
	}

	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "L and J can touch - perfect fit",
			text: "LJ",
			// L ends at col 2 on row 2, J starts at col 0 - actually touching!
			want: "L  J \nL  J \nLLLJJJ ",
		},
		{
			name: "T and I - zero gap possible",
			text: "TI",
			// T ends at col 2 on row 0, I starts at col 1 - but row 1 and 2
			// have T ending at col 1, I starting at col 1 - zero gap (touching)
			want: "TTT I  \n T I  \n T I  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Render(tt.text, font, &Options{
				Layout:         common.FitKerning,
				PrintDirection: 0,
			})
			if err != nil {
				t.Errorf("Render() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Render() zero-gap mismatch:\ngot:\n%q\nwant:\n%q", got, tt.want)
				t.Logf("Visual:\nGot:\n%s\n\nWant:\n%s", got, tt.want)
			}
		})
	}
}

func TestRenderKerningDefaultFromFont(t *testing.T) {
	// Test that kerning is used when font defaults to it
	font := &parser.Font{
		Hardblank:      '$',
		Height:         2,
		Baseline:       1,
		MaxLength:      4,
		OldLayout:      0, // Default to kerning
		PrintDirection: 0,
		Characters: map[rune][]string{
			'I': {
				" I ",
				" I ",
			},
			'T': {
				"TTT",
				" T ",
			},
		},
	}

	// With nil options, should use font's default (kerning)
	got, err := Render("IT", font, nil)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	// Kerning should tighten I and T
	// I ends at col 1, T starts at col 0 - touching allowed
	want := " ITTT\n I T "
	if got != want {
		t.Errorf("Render() with default kerning:\ngot:\n%q\nwant:\n%q", got, want)
	}
}
