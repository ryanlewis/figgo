package figgo

import "errors"

// Font represents an immutable FIGfont that can be safely shared across goroutines.
//
// Font data is loaded once and never modified, making it safe for concurrent use
// without locking.
type Font struct {
	// Name is the font name (e.g., "standard")
	Name string

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

	// FullLayout contains the full layout bitmask combining fitting mode and smushing rules
	FullLayout Layout

	// PrintDirection specifies the print direction (0=LTR, 1=RTL)
	PrintDirection int

	// CommentLines is the number of comment lines in the font file
	CommentLines int

	// Glyphs maps runes to their multi-line ASCII art representations
	Glyphs map[rune][]string
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
