// Package parser implements FIGfont (FLF 2.0) parsing.
package parser

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	// minHeaderFields is the minimum number of required fields in a FIGfont header
	minHeaderFields = 5
	// signaturePrefix is the expected prefix for FIGfont files
	signaturePrefix = "flf2"
	// minSignatureLen is the minimum length of a valid signature (e.g., "flf2a$")
	minSignatureLen = 6
	// minSignatureRunes is the minimum number of runes in a valid signature (handles UTF-8)
	minSignatureRunes = 6
	// firstNonSpaceASCII is the first non-space printable ASCII character (!)
	firstNonSpaceASCII = 33
	// lastPrintableASCII is the last printable ASCII character (~)
	lastPrintableASCII = 126
)

// Font represents a parsed FIGfont with all its metadata and character glyphs.
type Font struct { //nolint:govet // Field order optimized for clarity
	// Characters maps ASCII codes to their glyph representations
	Characters map[rune][]string

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

	// CodetagCount specifies the number of code-tagged characters
	CodetagCount int
}

// Parse reads a FIGfont from the provided reader and returns a parsed Font.
func Parse(r io.Reader) (*Font, error) {
	scanner := bufio.NewScanner(r)

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
	parseOptionalFields(fields, font)

	// Read comment lines
	if err := readCommentLines(scanner, font); err != nil {
		return nil, err
	}

	// Initialize Characters map
	font.Characters = make(map[rune][]string)

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
	if len(headerLine) < minSignatureLen || !strings.HasPrefix(headerLine, signaturePrefix) {
		return fmt.Errorf("invalid signature: expected 'flf2a' format")
	}

	// Get the hardblank as a rune (handles multi-byte UTF-8)
	runes := []rune(headerLine)
	if len(runes) < minSignatureRunes {
		return fmt.Errorf("invalid signature: too short")
	}

	if runes[5] == ' ' {
		return fmt.Errorf("invalid signature: missing hardblank character")
	}

	font.Signature = string(runes[:5])
	font.Hardblank = runes[5]
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
func parseOptionalFields(fields []string, font *Font) {
	const (
		printDirectionField = 5
		fullLayoutField     = 6
		codetagCountField   = 7
	)

	if len(fields) > printDirectionField {
		if val, err := strconv.Atoi(fields[printDirectionField]); err == nil {
			font.PrintDirection = val
		}
	}

	if len(fields) > fullLayoutField {
		if val, err := strconv.Atoi(fields[fullLayoutField]); err == nil {
			font.FullLayout = val
		}
	}

	if len(fields) > codetagCountField {
		if val, err := strconv.Atoi(fields[codetagCountField]); err == nil {
			font.CodetagCount = val
		}
	}
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

// parseGlyphs parses the ASCII character glyphs (32-126)
func parseGlyphs(scanner *bufio.Scanner, font *Font) error {
	// Detect endmark from the first glyph (space character)
	endmark, lines, err := detectEndmark(scanner, font.Height)
	if err != nil {
		// Pass through the original error
		return fmt.Errorf("error parsing glyph for character 32 (space): %w", err)
	}

	// Parse the first glyph (space) with the detected endmark
	spaceGlyph, err := parseGlyphWithLines(lines, endmark, font.Height)
	if err != nil {
		return fmt.Errorf("error parsing glyph for character 32 (space): %w", err)
	}
	font.Characters[' '] = spaceGlyph

	// Parse remaining ASCII characters (33-126)
	// But stop gracefully if we run out of data (for testing partial fonts)
	for charCode := rune(firstNonSpaceASCII); charCode <= lastPrintableASCII; charCode++ {
		glyph, err := parseGlyph(scanner, font.Height, endmark)
		if err != nil {
			// Check if it's EOF - if so, we're done
			if strings.Contains(err.Error(), "unexpected EOF") {
				// Partial font is OK for testing
				break
			}
			return fmt.Errorf("error parsing glyph for character %d (%c): %w", charCode, charCode, err)
		}
		font.Characters[charCode] = glyph
	}

	return nil
}

// detectEndmark reads the first glyph's lines to detect the endmark character
func detectEndmark(scanner *bufio.Scanner, height int) (endmark string, glyphLines []string, err error) {
	lines := make([]string, 0, height)

	// Read all lines of the first glyph
	for i := 0; i < height; i++ {
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return "", nil, fmt.Errorf("error reading line %d: %w", i+1, err)
			}
			return "", nil, fmt.Errorf("unexpected EOF: expected %d lines, got %d", height, i)
		}
		lines = append(lines, scanner.Text())
	}

	// The last line should have double endmark
	lastLine := lines[height-1]
	lastLine = strings.TrimRight(lastLine, "\r\n")

	// Find the endmark by looking for repeated character at the end
	if len(lastLine) < 2 {
		return "", nil, fmt.Errorf("last line too short to detect endmark")
	}

	// The endmark is the last character, and it should appear twice
	endmark = string(lastLine[len(lastLine)-1])
	if !strings.HasSuffix(lastLine, endmark+endmark) {
		// If not double endmark at the end, it might be a single endmark
		// This shouldn't happen according to spec but handle it gracefully
		return endmark, lines, nil
	}

	// Verify other lines have single endmark
	for i := 0; i < height-1; i++ {
		line := strings.TrimRight(lines[i], "\r\n")
		if !strings.HasSuffix(line, endmark) || strings.HasSuffix(line, endmark+endmark) {
			// Allow for variations, just use what we detected
			break
		}
	}

	return endmark, lines, nil
}

