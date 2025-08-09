// Package renderer implements ASCII art rendering from parsed FIGfonts.
package renderer

import (
	"fmt"
	"strings"

	"github.com/ryanlewis/figgo/internal/common"
	"github.com/ryanlewis/figgo/internal/parser"
)

// Options configures rendering behavior.
type Options struct {
	// Layout controls the fitting/smushing rules
	Layout int

	// PrintDirection overrides the font's default print direction
	PrintDirection int
}

// normalizeFit validates and normalizes a layout value
func normalizeFit(l int) (int, error) {
	// Extract fitting mode and rule bits separately
	fittingMask := common.FitKerning | common.FitSmushing
	ruleMask := common.RuleEqualChar | common.RuleUnderscore | common.RuleHierarchy |
		common.RuleOppositePair | common.RuleBigX | common.RuleHardblank

	fitting := l & fittingMask
	rules := l & ruleMask

	// Count fitting mode bits
	fitBits := 0
	if (fitting & common.FitKerning) != 0 {
		fitBits++
	}
	if (fitting & common.FitSmushing) != 0 {
		fitBits++
	}

	// Validate at most one fitting mode
	if fitBits > 1 {
		return 0, common.ErrLayoutConflict
	}

	// If no fitting bits set, default to full-width
	// Preserve rule bits even though they're ignored without smushing
	if fitBits == 0 {
		return common.FitFullWidth | rules, nil
	}

	// Return the fitting mode with preserved rule bits
	return fitting | rules, nil
}

// pickLayout determines the effective layout from font defaults and options
func pickLayout(font *parser.Font, opts *Options) (int, error) {
	if opts != nil {
		return normalizeFit(opts.Layout)
	}

	// Prefer FullLayout if available (per PRD §7)
	if font.FullLayoutSet {
		// FullLayout=0 with FullLayoutSet=true means full width
		if font.FullLayout == 0 {
			return common.FitFullWidth, nil
		}
		return normalizeFit(font.FullLayout)
	}

	// Fallback to OldLayout
	switch font.OldLayout {
	case -1:
		return common.FitFullWidth, nil
	case 0:
		return common.FitKerning, nil
	default:
		return common.FitSmushing, nil
	}
}

