package renderer

import (
	"strings"

	"github.com/ryanlewis/figgo/internal/common"
	"github.com/ryanlewis/figgo/internal/parser"
)

// isBorderChar checks if a rune is a border character for Rule 2
func isBorderChar(r rune) bool {
	switch r {
	case '|', '/', '\\', '[', ']', '{', '}', '(', ')', '<', '>':
		return true
	default:
		return false
	}
}

// getHierarchyClass returns the hierarchy class for Rule 3
// Class priority order: | > /\ > [] > {} > () > <>
// Higher numeric constant = earlier in priority list (higher priority)
func getHierarchyClass(r rune) int {
	const (
		classPipe    = 6 // | class
		classSlash   = 5 // /\ class
		classBracket = 4 // [] class
		classBrace   = 3 // {} class
		classParen   = 2 // () class
		classAngle   = 1 // <> class
		classNone    = 0 // not a hierarchy char
	)

	switch r {
	case '|':
		return classPipe
	case '/', '\\':
		return classSlash
	case '[', ']':
		return classBracket
	case '{', '}':
		return classBrace
	case '(', ')':
		return classParen
	case '<', '>':
		return classAngle
	default:
		return classNone
	}
}

// isOppositePair checks if two runes form an opposite pair for Rule 4
func isOppositePair(left, right rune) bool {
	switch {
	case left == '(' && right == ')':
		return true
	case left == ')' && right == '(':
		return true
	case left == '[' && right == ']':
		return true
	case left == ']' && right == '[':
		return true
	case left == '{' && right == '}':
		return true
	case left == '}' && right == '{':
		return true
	default:
		return false
	}
}

// smushPair determines if two characters can be smushed and returns the result
// Returns the smushed character and true if smushing is possible, or (0, false) if not
// This implements controlled smushing rules 1-6 with strict precedence
//
//nolint:gocognit,gocyclo // High complexity is inherent to the FIGfont spec with 6 rules and strict precedence
func smushPair(left, right rune, layout int, hardblank rune) (rune, bool) {
	// Check if smushing mode is enabled
	if (layout & common.FitSmushing) == 0 {
		return 0, false
	}

	// Rule 1: Equal Character (takes precedence)
	if (layout&common.RuleEqualChar) != 0 && left == right && left != ' ' && left != hardblank {
		return left, true
	}

	// Rule 2: Underscore
	if (layout & common.RuleUnderscore) != 0 {
		if left == '_' && isBorderChar(right) {
			return right, true
		}
		if right == '_' && isBorderChar(left) {
			return left, true
		}
	}

	// Rule 3: Hierarchy (only when classes differ)
	if (layout & common.RuleHierarchy) != 0 {
		leftClass := getHierarchyClass(left)
		rightClass := getHierarchyClass(right)
		if leftClass > 0 && rightClass > 0 && leftClass != rightClass {
			if leftClass > rightClass {
				return left, true
			}
			return right, true
		}
	}

	// Rule 4: Opposite Pairs
	if (layout & common.RuleOppositePair) != 0 {
		if isOppositePair(left, right) {
			return '|', true
		}
	}

	// Rule 5: Big X (per FIGfont v2 spec)
	// /\ → '|', \/ → 'Y', >< → 'X'
	if (layout & common.RuleBigX) != 0 {
		if left == '/' && right == '\\' {
			return '|', true
		}
		if left == '\\' && right == '/' {
			return 'Y', true
		}
		if left == '>' && right == '<' {
			return 'X', true
		}
	}

	// Rule 6: Hardblank
	if (layout & common.RuleHardblank) != 0 {
		if left == hardblank && right == hardblank {
			return hardblank, true
		}
	}

	// Check if any controlled smushing rules are defined
	hasRules := (layout & (common.RuleEqualChar | common.RuleUnderscore | common.RuleHierarchy |
		common.RuleOppositePair | common.RuleBigX | common.RuleHardblank)) != 0

	if !hasRules {
		// Universal smushing (only when NO controlled rules are defined)
		// Per spec: later character overrides earlier at overlapping position
		// Visible chars override spaces AND hardblanks
		if right != ' ' && right != hardblank {
			return right, true // Right (later) char overrides
		}
		if left != ' ' && left != hardblank {
			return left, true // Keep left if right is space/hardblank
		}
		// Both are space or hardblank - keep the override (right)
		return right, true
	}

	// Controlled rules are defined but none matched
	// Fall back to universal smushing, but with restrictions:
	// - Hardblank vs hardblank collisions are NOT allowed
	// - Hardblanks are overridden by visible characters (per spec)
	// - Space vs visible character combinations are allowed

	// Block only hardblank-vs-hardblank in universal fallback
	if left == hardblank && right == hardblank {
		return 0, false
	}

	// If exactly one side is a hardblank and the other is visible, universal smushing
	// should choose the visible character (spec: hardblanks are overridden by visible)
	if left == hardblank && right != ' ' && right != hardblank {
		return right, true
	}
	if right == hardblank && left != ' ' && left != hardblank {
		return left, true
	}

	// Universal smushing fallback rules for other combinations:
	// - If left is space and right is visible → take right
	// - If right is space and left is visible → take left
	// - If both are spaces → take space (allow overlap)
	// - If both are visible (non-hardblank) → no smush (fall back to kerning)
	if left == ' ' && right != ' ' {
		return right, true
	}
	if right == ' ' && left != ' ' {
		return left, true
	}
	if left == ' ' && right == ' ' {
		return ' ', true // Both spaces - allow overlap with space
	}

	// Both visible (non-hardblank) - no smushing possible
	return 0, false
}

