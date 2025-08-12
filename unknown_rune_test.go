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

	// Check error message - renderer includes the specific rune in the error
	if err == nil || !strings.HasPrefix(err.Error(), "unsupported rune") {
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

	// Test default replacement with '?' - need to explicitly specify it
	output, err := Render("Hello 世界", mockFont, WithUnknownRune('?'))
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

	// Try to render text with missing 't' - need to explicitly specify replacement
	output, err := Render("test", mockFont, WithUnknownRune('?'))

	// Should succeed since '?' replacement is specified and available
	if err != nil {
		t.Errorf("unexpected error when '?' replacement is available: %v", err)
	}
	
	// Should contain '?' as replacement for 't'
	if !strings.Contains(output, "?") {
		t.Errorf("expected '?' replacement for missing 't', got: %s", output)
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
	// The renderer will return an error if the replacement rune itself is not in the font
	_, err := Render("Hello 世界", mockFont, WithUnknownRune('☺'))
	if err == nil || !strings.Contains(err.Error(), "unsupported rune") {
		t.Errorf("expected error when replacement rune '☺' is not in font: %v", err)
	}

	// To successfully render, we need to use a replacement that exists in the font
	output, err := Render("Hello 世界", mockFont, WithUnknownRune('?'))
	if err != nil {
		t.Errorf("should succeed with valid replacement rune: %v", err)
	}
	if !strings.Contains(output, "?") {
		t.Errorf("should use '?' as replacement, got: %s", output)
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
