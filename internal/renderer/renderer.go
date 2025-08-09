// Package renderer implements ASCII art rendering from parsed FIGfonts.
package renderer

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ryanlewis/figgo/internal/parser"
)

// Options configures rendering behavior.
type Options struct {
	// Layout controls the fitting/smushing rules
	Layout int

	// PrintDirection overrides the font's default print direction
	PrintDirection int
}

// Layout bit constants (matching the public API layout.go)
const (
	// FitFullWidth displays characters at full width with no overlap (bit 6)
	FitFullWidth = 0x00000040
	// FitKerning displays characters with minimal spacing, no overlap (bit 7)
	FitKerning = 0x00000080
	// FitSmushing allows characters to overlap using smushing rules (bit 8)
	FitSmushing = 0x00000100
)

// Common errors
var (
	// ErrUnknownFont is returned when font is nil
	ErrUnknownFont = errors.New("unknown font")
	// ErrUnsupportedRune is returned when a rune is not supported by the font
	ErrUnsupportedRune = errors.New("unsupported rune")
	// ErrBadFontFormat is returned when font has invalid structure
	ErrBadFontFormat = errors.New("bad font format")
)

// Render converts text to ASCII art using the specified font and options.
func Render(text string, font *parser.Font, opts *Options) (string, error) {
	// Check font validity
	if font == nil {
		return "", ErrUnknownFont
	}
	if font.Height <= 0 {
		return "", ErrBadFontFormat
	}

	// Determine layout mode
	layout := FitFullWidth // Default to full-width
	if opts != nil && opts.Layout != 0 {
		layout = opts.Layout
	}

	// Determine print direction
	printDir := font.PrintDirection
	if opts != nil && opts.PrintDirection != 0 {
		// Options override font default
		printDir = opts.PrintDirection
	}

	// For now, only implement Full-Width mode
	if layout == FitFullWidth {
		return renderFullWidth(text, font, printDir)
	}

	// Kerning and Smushing modes not yet implemented
	return "", fmt.Errorf("layout mode %x not yet implemented", layout)
}

// renderFullWidth renders text using Full-Width layout (no overlap)
func renderFullWidth(text string, font *parser.Font, printDir int) (string, error) {
	if font == nil {
		return "", ErrUnknownFont
	}

	h := font.Height
	if h <= 0 {
		return "", ErrBadFontFormat
	}

	// Handle empty text
	if text == "" {
		// Return empty lines matching font height
		lines := make([]string, h)
		return strings.Join(lines, "\n"), nil
	}

	// Convert text to runes
	runes := []rune(text)

	// Prepare line builders
	lines := make([]strings.Builder, h)
	// Pre-allocate reasonable capacity to minimize allocations
	const avgGlyphWidth = 5 // Average glyph width estimate
	estimatedWidth := len(runes) * avgGlyphWidth
	for i := range lines {
		lines[i].Grow(estimatedWidth)
	}

	// Compose glyphs in logical order (LTR)
	for _, r := range runes {
		glyph, ok := font.Characters[r]
		if !ok {
			// Policy for this ticket: fail on missing glyph
			return "", ErrUnsupportedRune
		}

		// Validate glyph height matches font height
		if len(glyph) != h {
			return "", ErrBadFontFormat
		}

		// Append each row as-is, no overlap/trim
		for i := 0; i < h; i++ {
			lines[i].WriteString(glyph[i])
		}
	}

	// Post-process: replace hardblank with space
	hb := font.Hardblank
	for i := 0; i < h; i++ {
		line := lines[i].String()
		// Replace all hardblank characters with spaces
		line = strings.ReplaceAll(line, string(hb), " ")
		lines[i].Reset()
		lines[i].WriteString(line)
	}

	// Apply print direction (1 = RTL): reverse each line
	if printDir == 1 {
		for i := 0; i < h; i++ {
			line := lines[i].String()
			// Reverse the line (ASCII-safe byte reversal)
			reversed := reverseString(line)
			lines[i].Reset()
			lines[i].WriteString(reversed)
		}
	}

	// Join lines with newlines
	result := make([]string, h)
	for i := 0; i < h; i++ {
		result[i] = lines[i].String()
	}

	return strings.Join(result, "\n"), nil
}

// reverseString reverses a string byte-by-byte (safe for ASCII)
func reverseString(s string) string {
	b := []byte(s)
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}
