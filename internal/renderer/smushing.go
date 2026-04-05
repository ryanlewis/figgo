package renderer

import "github.com/ryanlewis/figgo/internal/debug"

// smush attempts to combine two characters into one according to the smush mode.
// Returns the smushed character or 0 if no smushing can be done.
//
// Smushing Rule Precedence (CRITICAL):
// The rules are checked in a specific order that affects the output:
// 1. Spaces always combine (fundamental rule)
// 2. Width check prevents invalid overlaps
// 3. Universal smushing (if no specific rules)
// 4. Hardblank rule (highest precedence among controlled rules)
// 5. Equal character rule
// 6. Underscore rule
// 7. Hierarchy rule (complex multi-level precedence)
// 8. Opposite pair rule
// 9. Big X rule (lowest precedence)
//
// This order is not arbitrary - it ensures predictable, aesthetically
// pleasing output. Earlier rules override later ones when multiple
// rules could apply to the same character pair.
//
// Character Direction:
// In RTL mode, the left character (lch) is preferred in universal smushing
// because it appears later in the user's text (right-to-left reading).
func (state *renderState) smush(lch, rch rune) rune {
	// Handle spaces first
	if lch == ' ' {
		return rch
	}
	if rch == ' ' {
		return lch
	}

	// If not in smushing mode, return 0 (kerning)
	if (state.smushMode & SMSmush) == 0 {
		return 0
	}

	// Width check: characters must be at least 2 wide to smush
	// (matches FIGlet's smushem function behavior)
	if state.previousCharWidth < 2 || state.currentCharWidth < 2 {
		return 0
	}

	// Universal smushing mode (no specific rules set)
	if (state.smushMode & 63) == 0 {
		return state.smushUniversal(lch, rch)
	}

	return state.smushControlled(lch, rch)
}

// smushUniversal handles universal smushing (no specific rules set).
func (state *renderState) smushUniversal(lch, rch rune) rune {
	// Handle hardblanks - prefer visible characters
	if lch == state.hardblank {
		return rch
	}
	if rch == state.hardblank {
		return lch
	}

	// For right-to-left, prefer left character (latter in user's text)
	if state.right2left == 1 {
		return lch
	}
	return rch
}

// smushControlled applies controlled smushing rules in order of precedence.
func (state *renderState) smushControlled(lch, rch rune) rune {
	// Rule 6: Hardblank smushing
	if (state.smushMode&SMHardblank) != 0 && lch == state.hardblank && rch == state.hardblank {
		return lch
	}

	// If either character is hardblank, no further smushing
	if lch == state.hardblank || rch == state.hardblank {
		return 0
	}

	// Rule 1: Equal character smushing
	if (state.smushMode&SMEqual) != 0 && lch == rch {
		return lch
	}

	// Rule 2: Underscore smushing
	if r := smushUnderscore(lch, rch, state.smushMode); r != 0 {
		return r
	}

	// Rule 3: Hierarchy smushing
	if r := smushHierarchy(lch, rch, state.smushMode); r != 0 {
		return r
	}

	// Rule 4: Opposite pair smushing
	if r := smushPair(lch, rch, state.smushMode); r != 0 {
		return r
	}

	// Rule 5: Big X smushing
	if r := smushBigX(lch, rch, state.smushMode); r != 0 {
		return r
	}

	return 0
}

// smushUnderscore applies underscore smushing (Rule 2).
func smushUnderscore(lch, rch rune, smushMode int) rune {
	if (smushMode & SMLowline) == 0 {
		return 0
	}
	if lch == '_' && underscoreBorders[rch] {
		return rch
	}
	if rch == '_' && underscoreBorders[lch] {
		return lch
	}
	return 0
}

// hierarchyLevel maps hierarchy characters to their class level.
// Lower level = stronger. Returns -1 if not a hierarchy character.
var hierarchyLevelMap = map[rune]int{
	'|': 0, '/': 1, '\\': 1,
	'[': 2, ']': 2, '{': 3, '}': 3,
	'(': 4, ')': 4, '<': 5, '>': 5,
}

