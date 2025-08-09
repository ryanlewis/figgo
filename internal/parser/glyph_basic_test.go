package parser

import (
	"strings"
	"testing"
)

// parseAndValidate is a helper function to parse input and validate the result
func parseAndValidate(t *testing.T, input string) *Font {
	t.Helper()
	r := strings.NewReader(input)
	font, err := Parse(r)
	if err != nil {
		t.Fatalf("Parse() unexpected error = %v", err)
	}
	return font
}

// TestParseGlyphs_SingleGlyph tests parsing a single glyph
func TestParseGlyphs_SingleGlyph(t *testing.T) {
	input := `flf2a$ 3 2 10 0 1
Single line comment
  _ @
 | |@
 |_|@@
`
	font := parseAndValidate(t, input)

	if font.Hardblank != '$' {
		t.Errorf("Hardblank = %q, want %q", font.Hardblank, '$')
	}
	if font.Height != 3 {
		t.Errorf("Height = %d, want %d", font.Height, 3)
	}

	ValidateSpace(t, font, []string{"  _ ", " | |", " |_|"})
}

// TestParseGlyphs_MultipleGlyphs tests parsing multiple glyphs
func TestParseGlyphs_MultipleGlyphs(t *testing.T) {
	input := `flf2a# 2 2 8 0 0
 ##
!##
"##
?##
`
	font := parseAndValidate(t, input)

	if font.Hardblank != '#' {
		t.Errorf("Hardblank = %q, want %q", font.Hardblank, '#')
	}

	ValidateSpace(t, font, []string{" ", "!"})

	// Check exclamation mark (ASCII 33)
	excl, exists := font.Characters['!']
	if !exists {
		t.Fatal("Exclamation character not found")
	}
	if len(excl) != 2 {
		t.Errorf("Exclamation has %d lines, want 2", len(excl))
	}
	if excl[0] != "\"" || excl[1] != "?" {
		t.Errorf("Exclamation = %v, want [%q, %q]", excl, "\"", "?")
	}
}

// TestParseGlyphs_AllASCII tests parsing all ASCII printable characters
func TestParseGlyphs_AllASCII(t *testing.T) {
	font := parseAndValidate(t, generateFullASCIIFont())

	// Should have exactly 95 characters (32-126)
	if len(font.Characters) != 95 {
		t.Errorf("Got %d characters, want 95", len(font.Characters))
	}

	// Check all ASCII printable characters are present
	for r := rune(32); r <= 126; r++ {
		if _, exists := font.Characters[r]; !exists {
			t.Errorf("Missing character %d (%c)", r, r)
		}
	}
}

// TestParseGlyphs_EmptyLines tests glyph with empty lines
func TestParseGlyphs_EmptyLines(t *testing.T) {
	input := `flf2a@ 4 3 8 0 0
   @
   @
   @
   @@
`
	font := parseAndValidate(t, input)
	ValidateSpace(t, font, []string{"   ", "   ", "   ", "   "})
}

// TestParseGlyphs_TrailingSpaces tests glyph with trailing spaces
func TestParseGlyphs_TrailingSpaces(t *testing.T) {
	input := `flf2a@ 2 2 10 0 0
test   @
line   @@
`
	font := parseAndValidate(t, input)
	ValidateSpace(t, font, []string{"test   ", "line   "})
}

// TestParseGlyphs_UnicodeEndmark tests unicode endmark handling
func TestParseGlyphs_UnicodeEndmark(t *testing.T) {
	input := "flf2a\u00A3 2 2 10 0 0\ntest\u00A3\ndata\u00A3\u00A3\u00A3\n"
	font := parseAndValidate(t, input)

	if font.Hardblank != '£' {
		t.Errorf("Hardblank = %q, want %q", font.Hardblank, '£')
	}
	ValidateSpace(t, font, []string{"test", "data"})
}

// TestParseGlyphs_HardblankInGlyph tests hardblank preservation in glyphs
func TestParseGlyphs_HardblankInGlyph(t *testing.T) {
	input := `flf2a$ 2 2 8 0 0
te$t@@
da$t@@
`
	font := parseAndValidate(t, input)
	ValidateSpace(t, font, []string{"te$t", "da$t"})
}

// TestParseGlyphs_DoubleEndmark tests that ALL trailing endmarks are stripped
func TestParseGlyphs_DoubleEndmark(t *testing.T) {
	input := `flf2a@ 2 2 8 0 0
test@@
data@@@
`
	font := parseAndValidate(t, input)
	// Per spec: "eliminate the last block of consecutive equal characters"
	// So test@@ becomes test, data@@@ becomes data
	ValidateSpace(t, font, []string{"test", "data"})
}

// TestParseGlyphs_EndmarkOnlyLines tests endmark-only lines
func TestParseGlyphs_EndmarkOnlyLines(t *testing.T) {
	input := `flf2a@ 3 2 8 0 0
@
@@
@@@
`
	font := parseAndValidate(t, input)
	// Lines with ONLY endmarks should be empty (zero-width)
	ValidateSpace(t, font, []string{"", "", ""})
}

// TestParseGlyphs_ErrorCases tests error handling
func TestParseGlyphs_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		errContains string
	}{
		{
			name: "incorrect_line_count",
			input: `flf2a@ 3 2 8 0 0
line1@
line2@@
`,
			errContains: "expected 3 lines",
		},
		// Removed "missing_endmark" test - the new parser is more lenient
		// and handles lines without explicit endmarks by treating the last
		// character as a single-character endmark run
		{
			name: "empty_glyph_data",
			input: `flf2a@ 2 2 8 0 0
`,
			errContains: "unexpected EOF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			_, err := Parse(r)
			if err == nil {
				t.Errorf("Parse() error = nil, want error containing %q", tt.errContains)
				return
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("Parse() error = %v, want error containing %q", err, tt.errContains)
			}
		})
	}
}
