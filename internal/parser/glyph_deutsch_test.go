package parser

import (
	"fmt"
	"strings"
	"testing"
)

// TestParseGlyphs_RequiredDeutschCharacters tests that the parser correctly
// handles the 7 required German characters after ASCII 126, as specified in
// the FIGfont v2 specification (lines 1046-1060).
func TestParseGlyphs_RequiredDeutschCharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		validate func(t *testing.T, f *Font)
	}{
		{
			name:  "all_102_required_characters",
			input: GenerateFontWithDeutschChars(),
			validate: func(t *testing.T, f *Font) {
				ValidateCharCount(t, f, 102)
				ValidateAllASCIIChars(t, f)
				ValidateAllGermanChars(t, f)
			},
		},
		{
			name: "deutsch_characters_with_content",
			input: `flf2a@ 3 2 10 0 0
  _ @
 | |@
 |_|@@
!@
!@
!@@
` + strings.Repeat("x@\nx@\nx@@\n", 92) + // ASCII 33-125 except tilde
				`~@
~@
~@@
 _   _ @
(_) (_)@
|_| |_|@@
 _   _ @
(_) (_)@
| | | |@@
 _   _ @
(_) (_)@
| | | |@@
 _   _ @
(_) (_)@
| |_|_|@@
 _   _ @
(_) (_)@
|_| |_|@@
 _   _ @
(_) (_)@
|_|_|_|@@
  ___ @
 / _ \@
| |_/ @@`,
			validate: func(t *testing.T, f *Font) {
				ValidateGermanChar(t, f, 196, []string{" _   _ ", "(_) (_)", "|_| |_|"}, "Ä")
				ValidateGermanChar(t, f, 223, []string{"  ___ ", " / _ \\", "| |_/ "}, "ß")
			},
		},
		{
			name: "deutsch_characters_as_empty",
			input: `flf2a@ 2 2 5 0 0
 @
 @@
` + strings.Repeat("x@\nx@@\n", 94) + // ASCII 33-126
				`@@
@@
@@
@@
@@
@@
@@
@@
@@
@@
@@
@@
@@
@@`,
			validate: func(t *testing.T, f *Font) {
				ValidateGermanCharsEmpty(t, f)
			},
		},
		{
			name: "partial_font_without_deutsch_chars",
			input: `flf2a@ 2 2 5 0 0
 @
 @@
` + strings.Repeat("x@\nx@@\n", 94), // Only ASCII 33-126, no German chars
			validate: func(t *testing.T, f *Font) {
				ValidateCharCount(t, f, 95)
				ValidateGermanCharsAbsent(t, f)
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

// TestParseGlyphs_DeutschCharacterOrder verifies that German characters
// are parsed in the exact order specified by the FIGfont spec.
func TestParseGlyphs_DeutschCharacterOrder(t *testing.T) {
	input := generateFontWithNumberedGlyphs()
	r := strings.NewReader(input)
	font, err := Parse(r)
	if err != nil {
		t.Fatalf("Parse() unexpected error = %v", err)
	}

	// The spec requires this exact order after ASCII 126
	expectedOrder := []struct {
		code rune
		name string
		num  string
	}{
		{196, "Ä", "96"},
		{214, "Ö", "97"},
		{220, "Ü", "98"},
		{228, "ä", "99"},
		{246, "ö", "100"},
		{252, "ü", "101"},
		{223, "ß", "102"},
	}

	for _, expected := range expectedOrder {
		glyph, exists := font.Characters[expected.code]
		if !exists {
			t.Errorf("Missing German character %d (%s)", expected.code, expected.name)
			continue
		}

		// Check that it has the right number (proving order is correct)
		if !strings.Contains(glyph[0], expected.num) {
			t.Errorf("German char %s: expected to contain %q in glyph, got %q",
				expected.name, expected.num, glyph[0])
		}
	}
}

// generateFontWithNumberedGlyphs creates a font where each glyph contains its position number
func generateFontWithNumberedGlyphs() string {
	var sb strings.Builder
	// Header: height=1, baseline=1, maxlength=10, oldlayout=0, comments=0
	sb.WriteString("flf2a@ 1 1 10 0 0\n")

	// Generate glyphs with their position number
	for i := 1; i <= 102; i++ {
		sb.WriteString(fmt.Sprintf("%d@@\n", i))
	}

	return sb.String()
}
