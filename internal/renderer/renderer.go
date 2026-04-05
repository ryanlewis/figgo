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

// RenderTo writes ASCII art directly to the provided writer using the font and options.
// This is more efficient than Render as it avoids allocating a string for the result.
func RenderTo(w io.Writer, text string, font *parser.Font, opts *Options) error {
	if font == nil {
		return ErrNilFont
	}

	state := acquireRenderState(font.Height, font.Hardblank, len(text))
	defer releaseRenderState(state)

	state.initFromOptions(font, opts)
	startTime := state.emitRenderStart(text)

	if err := state.processText(text, font, opts); err != nil {
		return err
	}

	return state.writeOutput(w, text, startTime, font.Height)
}

// initFromOptions configures the render state from font and options.
func (state *renderState) initFromOptions(font *parser.Font, opts *Options) {
	if opts != nil && opts.Debug != nil {
		state.debug = opts.Debug
	}

	state.outlineLenLimit = 10000 // Default for golden test compatibility
	if opts != nil {
		state.trimWhitespace = opts.TrimWhitespace
		if opts.Width != nil && *opts.Width > 0 {
			state.outlineLenLimit = *opts.Width - 1
		}
	}

	if opts != nil && opts.PrintDirection != nil {
		state.right2left = *opts.PrintDirection
	} else {
		state.right2left = font.PrintDirection
	}

	state.smushMode = resolveSmushMode(font, opts)
}

// resolveSmushMode determines the smushing mode from font and options.
func resolveSmushMode(font *parser.Font, opts *Options) int {
	if opts == nil {
		return fontDefaultSmushMode(font)
	}
	// FitSmushing (bit 7) without rule bits means "use font's default rules"
	if opts.Layout == (1 << 7) {
		mode := fontDefaultSmushMode(font)
		return mode | SMSmush
	}
	return layoutToSmushMode(opts.Layout)
}

// fontDefaultSmushMode extracts the default smush mode from a font.
func fontDefaultSmushMode(font *parser.Font) int {
	if font.FullLayoutSet && font.FullLayout != 0 {
		return font.FullLayout & 0xFF
	}
	return oldLayoutToSmushMode(font.OldLayout)
}

// emitRenderStart emits a render start debug event and returns the start time.
func (state *renderState) emitRenderStart(text string) time.Time {
	if state.debug == nil {
		return time.Time{}
	}
	start := time.Now()
	state.debug.Emit("render", "Start", debug.RenderStartData{
		Text:       text,
		TextLength: len(text),
		CharHeight: state.charHeight,
		Hardblank:  state.hardblank,
		WidthLimit: state.outlineLenLimit,
		PrintDir:   state.right2left,
		SmushMode:  state.smushMode,
		SmushRules: debug.FormatSmushRules(state.smushMode),
	})
	return start
}

// processText iterates over the input text and builds the rendered output.
func (state *renderState) processText(text string, font *parser.Font, opts *Options) error {
	for charIdx, r := range text {
		if r == '\t' {
			r = ' '
		}

		if r == '\n' {
			state.handleNewline(charIdx)
			continue
		}

		if state.wordbreakmode == -1 && r == ' ' {
			continue
		}
		if r < ' ' {
			continue
		}

		if err := state.processChar(r, charIdx, font, opts); err != nil {
			return err
		}
	}

	// Flush any remaining line
	if state.outlineLen > 0 {
		state.emitSplit("end", 0, state.inputCount)
		state.flushLine()
	}
	return nil
}

// handleNewline processes a newline character in the input.
func (state *renderState) handleNewline(charIdx int) {
	if state.outlineLen > 0 {
		state.emitSplit("newline", 0, charIdx)
		state.flushLine()
	}
	state.wordbreakmode = 0
	state.inputCount = 0
	state.lastWordBreak = -1
}

// emitSplit emits a split debug event.
func (state *renderState) emitSplit(reason string, fsmNext, position int) {
	if state.debug == nil {
		return
	}
	state.debug.Emit("render", "Split", debug.SplitData{
		Reason:     reason,
		FSMPrev:    state.wordbreakmode,
		FSMNext:    fsmNext,
		OutlineLen: state.outlineLen,
		Position:   position,
	})
}

// processChar handles a single character with retry logic for line wrapping.
func (state *renderState) processChar(r rune, charIdx int, font *parser.Font, opts *Options) error {
	retry := true
	for retry {
		retry = false

		glyph, resolvedRune, err := state.lookupGlyph(r, font, opts)
		if err != nil {
			return err
		}
		r = resolvedRune

		state.processingSpaceGlyph = (r == ' ')
		state.emitGlyphEvent(r, glyph)

		if state.addChar(glyph) {
			state.processingSpaceGlyph = false
			state.recordSuccess(r)
		} else {
			state.processingSpaceGlyph = false
			retry = state.handleAddFailure(r, glyph, charIdx, font, opts)
		}
	}
	return nil
}

