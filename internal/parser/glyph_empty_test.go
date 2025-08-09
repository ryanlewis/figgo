package parser

import (
	"strings"
	"testing"
)

// Helper functions to reduce test complexity

// validateEmptyGlyph checks that all lines in a glyph are empty (local helper for specific tests)
func validateEmptyGlyph(t *testing.T, glyph []string, charName string) {
	t.Helper()
	for i, line := range glyph {
		if line != "" {
			t.Errorf("%s line %d should be empty, got %q", charName, i, line)
		}
	}
}

// validateGlyphContent checks that a glyph has expected content (local helper for specific tests)
func validateGlyphContent(t *testing.T, glyph, expected []string, charName string) {
	t.Helper()
	if len(glyph) != len(expected) {
		t.Errorf("%s should have %d lines, got %d", charName, len(expected), len(glyph))
		return
	}
	for i, expectedLine := range expected {
		if glyph[i] != expectedLine {
			t.Errorf("%s line %d = %q, want %q", charName, i, glyph[i], expectedLine)
		}
	}
}

// TestParseGlyphs_EmptyFIGcharacters tests support for empty FIGcharacters
// as specified in the FIGfont spec (lines 1062-1064): "You MAY create 'empty'
// FIGcharacters by placing endmarks flush with the left margin."
func TestParseGlyphs_EmptyFIGcharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		validate func(t *testing.T, f *Font)
	}{
		{
			name: "empty_space_character",
			input: `flf2a@ 3 3 5 0 0
@@
@@
@@@
`,
			validate: func(t *testing.T, f *Font) {
				space := ValidateCharExists(t, f, ' ', "Space")
				if len(space) != 3 {
					t.Errorf("Space should have 3 lines, got %d", len(space))
				}
				validateEmptyGlyph(t, space, "Space")
			},
		},
		{
			name: "mixed_empty_and_content_characters",
			input: `flf2a@ 2 2 8 0 0
@@
@@
 _ @
|_|@@
@@
@@
X@
X@@
`,
			validate: func(t *testing.T, f *Font) {
				// Space (32) should be empty
				space := ValidateCharExists(t, f, ' ', "Space")
				validateEmptyGlyph(t, space, "Space")

				// ! (33) should have content
				excl := ValidateCharExists(t, f, '!', "!")
				validateGlyphContent(t, excl, []string{" _ ", "|_|"}, "!")

				// " (34) should be empty
				quote := ValidateCharExists(t, f, '"', "Quote")
				validateEmptyGlyph(t, quote, "Quote")

				// # (35) should have content
				hash := ValidateCharExists(t, f, '#', "#")
				validateGlyphContent(t, hash, []string{"X", "X"}, "#")
			},
		},
		{
			name: "single_endmark_empty_glyph",
			input: `flf2a@ 4 4 5 0 0
@
@
@
@@
`,
			validate: func(t *testing.T, f *Font) {
				space := ValidateCharExists(t, f, ' ', "Space")
				validateEmptyGlyph(t, space, "Space")
			},
		},
		{
			name: "empty_with_different_endmark",
			input: `flf2a# 2 2 5 0 0
##
###
X#
X##
`,
			validate: func(t *testing.T, f *Font) {
				// Space should be empty
				space := ValidateCharExists(t, f, ' ', "Space")
				validateEmptyGlyph(t, space, "Space")

				// ! should have content
				excl := ValidateCharExists(t, f, '!', "!")
				validateGlyphContent(t, excl, []string{"X", "X"}, "!")
			},
		},
		{
			name: "zero_width_after_processing",
			input: `flf2a@ 3 3 10 0 0
   @@
   @@
   @@@
ABC@
DEF@
GHI@@
`,
			validate: func(t *testing.T, f *Font) {
				// Space has spaces before endmarks - these should be preserved
				space := ValidateCharExists(t, f, ' ', "Space")
				// After endmark removal, the spaces before endmarks are preserved
				// ALL trailing @ are stripped, leaving just the spaces
				validateGlyphContent(t, space, []string{"   ", "   ", "   "}, "Space")
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

// TestParseGlyphs_EmptyGermanCharacters tests that German characters
// can be defined as empty, which is a common pattern in FIGfonts
func TestParseGlyphs_EmptyGermanCharacters(t *testing.T) {
	// Generate font with ASCII content but empty German chars
	var sb strings.Builder
	sb.WriteString("flf2a@ 2 2 5 0 0\n")

	// ASCII characters with content
	for i := 32; i <= 126; i++ {
		if i == 32 {
			sb.WriteString(" @\n @@\n")
		} else {
			sb.WriteString("X@\nX@@\n")
		}
	}

	// German characters as empty (just endmarks)
	for i := 0; i < 7; i++ {
		sb.WriteString("@@\n@@\n")
	}

	r := strings.NewReader(sb.String())
	font, err := Parse(r)
	if err != nil {
		t.Fatalf("Parse() unexpected error = %v", err)
	}

	// Check ASCII chars have content
	ValidateASCIICharsHaveContent(t, font, "X")

	// Check German chars are empty
	ValidateGermanCharsEmpty(t, font)
}
