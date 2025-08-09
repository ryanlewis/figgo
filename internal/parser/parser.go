// Package parser implements FIGfont (FLF 2.0) parsing.
package parser

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	// minHeaderFields is the minimum number of required fields in a FIGfont header
	minHeaderFields = 5
	// minSignatureRunes is the minimum number of runes in a valid signature (handles UTF-8)
	minSignatureRunes = 6
	// firstNonSpaceASCII is the first non-space printable ASCII character (!)
	firstNonSpaceASCII = 33
	// lastPrintableASCII is the last printable ASCII character (~)
	lastPrintableASCII = 126

	// Buffer size constants
	defaultBufferSize = 64 * 1024
	maxBufferSize     = 4 * 1024 * 1024

	// ASCII threshold for fast-path optimization
	asciiThreshold = 0x80
)

// GlyphTrim contains precomputed trim information for a glyph row
type GlyphTrim struct {
	LeftmostVisible  int // Index of leftmost non-space character (-1 if all spaces)
	RightmostVisible int // Index of rightmost non-space character (-1 if all spaces)
}

// computeGlyphTrims precomputes the trim information for a glyph
// Note: Only ASCII space ' ' is considered blank. Hardblanks are treated as visible.
func computeGlyphTrims(glyph []string) []GlyphTrim {
	trims := make([]GlyphTrim, len(glyph))
	for i, row := range glyph {
		leftmost := -1
		rightmost := -1

		// Find leftmost visible
		for j := 0; j < len(row); j++ {
			if row[j] != ' ' {
				leftmost = j
				break
			}
		}

		// Find rightmost visible
		for j := len(row) - 1; j >= 0; j-- {
			if row[j] != ' ' {
				rightmost = j
				break
			}
		}

		trims[i] = GlyphTrim{
			LeftmostVisible:  leftmost,
			RightmostVisible: rightmost,
		}
	}
	return trims
}

// Font represents a parsed FIGfont with all its metadata and character glyphs.
type Font struct {
	// Characters maps ASCII codes to their glyph representations
	Characters map[rune][]string

	// CharacterTrims maps ASCII characters to their precomputed trim data per row
	// This is computed during parsing for performance
	CharacterTrims map[rune][]GlyphTrim

	// Comments contains the font comments
	Comments []string

	// Signature contains the FIGfont signature (e.g., "flf2a")
	Signature string

	// Hardblank is the character used for hard blanks
	Hardblank rune

	// Height is the number of lines per character
	Height int

	// Baseline is the number of lines from the top to the baseline
	Baseline int

	// MaxLength is the maximum character width
	MaxLength int

	// OldLayout is the old layout value for backward compatibility
	OldLayout int

	// CommentLines is the number of comment lines after the header
	CommentLines int

	// PrintDirection specifies the print direction (0=LTR, 1=RTL)
	PrintDirection int

	// FullLayout contains the full layout value
	FullLayout int

	// FullLayoutSet indicates whether FullLayout was present in the header
	FullLayoutSet bool

	// CodetagCount specifies the number of code-tagged characters
	CodetagCount int

	// Warnings contains any non-fatal issues encountered during parsing
	Warnings []string
}

// Parse reads a FIGfont from the provided reader and returns a parsed Font.
func Parse(r io.Reader) (*Font, error) {
	scanner := bufio.NewScanner(r)
	// Increase buffer size for large fonts (default is 64KB, set max to 4MB)
	scanner.Buffer(make([]byte, 0, defaultBufferSize), maxBufferSize)

	// Parse header and comments first
	font, err := parseHeaderWithScanner(scanner)
	if err != nil {
		return nil, err
	}

	// Parse character glyphs
	if err := parseGlyphs(scanner, font); err != nil {
		return nil, err
	}

	return font, nil
}

// ParseHeader parses the FIGfont header and comment lines.
// It reads the signature line, validates all required fields, and reads
// the specified number of comment lines.
func ParseHeader(r io.Reader) (*Font, error) {
	scanner := bufio.NewScanner(r)
	// Increase buffer size for large fonts
	scanner.Buffer(make([]byte, 0, defaultBufferSize), maxBufferSize)
	return parseHeaderWithScanner(scanner)
}

