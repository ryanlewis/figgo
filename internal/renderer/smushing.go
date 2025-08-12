package renderer

import "strings"

// smushem implements the character smushing logic from figlet.c (lines 1358-1434)
// Given 2 characters, attempts to smush them into 1, according to smushmode.
// Returns smushed character or 0 if no smushing can be done.
func (state *renderState) smushem(lch, rch rune) rune {
	// Handle spaces first
	if lch == ' ' {
		return rch
	}
	if rch == ' ' {
		return lch
	}

	// Disallow overlapping if the previous character or current character has width < 2
	if state.previousCharWidth < 2 || state.currCharWidth < 2 {
		return 0
	}

	// If not in smushing mode, return 0 (kerning)
	if (state.smushMode & SM_SMUSH) == 0 {
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
	if (state.smushMode & SM_HARDBLANK) != 0 {
		if lch == state.hardblank && rch == state.hardblank {
			return lch
		}
	}

	// If either character is hardblank and we're not doing hardblank rule, no smushing
	if lch == state.hardblank || rch == state.hardblank {
		return 0
	}

	// Rule 1: Equal character smushing
	if (state.smushMode & SM_EQUAL) != 0 {
		if lch == rch {
			return lch
		}
	}

	// Rule 2: Underscore smushing
	if (state.smushMode & SM_LOWLINE) != 0 {
		if lch == '_' && strings.ContainsRune("|/\\[]{}()<>", rch) {
			return rch
		}
		if rch == '_' && strings.ContainsRune("|/\\[]{}()<>", lch) {
			return lch
		}
	}

	// Rule 3: Hierarchy smushing
	if (state.smushMode & SM_HIERARCHY) != 0 {
		// "|" replaces "/\", "[]", "{}", "()", "<>"
		if lch == '|' && strings.ContainsRune("/\\[]{}()<>", rch) {
			return rch
		}
		if rch == '|' && strings.ContainsRune("/\\[]{}()<>", lch) {
			return lch
		}
		
		// "/\" replaces "[]", "{}", "()", "<>"
		if strings.ContainsRune("/\\", lch) && strings.ContainsRune("[]{}()<>", rch) {
			return rch
		}
		if strings.ContainsRune("/\\", rch) && strings.ContainsRune("[]{}()<>", lch) {
			return lch
		}
		
		// "[]" replaces "{}", "()", "<>"
		if strings.ContainsRune("[]", lch) && strings.ContainsRune("{}()<>", rch) {
			return rch
		}
		if strings.ContainsRune("[]", rch) && strings.ContainsRune("{}()<>", lch) {
			return lch
		}
		
		// "{}" replaces "()", "<>"
		if strings.ContainsRune("{}", lch) && strings.ContainsRune("()<>", rch) {
			return rch
		}
		if strings.ContainsRune("{}", rch) && strings.ContainsRune("()<>", lch) {
			return lch
		}
		
		// "()" replaces "<>"
		if strings.ContainsRune("()", lch) && strings.ContainsRune("<>", rch) {
			return rch
		}
		if strings.ContainsRune("()", rch) && strings.ContainsRune("<>", lch) {
			return lch
		}
	}

	// Rule 4: Opposite pair smushing
	if (state.smushMode & SM_PAIR) != 0 {
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
	if (state.smushMode & SM_BIGX) != 0 {
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

// smushAmt returns the maximum amount that the current character can be smushed
// into the current line (from figlet.c lines 1446-1485)
func (state *renderState) smushAmt() int {
	// If not in kerning or smushing mode, no overlap
	if (state.smushMode & (SM_SMUSH | SM_KERN)) == 0 {
		return 0
	}

	// Note: figlet.c does NOT return 0 when output is empty!
	// It continues to calculate overlap even for the first character
	
	maxSmush := state.currCharWidth
	
	for row := 0; row < state.charHeight; row++ {
		var amt int
		var ch1, ch2 rune
		var lineBd, charBd int  // Declare here for debug access
		
		if state.right2left != 0 {
			// Right-to-left processing
			if maxSmush > len(state.outputLine[row]) {
				maxSmush = len(state.outputLine[row])
			}
			
			// Find rightmost non-space in current character
			charBd = len(state.currChar[row])
			currRunes := []rune(state.currChar[row])
			for charBd > 0 {
				if charBd-1 < len(currRunes) {
					ch1 = currRunes[charBd-1]
				} else {
					ch1 = 0
				}
				if ch1 != 0 && ch1 != ' ' {
					break
				}
				charBd--
			}
			
			// Find leftmost non-space in output line
			lineBd = 0
			for lineBd < len(state.outputLine[row]) {
				ch2 = state.outputLine[row][lineBd]
				if ch2 != ' ' {
					break
				}
				lineBd++
			}
			
			amt = lineBd + state.currCharWidth - 1 - charBd
		} else {
			// Left-to-right processing (exact figlet.c logic)
			// Find the rightmost non-space character in output line
			// figlet.c: for (linebd=STRLEN(outputline[row]);
			//   ch1 = outputline[row][linebd],(linebd>0&&(!ch1||ch1==' '));linebd--)
			// Use row-specific length (emulates C's strlen)
			lineBd = state.rowLengths[row]
			
			// Emulate figlet.c's loop exactly
			for {
				// Get character at linebd position
				// In C, accessing string[strlen] gives null terminator
				// In Go, we need to handle this explicitly
				if lineBd < len(state.outputLine[row]) {
					ch1 = state.outputLine[row][lineBd]
				} else {
					ch1 = 0 // Emulate C's null terminator at end of string
				}
				
				// Check condition: linebd>0 && (!ch1 || ch1==' ')
				if !(lineBd > 0 && (ch1 == 0 || ch1 == ' ')) {
					break
				}
				lineBd--
			}
			// Now lineBd points to rightmost non-space character
			// ch1 already has the correct value from the loop above
			
			// Find the leftmost non-space character in the current character
			// figlet.c: for (charbd=0;ch2=currchar[row][charbd],ch2==' ';charbd++)
			// CRITICAL: In C, this loop continues until it hits a non-space OR null terminator
			charBd = 0
			currRunes := []rune(state.currChar[row])
			
			// Emulate figlet.c's loop with null terminator handling
			for {
				// Get character at charbd position
				if charBd < len(currRunes) {
					ch2 = currRunes[charBd]
				} else {
					ch2 = 0 // Emulate C's null terminator when past end
					break   // Exit loop when we hit the "null terminator"
				}
				
				// Check if it's a space - if not, exit loop
				if ch2 != ' ' {
					break
				}
				charBd++
			}
			// charBd is the 0-based index of leftmost non-space (or length if all spaces)
			// ch2 has the character at that position (or 0 if all spaces)
			
			// Calculate overlap amount using figlet.c formula
			amt = charBd + state.outlineLen - 1 - lineBd
		}

		// Adjust amount based on character overlap rules
		if ch1 == 0 || ch1 == ' ' {
			amt++
		} else if ch2 != 0 {
			if state.smushem(ch1, ch2) != 0 {
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