// smushHierarchy applies hierarchy smushing (Rule 3).
// The WEAKER character is returned, matching figlet.c's implementation.
func smushHierarchy(lch, rch rune, smushMode int) rune {
	if (smushMode & SMHierarchy) == 0 {
		return 0
	}

	lLevel, lOk := hierarchyLevelMap[lch]
	rLevel, rOk := hierarchyLevelMap[rch]
	if !lOk || !rOk || lLevel == rLevel {
		return 0
	}

	// Return the character from the STRONGER (lower level) class
	if lLevel < rLevel {
		return rch
	}
	return lch
}

// smushPair applies opposite pair smushing (Rule 4).
func smushPair(lch, rch rune, smushMode int) rune {
	if (smushMode & SMPair) == 0 {
		return 0
	}
	switch {
	case lch == '[' && rch == ']', rch == '[' && lch == ']',
		lch == '{' && rch == '}', rch == '{' && lch == '}',
		lch == '(' && rch == ')', rch == '(' && lch == ')':
		return '|'
	}
	return 0
}

// smushBigX applies Big X smushing (Rule 5).
func smushBigX(lch, rch rune, smushMode int) rune {
	if (smushMode & SMBigX) == 0 {
		return 0
	}
	if lch == '/' && rch == '\\' {
		return '|'
	}
	if rch == '/' && lch == '\\' {
		return 'Y'
	}
	if lch == '>' && rch == '<' {
		return 'X'
	}
	return 0
}

// smushAmount returns the maximum amount that the current character can overlap
// with the current output line.
//
// Dual-Direction Algorithm:
// The function handles both LTR and RTL rendering with different boundary
// calculations:
//
// LTR (Left-to-Right):
// 1. Find rightmost non-space in output line (lineBoundary)
// 2. Find leftmost non-space in new character (charBoundary)
// 3. Calculate potential overlap: charBoundary + outlineLen - 1 - lineBoundary
//
// RTL (Right-to-Left):
// 1. Find leftmost non-space in output line (lineBoundary)
// 2. Find rightmost non-space in new character (charBoundary)
// 3. Calculate potential overlap: lineBoundary + currentCharWidth - 1 - charBoundary
//
// The function checks each row independently and returns the MINIMUM overlap
// across all rows. This ensures no row exceeds safe overlap limits.
//
// Special Cases:
// - Empty spaces at boundaries allow additional overlap (+1)
// - Characters that can smush together allow additional overlap (+1)
// - First character in a line gets special handling
func (state *renderState) smushAmount() int {
	// If not in kerning or smushing mode, no overlap
	if (state.smushMode & (SMSmush | SMKern)) == 0 {
		return 0
	}

	// Get a pooled rune buffer for conversions
	runeBuffer := acquireRuneSlice()
	defer releaseRuneSlice(runeBuffer)

	maxSmush := state.currentCharWidth

	for row := 0; row < state.charHeight; row++ {
		// Apply RTL cap before row calculation
		if state.right2left != 0 && maxSmush > len(state.outputLine[row]) {
			maxSmush = len(state.outputLine[row])
		}

		var rowResult smushRowResult
		if state.right2left != 0 {
			runeBuffer, rowResult = state.smushAmountRTL(row, runeBuffer, maxSmush)
		} else {
			runeBuffer, rowResult = state.smushAmountLTR(row, runeBuffer)
		}

		// Adjust amount based on character overlap rules
		amt, reason := state.adjustSmushAmount(rowResult)

		// Emit debug event for this row's calculation
		if state.debug != nil {
			state.debug.Emit("render", "SmushAmountRow", debug.SmushAmountRowData{
				GlyphIdx:        state.inputCount,
				Row:             row,
				LineBoundaryIdx: rowResult.lineBoundary,
				CharBoundaryIdx: rowResult.charBoundary,
				Ch1:             rowResult.ch1,
				Ch2:             rowResult.ch2,
				AmountBefore:    rowResult.amt,
				AmountAfter:     amt,
				Reason:          reason,
				RTL:             state.right2left != 0,
			})
		}

		if amt < maxSmush {
			maxSmush = amt
		}
	}

	return maxSmush
}

