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
// Classes in order: | /\ [] {} () <>
// Per spec: "the one from the latter class is used"
// Lower numeric value = earlier in list, higher value = latter class (wins)
func getHierarchyClass(r rune) int {
	const (
		classPipe    = 1 // | class (earliest)
		classSlash   = 2 // /\ class
		classBracket = 3 // [] class
		classBrace   = 4 // {} class
		classParen   = 5 // () class
		classAngle   = 6 // <> class (latest, wins over all)
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

// universalSmush implements universal smushing logic per FIGfont v2 spec
// Universal smushing: later character overrides earlier at overlap position
// Key spec behavior:
//   - Space vs non-space: take the non-space
//   - Visible vs visible: later wins (when allowVisibleCollision=true)
//   - Hardblank vs visible: visible wins (hardblanks are overridden per spec)
//   - Hardblank vs hardblank: NOT allowed (prevents illegibility)
//   - Hardblank vs space: later wins
//
// allowVisibleCollision: true for pure universal, false for fallback universal
//
//nolint:gocyclo,gocognit // Multiple decision paths inherent to universal smushing spec
func universalSmush(left, right, hardblank rune, allowVisibleCollision bool) (rune, bool) {
	// Check if either is hardblank
	leftIsHardblank := left == hardblank
	rightIsHardblank := right == hardblank

	// Block hardblank vs hardblank (spec: prevents illegibility)
	if leftIsHardblank && rightIsHardblank {
		return 0, false
	}

	// Check visibility (non-space, non-hardblank)
	leftVisible := left != ' ' && left != hardblank
	rightVisible := right != ' ' && right != hardblank

	// Hardblank vs visible: visible wins per FIGfont spec
	// "Hardblanks ARE overridden by any visible sub-character"
	if leftIsHardblank && rightVisible {
		return right, true // Visible overrides hardblank
	}
	if leftVisible && rightIsHardblank {
		return left, true // Visible overrides hardblank
	}

	// Space vs visible: visible wins
	if left == ' ' && rightVisible {
		return right, true
	}
	if leftVisible && right == ' ' {
		return left, true
	}

	// Visible vs visible
	if leftVisible && rightVisible {
		if allowVisibleCollision {
			// Pure universal: later wins
			return right, true
		}
		// Fallback universal: no smush
		return 0, false
	}

	// Hardblank vs space: later wins
	if leftIsHardblank && right == ' ' {
		return ' ', true
	}
	if left == ' ' && rightIsHardblank {
		return hardblank, true
	}

	// Both are spaces
	if left == ' ' && right == ' ' {
		return ' ', true
	}

	// Should not reach here
	return 0, false
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
			// Per spec: "the one from the latter class is used"
			// Higher class number = latter in list = wins
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
		// Pure universal smushing (only when NO controlled rules are defined)
		// In pure universal mode, visible characters can smush (later wins)
		return universalSmush(left, right, hardblank, true)
	}

	// Controlled rules are defined but none matched - fall back to universal smushing
	// In fallback mode, visible-visible collisions are NOT allowed
	return universalSmush(left, right, hardblank, false)
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
			// Get precomputed trims if available for efficiency
			var trims []parser.GlyphTrim
			if font.CharacterTrims != nil {
				trims = font.CharacterTrims[r]
			}

			// Try to smush with previous character using optimal overlap algorithm
			overlap := calculateOptimalOverlap(lines, glyph, layout, font.Hardblank, trims, h)
			if overlap > 0 {
				applySmushing(lines, glyph, overlap, layout, font.Hardblank)
			} else {
				// Fall back to kerning distance if no smushing possible
				distance := calculateKerningDistance(lines, glyph, trims, h)
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