// Render converts text to ASCII art using the specified font and options.
func Render(text string, font *parser.Font, opts *Options) (string, error) {
	// Check font validity
	if font == nil {
		return "", common.ErrUnknownFont
	}
	if font.Height <= 0 {
		return "", common.ErrBadFontFormat
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
	// Validate print direction
	if printDir != 0 && printDir != 1 {
		printDir = 0 // default to LTR
	}

	// Extract the fitting mode (ignore rule bits)
	fittingMode := layout & (common.FitKerning | common.FitSmushing)

	switch fittingMode {
	case 0: // No fitting bits set = full-width
		return renderFullWidth(text, font, printDir)
	case common.FitKerning:
		return renderKerning(text, font, printDir)
	case common.FitSmushing:
		// Smushing mode not yet implemented
		return "", fmt.Errorf("smushing mode not yet implemented")
	default:
		return "", common.ErrLayoutConflict
	}
}

// filterNonASCII replaces non-ASCII characters with '?' (PRD MVP policy)
func filterNonASCII(runes []rune) {
	for i, r := range runes {
		if r < 32 || r > 126 {
			runes[i] = '?'
		}
	}
}

// composeGlyphs assembles glyphs into lines
func composeGlyphs(runes []rune, font *parser.Font, h int) ([][]byte, error) {
	const avgGlyphWidth = 10 // Average glyph width estimate (increased to reduce reslices)
	estimatedWidth := len(runes) * avgGlyphWidth
	lines := make([][]byte, h)
	for i := range lines {
		lines[i] = make([]byte, 0, estimatedWidth)
	}

	for _, r := range runes {
		glyph, ok := font.Characters[r]
		if !ok {
			// If '?' is missing, try space as a last resort
			if r == '?' {
				if spaceGlyph, hasSpace := font.Characters[' ']; hasSpace {
					glyph = spaceGlyph
					ok = true
				}
			}
			if !ok {
				return nil, common.ErrUnsupportedRune
			}
		}
		if len(glyph) != h {
			return nil, common.ErrBadFontFormat
		}
		for i := 0; i < h; i++ {
			lines[i] = append(lines[i], []byte(glyph[i])...)
		}
	}
	return lines, nil
}

// replaceHardblanks replaces hardblank characters with spaces in-place
func replaceHardblanks(lines [][]byte, hb byte) {
	for i := range lines {
		b := lines[i]
		for j := range b {
			if b[j] == hb {
				b[j] = ' '
			}
		}
	}
}

// renderFullWidth renders text using Full-Width layout (no overlap)
func renderFullWidth(text string, font *parser.Font, printDir int) (string, error) {
	if font == nil {
		return "", common.ErrUnknownFont
	}

	h := font.Height
	if h <= 0 {
		return "", common.ErrBadFontFormat
	}

	// Handle empty text
	if text == "" {
		lines := make([]string, h)
		return strings.Join(lines, "\n"), nil
	}

	// Convert text to runes and filter non-ASCII
	runes := []rune(text)
	filterNonASCII(runes)

	// Compose glyphs
	lines, err := composeGlyphs(runes, font, h)
	if err != nil {
		return "", err
	}

	// Replace hardblanks with spaces
	replaceHardblanks(lines, byte(font.Hardblank))

	// Apply print direction (1 = RTL)
	if printDir == 1 {
		for i := range lines {
			reverseBytes(lines[i])
		}
	}

	// Convert to strings and join
	result := make([]string, h)
	for i := 0; i < h; i++ {
		result[i] = string(lines[i])
	}

	return strings.Join(result, "\n"), nil
}

// reverseBytes reverses a byte slice in-place (safe for ASCII)
func reverseBytes(b []byte) {
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
}

// lookupGlyph finds a glyph for a rune in the font
// Returns ErrUnsupportedRune if the glyph is missing (no fallback)
func lookupGlyph(font *parser.Font, r rune, h int) ([]string, error) {
	glyph, ok := font.Characters[r]
	if !ok {
		return nil, common.ErrUnsupportedRune
	}

	if len(glyph) != h {
		return nil, common.ErrBadFontFormat
	}

	return glyph, nil
}

// appendGlyph adds a glyph to the output lines
func appendGlyph(lines [][]byte, glyph []string) {
	for i := range lines {
		lines[i] = append(lines[i], []byte(glyph[i])...)
	}
}

// findRightmostVisible finds the rightmost non-space character in a line
// Note: Only ASCII space ' ' is considered blank. Hardblanks are treated as visible.
// TODO(perf): Consider caching trailing-space counts per composed line segment to reduce scans.
func findRightmostVisible(line []byte) int {
	for j := len(line) - 1; j >= 0; j-- {
		if line[j] != ' ' {
			return j
		}
	}
	return -1
}

// findLeftmostVisible finds the leftmost non-space character in a string
// Note: Only ASCII space ' ' is considered blank. Hardblanks are treated as visible.
func findLeftmostVisible(s string) int {
	for j := 0; j < len(s); j++ {
		if s[j] != ' ' {
			return j
		}
	}
	return len(s)
}

// calculateKerningDistance calculates the maximum required gap to avoid collision
// Returns the maximum gap needed across all rows (touching is allowed when gap=0)
//
// INVARIANT: Blank = ASCII space only; hardblank is visible.
// Trailing spaces are trimmed later in applyKerning, so the computed gap
// assumes those spaces are not preserved.
func calculateKerningDistance(lines [][]byte, glyph []string, trims []parser.GlyphTrim, h int) int {
	maxRequired := 0

	for row := 0; row < h; row++ {
		rightmost := findRightmostVisible(lines[row])
		// Use precomputed trim if available, otherwise compute on the fly
		var leftmost int
		if trims != nil && row < len(trims) {
			if trims[row].LeftmostVisible == -1 {
				leftmost = len(glyph[row]) // All spaces
			} else {
				leftmost = trims[row].LeftmostVisible
			}
		} else {
			leftmost = findLeftmostVisible(glyph[row])
		}

		var need int
		switch {
		case rightmost == -1 && leftmost == len(glyph[row]):
			// Both current line and new glyph line are all blanks
			// No gap needed when both sides are blank
			need = 0
		case rightmost == -1:
			// Current line is all blanks
			need = leftmost
		case leftmost == len(glyph[row]):
			// New glyph line is all blanks
			need = 0
		default:
			// Both have visible characters
			// Calculate: leftmost - trailing spaces in current line
			trailing := len(lines[row]) - rightmost - 1
			need = leftmost - trailing
			if need < 0 {
				need = 0 // Touching is allowed (zero gap valid)
			}
		}

		if need > maxRequired {
			maxRequired = need
		}
	}

	return maxRequired
}

// applyKerning adds a glyph with calculated kerning distance
func applyKerning(lines [][]byte, glyph []string, distance int) {
	for i := range lines {
		// Trim trailing spaces from current line (but not hardblanks)
		for len(lines[i]) > 0 && lines[i][len(lines[i])-1] == ' ' {
			lines[i] = lines[i][:len(lines[i])-1]
		}

		// Add spacing
		for j := 0; j < distance; j++ {
			lines[i] = append(lines[i], ' ')
		}

		// Append new glyph
		lines[i] = append(lines[i], []byte(glyph[i])...)
	}
}

// renderKerning renders text using kerning layout (minimal spacing without overlap)
func renderKerning(text string, font *parser.Font, printDir int) (string, error) {
	if font == nil {
		return "", common.ErrUnknownFont
	}

	h := font.Height
	if h <= 0 {
		return "", common.ErrBadFontFormat
	}

	// Handle empty text
	if text == "" {
		lines := make([]string, h)
		return strings.Join(lines, "\n"), nil
	}

	// Convert text to runes and filter non-ASCII
	runes := []rune(text)
	filterNonASCII(runes)

	// For RTL, reverse the order of runes (not the glyphs themselves)
	if printDir == 1 {
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
	}

	// Build output line by line, character by character
	const avgGlyphWidth = 10 // Average glyph width estimate (increased to reduce reslices)
	lines := make([][]byte, h)
	for i := range lines {
		lines[i] = make([]byte, 0, len(runes)*avgGlyphWidth)
	}

	// Process each character
	for idx, r := range runes {
		glyph, err := lookupGlyph(font, r, h)
		if err != nil {
			return "", err
		}

		if idx == 0 {
			// First character - just append as-is
			appendGlyph(lines, glyph)
		} else {
			// Calculate and apply kerning for all non-first characters
			// (including spaces - they're controlled by font design)
			// Use precomputed trims if available
			var trims []parser.GlyphTrim
			if font.CharacterTrims != nil {
				trims = font.CharacterTrims[r]
			}
			distance := calculateKerningDistance(lines, glyph, trims, h)
			applyKerning(lines, glyph, distance)
		}
	}

	// Replace hardblanks with spaces
	replaceHardblanks(lines, byte(font.Hardblank))

	// Convert to strings and join
	result := make([]string, h)
	for i := 0; i < h; i++ {
		result[i] = string(lines[i])
	}

	return strings.Join(result, "\n"), nil
}