// parseHeaderWithScanner parses the header using an existing scanner
func parseHeaderWithScanner(scanner *bufio.Scanner) (*Font, error) {
	// Read and validate header line
	headerLine, err := readHeaderLine(scanner)
	if err != nil {
		return nil, err
	}

	// Parse header into font structure
	font := &Font{}

	// Extract signature and hardblank
	if err := parseSignature(headerLine, font); err != nil {
		return nil, err
	}

	// Parse numeric fields - need to skip past the hardblank character properly
	// Convert to runes to handle multi-byte characters correctly
	runes := []rune(headerLine)
	if len(runes) < minSignatureRunes {
		return nil, fmt.Errorf("header line too short")
	}
	// Skip "flf2a" (5 runes) and hardblank (1 rune) = 6 runes total
	remainingHeader := string(runes[minSignatureRunes:])
	fields := strings.Fields(remainingHeader)
	if len(fields) < minHeaderFields {
		return nil, fmt.Errorf("insufficient header fields: got %d, need at least %d", len(fields), minHeaderFields)
	}

	// Parse required fields
	if err := parseRequiredFields(fields, font); err != nil {
		return nil, err
	}

	// Parse optional fields
	if err := parseOptionalFields(fields, font); err != nil {
		return nil, err
	}

	// Read comment lines
	if err := readCommentLines(scanner, font); err != nil {
		return nil, err
	}

	// Initialize Characters map with capacity for ASCII (95) + German (7) + some extras
	font.Characters = make(map[rune][]string, 128)
	font.CharacterTrims = make(map[rune][]GlyphTrim, 128)

	return font, nil
}

// readHeaderLine reads the first non-empty line from the scanner
func readHeaderLine(scanner *bufio.Scanner) (string, error) {
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("error reading header: %w", err)
		}
		return "", fmt.Errorf("empty font data")
	}

	headerLine := strings.TrimSpace(scanner.Text())

	// Strip UTF-8 BOM from the first line if present
	// Some .flf files in the wild carry a UTF-8 BOM (U+FEFF)
	const utf8BOM = "\uFEFF"
	headerLine = strings.TrimPrefix(headerLine, utf8BOM)
	if headerLine == "" {
		// Try to find non-empty line
		for scanner.Scan() {
			headerLine = strings.TrimSpace(scanner.Text())
			if headerLine != "" {
				break
			}
		}
		if headerLine == "" {
			return "", fmt.Errorf("empty font data")
		}
	}

	return headerLine, nil
}

// parseSignature validates and extracts the signature and hardblank
func parseSignature(headerLine string, font *Font) error {
	// Get the hardblank as a rune (handles multi-byte UTF-8)
	runes := []rune(headerLine)
	if len(runes) < minSignatureRunes {
		return fmt.Errorf("invalid signature: too short")
	}

	// Spec says the signature must be exactly "flf2a" (5th char is 'a' and cannot be omitted)
	signature := string(runes[:5])
	if signature != "flf2a" {
		return fmt.Errorf("invalid signature: expected 'flf2a', got %q", signature)
	}

	hardblank := runes[5]
	// Spec forbids hardblank being space/CR/LF/NUL
	if hardblank == ' ' || hardblank == '\r' || hardblank == '\n' || hardblank == '\x00' {
		return fmt.Errorf("invalid hardblank character: cannot be space, CR, LF, or NUL")
	}

	font.Signature = signature
	font.Hardblank = hardblank
	return nil
}

