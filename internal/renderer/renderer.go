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

		// Set width limit
		if opts.Width != nil && *opts.Width > 0 {
			state.outlineLenLimit = *opts.Width - 1 // -1 to match figlet behavior
		} else {
			state.outlineLenLimit = 79 // Default: 80 - 1
		}
	} else {
		state.outlineLenLimit = 79 // Default when no options
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
		// Use FullLayout when available, fall back to OldLayout
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
			// Flush current line and start new one
			if state.outlineLen > 0 {
				state.flushLine()
			}
			state.inputCount = 0
			state.lastWordBreak = -1
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

		// Track input characters for word boundaries
		if len(state.inputBuffer) <= state.inputCount {
			// Grow inputBuffer if needed
			state.inputBuffer = append(state.inputBuffer, r)
		} else {
			state.inputBuffer[state.inputCount] = r
		}

		// Track word boundaries (spaces)
		if r == ' ' {
			state.lastWordBreak = state.inputCount
		}
		state.inputCount++

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

		// Try to add character to output
		if !state.addChar(glyph) {
			// Character didn't fit
			if state.outlineLen == 0 {
				// Line is empty but character doesn't fit - force add it
				// This prevents infinite loops with very narrow widths
				oldLimit := state.outlineLenLimit
				state.outlineLenLimit = 10000 // Temporarily allow
				state.addChar(glyph)
				state.outlineLenLimit = oldLimit
			} else if state.lastWordBreak > 0 {
				// Try to split at word boundary
				if state.splitLine(font, opts) {
					// Successfully split, character is already on new line
					// No need to add it again
				} else {
					// Split failed, try adding character again
					if !state.addChar(glyph) {
						// Still doesn't fit, force it
						oldLimit := state.outlineLenLimit
						state.outlineLenLimit = 10000
						state.addChar(glyph)
						state.outlineLenLimit = oldLimit
					}
				}
			} else {
				// No word boundary, just flush and continue
				state.flushLine()
				// Reset input tracking for new line
				state.inputCount = 0
				state.lastWordBreak = -1
				// Try adding character on new line
				if !state.addChar(glyph) {
					// Force it if still doesn't fit
					oldLimit := state.outlineLenLimit
					state.outlineLenLimit = 10000
					state.addChar(glyph)
					state.outlineLenLimit = oldLimit
				}
				// Track this character in inputBuffer for new line
				if len(state.inputBuffer) <= state.inputCount {
					state.inputBuffer = append(state.inputBuffer, r)
				} else {
					state.inputBuffer[state.inputCount] = r
				}
				if r == ' ' {
					state.lastWordBreak = state.inputCount
				}
				state.inputCount++
			}
		}
	}

	// Flush any remaining line
	if state.outlineLen > 0 {
		state.flushLine()
	}

	// Write accumulated output to writer
	if len(state.outputBuffer) > 0 {
		// Remove the trailing newline from the last line
		if state.outputBuffer[len(state.outputBuffer)-1] == '\n' {
			state.outputBuffer = state.outputBuffer[:len(state.outputBuffer)-1]
		}
		_, err := w.Write(state.outputBuffer)
		return err
	}

	return nil
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

// layoutToSmushMode converts figgo Layout bitmask to smush mode.
//
// Bitmask Conversion:
// The figgo Layout type uses a different bit layout than the internal smush mode:
// - figgo: Fitting modes in bits 6-7, rules in bits 0-5
// - smush: Combined mode and rules in a single integer
//
// This function bridges the two representations:
// 1. Extracts fitting mode from bits 6-7 (FitKerning/FitSmushing)
// 2. If smushing is enabled, extracts rules from bits 0-5
// 3. Maps each rule bit to the corresponding SM* constant
//
// The conversion is necessary because figgo's public API uses a cleaner
// bitmask design, while the renderer uses the original FIGlet constants
// for compatibility with the smushing algorithm.
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

// oldLayoutToSmushMode converts font OldLayout (-1 or 0..63) to smush mode.
//
// OldLayout Interpretation (from FIGfont v2 spec):
// - -1: Full-width mode (no character overlap)
// - 0: Kerning mode (minimal spacing, no overlap)
// - 1-63: Smushing mode with rules encoded in bits 0-5
//
// The bits directly map to smushing rules:
// - Bit 0: Equal character smushing
// - Bit 1: Underscore smushing
// - Bit 2: Hierarchy smushing
// - Bit 3: Opposite pair smushing
// - Bit 4: Big X smushing
// - Bit 5: Hardblank smushing
//
// Invalid values (<-1) default to full-width for safety.
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

