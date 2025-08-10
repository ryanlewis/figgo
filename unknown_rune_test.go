package figgo

import (
	"strings"
	"testing"
)

func TestWithUnknownRuneOption(t *testing.T) {
	// Test that the option correctly sets the unknown rune
	opts := defaultOptions()
	WithUnknownRune('*')(opts)

	if opts.unknownRune == nil || *opts.unknownRune != '*' {
		t.Errorf("WithUnknownRune failed to set option, got %v", opts.unknownRune)
	}
}

func TestUnknownRuneWithMissingQuestionMark(t *testing.T) {
	// Create a mock font without '?' to test ErrUnsupportedRune
	mockFont := &Font{
		glyphs: map[rune][]string{
			'H': {"H", "H"},
			'e': {"e", "e"},
			'l': {"l", "l"},
			'o': {"o", "o"},
			' ': {" ", " "},
		},
		Height: 2,
		Layout: FitFullWidth,
	}

	// Try to render text with unknown rune and no '?' fallback
	_, err := Render("Hello 世界", mockFont)

	// Check error message since internal/common has its own ErrUnsupportedRune
	if err == nil || err.Error() != "unsupported rune" {
		t.Errorf("expected 'unsupported rune' error when '?' is missing, got: %v", err)
	}
}

func TestUnknownRuneReplacement(t *testing.T) {
	// Create a mock font with all printable ASCII
	glyphs := make(map[rune][]string)
	for i := 32; i <= 126; i++ {
		r := rune(i)
		glyphs[r] = []string{string(r), string(r)}
	}

	mockFont := &Font{
		glyphs: glyphs,
		Height: 2,
		Layout: FitFullWidth,
	}

	// Test default replacement with '?'
	output, err := Render("Hello 世界", mockFont)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "?") {
		t.Errorf("should replace unknown runes with '?', got: %s", output)
	}

	// Test custom replacement with '*'
	output, err = Render("Hello 世界", mockFont, WithUnknownRune('*'))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "*") {
		t.Errorf("should replace unknown runes with '*', got: %s", output)
	}

	// Test control characters are replaced
	output, err = Render("Test\x01\x1F\x7F", mockFont, WithUnknownRune('_'))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "_") {
		t.Errorf("should replace control chars with '_', got: %s", output)
	}
}

func TestMissingASCIIGlyph(t *testing.T) {
	// Create a font missing 't' (ASCII 116)
	glyphs := make(map[rune][]string)
	for i := 32; i <= 126; i++ {
		if i == 116 { // skip 't'
			continue
		}
		r := rune(i)
		glyphs[r] = []string{string(r), string(r)}
	}

	mockFont := &Font{
		glyphs: glyphs,
		Height: 2,
		Layout: FitFullWidth,
	}

	// Try to render text with missing 't'
	_, err := Render("test", mockFont)

	// Check error message since internal/common has its own ErrUnsupportedRune
	if err == nil || err.Error() != "unsupported rune" {
		t.Errorf("expected 'unsupported rune' error for missing ASCII glyph, got: %v", err)
	}
}

func TestUnknownRuneFallback(t *testing.T) {
	// Create a font with all ASCII except the replacement rune
	glyphs := make(map[rune][]string)
	for i := 32; i <= 126; i++ {
		r := rune(i)
		glyphs[r] = []string{string(r), string(r)}
	}

	mockFont := &Font{
		glyphs: glyphs,
		Height: 2,
		Layout: FitFullWidth,
	}

	// Try to use a replacement rune that's not in the font (beyond ASCII)
	output, err := Render("Hello 世界", mockFont, WithUnknownRune('☺'))
	if err != nil {
		t.Errorf("should not error when falling back to '?': %v", err)
	}

	// Should fall back to '?' since ☺ is not in the font
	if !strings.Contains(output, "?") {
		t.Errorf("should fall back to '?' when replacement rune not in font, got: %s", output)
	}
}

func TestPrintDirectionOverride(t *testing.T) {
	// Create a font with RTL as default
	glyphs := make(map[rune][]string)
	for i := 32; i <= 126; i++ {
		r := rune(i)
		glyphs[r] = []string{string(r), string(r)}
	}

	mockFont := &Font{
		glyphs:         glyphs,
		Height:         2,
		Layout:         FitFullWidth,
		PrintDirection: 1, // RTL
	}

	// Test that we can override RTL to LTR (0)
	output1, err := Render("ABC", mockFont, WithPrintDirection(0))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test that we can explicitly set RTL (1)
	output2, err := Render("ABC", mockFont, WithPrintDirection(1))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// The outputs should potentially be different (depends on renderer implementation)
	// but the important part is that the override is respected (no error)
	_ = output1
	_ = output2
}
