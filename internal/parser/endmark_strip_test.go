package parser

import (
	"bytes"
	"strings"
	"testing"
)

// Helper function to validate a specific character's glyph
func validateCharacterGlyph(t *testing.T, f *Font, char rune, expected []string, charName string) {
	t.Helper()
	glyph, exists := f.Characters[char]
	if !exists {
		t.Fatalf("%s character not found", charName)
	}
	if len(glyph) != len(expected) {
		t.Errorf("%s glyph length = %d, want %d", charName, len(glyph), len(expected))
		return
	}
	for i, expectedLine := range expected {
		if glyph[i] != expectedLine {
			t.Errorf("%s line %d = %q, want %q", charName, i, glyph[i], expectedLine)
		}
	}
}

// Helper for validating multiple character glyphs in sequence
func validateMultipleGlyphs(t *testing.T, f *Font, validations []glyphValidation) {
	t.Helper()
	for _, v := range validations {
		validateCharacterGlyph(t, f, v.char, v.expected, v.name)
	}
}

type glyphValidation struct {
	char     rune
	name     string
	expected []string
}

// Helper specifically for validating space character (commonly used)
func validateSpaceGlyphWithLines(t *testing.T, f *Font, expected []string) {
	t.Helper()
	validateCharacterGlyph(t, f, ' ', expected, "space")
}

// TestParseGlyphs_StripEntireTrailingRun tests that the parser strips the ENTIRE
// trailing run of the endmark character, not just 1 or 2 occurrences.
// Per spec: "eliminate the last block of consecutive equal characters"
func TestParseGlyphs_StripEntireTrailingRun(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		validate func(t *testing.T, f *Font)
	}{
		{
			name: "strip_all_trailing_endmarks",
			input: `flf2a@ 3 3 11 0 0
test@@@@@
data######
end!$$$$$$$
`,
			validate: func(t *testing.T, f *Font) {
				validateSpaceGlyphWithLines(t, f, []string{testContent, dataContent, "end!"})
			},
		},
		{
			name: "strip_massive_trailing_run",
			input: `flf2a@ 2 2 39 0 0
hello@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
world@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
`,
			validate: func(t *testing.T, f *Font) {
				validateSpaceGlyphWithLines(t, f, []string{"hello", "world"})
			},
		},
		{
			name: "endmark_inside_content_preserved",
			input: `flf2a@ 2 2 10 0 0
te@st@@@
da@ta@@@
`,
			validate: func(t *testing.T, f *Font) {
				validateSpaceGlyphWithLines(t, f, []string{"te@st", "da@ta"})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			font, err := Parse(r)
			if err != nil {
				t.Fatalf("Parse() unexpected error = %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, font)
			}
		})
	}
}

// TestParseGlyphs_PerGlyphEndmark tests that each glyph can have a different
// endmark character (the last character of each line determines the endmark)
func TestParseGlyphs_PerGlyphEndmark(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		validate func(t *testing.T, f *Font)
	}{
		{
			name: "different_endmark_per_glyph",
			input: `flf2a@ 2 2 10 0 0
test@@
data@@
next##
line##
foo$$
bar$$
`,
			validate: func(t *testing.T, f *Font) {
				validateMultipleGlyphs(t, f, []glyphValidation{
					{' ', "space", []string{testContent, dataContent}},
					{'!', "exclamation", []string{"next", "line"}},
					{'"', "quote", []string{"foo", "bar"}},
				})
			},
		},
		{
			name: "glyph_using_at_in_content",
			input: `flf2a@ 2 2 10 0 0
test@@
data@@
@art##
@pic##
`,
			validate: func(t *testing.T, f *Font) {
				validateMultipleGlyphs(t, f, []glyphValidation{
					{' ', "space", []string{testContent, dataContent}},
					{'!', "exclamation", []string{"@art", "@pic"}},
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			font, err := Parse(r)
			if err != nil {
				t.Fatalf("Parse() unexpected error = %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, font)
			}
		})
	}
}

// TestParseGlyphs_MultiByteEndmark tests that multi-byte (UTF-8) endmarks
// are handled correctly using rune-aware operations
func TestParseGlyphs_MultiByteEndmark(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		validate func(t *testing.T, f *Font)
	}{
		{
			name:  "emoji_endmark",
			input: "flf2a@ 2 2 12 0 0\ntest😀😀\ndata😀😀\n",
			validate: func(t *testing.T, f *Font) {
				validateSpaceGlyphWithLines(t, f, []string{testContent, dataContent})
			},
		},
		{
			name:  "chinese_character_endmark",
			input: "flf2a@ 2 2 14 0 0\nhello中中中\nworld中中中\n",
			validate: func(t *testing.T, f *Font) {
				validateSpaceGlyphWithLines(t, f, []string{"hello", "world"})
			},
		},
		{
			name:  "mixed_endmarks_with_multibyte",
			input: "flf2a@ 3 3 13 0 0\ntest@@\ndata££\nend!世世世\n",
			validate: func(t *testing.T, f *Font) {
				validateSpaceGlyphWithLines(t, f, []string{testContent, dataContent, "end!"})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			font, err := Parse(r)
			if err != nil {
				t.Fatalf("Parse() unexpected error = %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, font)
			}
		})
	}
}

// TestParseGlyphs_WidthConsistency tests that all lines in a glyph have
// the same width after endmark stripping
func TestParseGlyphs_WidthConsistency(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		errContains string
		wantErr     bool
	}{
		{
			name: "inconsistent_width_should_error",
			input: `flf2a@ 3 3 10 0 0
test@@
da@@
longer@@
`,
			wantErr:     true,
			errContains: "inconsistent",
		},
		{
			name: "consistent_width_ok",
			input: `flf2a@ 3 3 10 0 0
test@@
data@@
line@@
`,
			wantErr: false,
		},
		{
			name: "varying_endmark_count_but_consistent_width",
			input: `flf2a@ 3 3 10 0 0
test@@
data@@@
line@@@@@@
`,
			wantErr: false, // After stripping, all are 4 chars wide
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			_, err := Parse(r)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Parse() error = nil, want error containing %q", tt.errContains)
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Parse() error = %v, want error containing %q", err, tt.errContains)
				}
			} else if err != nil {
				t.Errorf("Parse() unexpected error = %v", err)
			}
		})
	}
}

