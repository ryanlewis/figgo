package renderer

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/ryanlewis/figgo/internal/parser"
)

// Render converts text to ASCII art using the font and options
func Render(text string, font *parser.Font, opts *Options) (string, error) {
	if font == nil {
		return "", errors.New("font cannot be nil")
	}

	// Initialize render state
	state := &renderState{
		charHeight:      font.Height,
		hardblank:       font.Hardblank,
		outlineLenLimit: 10000, // Default large limit
	}
	
	// Set trim whitespace option
	if opts != nil {
		state.trimWhitespace = opts.TrimWhitespace
	}

	// Set print direction from options or font
	if opts != nil && opts.PrintDirection != nil {
		state.right2left = *opts.PrintDirection
	} else {
		state.right2left = font.PrintDirection
	}

	// Convert layout to smush mode
	if opts != nil {
		state.smushMode = layoutToSmushMode(opts.Layout)
	} else {
		// Use font's default layout
		// figlet.c uses FullLayout when available, falls back to OldLayout
		if font.FullLayoutSet && font.FullLayout != 0 {
			// Extract horizontal layout from FullLayout
			// FullLayout contains both horizontal (bits 0-7) and vertical (bits 8-14) layout
			// We only need horizontal for now
			state.smushMode = font.FullLayout & 0xFF // Get bits 0-7 for horizontal layout
		} else {
			// Fall back to converting OldLayout
			state.smushMode = oldLayoutToSmushMode(font.OldLayout)
		}
	}

	// Initialize output lines with pre-allocated buffer (like figlet.c)
	// This enables future buffer pooling and matches figlet.c's behavior
	state.outputLine = make([][]rune, state.charHeight)
	state.rowLengths = make([]int, state.charHeight)
	for i := range state.outputLine {
		// Pre-allocate full buffer size, not just capacity
		state.outputLine[i] = make([]rune, state.outlineLenLimit)
		// Note: in Go, rune zero value is 0, which works like C's null terminator
		state.rowLengths[i] = 0 // Start with empty rows (like clearline in figlet.c)
	}

	// Process each character in the input text
	for _, r := range text {
		// Handle newlines and special characters
		if r == '\n' {
			// Print current line and reset
			// For now, we'll just continue processing - line breaks will be added later
			continue
		}

		// Skip control characters except space
		if r < ' ' && r != '\n' {
			continue
		}

		// Normalize whitespace
		if r == '\t' {
			r = ' '
		}

		// Get character glyph
		glyph, exists := font.Characters[r]
		if !exists {
			// Handle unknown character
			if opts != nil && opts.UnknownRune != nil {
				r = *opts.UnknownRune
				glyph, exists = font.Characters[r]
				if !exists {
					return "", errors.New("unsupported rune: " + string(r))
				}
			} else {
				return "", errors.New("unsupported rune: " + string(r))
			}
		}

		// Add character to output
		if !state.addChar(glyph) {
			// Character didn't fit - for now just continue
			// Line breaking logic would go here
		}
	}

	// Convert output to string
	return state.outputToString(), nil
}

// layoutToSmushMode converts figgo Layout bitmask to figlet.c smush mode
func layoutToSmushMode(layout int) int {
	// Layout constants from figgo/layout.go:
	// FitFullWidth = 0
	// FitKerning = 1 << 6 = 64
	// FitSmushing = 1 << 7 = 128
	// Rules are in bits 0-5

	smushMode := 0

	// Check fitting mode
	if layout&(1<<6) != 0 { // FitKerning
		smushMode |= SM_KERN
	} else if layout&(1<<7) != 0 { // FitSmushing
		smushMode |= SM_SMUSH
		
		// Add smushing rules (bits 0-5)
		if layout&(1<<0) != 0 { // RuleEqualChar
			smushMode |= SM_EQUAL
		}
		if layout&(1<<1) != 0 { // RuleUnderscore
			smushMode |= SM_LOWLINE
		}
		if layout&(1<<2) != 0 { // RuleHierarchy
			smushMode |= SM_HIERARCHY
		}
		if layout&(1<<3) != 0 { // RuleOppositePair
			smushMode |= SM_PAIR
		}
		if layout&(1<<4) != 0 { // RuleBigX
			smushMode |= SM_BIGX
		}
		if layout&(1<<5) != 0 { // RuleHardblank
			smushMode |= SM_HARDBLANK
		}
	}
	// FitFullWidth = 0 means no bits set

	return smushMode
}