// lookupGlyph finds the glyph for a rune, handling unknown rune substitution.
func (state *renderState) lookupGlyph(r rune, font *parser.Font, opts *Options) ([]string, rune, error) {
	glyph, exists := font.Characters[r]
	if exists {
		return glyph, r, nil
	}
	if opts != nil && opts.UnknownRune != nil {
		originalRune := r
		r = *opts.UnknownRune
		glyph, exists = font.Characters[r]
		if exists {
			return glyph, r, nil
		}
		return nil, r, fmt.Errorf("%w: %s", ErrUnsupportedRune, string(originalRune))
	}
	return nil, r, fmt.Errorf("%w: %s", ErrUnsupportedRune, string(r))
}

// emitGlyphEvent emits a debug event for glyph processing.
func (state *renderState) emitGlyphEvent(r rune, glyph []string) {
	if state.debug == nil {
		return
	}
	glyphWidth := 0
	if len(glyph) > 0 {
		glyphWidth = getCachedRuneCount(glyph[0])
	}
	state.debug.Emit("render", "Glyph", debug.GlyphData{
		Index:        state.inputCount,
		Rune:         r,
		Width:        glyphWidth,
		SpaceGlyph:   state.processingSpaceGlyph,
		UnknownSubst: false,
	})
}

// recordSuccess records a successfully added character in the input buffer and updates FSM.
func (state *renderState) recordSuccess(r rune) {
	if len(state.inputBuffer) <= state.inputCount {
		state.inputBuffer = append(state.inputBuffer, r)
	} else {
		state.inputBuffer[state.inputCount] = r
	}

	if r == ' ' {
		state.lastWordBreak = state.inputCount
	}
	state.inputCount++

	state.updateFSMSuccess(r)
}

// updateFSMSuccess updates the word-break FSM after a successful character addition.
func (state *renderState) updateFSMSuccess(r rune) {
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
}

// handleAddFailure handles the case when addChar returns false (character doesn't fit).
// Returns true if the character should be retried.
func (state *renderState) handleAddFailure(r rune, glyph []string, charIdx int, font *parser.Font, opts *Options) bool {
	switch {
	case state.outlineLen == 0:
		return state.handleOversizedFirstChar(glyph, charIdx)
	case r == ' ':
		state.handleSpaceFailure(font, opts)
		return false
	default:
		return state.handleNonSpaceFailure(font, opts)
	}
}

// handleOversizedFirstChar handles a glyph that's too wide even for an empty line.
func (state *renderState) handleOversizedFirstChar(glyph []string, charIdx int) bool {
	if len(glyph) != state.charHeight {
		return false // malformed font - skip
	}
	for row := 0; row < state.charHeight; row++ {
		runeSlice := []rune(glyph[row])
		if state.right2left != 0 && state.outlineLenLimit > 1 {
			start := len(runeSlice) - state.outlineLenLimit
			if start < 0 {
				start = 0
			}
			truncated := runeSlice[start:]
			copy(state.outputLine[row], truncated)
			state.rowLengths[row] = len(truncated)
		} else {
			limit := len(runeSlice)
			if limit > state.outlineLenLimit {
				limit = state.outlineLenLimit
			}
			copy(state.outputLine[row], runeSlice[:limit])
			state.rowLengths[row] = limit
		}
	}
	state.outlineLen = state.rowLengths[0]
	state.emitSplit("width", -1, charIdx)
	state.flushLine()
	state.wordbreakmode = -1
	return false
}

// handleSpaceFailure handles when a space character doesn't fit on the line.
func (state *renderState) handleSpaceFailure(font *parser.Font, opts *Options) {
	if state.wordbreakmode == 2 {
		state.splitLine(font, opts)
	} else {
		state.flushLine()
	}
	state.wordbreakmode = -1
}

// handleNonSpaceFailure handles when a non-space character doesn't fit.
// Returns true to indicate the character should be retried on the new line.
func (state *renderState) handleNonSpaceFailure(font *parser.Font, opts *Options) bool {
	prevState := state.wordbreakmode

	if state.wordbreakmode >= 2 {
		if !state.splitLine(font, opts) {
			state.flushLine()
		}
	} else {
		state.flushLine()
	}

	if prevState == 3 {
		state.wordbreakmode = 1
	} else {
		state.wordbreakmode = 0
	}
	return true
}

