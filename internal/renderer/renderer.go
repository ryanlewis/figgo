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

	// For now, only implement Full-Width mode
	// Extract the fitting mode (ignore rule bits)
	fittingMode := layout & (common.FitKerning | common.FitSmushing)
	if fittingMode == 0 { // No fitting bits set = full-width
		return renderFullWidth(text, font, printDir)
	}

	// Kerning and Smushing modes not yet implemented
	return "", fmt.Errorf("layout mode %x not yet implemented", layout)
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
	const avgGlyphWidth = 5 // Average glyph width estimate
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
