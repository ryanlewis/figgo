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

// Layout bit constants aligned with PRD §7 / public API
const (
	// FitFullWidth displays characters at full width with no overlap
	FitFullWidth = 0
	// FitKerning displays characters with minimal spacing, no overlap (bit 6)
	FitKerning = 1 << 6
	// FitSmushing allows characters to overlap using smushing rules (bit 7)
	FitSmushing = 1 << 7
)

// Common errors
var (
	// ErrUnknownFont is returned when font is nil
	ErrUnknownFont = errors.New("unknown font")
	// ErrUnsupportedRune is returned when a rune is not supported by the font
	ErrUnsupportedRune = errors.New("unsupported rune")
	// ErrBadFontFormat is returned when font has invalid structure
	ErrBadFontFormat = errors.New("bad font format")
	// ErrLayoutConflict mirrors public API intent
	ErrLayoutConflict = errors.New("layout conflict")
)

// pickLayout determines the effective layout from font defaults and options
func pickLayout(font *parser.Font, opts *Options) (int, error) {
	// 1) start from opts if provided; else fall back to font defaults
	layout := FitFullWidth
	if opts != nil {
		layout = opts.Layout // 0 (full) is valid
	} else {
		// Fallback from font.OldLayout (per spec): -1 full, 0 kern, >0 smush
		switch font.OldLayout {
		case -1:
			layout = FitFullWidth
		case 0:
			layout = FitKerning
		default:
			layout = FitSmushing
		}
	}

	// 2) validation per PRD §7
	fitBits := 0
	if layout&FitKerning != 0 {
		fitBits++
	}
	if layout&FitSmushing != 0 {
		fitBits++
	}
	if fitBits > 1 {
		return 0, ErrLayoutConflict
	}
	if fitBits == 0 {
		layout = FitFullWidth // neither set -> full-width
	}
	return layout, nil
}

// Render converts text to ASCII art using the specified font and options.
func Render(text string, font *parser.Font, opts *Options) (string, error) {
	// Check font validity
	if font == nil {
		return "", ErrUnknownFont
	}
	if font.Height <= 0 {
		return "", ErrBadFontFormat
	}

	// Determine layout mode with validation
	layout, err := pickLayout(font, opts)
	if err != nil {
		return "", err
	}

	// Determine print direction (0 LTR, 1 RTL)
	printDir := font.PrintDirection
	if opts != nil {
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