// writeOutput writes the accumulated render output to the writer.
func (state *renderState) writeOutput(w io.Writer, text string, startTime time.Time, fontHeight int) error {
	if len(state.outputBuffer) > 0 {
		if state.outputBuffer[len(state.outputBuffer)-1] == '\n' {
			state.outputBuffer = state.outputBuffer[:len(state.outputBuffer)-1]
		}
		bytesWritten := len(state.outputBuffer)
		_, err := w.Write(state.outputBuffer)
		state.emitRenderEnd(text, startTime, strings.Count(string(state.outputBuffer), "\n")+1, bytesWritten)
		return err
	}

	// Handle empty output - return height-1 blank lines
	if fontHeight > 1 {
		blankLines := make([]byte, fontHeight-1)
		for i := range blankLines {
			blankLines[i] = '\n'
		}
		bytesWritten := len(blankLines)
		_, err := w.Write(blankLines)
		state.emitRenderEnd("", startTime, fontHeight-1, bytesWritten)
		return err
	}

	state.emitRenderEnd(text, startTime, 0, 0)
	return nil
}

// emitRenderEnd emits a render end debug event.
func (state *renderState) emitRenderEnd(text string, startTime time.Time, totalLines, bytesWritten int) {
	if state.debug == nil {
		return
	}
	elapsed := time.Since(startTime)
	state.debug.Emit("render", "End", debug.RenderEndData{
		TotalLines:   totalLines,
		TotalRunes:   len([]rune(text)),
		TotalGlyphs:  state.inputCount,
		ElapsedMs:    elapsed.Milliseconds(),
		BytesWritten: bytesWritten,
	})
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
	switch {
	case oldLayout == -1:
		// Full-width mode
		return 0
	case oldLayout == 0:
		// Kerning mode
		return SMKern
	case oldLayout < 0:
		// Invalid, default to full-width
		return 0
	default:
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
		return false
	}

	// Save previous width BEFORE updating current character
	state.previousCharWidth = state.currentCharWidth
	state.currentChar = glyph

	if len(glyph) > 0 {
		state.currentCharWidth = getCachedRuneCount(glyph[0])
	} else {
		state.currentCharWidth = 0
	}

	smushAmt := state.smushAmount()
	if smushAmt < 0 {
		smushAmt = 0
	}

	if state.outlineLen+state.currentCharWidth-smushAmt > state.outlineLenLimit {
		return false
	}

	// Get pooled buffer for rune conversion
	runeBuffer := acquireRuneSlice()
	defer releaseRuneSlice(runeBuffer)

	// Get temp buffer from pool for RTL processing
	var tempLine []rune
	if state.right2left != 0 {
		tempLine = acquireTempLine()
		defer releaseTempLine(tempLine)
	}

	for row := 0; row < state.charHeight; row++ {
		runeBuffer = runeBufferFromString(runeBuffer, glyph[row])
		rowRunes := runeBuffer

		if state.right2left != 0 {
			state.addCharRowRTL(row, rowRunes, tempLine, smushAmt)
		} else {
			state.addCharRowLTR(row, rowRunes, smushAmt)
		}
	}

	state.outlineLen = state.rowLengths[0]
	return true
}

// addCharRowRTL processes a single row for right-to-left character addition.
func (state *renderState) addCharRowRTL(row int, rowRunes, tempLine []rune, smushAmt int) {
	end := state.rowLengths[row]
	copy(tempLine, rowRunes)

	// Apply smushing at overlap positions
	for k := 0; k < smushAmt; k++ {
		column := state.currentCharWidth - smushAmt + k
		var existing rune
		if k < end {
			existing = state.outputLine[row][k]
		}

		if k < len(rowRunes) && column >= 0 && column < len(tempLine) {
			left := tempLine[column]
			smushResult := state.smush(left, existing)
			if smushResult != 0 {
				state.emitSmushDecision(row, column, left, existing, smushResult)
				tempLine[column] = smushResult
			} else {
				tempLine[column] = 0 // Mark for truncation
			}
		}
	}

	// Build final output: find content end, then append remaining output
	tempEnd := 0
	for i := 0; i < state.currentCharWidth; i++ {
		if tempLine[i] != 0 {
			tempEnd = i + 1
		} else {
			break
		}
	}

	appendStart := tempEnd
	if smushAmt < end && tempEnd == state.currentCharWidth {
		for i := smushAmt; i < end; i++ {
			tempLine[tempEnd] = state.outputLine[row][i]
			tempEnd++
		}
		state.emitRowAppend(row, appendStart, end-smushAmt, appendStart, tempEnd)
	}

	copy(state.outputLine[row][:tempEnd], tempLine[:tempEnd])
	state.rowLengths[row] = tempEnd
}

