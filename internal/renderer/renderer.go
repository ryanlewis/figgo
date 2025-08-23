package renderer

import (
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ryanlewis/figgo/internal/debug"
	"github.com/ryanlewis/figgo/internal/parser"
)

// singleSpaceGlyph creates a 1-column space glyph for the given height.
// This is used in smushing mode to ensure input spaces contribute exactly 1 column.
func singleSpaceGlyph(height int) []string {
	rows := make([]string, height)
	for i := 0; i < height; i++ {
		rows[i] = " "
	}
	return rows
}

// RenderTo writes ASCII art directly to the provided writer using the font and options.
// This is more efficient than Render as it avoids allocating a string for the result.
func RenderTo(w io.Writer, text string, font *parser.Font, opts *Options) error {
	if font == nil {
		return ErrNilFont
	}

	// Get render state from pool
	state := acquireRenderState(font.Height, font.Hardblank, len(text))
	defer releaseRenderState(state)
	
	// Set debug session if provided
	if opts != nil && opts.Debug != nil {
		state.debug = opts.Debug
	}

	// Set trim whitespace option
	if opts != nil {
		state.trimWhitespace = opts.TrimWhitespace

		// Set width limit
		if opts.Width != nil && *opts.Width > 0 {
			state.outlineLenLimit = *opts.Width - 1 // -1 to match figlet behavior
		} else {
			// Use large default for golden test compatibility
			// Golden tests were generated with effectively no wrapping
			state.outlineLenLimit = 10000
		}
	} else {
		// Use large default for golden test compatibility
		state.outlineLenLimit = 10000
	}

	// Set print direction from options or font
	if opts != nil && opts.PrintDirection != nil {
		state.right2left = *opts.PrintDirection
	} else {
		state.right2left = font.PrintDirection
	}

	// Convert layout to smush mode
	if opts != nil {
		// Check if FitSmushing (bit 7) is specified without rule bits (bits 0-5)
		// This means "use font's default smushing rules"
		if opts.Layout == (1<<7) {
			// Use font's default smushing rules
			if font.FullLayoutSet && font.FullLayout != 0 {
				// Extract horizontal layout from FullLayout
				state.smushMode = font.FullLayout & 0xFF
			} else {
				// Fall back to converting OldLayout
				state.smushMode = oldLayoutToSmushMode(font.OldLayout)
			}
			// Ensure smushing mode is enabled
			state.smushMode |= SMSmush
		} else {
			state.smushMode = layoutToSmushMode(opts.Layout)
		}
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
	
	// Log render start
	var startTime time.Time
	if state.debug != nil {
		startTime = time.Now()
		state.debug.Emit("render", "Start", debug.RenderStartData{
			Text:        text,
			TextLength:  len(text),
			CharHeight:  state.charHeight,
			Hardblank:   state.hardblank,
			WidthLimit:  state.outlineLenLimit,
			PrintDir:    state.right2left,
			SmushMode:   state.smushMode,
			SmushRules:  debug.FormatSmushRules(state.smushMode),
		})
	}

	// Process each character in the input text
	for _, r := range text {
		// FIRST: Normalize whitespace
		if r == '\t' {
			r = ' '
		}
		
		// Handle newlines
		if r == '\n' {
			// Flush current line and start new one
			if state.outlineLen > 0 {
				state.flushLine()
			}
			// Reset FSM and buffer state unconditionally
			state.wordbreakmode = 0
			state.inputCount = 0
			state.lastWordBreak = -1
			continue
		}

		// Early space absorption for wordbreakmode == -1
		if state.wordbreakmode == -1 && r == ' ' {
			continue // Absorb space (includes normalized tabs)
		}

		// Skip control characters
		if r < ' ' {
			continue
		}
		
		// Retry loop for character processing
		retry := true
		for retry {
			retry = false
			
			// Get character glyph
			glyph, exists := font.Characters[r]
			unknownSubstituted := false
			if !exists {
				// Handle unknown character
				if opts != nil && opts.UnknownRune != nil {
					originalRune := r
					r = *opts.UnknownRune
					glyph, exists = font.Characters[r]
					unknownSubstituted = true
					if !exists {
						return fmt.Errorf("%w: %s", ErrUnsupportedRune, string(originalRune))
					}
				} else {
					return fmt.Errorf("%w: %s", ErrUnsupportedRune, string(r))
				}
			}

			// In smushing mode, use a normalized 1-column space glyph
			if r == ' ' && (state.smushMode & SMSmush) != 0 {
				glyph = singleSpaceGlyph(state.charHeight)
				state.processingSpaceGlyph = true
			} else {
				state.processingSpaceGlyph = false
			}
			
			// Emit glyph event
			if state.debug != nil {
				glyphWidth := 0
				if len(glyph) > 0 {
					glyphWidth = getCachedRuneCount(glyph[0])
				}
				state.debug.Emit("render", "Glyph", debug.GlyphData{
					Index:        state.inputCount,
					Rune:         r,
					Width:        glyphWidth,
					SpaceGlyph:   state.processingSpaceGlyph,
					UnknownSubst: unknownSubstituted,
				})
			}
			
			// Try to add character to output
			if state.addChar(glyph) {
				// Clear flag after add
				state.processingSpaceGlyph = false
				
				// SUCCESS - NOW append to buffer
				if len(state.inputBuffer) <= state.inputCount {
					state.inputBuffer = append(state.inputBuffer, r)
				} else {
					state.inputBuffer[state.inputCount] = r
				}
				
				// Track word boundaries
				if r == ' ' {
					state.lastWordBreak = state.inputCount
				}
				state.inputCount++
				
				// Debug: log successful addition
				// fmt.Fprintf(os.Stderr, "Added '%c' at position %d (inputCount now %d)\n", r, state.inputCount-1, state.inputCount)
				
				// Update FSM on success
				if r != ' ' {
					if state.wordbreakmode >= 2 {
						state.wordbreakmode = 3
					} else {
						state.wordbreakmode = 1
					}
				} else {
					if state.wordbreakmode > 0 {
						state.wordbreakmode = 2
					} else {
						state.wordbreakmode = 0
					}
				}
			} else {
				// Clear flag after failed add
				state.processingSpaceGlyph = false
				
				// FAILURE - handle based on context without buffer contamination
				if state.outlineLen == 0 {
					// Oversized first char - print truncated version
					for row := 0; row < state.charHeight; row++ {
						glyphRow := glyph[row]
						runeSlice := []rune(glyphRow)
						
						if state.right2left != 0 && state.outlineLenLimit > 1 {
							// RTL: Copy rightmost outlineLenLimit runes
							start := len(runeSlice) - state.outlineLenLimit
							if start < 0 {
								start = 0
							}
							truncated := runeSlice[start:]
							copy(state.outputLine[row], truncated)
							state.rowLengths[row] = len(truncated)
						} else {
							// LTR: Copy leftmost outlineLenLimit runes
							limit := len(runeSlice)
							if limit > state.outlineLenLimit {
								limit = state.outlineLenLimit
							}
							copy(state.outputLine[row], runeSlice[:limit])
							state.rowLengths[row] = limit
						}
					}
					state.outlineLen = state.rowLengths[0]
					state.flushLine()
					state.wordbreakmode = -1 // Enter absorption mode
					
				} else if r == ' ' {
					// Space failure
					if state.wordbreakmode == 2 {
						state.splitLine(font, opts)
					} else {
						state.flushLine()
					}
					state.wordbreakmode = -1 // Enter absorption mode
					// NO RETRY - space is consumed
					
				} else {
					// Non-space failure
					// Capture previous state BEFORE split/flush
					prevState := state.wordbreakmode
					
					if state.wordbreakmode >= 2 {
						split := state.splitLine(font, opts)
						if !split {
							state.flushLine()
						}
					} else {
						state.flushLine()
					}
					
					// Update FSM using PREVIOUS state for correct transition
					if prevState == 3 {
						state.wordbreakmode = 1
					} else {
						state.wordbreakmode = 0
					}
					
					// ALWAYS retry non-space (unconditional)
					retry = true
				}
			}
		}
	}

	// Flush any remaining line
	if state.outlineLen > 0 {
		state.flushLine()
	}

	// Write accumulated output to writer
	bytesWritten := 0
	if len(state.outputBuffer) > 0 {
		// Remove the trailing newline from the last line
		if state.outputBuffer[len(state.outputBuffer)-1] == '\n' {
			state.outputBuffer = state.outputBuffer[:len(state.outputBuffer)-1]
		}
		bytesWritten = len(state.outputBuffer)
		_, err := w.Write(state.outputBuffer)
		
		// Log render end
		if state.debug != nil {
			elapsed := time.Since(startTime)
			state.debug.Emit("render", "End", debug.RenderEndData{
				TotalLines:   strings.Count(string(state.outputBuffer), "\n") + 1,
				TotalRunes:   len([]rune(text)),
				TotalGlyphs:  state.inputCount,
				ElapsedMs:    elapsed.Milliseconds(),
				BytesWritten: bytesWritten,
			})
		}
		
		return err
	}

	// Handle empty input - return height-1 blank lines
	// (Tests expect height-1 newlines since final trailing newline is trimmed)
	if len(text) == 0 && font.Height > 1 {
		blankLines := make([]byte, font.Height-1)
		for i := 0; i < font.Height-1; i++ {
			blankLines[i] = '\n'
		}
		bytesWritten = len(blankLines)
		_, err := w.Write(blankLines)
		
		// Log render end for empty input
		if state.debug != nil {
			elapsed := time.Since(startTime)
			state.debug.Emit("render", "End", debug.RenderEndData{
				TotalLines:   font.Height - 1,
				TotalRunes:   0,
				TotalGlyphs:  0,
				ElapsedMs:    elapsed.Milliseconds(),
				BytesWritten: bytesWritten,
			})
		}
		
		return err
	}
	
	// Log render end for zero output
	if state.debug != nil {
		elapsed := time.Since(startTime)
		state.debug.Emit("render", "End", debug.RenderEndData{
			TotalLines:   0,
			TotalRunes:   len([]rune(text)),
			TotalGlyphs:  state.inputCount,
			ElapsedMs:    elapsed.Milliseconds(),
			BytesWritten: 0,
		})
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

	// Calculate character width using full glyph width
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
			// Track per-row end position to emulate C string truncation
			end := state.rowLengths[row]
			
			// Get temp buffer from pool
			tempLine := acquireTempLine()
			defer releaseTempLine(tempLine)

			// Copy current character to temp buffer
			copy(tempLine, rowRunes)

			// Apply smushing at overlap positions
			for k := 0; k < smushAmount; k++ {
				column := state.currentCharWidth - smushAmount + k
				
				// CRITICAL: Use 0 for past-end (not space) to match figlet behavior
				var existing rune = 0
				if k < end {
					existing = state.outputLine[row][k]
				}
				
				if k < len(rowRunes) && column >= 0 && column < len(tempLine) {
					smushResult := state.smush(tempLine[column], existing)
					if smushResult != 0 {
						// Write result 
						tempLine[column] = smushResult
						
						// Emit debug event for smush decision
						if state.debug != nil {
							state.debug.Emit("render", "SmushDecision", debug.SmushDecisionData{
								Row:    row,
								Col:    k,
								Lch:    tempLine[column],
								Rch:    existing,
								Result: smushResult,
								Rule:   debug.ClassifySmushRule(tempLine[column], existing, smushResult, state.smushMode),
							})
						}
					} else {
						// Emulate truncation for RTL
						// For RTL, we need to track how this affects the merge
						// The temp buffer will be copied back, so we handle it there
						tempLine[column] = 0  // Mark for truncation
					}
				}
			}

			// Build final output considering truncation
			finalEnd := 0
			
			// Copy tempLine handling truncation
			for i := 0; i < state.currentCharWidth; i++ {
				if tempLine[i] != 0 {
					state.outputLine[row][finalEnd] = tempLine[i]
					finalEnd++
				} else {
					// Hit truncation point
					break
				}
			}
			
			// Append remaining output line after smush region if no truncation
			if smushAmount < end && finalEnd == state.currentCharWidth {
				// Copy the part of outputline after smush region
				remaining := state.outputLine[row][smushAmount:end]
				startPos := finalEnd
				copy(state.outputLine[row][finalEnd:], remaining)
				finalEnd += end - smushAmount
				
				// Emit debug event for row append
				if state.debug != nil {
					state.debug.Emit("render", "RowAppend", debug.RowAppendData{
						Row:       row,
						StartPos:  startPos,
						CharCount: end - smushAmount,
						EndBefore: startPos,
						EndAfter:  finalEnd,
					})
				}
			}

			// Update row length
			state.rowLengths[row] = finalEnd
		} else {
			// Left-to-right processing
			// Track per-row end position to emulate C string truncation
			end := state.rowLengths[row]
			
			// Apply smushing at overlap positions
			for k := 0; k < smushAmount; k++ {
				column := state.outlineLen - smushAmount + k
				if column < 0 {
					column = 0
				}
				
				// CRITICAL: Use 0 for past-end (not space) to match figlet behavior
				var existing rune = 0
				if column < end {
					existing = state.outputLine[row][column]
				}
				
				// Smush characters
				if k < len(rowRunes) {
					smushResult := state.smush(existing, rowRunes[k])
					if smushResult != 0 {
						// Write result and extend end if needed
						state.outputLine[row][column] = smushResult
						if column >= end {
							end = column + 1
						}
						
						// Emit debug event for smush decision
						if state.debug != nil {
							state.debug.Emit("render", "SmushDecision", debug.SmushDecisionData{
								Row:    row,
								Col:    column,
								Lch:    existing,
								Rch:    rowRunes[k],
								Result: smushResult,
								Rule:   debug.ClassifySmushRule(existing, rowRunes[k], smushResult, state.smushMode),
							})
						}
					} else {
						// Emulate C string truncation - shrink end
						// DO NOT write 0 rune
						if column < end {
							end = column
						}
					}
				}
			}

			// CRITICAL: Append at per-row end, not at outlineLen
			// This emulates figlet's STRCAT behavior after NUL truncation
			if smushAmount < len(rowRunes) {
				remaining := rowRunes[smushAmount:]
				dest := end  // Use per-row end, not outlineLen!
				startPos := dest
				copy(state.outputLine[row][dest:], remaining)
				end += len(remaining)
				
				// Emit debug event for row append
				if state.debug != nil {
					state.debug.Emit("render", "RowAppend", debug.RowAppendData{
						Row:       row,
						StartPos:  startPos,
						CharCount: len(remaining),
						EndBefore: startPos,
						EndAfter:  end,
					})
				}
			}
			
			// Update row length with new end position
			state.rowLengths[row] = end
		}
	}

	// Update output length based on row 0's length
	// In figlet.c, this is done with: outlinelen = STRLEN(outputline[0])
	// Trust the rowLengths we maintained during merge operations
	state.outlineLen = state.rowLengths[0]
	
	// No need to rescan - we trust rowLengths maintained during operations

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
	// Clear output line content only up to rowLengths[i]
	for i := 0; i < state.charHeight; i++ {
		// Clear only the used portion
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
	// Clear output line content only up to rowLengths[i]
	for i := 0; i < state.charHeight; i++ {
		// Clear only the used portion
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
// Returns the count of characters successfully rendered.
func (state *renderState) renderCharacterRange(font *parser.Font, start, end int, opts *Options) (int, error) {
	// Bounds check
	if start < 0 || end > len(state.inputBuffer) || start >= end {
		return 0, nil
	}

	renderedCount := 0
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
					return renderedCount, fmt.Errorf("%w: %s", ErrUnsupportedRune, string(r))
				}
			} else {
				return renderedCount, fmt.Errorf("%w: %s", ErrUnsupportedRune, string(r))
			}
		}

		// In smushing mode, use a normalized 1-column space glyph
		if r == ' ' && (state.smushMode & SMSmush) != 0 {
			glyph = singleSpaceGlyph(state.charHeight)
			state.processingSpaceGlyph = true
		} else {
			state.processingSpaceGlyph = false
		}
		
		// Add character to output (without updating inputBuffer)
		if !state.addChar(glyph) {
			// Character doesn't fit - return what we've rendered so far
			state.processingSpaceGlyph = false
			break
		}
		
		// Clear flag
		state.processingSpaceGlyph = false
		renderedCount++
	}

	return renderedCount, nil
}

