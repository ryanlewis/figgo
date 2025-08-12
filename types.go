package figgo

import "errors"

// Font represents an immutable FIGfont that can be safely shared across goroutines.
//
// Font data is loaded once and never modified, making it safe for concurrent use
// without locking.
type Font struct {
	// glyphs maps runes to their multi-line ASCII art representations (unexported for immutability)
	glyphs map[rune][]string

	// Name is the font name (e.g., "standard")
	Name string

	// Layout contains the normalized horizontal layout bitmask combining fitting mode and smushing rules.
	// This is the processed layout derived from the font header's OldLayout/FullLayout values.
	// See NormalizeLayoutFromHeader for the normalization process.
	// Only horizontal layout is currently used by the renderer.
	Layout Layout

	// Hardblank is the character used for hard blanks in the font
	Hardblank rune

	// Height is the number of lines per character
	Height int

	// Baseline is the number of lines from the top to the baseline
	Baseline int

	// MaxLen is the maximum character width
	MaxLen int

	// OldLayout is the old layout value for backward compatibility (-1 if not present)
	OldLayout int

	// PrintDirection specifies the print direction (0=LTR, 1=RTL)
	PrintDirection int

	// CommentLines is the number of comment lines in the font file
	CommentLines int
}

// Glyph returns the ASCII art representation for a rune, or false if not found.
// The returned slice should not be modified by the caller.
func (f *Font) Glyph(r rune) ([]string, bool) {
	if f == nil || f.glyphs == nil {
		return nil, false
	}
	glyph, ok := f.glyphs[r]
	return glyph, ok
}

// Common errors returned by the figgo package
var (
	// ErrUnknownFont is returned when a requested font cannot be found
	ErrUnknownFont = errors.New("unknown font")

	// ErrUnsupportedRune is returned when a rune is not supported by the font
	ErrUnsupportedRune = errors.New("unsupported rune")

	// ErrBadFontFormat is returned when a font file has an invalid format
	ErrBadFontFormat = errors.New("bad font format")
)

// WithLayout sets the layout mode for rendering, overriding the font's default
func WithLayout(layout Layout) Option {
	return func(opts *options) {
		opts.layout = &layout
	}
}

// WithPrintDirection sets the print direction (0=LTR, 1=RTL)
func WithPrintDirection(direction int) Option {
	return func(opts *options) {
		opts.printDirection = &direction
	}
}

// WithUnknownRune sets the rune used to replace unknown/unsupported runes during rendering.
// Default is '?' when not set.
func WithUnknownRune(r rune) Option {
	return func(opts *options) {
		opts.unknownRune = &r
	}
}

// WithTrimWhitespace enables trimming of trailing whitespace from each line.
// By default, figgo preserves trailing spaces to match figlet's behavior.
func WithTrimWhitespace(trim bool) Option {
	return func(opts *options) {
		opts.trimWhitespace = trim
	}
}