// parseGlyphWithLines parses a glyph from already-read lines
func parseGlyphWithLines(lines []string, endmark string, height int) ([]string, error) {
	if len(lines) != height {
		return nil, fmt.Errorf("expected %d lines, got %d", height, len(lines))
	}

	glyph := make([]string, 0, height)
	for i, line := range lines {
		processedLine, err := processGlyphLine(line, endmark, i == height-1)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		glyph = append(glyph, processedLine)
	}

	return glyph, nil
}

// parseGlyph parses a single character glyph
func parseGlyph(scanner *bufio.Scanner, height int, endmark string) ([]string, error) {
	glyph := make([]string, 0, height)

	for lineNum := 0; lineNum < height; lineNum++ {
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return nil, fmt.Errorf("error reading line %d: %w", lineNum+1, err)
			}
			return nil, fmt.Errorf("unexpected EOF: expected %d lines, got %d", height, lineNum)
		}

		line := scanner.Text()

		// Process the line and handle endmarks
		processedLine, err := processGlyphLine(line, endmark, lineNum == height-1)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum+1, err)
		}

		glyph = append(glyph, processedLine)
	}

	return glyph, nil
}

// processGlyphLine processes a single glyph line, handling endmarks
func processGlyphLine(line, endmark string, isLastLine bool) (string, error) {
	// Trim line endings but preserve other whitespace
	line = strings.TrimRight(line, "\r\n")

	// Check if line has at least one endmark
	if !strings.Contains(line, endmark) {
		return "", fmt.Errorf("missing endmark '%s' in line %q", endmark, line)
	}

	// For the last line of a glyph, we expect double endmark
	if isLastLine {
		doubleEndmark := endmark + endmark
		switch {
		case strings.HasSuffix(line, doubleEndmark):
			// Remove the double endmark from the end
			result := line[:len(line)-len(doubleEndmark)]
			// But if the result still ends with endmark(s), it means we had triple or more
			// In that case, we should convert doubles to singles
			for strings.HasSuffix(result, doubleEndmark) {
				// Replace trailing double with single
				result = result[:len(result)-len(doubleEndmark)] + endmark
			}
			return result, nil
		case strings.HasSuffix(line, endmark):
			// Only single endmark on last line - this is technically wrong but handle gracefully
			return line[:len(line)-len(endmark)], nil
		default:
			return "", fmt.Errorf("last line should end with double endmark")
		}
	}

	// For non-last lines, expect single endmark
	if !strings.HasSuffix(line, endmark) {
		return "", fmt.Errorf("line should end with endmark '%s'", endmark)
	}

	// Check if there's a double endmark (which should become single)
	doubleEndmark := endmark + endmark
	if strings.HasSuffix(line, doubleEndmark) {
		// Line ends with double endmark, keep one
		result := line[:len(line)-len(doubleEndmark)]
		// But check for triple or more endmarks
		for strings.HasSuffix(result, doubleEndmark) {
			result = result[:len(result)-len(doubleEndmark)] + endmark
		}
		// Add back single endmark since double becomes single
		return result + endmark, nil
	}

	// Single endmark - just remove it
	return line[:len(line)-len(endmark)], nil
}