// TestParseHeader_BaselineValidation tests that baseline must be >= 1
func TestParseHeader_BaselineValidation(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		errContains string
		wantErr     bool
	}{
		{
			name: "baseline_zero_should_error",
			input: `flf2a@ 5 0 10 0 0
`,
			wantErr:     true,
			errContains: "baseline must be",
		},
		{
			name: "baseline_negative_should_error",
			input: `flf2a@ 5 -1 10 0 0
`,
			wantErr:     true,
			errContains: "baseline",
		},
		{
			name: "baseline_one_ok",
			input: `flf2a@ 5 1 10 0 0
`,
			wantErr: false,
		},
		{
			name: "baseline_equals_height_ok",
			input: `flf2a@ 5 5 10 0 0
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			_, err := ParseHeader(r)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseHeader() error = nil, want error containing %q", tt.errContains)
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ParseHeader() error = %v, want error containing %q", err, tt.errContains)
				}
			} else if err != nil {
				t.Errorf("ParseHeader() unexpected error = %v", err)
			}
		})
	}
}

// TestParseGlyphs_PartialFontEOF tests that the parser accepts a truncated font
// mid-stream (breaks on io.ErrUnexpectedEOF and returns what it has)
func TestParseGlyphs_PartialFontEOF(t *testing.T) {
	// space + '!' ok, then EOF in the middle of the '"' glyph
	input := "flf2a@ 2 2 10 0 0\n" +
		"sp@@\nce@@\n" + // ' '
		"ex@@\ncl@@\n" + // '!'
		"qu@@\n" // only 1 line of height=2, triggers ErrUnexpectedEOF

	f, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	// Space should be parsed
	space := mustGlyph(t, f, ' ')
	if space[0] != "sp" || space[1] != "ce" {
		t.Errorf("space = %v, want [sp, ce]", space)
	}

	// '!' should be parsed
	excl := mustGlyph(t, f, '!')
	if excl[0] != "ex" || excl[1] != "cl" {
		t.Errorf("! = %v, want [ex, cl]", excl)
	}

	// '"' should NOT be parsed (incomplete)
	if _, ok := f.Characters['"']; ok {
		t.Fatal("expected '\"' to be missing due to EOF")
	}
}

// TestParseGlyphs_EmptyRows tests that lines of only endmarks become empty strings
func TestParseGlyphs_EmptyRows(t *testing.T) {
	input := "flf2a# 2 2 10 0 0\n###\n###\n"
	f, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	space := mustGlyph(t, f, ' ')
	if space[0] != "" || space[1] != "" {
		t.Fatalf("expected empty rows, got %q", space)
	}
}

// TestParseGlyphs_WidthByRunes tests that width check counts runes not bytes
// One line uses multibyte runes; another uses same rune count in ASCII → should pass
func TestParseGlyphs_WidthByRunes(t *testing.T) {
	input := "flf2a@ 2 2 10 0 0\néé@@\nab@@\n" // both 2 runes after stripping
	if _, err := Parse(strings.NewReader(input)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestParseGlyphs_InvalidUTF8Endmark tests the fallback for invalid UTF-8 at line end
func TestParseGlyphs_InvalidUTF8Endmark(t *testing.T) {
	// Build input with a trailing lone continuation byte
	b := append([]byte("flf2a@ 1 1 10 0 0\nx"), 0x80, 0x80, '\n')
	f, err := Parse(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	space := mustGlyph(t, f, ' ')
	if space[0] != "x" {
		t.Fatalf("got %q, want \"x\"", space[0])
	}
}