// addChar attempts to add a character glyph to the current output line.
//
// Character Addition Algorithm:
// 1. Validates glyph height matches font height
// 2. Saves previous character width for smushing calculations
// 3. Calculates how much characters can overlap (smushAmount)
// 4. Checks if the new character fits within line limits
// 5. Applies smushing at overlap positions (row by row)
// 6. Handles both LTR and RTL rendering directions
//
// State Management:
// - previousCharWidth: Width of last added character (for smushing)
// - currentChar: The glyph being added
// - currentCharWidth: Width of current glyph
// - outlineLen: Total length of output line so far
// - rowLengths: Actual content length per row (for trimming)
//
// The function uses pooled buffers for rune conversion to minimize allocations.
// RTL processing requires a temporary buffer to reverse the merge order.
func (state *renderState) addChar(glyph []string) bool {
	if len(glyph) != state.charHeight {
		return false // Invalid glyph height
	}

	// Save previous width BEFORE updating current character
	state.previousCharWidth = state.currentCharWidth

	// Set current character data
	state.currentChar = glyph

	// Calculate character width using only first row's length
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
			// Right-to-left processing
			// Get temp buffer from pool
			tempLine := acquireTempLine()
			defer releaseTempLine(tempLine)

			// Copy current character to temp buffer
			copy(tempLine, rowRunes)

			// Apply smushing at overlap positions
			for k := 0; k < smushAmount; k++ {
				if k < len(rowRunes) {
					// Always assign result
					tempLine[state.currentCharWidth-smushAmount+k] =
						state.smush(tempLine[state.currentCharWidth-smushAmount+k], state.outputLine[row][k])
				}
			}

			// Append remaining output line after smush region
			if smushAmount < state.rowLengths[row] {
				// Copy the part of outputline after smush region
				copy(tempLine[state.currentCharWidth:], state.outputLine[row][smushAmount:state.rowLengths[row]])
			}

			// Copy temp buffer back to output line
			copy(state.outputLine[row], tempLine)

			// Update row length
			state.rowLengths[row] = state.currentCharWidth + state.rowLengths[row] - smushAmount
		} else {
			// Left-to-right processing
			// Apply smushing at overlap positions
			for k := 0; k < smushAmount; k++ {
				column := state.outlineLen - smushAmount + k
				if column < 0 {
					column = 0
				}
				// Use currchar[row][k] directly - no adjustment for leading spaces
				// With pre-allocated buffer, we don't need to check outputLine bounds
				if k < len(rowRunes) {
					// Smush the characters - always assign result
					state.outputLine[row][column] = state.smush(state.outputLine[row][column], rowRunes[k])
				}
			}

			// Append character after smush region to output
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
	// Update output length
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
//
// Buffer Management Strategy:
// 1. Uses a pooled byte buffer for UTF-8 encoding
// 2. Processes each row independently
// 3. Replaces hardblanks with spaces during output
// 4. Optionally trims trailing whitespace per row
// 5. Flushes buffer when approaching capacity (>250 bytes)
//
// The function minimizes memory allocations by:
// - Reusing a single byte buffer across all rows
// - Writing directly to the io.Writer in chunks
// - Using utf8.AppendRune for efficient rune encoding
//
// Row Processing:
// - Uses row-specific lengths (rowLengths[i]) for accurate content
// - Trims trailing spaces if trimWhitespace is enabled
// - Preserves internal spaces for ASCII art alignment
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

// flushLine writes the current output line to the buffer and resets for the next line.
// This is called when a line is complete and needs to be output.
func (state *renderState) flushLine() {
	if state.charHeight == 0 || state.outlineLen == 0 {
		return
	}

	// Get buffer from pool for UTF-8 encoding
	buf := acquireWriteBuffer()
	defer releaseWriteBuffer(buf)

	// Ensure buffer has reasonable capacity
	if cap(buf) < 256 {
		buf = make([]byte, 0, 256)
	}

	// Process each row of the current line
	for i := 0; i < state.charHeight; i++ {
		// Extract only the actual content using row-specific length
		actualLine := state.outputLine[i][:state.rowLengths[i]]

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
		}

		// Append to output buffer
		state.outputBuffer = append(state.outputBuffer, buf...)

		// Add newline after each row
		state.outputBuffer = append(state.outputBuffer, '\n')
	}

	// Reset the line state for next line
	state.resetLine()
}

