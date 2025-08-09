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
			input: generateFontWithDeutschCharacters(),
			validate: func(t *testing.T, f *Font) {
				// Should have exactly 102 characters (95 ASCII + 7 German)
				if len(f.Characters) != 102 {
					t.Errorf("Expected 102 characters, got %d", len(f.Characters))
				}

				// Check all ASCII characters are present
				for r := rune(32); r <= 126; r++ {
					if _, exists := f.Characters[r]; !exists {
						t.Errorf("Missing ASCII character %d (%c)", r, r)
					}
				}

				// Check all German characters are present
				deutschChars := []rune{196, 214, 220, 228, 246, 252, 223}
				deutschNames := []string{"Ä", "Ö", "Ü", "ä", "ö", "ü", "ß"}
				for i, r := range deutschChars {
					if _, exists := f.Characters[r]; !exists {
						t.Errorf("Missing German character %d (%s)", r, deutschNames[i])
					}
				}
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
				// Check German Ä (196)
				if glyph, exists := f.Characters[196]; exists {
					if len(glyph) != 3 {
						t.Errorf("German Ä should have 3 lines, got %d", len(glyph))
					}
					if glyph[0] != " _   _ " {
						t.Errorf("German Ä line 0 = %q, want %q", glyph[0], " _   _ ")
					}
				} else {
					t.Error("German Ä (196) not found")
				}

				// Check German ß (223)
				if glyph, exists := f.Characters[223]; exists {
					if len(glyph) != 3 {
						t.Errorf("German ß should have 3 lines, got %d", len(glyph))
					}
					if glyph[0] != "  ___ " {
						t.Errorf("German ß line 0 = %q, want %q", glyph[0], "  ___ ")
					}
				} else {
					t.Error("German ß (223) not found")
				}
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
				// All German characters should be empty (zero-width)
				deutschChars := []rune{196, 214, 220, 228, 246, 252, 223}
				for _, r := range deutschChars {
					if glyph, exists := f.Characters[r]; exists {
						for i, line := range glyph {
							if line != "" {
								t.Errorf("German char %d line %d should be empty, got %q", r, i, line)
							}
						}
					} else {
						t.Errorf("German character %d not found", r)
					}
				}
			},
		},
		{
			name: "partial_font_without_deutsch_chars",
			input: `flf2a@ 2 2 5 0 0
 @
 @@
` + strings.Repeat("x@\nx@@\n", 94), // Only ASCII 33-126, no German chars
			validate: func(t *testing.T, f *Font) {
				// Should have 95 ASCII characters
				if len(f.Characters) != 95 {
					t.Errorf("Expected 95 ASCII characters, got %d", len(f.Characters))
				}

				// German characters should NOT be present (graceful degradation)
				deutschChars := []rune{196, 214, 220, 228, 246, 252, 223}
				for _, r := range deutschChars {
					if _, exists := f.Characters[r]; exists {
						t.Errorf("German character %d should not exist in partial font", r)
					}
				}
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

// generateFontWithDeutschCharacters creates a test font with all 102 required characters
func generateFontWithDeutschCharacters() string {
	var sb strings.Builder
	// Header: height=2, baseline=2, maxlength=10, oldlayout=0, comments=0
	sb.WriteString("flf2a@ 2 2 10 0 0\n")

	// ASCII 32 (space)
	sb.WriteString(" @\n")
	sb.WriteString(" @@\n")

	// ASCII 33-126 (simplified)
	for i := 33; i <= 126; i++ {
		sb.WriteString("x@\n")
		sb.WriteString("x@@\n")
	}

	// German characters (196, 214, 220, 228, 246, 252, 223)
	// Using simple 'G' prefix to indicate German chars
	for i := 0; i < 7; i++ {
		sb.WriteString("G@\n")
		sb.WriteString("G@@\n")
	}

	return sb.String()
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
