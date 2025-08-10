package renderer

import "github.com/ryanlewis/figgo/internal/parser"

// calculateMaxCandidateOverlap determines the maximum possible overlap
// Based on issue #14: max overlap is min(left.trailingSpaces, right.leadingSpaces)
// But we also allow full overlap for smushing evaluation
func calculateMaxCandidateOverlap(_ [][]byte, glyph []string, _ []parser.GlyphTrim, h int) int {
	if h == 0 || len(glyph) == 0 {
		return 0
	}

	// Find the minimum glyph width across all rows (absolute maximum)
	minGlyphWidth := len(glyph[0])
	for row := 1; row < h; row++ {
		if row < len(glyph) && len(glyph[row]) < minGlyphWidth {
			minGlyphWidth = len(glyph[row])
		}
	}

	// For smushing mode, we try overlapping up to the full glyph width
	// The validation step will determine what's actually allowed
	// This allows for rules like equal char to work even with no trailing/leading spaces
	return minGlyphWidth
}

// ValidateOverlap checks if all overlapped columns satisfy smushing rules
// Exported for testing
func ValidateOverlap(lines [][]byte, glyph []string, overlap, layout int, hardblank rune, h int) bool {
	if overlap <= 0 {
		return true // No overlap is always valid
	}

	// Check each row
	for row := 0; row < h; row++ {
		var lineRow []byte
		var glyphRow string

		if row < len(lines) {
			lineRow = lines[row]
		}
		if row < len(glyph) {
			glyphRow = glyph[row]
		}

		lineLen := len(lineRow)

		// Check each overlapped column
		for col := 0; col < overlap; col++ {
			// Calculate positions
			linePos := lineLen - overlap + col
			glyphPos := col

			// Get characters at this position
			var leftChar, rightChar rune

			if linePos >= 0 && linePos < lineLen {
				leftChar = rune(lineRow[linePos])
			} else {
				leftChar = ' '
			}

			if glyphPos < len(glyphRow) {
				rightChar = rune(glyphRow[glyphPos])
			} else {
				rightChar = ' '
			}

			// Check if these can smush
			_, ok := smushPair(leftChar, rightChar, layout, hardblank)
			if !ok {
				return false // This overlap is invalid
			}
		}
	}

	return true
}

// calculateOptimalOverlap finds the maximum valid overlap for smushing
// Implements the algorithm from issue #14: start at max, decrement to find valid overlap
func calculateOptimalOverlap(
	lines [][]byte, glyph []string, layout int, hardblank rune, trims []parser.GlyphTrim, h int,
) int {
	// Check if left side is empty - no overlap possible
	allEmpty := true
	for row := 0; row < h && row < len(lines); row++ {
		if len(lines[row]) > 0 {
			allEmpty = false
			break
		}
	}
	if allEmpty {
		return 0 // Can't overlap with empty left side
	}

	// Calculate maximum candidate overlap based on spacing
	maxCandidate := calculateMaxCandidateOverlap(lines, glyph, trims, h)

	// Also limit by the minimum glyph width
	minGlyphWidth := len(glyph[0])
	for i := 1; i < h; i++ {
		if i < len(glyph) && len(glyph[i]) < minGlyphWidth {
			minGlyphWidth = len(glyph[i])
		}
	}

	if minGlyphWidth < maxCandidate {
		maxCandidate = minGlyphWidth
	}

	// Start from maximum and work down to find the first valid overlap
	for overlap := maxCandidate; overlap > 0; overlap-- {
		if ValidateOverlap(lines, glyph, overlap, layout, hardblank, h) {
			return overlap
		}
	}

	// No valid overlap found
	return 0
}