// clearOutputLine clears just the output line buffer without resetting other state.
// This is used during word wrapping to re-render from a specific point.
func (state *renderState) clearOutputLine() {
	// Clear output line content
	for i := 0; i < state.charHeight; i++ {
		// Just reset the length, keep the allocated buffer
		for j := 0; j < state.rowLengths[i]; j++ {
			state.outputLine[i][j] = ' '
		}
		state.rowLengths[i] = 0
	}

	// Reset output tracking
	state.outlineLen = 0
	state.previousCharWidth = 0
	state.currentCharWidth = 0
}

// resetLine clears the current output line for the next line of text.
func (state *renderState) resetLine() {
	// Clear output line content
	for i := 0; i < state.charHeight; i++ {
		// Just reset the length, keep the allocated buffer
		for j := 0; j < state.rowLengths[i]; j++ {
			state.outputLine[i][j] = ' '
		}
		state.rowLengths[i] = 0
	}

	// Reset line tracking
	state.outlineLen = 0
	state.previousCharWidth = 0
	state.currentCharWidth = 0

	// Reset input line tracking
	state.inputCount = 0
	state.lastWordBreak = -1
}

// renderCharacterRange renders a specific range of characters from inputBuffer.
// This is used during word wrapping to re-render specific portions of the input.
func (state *renderState) renderCharacterRange(font *parser.Font, start, end int, opts *Options) error {
	// Bounds check
	if start < 0 || end > len(state.inputBuffer) || start >= end {
		return nil
	}

	// Render each character in the range
	for i := start; i < end; i++ {
		r := state.inputBuffer[i]

		// Skip newlines during re-rendering
		if r == '\n' {
			continue
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

		// Add character to output (without updating inputBuffer)
		if !state.addChar(glyph) {
			// Character doesn't fit - this shouldn't happen during re-rendering
			// as we're rendering a known-good range
			break
		}
	}

	return nil
}

// splitLine splits the current line at the last word boundary.
// Returns true if a split was performed and the current character was re-added, false otherwise.
func (state *renderState) splitLine(font *parser.Font, opts *Options) bool {
	if state.lastWordBreak <= 0 || state.lastWordBreak >= state.inputCount {
		return false
	}

	// Find the last group of spaces (like figlet does)
	// Scan backwards from the end to find where spaces start
	gotSpace := false
	lastSpace := state.lastWordBreak
	
	for i := state.inputCount - 1; i >= 0; i-- {
		if i >= len(state.inputBuffer) {
			continue
		}
		if !gotSpace && state.inputBuffer[i] == ' ' {
			gotSpace = true
			lastSpace = i
		}
		if gotSpace && state.inputBuffer[i] != ' ' {
			// Found non-space after spaces
			// Split point is after this non-space character
			break
		}
	}

	// First part ends at the last non-space before the space group
	// Second part starts after the space group
	firstPartEnd := lastSpace
	for firstPartEnd > 0 && state.inputBuffer[firstPartEnd-1] == ' ' {
		firstPartEnd--
	}
	
	// Skip all spaces to find where the second part starts
	secondPartStart := lastSpace
	for secondPartStart < state.inputCount && secondPartStart < len(state.inputBuffer) && state.inputBuffer[secondPartStart] == ' ' {
		secondPartStart++
	}

	// Save the characters that will go on the next line
	nextLineStart := secondPartStart
	nextLineEnd := state.inputCount

	// Clear the current output line
	state.clearOutputLine()

	// Re-render everything up to the first part end (before the spaces)
	if err := state.renderCharacterRange(font, 0, firstPartEnd, opts); err != nil {
		// If re-rendering fails, fall back to flushing
		state.flushLine()
		return false
	}

	// Flush the first part
	state.flushLine()

	// Now render the second part on the new line
	// Shift the remaining characters to the beginning of inputBuffer
	remainingLen := nextLineEnd - nextLineStart
	for i := 0; i < remainingLen; i++ {
		state.inputBuffer[i] = state.inputBuffer[nextLineStart+i]
	}
	state.inputCount = remainingLen
	state.lastWordBreak = -1

	// Find new word boundaries in the shifted text
	for i := 0; i < state.inputCount; i++ {
		if state.inputBuffer[i] == ' ' {
			state.lastWordBreak = i
		}
	}

	// Render the remaining characters (now at the start of inputBuffer)
	if err := state.renderCharacterRange(font, 0, state.inputCount, opts); err != nil {
		return false
	}

	// The current character that didn't fit has already been re-rendered
	// as part of renderCharacterRange. No need to add it again.
	return true
}