// smushRowResult holds the boundary calculation results for a single row.
type smushRowResult struct {
	amt          int
	ch1, ch2     rune
	lineBoundary int
	charBoundary int
}

// adjustSmushAmount applies the final overlap adjustment based on boundary characters.
func (state *renderState) adjustSmushAmount(r smushRowResult) (int, string) {
	amt := r.amt
	if r.ch1 == 0 {
		return amt + 1, "ch1_null"
	}
	if r.ch2 != 0 && state.smush(r.ch1, r.ch2) != 0 {
		return amt + 1, "smushable"
	}
	return amt, "none"
}

// smushAmountRTL calculates the overlap for a single row in RTL mode.
func (state *renderState) smushAmountRTL(row int, runeBuffer []rune, _ int) ([]rune, smushRowResult) {
	var r smushRowResult

	// Find rightmost non-space in current character
	rowStr := state.currentChar[row]
	runeBuffer = runeBufferFromString(runeBuffer, rowStr)
	currRunes := runeBuffer

	// Match figlet.c: charbd is the INDEX of the rightmost non-space
	r.charBoundary = len(currRunes)
	for {
		if r.charBoundary < len(currRunes) {
			r.ch1 = currRunes[r.charBoundary]
		} else {
			r.ch1 = 0
		}
		if r.charBoundary <= 0 || (r.ch1 != 0 && r.ch1 != ' ') {
			break
		}
		r.charBoundary--
	}

	// Find leftmost non-space in output line
	r.lineBoundary = 0
	for r.lineBoundary < state.rowLengths[row] {
		r.ch2 = state.outputLine[row][r.lineBoundary]
		if r.ch2 != ' ' {
			break
		}
		r.lineBoundary++
	}

	r.amt = r.lineBoundary + state.currentCharWidth - 1 - r.charBoundary
	return runeBuffer, r
}

// smushAmountLTR calculates the overlap for a single row in LTR mode.
func (state *renderState) smushAmountLTR(row int, runeBuffer []rune) ([]rune, smushRowResult) {
	var r smushRowResult

	// Find the rightmost non-space character in output line
	r.lineBoundary = state.rowLengths[row]
	for {
		if r.lineBoundary < state.rowLengths[row] {
			r.ch1 = state.outputLine[row][r.lineBoundary]
		} else {
			r.ch1 = 0
		}
		if r.lineBoundary <= 0 || (r.ch1 != 0 && r.ch1 != ' ') {
			break
		}
		r.lineBoundary--
	}

	// Find the leftmost non-space in current character
	r.charBoundary = 0
	rowStr := state.currentChar[row]
	runeBuffer = runeBufferFromString(runeBuffer, rowStr)
	currRunes := runeBuffer

	for {
		if r.charBoundary < len(currRunes) {
			r.ch2 = currRunes[r.charBoundary]
		} else {
			r.ch2 = 0
			break
		}
		if r.ch2 != ' ' {
			break
		}
		r.charBoundary++
	}

	r.amt = r.charBoundary + state.outlineLen - 1 - r.lineBoundary
	return runeBuffer, r
}

// runeBufferFromString converts a string to runes using a pooled buffer.
func runeBufferFromString(buf []rune, s string) []rune {
	needed := len(s)
	if cap(buf) < needed {
		buf = make([]rune, 0, needed)
	} else {
		buf = buf[:0]
	}
	for _, r := range s {
		buf = append(buf, r)
	}
	return buf
}