// parseRequiredFields parses the five required header fields
func parseRequiredFields(fields []string, font *Font) error {
	// Parse Height
	height, err := strconv.Atoi(fields[0])
	if err != nil {
		return fmt.Errorf("invalid height: %w", err)
	}
	if height <= 0 {
		return fmt.Errorf("height must be positive, got %d", height)
	}
	font.Height = height

	// Parse Baseline
	baseline, err := strconv.Atoi(fields[1])
	if err != nil {
		return fmt.Errorf("invalid baseline: %w", err)
	}
	if baseline < 1 {
		return fmt.Errorf("baseline must be at least 1, got %d", baseline)
	}
	if baseline > height {
		return fmt.Errorf("baseline exceeds height: %d > %d", baseline, height)
	}
	font.Baseline = baseline

	// Parse MaxLength
	maxLength, err := strconv.Atoi(fields[2])
	if err != nil {
		return fmt.Errorf("invalid maxlength: %w", err)
	}
	if maxLength <= 0 {
		return fmt.Errorf("maxlength must be positive, got %d", maxLength)
	}
	font.MaxLength = maxLength

	// Parse OldLayout
	oldLayout, err := strconv.Atoi(fields[3])
	if err != nil {
		return fmt.Errorf("invalid old layout: %w", err)
	}
	font.OldLayout = oldLayout

	// Parse CommentLines
	commentLines, err := strconv.Atoi(fields[4])
	if err != nil {
		return fmt.Errorf("invalid comment lines: %w", err)
	}
	if commentLines < 0 {
		return fmt.Errorf("comment lines must be non-negative, got %d", commentLines)
	}
	font.CommentLines = commentLines

	return nil
}

// parseOptionalFields parses the optional header fields if present
func parseOptionalFields(fields []string, font *Font) error {
	const (
		printDirectionField = 5
		fullLayoutField     = 6
		codetagCountField   = 7
	)

	if len(fields) > printDirectionField {
		if val, err := strconv.Atoi(fields[printDirectionField]); err == nil {
			// Validate PrintDirection (0=LTR, 1=RTL)
			if val != 0 && val != 1 {
				return fmt.Errorf("invalid print direction: %d (must be 0 or 1)", val)
			}
			font.PrintDirection = val
		}
	}

	if len(fields) > fullLayoutField {
		if val, err := strconv.Atoi(fields[fullLayoutField]); err == nil {
			font.FullLayout = val
			font.FullLayoutSet = true
		}
	}

	if len(fields) > codetagCountField {
		if val, err := strconv.Atoi(fields[codetagCountField]); err == nil {
			font.CodetagCount = val
		}
	}

	return nil
}

// readCommentLines reads the specified number of comment lines
func readCommentLines(scanner *bufio.Scanner, font *Font) error {
	font.Comments = make([]string, 0, font.CommentLines)
	for i := 0; i < font.CommentLines; i++ {
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("error reading comment line %d: %w", i+1, err)
			}
			return fmt.Errorf("unexpected EOF: expected %d comment lines, got %d", font.CommentLines, i)
		}
		// Preserve comment as-is, just trim line endings
		comment := strings.TrimRight(scanner.Text(), "\r\n")
		font.Comments = append(font.Comments, comment)
	}
	return nil
}

// parseGlyphs parses the required FIGcharacters: ASCII (32-126) and German (196,214,220,228,246,252,223)
func parseGlyphs(scanner *bufio.Scanner, font *Font) error {
	// Parse space character (ASCII 32)
	spaceGlyph, warnings, err := parseGlyph(scanner, font.Height, font.MaxLength)
	if err != nil {
		return fmt.Errorf("error parsing glyph for character 32 (space): %w", err)
	}
	font.Characters[' '] = spaceGlyph
	font.CharacterTrims[' '] = computeGlyphTrims(spaceGlyph)
	font.Warnings = append(font.Warnings, warnings...)

	// Parse remaining ASCII characters (33-126)
	for charCode := rune(firstNonSpaceASCII); charCode <= lastPrintableASCII; charCode++ {
		glyph, warnings, err := parseGlyph(scanner, font.Height, font.MaxLength)
		if err != nil {
			// Check if it's EOF - if so, we're done (partial font is OK)
			if errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return fmt.Errorf("error parsing glyph for character %d (%c): %w", charCode, charCode, err)
		}
		font.Characters[charCode] = glyph
		font.CharacterTrims[charCode] = computeGlyphTrims(glyph)
		font.Warnings = append(font.Warnings, warnings...)
	}

	// Parse the 7 required German/Deutsch characters in order
	// Per FIGfont spec: 196 (Ä), 214 (Ö), 220 (Ü), 228 (ä), 246 (ö), 252 (ü), 223 (ß)
	deutschChars := []rune{196, 214, 220, 228, 246, 252, 223}
	for _, charCode := range deutschChars {
		glyph, warnings, err := parseGlyph(scanner, font.Height, font.MaxLength)
		if err != nil {
			// Check if it's EOF - German chars are optional for backward compatibility
			if errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return fmt.Errorf("error parsing glyph for German character %d: %w", charCode, err)
		}
		font.Characters[charCode] = glyph
		font.CharacterTrims[charCode] = computeGlyphTrims(glyph)
		font.Warnings = append(font.Warnings, warnings...)
	}

	return nil
}

