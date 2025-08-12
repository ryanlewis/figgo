package renderer

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/ryanlewis/figgo/internal/parser"
)

// RenderTo writes ASCII art directly to the provided writer using the font and options.
// This is more efficient than Render as it avoids allocating a string for the result.
func RenderTo(w io.Writer, text string, font *parser.Font, opts *Options) error {
	if font == nil {
		return ErrNilFont
	}

	// Get render state from pool
	state := acquireRenderState(font.Height, font.Hardblank)
	defer releaseRenderState(state)

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
					return fmt.Errorf("%w: %s", ErrUnsupportedRune, string(r))
				}
			} else {
				return fmt.Errorf("%w: %s", ErrUnsupportedRune, string(r))
			}
		}

		// Add character to output
		if !state.addChar(glyph) {
			// Character didn't fit - for now just continue
			// Line breaking logic would go here
		}
	}

	// Write output directly to writer
	return state.writeTo(w)
}

// Render converts text to ASCII art using the font and options.
// It returns the rendered text as a string.
func Render(text string, font *parser.Font, opts *Options) (string, error) {
	var sb strings.Builder
	// Pre-size the builder for efficiency
	if font != nil {
		// Estimate: ~10 chars per input char * height
		estimatedSize := len(text) * 10 * font.Height
		if estimatedSize > 0 && estimatedSize < 1<<20 { // Cap at 1MB
			sb.Grow(estimatedSize)
		}
	}

	err := RenderTo(&sb, text, font, opts)
	if err != nil {
		return "", err
	}
	return sb.String(), nil
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
		smushMode |= SMKern
	} else if layout&(1<<7) != 0 { // FitSmushing
		smushMode |= SMSmush

		// Add smushing rules (bits 0-5)
		if layout&(1<<0) != 0 { // RuleEqualChar
			smushMode |= SMEqual
		}
		if layout&(1<<1) != 0 { // RuleUnderscore
			smushMode |= SMLowline
		}
		if layout&(1<<2) != 0 { // RuleHierarchy
			smushMode |= SMHierarchy
		}
		if layout&(1<<3) != 0 { // RuleOppositePair
			smushMode |= SMPair
		}
		if layout&(1<<4) != 0 { // RuleBigX
			smushMode |= SMBigX
		}
		if layout&(1<<5) != 0 { // RuleHardblank
			smushMode |= SMHardblank
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
		return SMKern
	} else if oldLayout < 0 {
		// Invalid, default to full-width
		return 0
	} else {
		// Smushing mode with rules (1..63)
		return SMSmush | (oldLayout & 63)
	}
}