// addCharRowLTR processes a single row for left-to-right character addition.
func (state *renderState) addCharRowLTR(row int, rowRunes []rune, smushAmt int) {
	end := state.rowLengths[row]

	for k := 0; k < smushAmt; k++ {
		column := state.outlineLen - smushAmt + k
		if column < 0 {
			column = 0
		}

		var existing rune
		if column < end {
			existing = state.outputLine[row][column]
		}

		if k < len(rowRunes) {
			smushResult := state.smush(existing, rowRunes[k])
			if smushResult != 0 {
				state.outputLine[row][column] = smushResult
				if column >= end {
					end = column + 1
				}
				state.emitSmushDecision(row, column, existing, rowRunes[k], smushResult)
			} else if column < end {
				end = column
			}
		}
	}

	if smushAmt < len(rowRunes) {
		remaining := rowRunes[smushAmt:]
		startPos := end
		copy(state.outputLine[row][end:], remaining)
		end += len(remaining)
		state.emitRowAppend(row, startPos, len(remaining), startPos, end)
	}

	state.rowLengths[row] = end
}

// emitSmushDecision emits a debug event for a smushing decision.
func (state *renderState) emitSmushDecision(row, col int, lch, rch, result rune) {
	if state.debug == nil {
		return
	}
	state.debug.Emit("render", "SmushDecision", debug.SmushDecisionData{
		Row:    row,
		Col:    col,
		Lch:    lch,
		Rch:    rch,
		Result: result,
		Rule:   debug.ClassifySmushRule(lch, rch, result, state.smushMode),
	})
}

// emitRowAppend emits a debug event for a row append operation.
func (state *renderState) emitRowAppend(row, startPos, charCount, endBefore, endAfter int) {
	if state.debug == nil {
		return
	}
	state.debug.Emit("render", "RowAppend", debug.RowAppendData{
		Row:       row,
		StartPos:  startPos,
		CharCount: charCount,
		EndBefore: endBefore,
		EndAfter:  endAfter,
	})
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
		var err error
		buf, err = state.writeRow(w, buf, line[:state.rowLengths[i]])
		if err != nil {
			return err
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

// writeRow writes a single row of output to w, replacing hardblanks and optionally trimming.
// It reuses buf for UTF-8 encoding and returns the (possibly reallocated) buffer.
func (state *renderState) writeRow(w io.Writer, buf []byte, actualLine []rune) ([]byte, error) {
	lastNonSpace := len(actualLine) - 1
	if state.trimWhitespace {
		for lastNonSpace >= 0 && actualLine[lastNonSpace] == ' ' {
			lastNonSpace--
		}
	}

	buf = buf[:0]
	for j := 0; j <= lastNonSpace; j++ {
		r := actualLine[j]
		if r == state.hardblank {
			r = ' '
		}
		buf = utf8.AppendRune(buf, r)

		if len(buf) > 250 {
			if _, err := w.Write(buf); err != nil {
				return buf, err
			}
			buf = buf[:0]
		}
	}

	if len(buf) > 0 {
		if _, err := w.Write(buf); err != nil {
			return buf, err
		}
	}

	return buf, nil
}

// outputToString converts the output lines to a final string.
//
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
	// strings.Builder's Write never returns an error, but check to satisfy linter
	if err := state.writeTo(&sb); err != nil {
		// This should never happen with strings.Builder, but handle gracefully
		return ""
	}
	return sb.String()
}

// flushLine writes the current output line to the buffer and resets for the next line.
// This is called when a line is complete and needs to be output.
func (state *renderState) flushLine() {
	if state.charHeight == 0 || state.outlineLen == 0 {
		return
	}

	// Capture row lengths before flush (capped at 32)
	var rowLengthsBefore []int
	if state.debug != nil {
		limit := state.charHeight
		if limit > 32 {
			limit = 32
		}
		rowLengthsBefore = make([]int, limit)
		copy(rowLengthsBefore, state.rowLengths[:limit])
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

	// Capture row lengths after reset (all zeros)
	var rowLengthsAfter []int
	if state.debug != nil {
		limit := state.charHeight
		if limit > 32 {
			limit = 32
		}
		rowLengthsAfter = make([]int, limit)
		// After reset, all row lengths will be 0
		// rowLengthsAfter is already all zeros from make()

		// Emit Flush event
		state.debug.Emit("render", "Flush", debug.FlushData{
			LineNumber:       -1, // TODO: track line number if needed
			RowLengthsBefore: rowLengthsBefore,
			RowLengthsAfter:  rowLengthsAfter,
		})
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

		// Track when processing a space character - spaces should not
		// overlap with adjacent characters in smushing mode, but we
		// preserve the font's original space glyph width (not 1-column)
		state.processingSpaceGlyph = (r == ' ')

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
		return false // No spaces found
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

	// Emit Split event for wordbreak
	if state.debug != nil {
		state.debug.Emit("render", "Split", debug.SplitData{
			Reason:     "wordbreak",
			FSMPrev:    state.wordbreakmode,
			FSMNext:    0,
			OutlineLen: state.outlineLen,
			Position:   lastSpaceStart,
		})
	}

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
