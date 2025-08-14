package renderer

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

	// Disallow overlapping if the previous character or current character has width < 2
	if state.previousCharWidth < 2 || state.currentCharWidth < 2 {
		return 0
	}

	// If not in smushing mode, return 0 (kerning)
	if (state.smushMode & SMSmush) == 0 {
		return 0
	}

	// Universal smushing mode (no specific rules set)
	if (state.smushMode & 63) == 0 {
		// This is smushing by universal overlapping

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

		// Default: prefer right character
		return rch
	}

	// Controlled smushing rules - check in order of precedence

	// Rule 6: Hardblank smushing
	if (state.smushMode & SMHardblank) != 0 {
		if lch == state.hardblank && rch == state.hardblank {
			return lch
		}
	}

	// If either character is hardblank and we're not doing hardblank rule, no smushing
	if lch == state.hardblank || rch == state.hardblank {
		return 0
	}

	// Rule 1: Equal character smushing
	if (state.smushMode & SMEqual) != 0 {
		if lch == rch {
			return lch
		}
	}

	// Rule 2: Underscore smushing
	if (state.smushMode & SMLowline) != 0 {
		if lch == '_' && underscoreBorders[rch] {
			return rch
		}
		if rch == '_' && underscoreBorders[lch] {
			return lch
		}
	}

	// Rule 3: Hierarchy smushing
	// Character hierarchy (strongest to weakest):
	// Level 0: | (strongest, survives against all)
	// Level 1: /\
	// Level 2: []
	// Level 3: {}
	// Level 4: ()
	// Level 5: <> (weakest)
	//
	// The stronger character replaces the weaker one.
	// This creates visually pleasing overlaps in ASCII art.
	if (state.smushMode & SMHierarchy) != 0 {
		// "|" replaces "/\", "[]", "{}", "()", "<>"
		if lch == '|' && hierarchyLevel1[rch] {
			return rch
		}
		if rch == '|' && hierarchyLevel1[lch] {
			return lch
		}

		// "/\" replaces "[]", "{}", "()", "<>"
		if (lch == '/' || lch == '\\') && hierarchyLevel2[rch] {
			return rch
		}
		if (rch == '/' || rch == '\\') && hierarchyLevel2[lch] {
			return lch
		}

		// "[]" replaces "{}", "()", "<>"
		if (lch == '[' || lch == ']') && hierarchyLevel3[rch] {
			return rch
		}
		if (rch == '[' || rch == ']') && hierarchyLevel3[lch] {
			return lch
		}

		// "{}" replaces "()", "<>"
		if (lch == '{' || lch == '}') && hierarchyLevel4[rch] {
			return rch
		}
		if (rch == '{' || rch == '}') && hierarchyLevel4[lch] {
			return lch
		}

		// "()" replaces "<>"
		if (lch == '(' || lch == ')') && hierarchyLevel5[rch] {
			return rch
		}
		if (rch == '(' || rch == ')') && hierarchyLevel5[lch] {
			return lch
		}
	}

	// Rule 4: Opposite pair smushing
	if (state.smushMode & SMPair) != 0 {
		if lch == '[' && rch == ']' {
			return '|'
		}
		if rch == '[' && lch == ']' {
			return '|'
		}
		if lch == '{' && rch == '}' {
			return '|'
		}
		if rch == '{' && lch == '}' {
			return '|'
		}
		if lch == '(' && rch == ')' {
			return '|'
		}
		if rch == '(' && lch == ')' {
			return '|'
		}
	}

	// Rule 5: Big X smushing
	if (state.smushMode & SMBigX) != 0 {
		if lch == '/' && rch == '\\' {
			return '|'
		}
		if rch == '/' && lch == '\\' {
			return 'Y'
		}
		if lch == '>' && rch == '<' {
			return 'X'
		}
		// Note: Don't want the reverse of above to give 'X'
	}

	// No smushing rule matched
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
	// Get a pooled rune buffer for conversions
	runeBuffer := acquireRuneSlice()
	defer releaseRuneSlice(runeBuffer)
	// If not in kerning or smushing mode, no overlap
	if (state.smushMode & (SMSmush | SMKern)) == 0 {
		return 0
	}

	// Calculate overlap even for the first character

	maxSmush := state.currentCharWidth

	for row := 0; row < state.charHeight; row++ {
		var amt int
		var ch1, ch2 rune
		var lineBoundary, charBoundary int // Declare here for debug access

		if state.right2left != 0 {
			// Right-to-left processing
			if maxSmush > len(state.outputLine[row]) {
				maxSmush = len(state.outputLine[row])
			}

			// Find rightmost non-space in current character
			charBoundary = len(state.currentChar[row])
			// Use pooled buffer for rune conversion
			rowStr := state.currentChar[row]
			needed := len(rowStr)
			if cap(runeBuffer) < needed {
				runeBuffer = make([]rune, needed)
			} else {
				runeBuffer = runeBuffer[:0]
			}
			for _, r := range rowStr {
				runeBuffer = append(runeBuffer, r)
			}
			currRunes := runeBuffer
			for charBoundary > 0 {
				if charBoundary-1 < len(currRunes) {
					ch1 = currRunes[charBoundary-1]
				} else {
					ch1 = 0
				}
				if ch1 != 0 && ch1 != ' ' {
					break
				}
				charBoundary--
			}

			// Find leftmost non-space in output line
			lineBoundary = 0
			for lineBoundary < len(state.outputLine[row]) {
				ch2 = state.outputLine[row][lineBoundary]
				if ch2 != ' ' {
					break
				}
				lineBoundary++
			}

			amt = lineBoundary + state.currentCharWidth - 1 - charBoundary
		} else {
			// Left-to-right processing
			// Find the rightmost non-space character in output line
			// Use row-specific length
			lineBoundary = state.rowLengths[row]

			// Find rightmost non-space in output line
			for {
				// Get character at linebd position
				// Handle end of string as null terminator
				if lineBoundary < len(state.outputLine[row]) {
					ch1 = state.outputLine[row][lineBoundary]
				} else {
					ch1 = 0 // Treat as null terminator at end
				}

				// Check condition: linebd>0 && (!ch1 || ch1==' ')
				if !(lineBoundary > 0 && (ch1 == 0 || ch1 == ' ')) {
					break
				}
				lineBoundary--
			}
			// Now lineBd points to rightmost non-space character
			// ch1 already has the correct value from the loop above

			// Find the leftmost non-space character in the current character
			// Find leftmost non-space in current character
			charBoundary = 0
			// Use pooled buffer for rune conversion
			rowStr := state.currentChar[row]
			needed := len(rowStr)
			if cap(runeBuffer) < needed {
				runeBuffer = make([]rune, needed)
			} else {
				runeBuffer = runeBuffer[:0]
			}
			for _, r := range rowStr {
				runeBuffer = append(runeBuffer, r)
			}
			currRunes := runeBuffer

			// Loop until we find a non-space or reach the end
			for {
				// Get character at charbd position
				if charBoundary < len(currRunes) {
					ch2 = currRunes[charBoundary]
				} else {
					ch2 = 0 // Treat as null when past end
					break   // Exit loop when we hit the "null terminator"
				}

				// Check if it's a space - if not, exit loop
				if ch2 != ' ' {
					break
				}
				charBoundary++
			}
			// charBd is the 0-based index of leftmost non-space (or length if all spaces)
			// ch2 has the character at that position (or 0 if all spaces)

			// Calculate overlap amount
			amt = charBoundary + state.outlineLen - 1 - lineBoundary
		}

		// Adjust amount based on character overlap rules
		// These adjustments determine if characters can overlap by one more position:
		// 1. If boundary character is space/null, safe to overlap (+1)
		// 2. If both characters exist and can smush, safe to overlap (+1)
		// 3. Otherwise, maintain current overlap amount (no adjustment)
		if ch1 == 0 || ch1 == ' ' {
			amt++
		} else if ch2 != 0 {
			if state.smush(ch1, ch2) != 0 {
				amt++
			}
		}

		// Take minimum overlap across all rows
		if amt < maxSmush {
			maxSmush = amt
		}
	}

	return maxSmush
}