// stripTrailingRun strips the trailing run of the last character from a line
// Returns the body (without trailing run), the endmark character, and the run length
//
// This implementation is intentionally permissive to handle real-world fonts:
// - Strips any trailing run length (not just single/double as per spec)
// - Doesn't enforce "last line has double endmark" convention
// - Handles multi-byte UTF-8 endmarks correctly
// This approach prioritizes compatibility over strict spec compliance.
func stripTrailingRun(line string) (body string, endmark rune, runLen int) {
	// Trim only CR; bufio.Scanner already strips LF
	// We special-case \r because Windows line endings (\r\n) leave a trailing \r
	// after Scanner removes the \n, and we need to strip it before processing endmarks
	line = strings.TrimSuffix(line, "\r")

	if line == "" {
		return "", 0, 0
	}

	// Fast-path for ASCII endmarks (common case)
	lastByte := line[len(line)-1]
	if lastByte < asciiThreshold {
		// ASCII character - do byte-wise operations for speed
		i := len(line) - 1
		for i >= 0 && line[i] == lastByte {
			i--
			runLen++
		}
		return line[:i+1], rune(lastByte), runLen
	}

	// Multi-byte UTF-8 or invalid UTF-8 path
	r, sz := utf8.DecodeLastRuneInString(line)
	if r == utf8.RuneError && sz == 1 {
		// Fallback: treat last byte as the endmark and strip the trailing run of that byte
		// This handles invalid UTF-8 at line end gracefully
		i := len(line) - 1
		for i >= 0 && line[i] == lastByte {
			i--
			runLen++
		}
		return line[:i+1], rune(lastByte), runLen
	}

	// Normal rune-aware path: count how many times this rune appears at the end
	i := len(line)
	for i > 0 {
		rr, s := utf8.DecodeLastRuneInString(line[:i])
		if rr != r {
			break
		}
		i -= s
		runLen++
	}

	return line[:i], r, runLen
}

// parseGlyph parses a single character glyph
func parseGlyph(scanner *bufio.Scanner, height, maxLength int) (glyph, warnings []string, err error) {
	glyph = make([]string, 0, height)
	width := -1 // -1 indicates width not yet set

	for row := 0; row < height; row++ {
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return nil, warnings, fmt.Errorf("error reading line %d: %w", row+1, err)
			}
			return nil, warnings, fmt.Errorf(
				"unexpected EOF: expected %d lines, got %d: %w",
				height, row, io.ErrUnexpectedEOF)
		}

		// Get the raw line
		rawLine := scanner.Text()

		// Strip the trailing run of the endmark character
		body, _, _ := stripTrailingRun(rawLine)

		// Validate MaxLength after stripping endmarks (advisory check)
		// The spec says MaxLength is "usually width + 2" for endmarks,
		// but many real fonts don't follow this strictly
		bodyWidth := utf8.RuneCountInString(body)
		if bodyWidth > maxLength {
			// Advisory warning only - many real fonts exceed their stated MaxLength
			warnings = append(warnings,
				fmt.Sprintf("line %d width (%d) exceeds MaxLength (%d)",
					row+1, bodyWidth, maxLength))
		}

		// Check width consistency
		w := utf8.RuneCountInString(body)
		if width == -1 {
			// First row: set the expected width
			width = w
		} else if w != width {
			// Note: We compare rune counts for width consistency, not visual width.
			// This means combining marks are counted as separate runes, which may
			// differ from visual width. FIGfonts overwhelmingly use ASCII art, so
			// this limitation rarely matters in practice.
			return nil, warnings, fmt.Errorf(
				"inconsistent row width in glyph: row %d has %d, expected %d",
				row+1, w, width)
		}

		glyph = append(glyph, body)
	}

	return glyph, warnings, nil
}