// splitLine splits the current line at the last word boundary.
// Returns true if a split was performed, false otherwise.
func (state *renderState) splitLine(font *parser.Font, opts *Options) bool {
	// Find LAST space group from the end
	lastSpaceStart := -1
	lastSpaceEnd := -1
	inSpaceGroup := false
	
	// Scan backwards to find last space group
	for i := state.inputCount - 1; i >= 0; i-- {
		if i >= len(state.inputBuffer) {
			continue
		}
		if state.inputBuffer[i] == ' ' {
			if !inSpaceGroup {
				lastSpaceEnd = i + 1
				inSpaceGroup = true
			}
			lastSpaceStart = i
		} else if inSpaceGroup {
			// Found complete space group
			break
		}
	}
	
	if lastSpaceStart < 0 {
		return false  // No spaces found
	}
	
	// Clear the current output line
	state.clearOutputLine()
	
	// Render everything before the space group
	if lastSpaceStart > 0 {
		if _, err := state.renderCharacterRange(font, 0, lastSpaceStart, opts); err != nil {
			return false
		}
	}
	
	// Save inputCount before flush (flush resets it to 0)
	savedInputCount := state.inputCount
	
	// Flush the first part
	state.flushLine()
	
	// Shift remainder to beginning of inputBuffer
	if lastSpaceEnd < savedInputCount {
		remainingCount := savedInputCount - lastSpaceEnd
		copy(state.inputBuffer[0:], state.inputBuffer[lastSpaceEnd:savedInputCount])
		
		// Re-render the remainder on new line and get actual rendered count
		renderedCount, err := state.renderCharacterRange(font, 0, remainingCount, opts)
		if err != nil {
			return false
		}
		
		// Set inputCount to actual rendered count
		state.inputCount = renderedCount
		
		// Recompute lastWordBreak only within rendered range
		state.lastWordBreak = -1
		for i := 0; i < state.inputCount; i++ {
			if state.inputBuffer[i] == ' ' {
				state.lastWordBreak = i
			}
		}
	} else {
		state.inputCount = 0
		state.lastWordBreak = -1
	}
	
	return true
}