// calculateSmushingDistance finds the maximum overlap where all columns can smush
// Returns the number of columns that can overlap (0 means no smushing possible)
//
//nolint:gocognit // Algorithm needs to test multiple overlaps and rows with smushing rules
func calculateSmushingDistance(lines [][]byte, glyph []string, layout int, hardblank rune, h int) int {
	maxOverlap := 0

	// Find the minimum glyph width to determine max possible overlap
	minGlyphWidth := len(glyph[0])
	for i := 1; i < h; i++ {
		if len(glyph[i]) < minGlyphWidth {
			minGlyphWidth = len(glyph[i])
		}
	}

	// Try increasing overlaps until we find the maximum valid one
	for overlap := 1; overlap <= minGlyphWidth; overlap++ {
		canSmush := true

		// Check if all rows can smush at this overlap distance
		for row := 0; row < h; row++ {
			lineLen := len(lines[row])
			glyphRow := glyph[row]

			// For each overlapped column
			for col := 0; col < overlap; col++ {
				// Calculate positions
				linePos := lineLen - overlap + col
				glyphPos := col

				// Get the characters at this position
				var leftChar, rightChar rune

				if linePos >= 0 && linePos < lineLen {
					leftChar = rune(lines[row][linePos])
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
					canSmush = false
					break
				}
			}

			if !canSmush {
				break
			}
		}

		if canSmush {
			maxOverlap = overlap
		} else {
			break // Can't overlap more if this overlap failed
		}
	}

	return maxOverlap
}

// applySmushing adds a glyph with smushing at the specified overlap
func applySmushing(lines [][]byte, glyph []string, overlap, layout int, hardblank rune) {
	h := len(lines)

	for row := 0; row < h; row++ {
		lineLen := len(lines[row])
		glyphRow := glyph[row]

		// Smush the overlapped columns
		for col := 0; col < overlap; col++ {
			linePos := lineLen - overlap + col
			glyphPos := col

			var leftChar, rightChar rune
			if linePos >= 0 && linePos < lineLen {
				leftChar = rune(lines[row][linePos])
			} else {
				leftChar = ' '
			}

			if glyphPos < len(glyphRow) {
				rightChar = rune(glyphRow[glyphPos])
			} else {
				rightChar = ' '
			}

			smushed, _ := smushPair(leftChar, rightChar, layout, hardblank)

			// Replace the character at linePos with the smushed result
			if linePos >= 0 && linePos < lineLen {
				lines[row][linePos] = byte(smushed)
			}
		}

		// Append the non-overlapped portion of the glyph
		if overlap < len(glyphRow) {
			lines[row] = append(lines[row], []byte(glyphRow[overlap:])...)
		}
	}
}

// renderSmushing renders text using smushing layout with controlled smushing rules
func renderSmushing(text string, font *parser.Font, layout, printDir int, replacement rune) (string, error) {
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

	// Convert text to runes and filter unsupported characters
	runes := []rune(text)
	filterUnsupportedRunes(runes, replacement)

	// For RTL, reverse the order of runes (not the glyphs themselves)
	if printDir == 1 {
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
	}

	// Build output line by line, character by character
	const avgGlyphWidth = 10
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
			// Try to smush with previous character
			overlap := calculateSmushingDistance(lines, glyph, layout, font.Hardblank, h)
			if overlap > 0 {
				applySmushing(lines, glyph, overlap, layout, font.Hardblank)
			} else {
				// Fall back to kerning distance if no smushing possible
				distance := calculateKerningDistance(lines, glyph, nil, h)
				applyKerning(lines, glyph, distance)
			}
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
