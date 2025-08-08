// Package parser implements FIGfont (FLF 2.0) parsing.
package parser

import (
	"fmt"
	"io"
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
	_ = r // TODO: Implement FIGfont parsing
	return nil, fmt.Errorf("parser not yet implemented")
}
