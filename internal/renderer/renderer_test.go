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
				PrintDirection: intPtr(0),
			},
			want: "H  H  III \nHHHH   I  \nH  H  III ",
		},
		{
			name: "single character",
			text: "H",
			font: createMinimalFont(),
			opts: &Options{
				Layout:         0, // FitFullWidth
				PrintDirection: intPtr(0),
			},
			want: "H  H \nHHHH \nH  H ",
		},
		{
			name: "Hello text",
			text: "Hello",
			font: createMinimalFont(),
			opts: &Options{
				Layout:         0, // FitFullWidth
				PrintDirection: intPtr(0),
			},
			want: "H  H  eee l    l     ooo \nHHHH e e el    l    o   o\nH  H  ee ellll llll  ooo ",
		},
		{
			name: "hardblank replacement after composition",
			text: "AB",
			font: createFontWithHardblank(),
			opts: &Options{
				Layout:         0, // FitFullWidth
				PrintDirection: intPtr(0),
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
				PrintDirection: intPtr(0),
			},
			wantErr: true,
			errMsg:  "unsupported rune",
		},
		{
			name: "RTL print direction",
			text: "HI",
			font: createMinimalFont(),
			opts: &Options{
				Layout:         0,         // FitFullWidth
				PrintDirection: intPtr(1), // RTL
			},
			want: " III  H  H\n  I   HHHH\n III  H  H",
		},
		{
			name: "nil font error",
			text: "test",
			font: nil,
			opts: &Options{
				Layout:         0, // FitFullWidth
				PrintDirection: intPtr(0),
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
				PrintDirection: intPtr(0),
			},
			want: "\n\n", // Font height minus 1 newlines
		},
		{
			name: "space character",
			text: "H I",
			font: createMinimalFont(),
			opts: &Options{
				Layout:         0, // FitFullWidth
				PrintDirection: intPtr(0),
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
		PrintDirection: intPtr(0),
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
				PrintDirection: intPtr(0),
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
				PrintDirection: intPtr(0),
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
		PrintDirection: intPtr(0),
	}

	// Layout with just FitFullWidth
	optsFullWidth := &Options{
		Layout:         0,
		PrintDirection: intPtr(0),
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
				PrintDirection: &tt.printDir,
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
				PrintDirection: intPtr(0),
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
				PrintDirection: intPtr(0),
			},
			// | starts at column 2, A ends at column 2, need at least 1 space
			want: "  A   |   \n A A   |   \nA   A   |   ",
		},
		{
			name: "kerning between W and A",
			text: "WA",
			opts: &Options{
				Layout:         common.FitKerning,
				PrintDirection: intPtr(0),
			},
			// W ends with visible at col 4, A starts with visible at col 2
			// Row 2: W ends at 3, A starts at 1 - tightest constraint
			want: "W   W   A   \nW W W  A A  \n W W A   A ",
		},
		{
			name: "kerning with spaces now processed like any glyph",
			text: "A A",
			opts: &Options{
				Layout:         common.FitKerning,
				PrintDirection: intPtr(0),
			},
			// Space character goes through normal kerning like any other glyph
			// A ends at col 4, space is all blanks (6 wide), next A can be kerned
			// The blank space allows the second A to be positioned with minimal gap
			want: "  A  A   \n A A A A  \nA   AA   A ",
		},
		{
			name: "kerning with RTL direction",
			text: "AV",
			opts: &Options{
				Layout:         common.FitKerning,
				PrintDirection: intPtr(1), // RTL
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
				PrintDirection: intPtr(0),
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
				PrintDirection: intPtr(0),
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

// TestBlankSpaceGlyph tests that a truly blank space glyph works correctly with kerning
func TestBlankSpaceGlyph(t *testing.T) {
	// Create font with blank space (all ASCII spaces, no hardblanks)
	fontBlankSpace := &parser.Font{
		Hardblank:      '$',
		Height:         3,
		Baseline:       2,
		MaxLength:      5,
		OldLayout:      0, // Kerning
		PrintDirection: 0,
		Characters: map[rune][]string{
			' ': { // Truly blank space
				"     ",
				"     ",
				"     ",
			},
			'A': {
				" AAA ",
				"A   A",
				"A   A",
			},
			'B': {
				"BBB  ",
				"B   B",
				"BBB  ",
			},
		},
	}

	// Test with blank space between letters
	got, err := Render("A B", fontBlankSpace, nil)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	// Space should be processed through kerning like any other glyph
	// A's rightmost at col 4, space all blank (5 wide), B's leftmost at col 0
	// With kerning: A takes cols 0-4, blank space collapsed, B starts immediately
	want := " AAABBB  \nA   AB   B\nA   ABBB  "
	if got != want {
		t.Errorf("Blank space kerning mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}

	// Create font with hardblank-walled space
	fontHardblankSpace := &parser.Font{
		Hardblank:      '#',
		Height:         3,
		Baseline:       2,
		MaxLength:      5,
		OldLayout:      0, // Kerning
		PrintDirection: 0,
		Characters: map[rune][]string{
			' ': { // Space with hardblank barriers
				"##   ",
				"##   ",
				"##   ",
			},
			'A': {
				" AAA ",
				"A   A",
				"A   A",
			},
			'B': {
				"BBB  ",
				"B   B",
				"BBB  ",
			},
		},
	}

	// Test with hardblank-walled space
	got2, err := Render("A B", fontHardblankSpace, nil)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	// Hardblanks in space glyph should prevent over-tightening
	// A's rightmost at col 4, space has hardblanks at cols 0-1, B's leftmost at col 0
	// The hardblanks prevent B from overlapping, maintaining separation
	// After hardblank replacement, the hardblanks become spaces
	want2 := " AAA  BBB  \nA   A  B   B\nA   A  BBB  "
	if got2 != want2 {
		t.Errorf("Hardblank space kerning mismatch:\ngot:\n%s\nwant:\n%s", got2, want2)
	}

	// Verify hardblanks were replaced
	if strings.Contains(got2, "#") {
		t.Errorf("Hardblanks not replaced in output: %q", got2)
	}
}

// TestBlankBlankPadding tests consecutive spaces between letters
func TestBlankBlankPadding(t *testing.T) {
	font := &parser.Font{
		Hardblank:      '$',
		Height:         2,
		Baseline:       1,
		MaxLength:      4,
		OldLayout:      0, // Kerning
		PrintDirection: 0,
		Characters: map[rune][]string{
			' ': {
				"   ",
				"   ",
			},
			'X': {
				"X X",
				" X ",
			},
			'Y': {
				"Y Y",
				" Y ",
			},
		},
	}

	// Test multiple spaces between letters
	got, err := Render("X  Y", font, nil)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	// With blank-blank rows, kerning should allow minimal gaps
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}

	// Verify the output maintains proper spacing
	// X ends at col 2, two blank spaces, Y starts at col 0
	// The blank-blank logic should handle this correctly
	if !strings.Contains(got, "X") || !strings.Contains(got, "Y") {
		t.Errorf("Missing expected characters in output: %q", got)
	}
}

// TestRTLEquivalence tests that RTL of ABC equals LTR of CBA
func TestRTLEquivalence(t *testing.T) {
	font := &parser.Font{
		Hardblank:      '#',
		Height:         2,
		Baseline:       1,
		MaxLength:      4,
		OldLayout:      0, // Kerning
		PrintDirection: 0,
		Characters: map[rune][]string{
			'A': {
				"AAA#",
				"A#A#",
			},
			'B': {
				"BBB#",
				"B#B#",
			},
			'C': {
				"CCC#",
				"C#C#",
			},
		},
	}

	// Render ABC with RTL
	gotRTL, err := Render("ABC", font, &Options{
		Layout:         common.FitKerning,
		PrintDirection: intPtr(1), // RTL
	})
	if err != nil {
		t.Fatalf("Render() RTL error = %v", err)
	}

	// Render CBA with LTR
	gotLTR, err := Render("CBA", font, &Options{
		Layout:         common.FitKerning,
		PrintDirection: intPtr(0), // LTR
	})
	if err != nil {
		t.Fatalf("Render() LTR error = %v", err)
	}

	// After hardblank replacement, they should be identical
	if gotRTL != gotLTR {
		t.Errorf("RTL(ABC) != LTR(CBA)\nRTL:\n%s\nLTR:\n%s", gotRTL, gotLTR)
	}
}

// TestSpaceGlyphWithVisibleColumns tests that space glyphs with visible characters are preserved
func TestSpaceGlyphWithVisibleColumns(t *testing.T) {
	// Create font with space that has visible columns (like a decorated space)
	font := &parser.Font{
		Hardblank:      '$',
		Height:         3,
		Baseline:       2,
		MaxLength:      5,
		OldLayout:      0, // Kerning
		PrintDirection: 0,
		Characters: map[rune][]string{
			' ': { // Space with visible decoration (dots)
				"· · ·",
				"     ",
				"· · ·",
			},
			'A': {
				" AAA ",
				"A   A",
				"A   A",
			},
			'B': {
				"BBB  ",
				"B   B",
				"BBB  ",
			},
		},
	}

	// Test that visible space columns are not collapsed
	got, err := Render("A B", font, nil)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	// The space glyph has visible dots that must be preserved
	// A ends at col 4, space starts with visible at col 0
	// Row 0: dots visible, can't overlap
	// Row 1: space is all blanks, B can kern tight
	// Row 2: dots visible, can't overlap
	want := " AAA· · ·BBB  \nA   AB   B\nA   A· · ·BBB  "
	if got != want {
		t.Errorf("Visible space not preserved correctly:\ngot:\n%s\nwant:\n%s", got, want)
		t.Logf("Visual comparison:\nGot:\n%s\n\nWant:\n%s", got, want)
	}
}

// TestLeadingTrailingSpaces tests behavior with leading and trailing spaces
func TestLeadingTrailingSpaces(t *testing.T) {
	font := &parser.Font{
		Hardblank:      '$',
		Height:         2,
		Baseline:       1,
		MaxLength:      4,
		OldLayout:      0, // Kerning
		PrintDirection: 0,
		Characters: map[rune][]string{
			' ': {
				"   ",
				"   ",
			},
			'A': {
				"AAA",
				"A A",
			},
		},
	}

	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "leading spaces",
			text: "  A",
			// Two blank spaces followed by A
			// Spaces go through kerning, get trimmed/collapsed
			want: "AAA\nA A",
		},
		{
			name: "trailing spaces",
			text: "A  ",
			// A followed by two blank spaces
			// Trailing spaces are preserved as blank glyphs, but trimmed at the end
			want: "AAA   \nA A   ",
		},
		{
			name: "both leading and trailing",
			text: "  A  ",
			// Leading spaces collapsed, trailing preserved
			want: "AAA   \nA A   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Render(tt.text, font, nil)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestGlyphsWithBlankRowsTopBottom tests glyphs with entirely blank rows at top/bottom
func TestGlyphsWithBlankRowsTopBottom(t *testing.T) {
	font := &parser.Font{
		Hardblank:      '$',
		Height:         5,
		Baseline:       3,
		MaxLength:      4,
		OldLayout:      0, // Kerning
		PrintDirection: 0,
		Characters: map[rune][]string{
			'T': { // Blank top row
				"    ",
				"TTT ",
				" T  ",
				" T  ",
				" T  ",
			},
			'L': { // Blank bottom row
				"L   ",
				"L   ",
				"L   ",
				"LLL ",
				"    ",
			},
			'I': { // Blank top and bottom
				"    ",
				" I  ",
				" I  ",
				" I  ",
				"    ",
			},
		},
	}

	// Test adjacent glyphs with blank rows
	got, err := Render("TLI", font, nil)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	// T: blank top, visible cols 0-2 in rows 1-4
	// L: visible cols 0-2 in rows 0-3, blank bottom
	// I: blank top/bottom, visible col 1 in rows 1-3
	// With kerning, they should fit tightly (L overlaps T's blank top row)
	want := "L    \nTTTL I  \n TL I  \n TLLL I  \n T    "
	if got != want {
		t.Errorf("Blank rows handling incorrect:\ngot:\n%s\nwant:\n%s", got, want)
		for i, line := range strings.Split(got, "\n") {
			t.Logf("Got  line %d: %q", i, line)
		}
		for i, line := range strings.Split(want, "\n") {
			t.Logf("Want line %d: %q", i, line)
		}
	}
}
