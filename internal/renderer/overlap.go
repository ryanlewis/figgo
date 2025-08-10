package renderer

import "github.com/ryanlewis/figgo/internal/parser"

// minGlyphWidth returns the minimum width across all rows of a glyph
func minGlyphWidth(glyph []string, h int) int {
	if h == 0 || len(glyph) == 0 {
		return 0
	}

	minWidth := len(glyph[0])
	for row := 1; row < h && row < len(glyph); row++ {
		if len(glyph[row]) < minWidth {
			minWidth = len(glyph[row])
		}
	}
	return minWidth
}

// minLineLength returns the minimum length across all lines
func minLineLength(lines [][]byte, h int) int {
	if h == 0 || len(lines) == 0 {
		return 0
	}

	minLen := len(lines[0])
	for row := 1; row < h && row < len(lines); row++ {
		if len(lines[row]) < minLen {
			minLen = len(lines[row])
		}
	}
	return minLen
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
	lines [][]byte, glyph []string, layout int, hardblank rune, _ []parser.GlyphTrim, h int,
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

	// Calculate maximum candidate overlap
	// Limited by both glyph width and line length to prevent data loss
	maxCandidate := minGlyphWidth(glyph, h)

	// CRITICAL: Limit by minimum line length to prevent out-of-bounds access
	// This prevents data loss when overlap would exceed line length
	minLineLen := minLineLength(lines, h)
	if minLineLen < maxCandidate {
		maxCandidate = minLineLen
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