// oldLayoutToSmushMode converts font OldLayout (-1 or 0..63) to smush mode
func oldLayoutToSmushMode(oldLayout int) int {
	if oldLayout == -1 {
		// Full-width mode
		return 0
	} else if oldLayout == 0 {
		// Kerning mode  
		return SM_KERN
	} else if oldLayout < 0 {
		// Invalid, default to full-width
		return 0
	} else {
		// Smushing mode with rules (1..63)
		return SM_SMUSH | (oldLayout & 63)
	}
}

// addChar attempts to add a character glyph to the current output line
func (state *renderState) addChar(glyph []string) bool {
	if len(glyph) != state.charHeight {
		return false // Invalid glyph height
	}

	// Save previous width BEFORE updating current character (like figlet.c getletter)
	state.previousCharWidth = state.currCharWidth
	
	// Set current character data
	state.currChar = glyph
	
	// Calculate character width - figlet.c uses ONLY first row's length
	// currcharwidth = STRLEN(currchar[0]);
	if len(glyph) > 0 {
		state.currCharWidth = utf8.RuneCountInString(glyph[0])
	} else {
		state.currCharWidth = 0
	}

	// Calculate smush amount
	smushAmount := state.smushAmt()
	
	// Ensure smushAmount is not negative
	if smushAmount < 0 {
		smushAmount = 0
	}

	// Check if character fits
	newLength := state.outlineLen + state.currCharWidth - smushAmount
	if newLength > state.outlineLenLimit {
		return false
	}

	// Add character to each row
	for row := 0; row < state.charHeight; row++ {
		rowRunes := []rune(glyph[row])
		
		if state.right2left != 0 {
			// Right-to-left processing (exact figlet.c logic)
			// Use pre-allocated temp buffer
			tempLine := make([]rune, state.outlineLenLimit)
			
			// STRCPY(templine,currchar[row])
			copy(tempLine, rowRunes)
			
			// Apply smushing at overlap positions
			for k := 0; k < smushAmount; k++ {
				if k < len(rowRunes) {
					// Always assign result like figlet.c
					tempLine[state.currCharWidth-smushAmount+k] = 
						state.smushem(tempLine[state.currCharWidth-smushAmount+k], state.outputLine[row][k])
				}
			}
			
			// STRCAT(templine,outputline[row]+smushamount)
			if smushAmount < state.rowLengths[row] {
				// Copy the part of outputline after smush region
				copy(tempLine[state.currCharWidth:], state.outputLine[row][smushAmount:state.rowLengths[row]])
			}
			
			// STRCPY(outputline[row],templine)
			copy(state.outputLine[row], tempLine)
			
			// Update row length
			state.rowLengths[row] = state.currCharWidth + state.rowLengths[row] - smushAmount
		} else {
			// Left-to-right processing (exact figlet.c logic)
			// Apply smushing at overlap positions
			for k := 0; k < smushAmount; k++ {
				column := state.outlineLen - smushAmount + k
				if column < 0 {
					column = 0
				}
				// Use currchar[row][k] directly - no adjustment for leading spaces
				// With pre-allocated buffer, we don't need to check outputLine bounds
				if k < len(rowRunes) {
					// Smush the characters - always assign result like figlet.c
					state.outputLine[row][column] = state.smushem(state.outputLine[row][column], rowRunes[k])
				}
			}

			// STRCAT(outputline[row],currchar[row]+smushamount)
			// Copy the part of the new character after the smush region
			if smushAmount < len(rowRunes) {
				remaining := rowRunes[smushAmount:]
				// Copy to the current end of this row's content
				// Note: rowLengths[row] hasn't been updated yet, so it's the old length
				copy(state.outputLine[row][state.rowLengths[row]:], remaining)
			}
		}
	}

	// Update output length and ensure all rows have consistent length
	// figlet.c: outlinelen = STRLEN(outputline[0]);
	state.outlineLen = newLength
	
	// Update all row lengths to match the new length
	// This ensures consistency across all rows
	for row := 0; row < state.charHeight; row++ {
		state.rowLengths[row] = newLength
	}
	return true
}

// outputToString converts the output lines to a final string
func (state *renderState) outputToString() string {
	if state.charHeight == 0 {
		return ""
	}

	var result strings.Builder
	for i, line := range state.outputLine {
		// Extract only the actual content using row-specific length
		// This emulates C's null-terminated string behavior
		actualLine := line[:state.rowLengths[i]]
		
		// Replace hardblanks with spaces
		lineStr := string(actualLine)
		lineStr = strings.ReplaceAll(lineStr, string(state.hardblank), " ")
		
		// Optionally trim trailing spaces (figlet preserves them by default)
		if state.trimWhitespace {
			lineStr = strings.TrimRight(lineStr, " ")
		}
		
		result.WriteString(lineStr)
		if i < len(state.outputLine)-1 {
			result.WriteString("\n")
		}
	}
	return result.String()
}