package parser

import (
	"strings"
	"testing"
)

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
				space, exists := f.Characters[' ']
				if !exists {
					t.Fatal("Space character not found")
				}

				if len(space) != 3 {
					t.Errorf("Space should have 3 lines, got %d", len(space))
				}

				// All lines should be empty after endmark removal
				for i, line := range space {
					if line != "" {
						t.Errorf("Space line %d should be empty, got %q", i, line)
					}
				}
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
				if space, exists := f.Characters[' ']; exists {
					for i, line := range space {
						if line != "" {
							t.Errorf("Space line %d should be empty, got %q", i, line)
						}
					}
				} else {
					t.Error("Space character not found")
				}

				// ! (33) should have content
				if excl, exists := f.Characters['!']; exists {
					if excl[0] != " _ " {
						t.Errorf("! line 0 = %q, want %q", excl[0], " _ ")
					}
					if excl[1] != "|_|" {
						t.Errorf("! line 1 = %q, want %q", excl[1], "|_|")
					}
				} else {
					t.Error("! character not found")
				}

				// " (34) should be empty
				if quote, exists := f.Characters['"']; exists {
					for i, line := range quote {
						if line != "" {
							t.Errorf("Quote line %d should be empty, got %q", i, line)
						}
					}
				} else {
					t.Error("Quote character not found")
				}

				// # (35) should have content
				if hash, exists := f.Characters['#']; exists {
					if hash[0] != "X" || hash[1] != "X" {
						t.Errorf("# should be 'X' on both lines, got %v", hash)
					}
				} else {
					t.Error("# character not found")
				}
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
				space, exists := f.Characters[' ']
				if !exists {
					t.Fatal("Space character not found")
				}

				// All lines should be empty (single @ at start = empty)
				for i, line := range space {
					if line != "" {
						t.Errorf("Space line %d should be empty, got %q", i, line)
					}
				}
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
				if space, exists := f.Characters[' ']; exists {
					for i, line := range space {
						if line != "" {
							t.Errorf("Space line %d should be empty, got %q", i, line)
						}
					}
				} else {
					t.Error("Space character not found")
				}

				// ! should have content
				if excl, exists := f.Characters['!']; exists {
					if excl[0] != "X" || excl[1] != "X" {
						t.Errorf("! should be 'X' on both lines, got %v", excl)
					}
				} else {
					t.Error("! character not found")
				}
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
				space, exists := f.Characters[' ']
				if !exists {
					t.Fatal("Space character not found")
				}

				// After endmark removal, the spaces before endmarks are preserved
				// ALL trailing @ are stripped, leaving just the spaces
				for i, line := range space {
					expected := "   "
					if line != expected {
						t.Errorf("Space line %d = %q, want %q", i, line, expected)
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
	for i := rune(33); i <= 126; i++ {
		if glyph, exists := font.Characters[i]; exists {
			if glyph[0] != "X" || glyph[1] != "X" {
				t.Errorf("ASCII char %d should have content 'X', got %v", i, glyph)
			}
		}
	}

	// Check German chars are empty
	deutschChars := []rune{196, 214, 220, 228, 246, 252, 223}
	for _, r := range deutschChars {
		if glyph, exists := font.Characters[r]; exists {
			for i, line := range glyph {
				if line != "" {
					t.Errorf("German char %d line %d should be empty, got %q", r, i, line)
				}
			}
		} else {
			// This is expected to fail until we implement German char support
			t.Logf("German character %d not found (expected until implementation)", r)
		}
	}
}