// addChar attempts to add a character glyph to the current output line
func (state *renderState) addChar(glyph []string) bool {
	if len(glyph) != state.charHeight {
		return false // Invalid glyph height
	}

	// Save previous width BEFORE updating current character (like figlet.c getletter)
	state.previousCharWidth = state.currentCharWidth

	// Set current character data
	state.currentChar = glyph

	// Calculate character width - figlet.c uses ONLY first row's length
	// currcharwidth = STRLEN(currchar[0]);
	if len(glyph) > 0 {
		state.currentCharWidth = getCachedRuneCount(glyph[0])
	} else {
		state.currentCharWidth = 0
	}

	// Calculate smush amount
	smushAmount := state.smushAmount()

	// Ensure smushAmount is not negative
	if smushAmount < 0 {
		smushAmount = 0
	}

	// Check if character fits
	newLength := state.outlineLen + state.currentCharWidth - smushAmount
	if newLength > state.outlineLenLimit {
		return false
	}

	// Get pooled buffer for rune conversion
	runeBuffer := acquireRuneSlice()
	defer releaseRuneSlice(runeBuffer)

	// Add character to each row
	for row := 0; row < state.charHeight; row++ {
		// Convert to runes using pooled buffer for efficiency
		rowStr := glyph[row]

		// Ensure buffer is large enough
		needed := len(rowStr) // Worst case: all ASCII
		if cap(runeBuffer) < needed {
			runeBuffer = make([]rune, needed)
		} else {
			runeBuffer = runeBuffer[:0] // Reset length
		}

		// Convert string to runes
		for _, r := range rowStr {
			runeBuffer = append(runeBuffer, r)
		}
		rowRunes := runeBuffer

		if state.right2left != 0 {
			// Right-to-left processing (exact figlet.c logic)
			// Get temp buffer from pool
			tempLine := acquireTempLine()
			defer releaseTempLine(tempLine)

			// STRCPY(templine,currchar[row])
			copy(tempLine, rowRunes)

			// Apply smushing at overlap positions
			for k := 0; k < smushAmount; k++ {
				if k < len(rowRunes) {
					// Always assign result like figlet.c
					tempLine[state.currentCharWidth-smushAmount+k] =
						state.smush(tempLine[state.currentCharWidth-smushAmount+k], state.outputLine[row][k])
				}
			}

			// STRCAT(templine,outputline[row]+smushamount)
			if smushAmount < state.rowLengths[row] {
				// Copy the part of outputline after smush region
				copy(tempLine[state.currentCharWidth:], state.outputLine[row][smushAmount:state.rowLengths[row]])
			}

			// STRCPY(outputline[row],templine)
			copy(state.outputLine[row], tempLine)

			// Update row length
			state.rowLengths[row] = state.currentCharWidth + state.rowLengths[row] - smushAmount
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
					state.outputLine[row][column] = state.smush(state.outputLine[row][column], rowRunes[k])
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

// writeTo writes the rendered output directly to an io.Writer.
// This avoids allocating a string for the entire output.
func (state *renderState) writeTo(w io.Writer) error {
	if state.charHeight == 0 {
		return nil
	}

	// Get buffer from pool for UTF-8 encoding
	buf := acquireWriteBuffer()
	defer releaseWriteBuffer(buf)

	// Ensure buffer has reasonable capacity
	if cap(buf) < 256 {
		buf = make([]byte, 0, 256)
	}

	for i, line := range state.outputLine {
		// Extract only the actual content using row-specific length
		actualLine := line[:state.rowLengths[i]]

		// Process the line, replacing hardblanks and optionally trimming
		lastNonSpace := len(actualLine) - 1
		if state.trimWhitespace {
			// Find last non-space character
			for lastNonSpace >= 0 && actualLine[lastNonSpace] == ' ' {
				lastNonSpace--
			}
		}

		// Write runes to buffer, replacing hardblanks
		buf = buf[:0] // Reset buffer
		for j := 0; j <= lastNonSpace; j++ {
			r := actualLine[j]
			if r == state.hardblank {
				r = ' '
			}

			// Append rune to buffer
			buf = utf8.AppendRune(buf, r)

			// Flush buffer if getting full (leave room for max rune size)
			if len(buf) > 250 {
				if _, err := w.Write(buf); err != nil {
					return err
				}
				buf = buf[:0]
			}
		}

		// Write any remaining buffer content
		if len(buf) > 0 {
			if _, err := w.Write(buf); err != nil {
				return err
			}
		}

		// Write newline between lines (but not after last line)
		if i < len(state.outputLine)-1 {
			if _, err := w.Write([]byte{'\n'}); err != nil {
				return err
			}
		}
	}

	return nil
}

// outputToString converts the output lines to a final string.
// Deprecated: Use writeTo for better performance.
func (state *renderState) outputToString() string {
	var sb strings.Builder
	// Pre-calculate capacity
	maxWidth := 0
	for _, len := range state.rowLengths {
		if len > maxWidth {
			maxWidth = len
		}
	}
	sb.Grow(calculateBuilderCapacity(state.charHeight, maxWidth))

	// Use writeTo to avoid duplication
	_ = state.writeTo(&sb) // strings.Builder's Write never returns an error
	return sb.String()
}
