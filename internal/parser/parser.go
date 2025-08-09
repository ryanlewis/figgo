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
	font, err := ParseHeader(r)
	if err != nil {
		return nil, err
	}

	// TODO: Parse character glyphs
	font.Characters = make(map[rune][]string)

	return font, nil
}

// ParseHeader parses the FIGfont header and comment lines.
// It reads the signature line, validates all required fields, and reads
// the specified number of comment lines.
func ParseHeader(r io.Reader) (*Font, error) {
	scanner := bufio.NewScanner(r)

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

	// Parse numeric fields
	fields := strings.Fields(headerLine[minSignatureLen:])
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

	if headerLine[5] == ' ' {
		return fmt.Errorf("invalid signature: missing hardblank character")
	}

	font.Signature = headerLine[:5]
	font.Hardblank = rune(headerLine[5])
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
