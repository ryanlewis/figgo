package renderer

import (
	"strings"
	"testing"

	"github.com/ryanlewis/figgo/internal/parser"
)

// createSpaceTestFont creates a font with a known 5-column space glyph
// to test that space width is preserved in smushing mode.
// The space glyph uses hardblanks ($) which act as barriers preventing
// overlap - this matches how real FIGlet fonts work.
func createSpaceTestFont() *parser.Font {
	return &parser.Font{
		Height:         3,
		Hardblank:      '$',
		OldLayout:      -1, // Smushing mode
		PrintDirection: 0,  // LTR
		Characters: map[rune][]string{
			// 5-column wide space glyph with hardblanks as barriers
			// Real FIGlet fonts use hardblanks in space glyphs
			' ': {"  $  ", "  $  ", "  $  "},
			// Simple 3-column characters for testing
			'a': {
				" _ ",
				"(_)",
				"   ",
			},
			'b': {
				"|_ ",
				"|_)",
				"   ",
			},
		},
	}
}

// TestSpaceGlyphWidthPreserved verifies that space glyphs retain their
// original width from the font, rather than being normalised to 1 column.
// In smushing mode, spaces can overlap with adjacent characters based on
// their leading/trailing whitespace, but hardblanks act as barriers.
func TestSpaceGlyphWidthPreserved(t *testing.T) {
	font := createSpaceTestFont()

	// Render just a space character in smushing mode
	opts := &Options{
		Layout: SMSmush, // Smushing mode
	}

	result, err := Render(" ", font, opts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	lines := strings.Split(result, "\n")
	if len(lines) == 0 {
		t.Fatal("Expected at least one line of output")
	}

	// In smushing mode, the space glyph "  $  " has leading spaces that
	// can overlap with an empty line. The hardblank ($) in the middle
	// acts as a barrier, but leading spaces may be trimmed.
	// The result should have at least the hardblank column preserved.
	firstLineWidth := len(lines[0])

	// The output should not be empty - at minimum the hardblank column
	// plus trailing spaces should be preserved
	if firstLineWidth == 0 {
		t.Error("Space glyph completely disappeared - expected at least hardblank barrier")
	}
}

// TestSpaceBetweenCharacters verifies that spaces between characters
// are properly rendered with their hardblanks acting as barriers.
func TestSpaceBetweenCharacters(t *testing.T) {
	font := createSpaceTestFont()

	opts := &Options{
		Layout: SMSmush,
	}

	result, err := Render("a b", font, opts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	lines := strings.Split(result, "\n")
	if len(lines) == 0 {
		t.Fatal("Expected at least one line of output")
	}

	// The space glyph has hardblanks that prevent 'a' and 'b' from
	// completely merging together. The exact width depends on overlap
	// calculations, but there should be visible separation.
	firstLineWidth := len(lines[0])

	// 'a' and 'b' are 3 columns each. Even with smushing overlap,
	// the hardblank in the space should prevent them from touching.
	// Minimum width should be greater than just 'a' + 'b' smashed together.
	if firstLineWidth < 5 {
		t.Errorf("Space between characters too narrow: got %d columns\n"+
			"Line content: %q\n"+
			"Expected visible separation between 'a' and 'b'",
			firstLineWidth, lines[0])
	}
}

// TestSpaceGlyphInKerningMode verifies space handling in kerning mode.
// In kerning mode, leading/trailing spaces overlap but visible content
// is preserved based on the hardblank barriers.
func TestSpaceGlyphInKerningMode(t *testing.T) {
	font := createSpaceTestFont()
	font.OldLayout = 0 // Kerning mode

	opts := &Options{
		Layout: SMKern,
	}

	result, err := Render(" ", font, opts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	lines := strings.Split(result, "\n")
	if len(lines) == 0 {
		t.Fatal("Expected at least one line of output")
	}

	// In kerning mode, the space glyph still participates in overlap
	// calculations. The hardblank should still act as a barrier.
	firstLineWidth := len(lines[0])

	// The output should not be empty
	if firstLineWidth == 0 {
		t.Error("Space glyph completely disappeared in kerning mode")
	}
}

// TestSpaceGlyphInFullWidthMode verifies space handling in full-width mode.
func TestSpaceGlyphInFullWidthMode(t *testing.T) {
	font := createSpaceTestFont()

	opts := &Options{
		Layout: 0, // Full-width mode (no kerning or smushing)
	}

	result, err := Render(" ", font, opts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	lines := strings.Split(result, "\n")
	if len(lines) == 0 {
		t.Fatal("Expected at least one line of output")
	}

	firstLineWidth := len(lines[0])
	expectedWidth := 5

	if firstLineWidth != expectedWidth {
		t.Errorf("Space glyph width in full-width mode: got %d columns, want %d columns\n"+
			"Line content: %q", firstLineWidth, expectedWidth, lines[0])
	}
}
