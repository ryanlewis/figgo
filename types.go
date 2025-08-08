package figgo

import "errors"

// Layout represents horizontal fitting and smushing rules as a bitmask.
// Bits 6-7 control fitting mode, bits 0-5 control smushing rules.
type Layout int

// Fitting modes (bits 6-7) - exactly one must be set
const (
	// FitFullWidth displays characters at full width with no fitting
	FitFullWidth Layout = 0

	// FitKerning moves characters closer until they touch (no overlap)
	FitKerning Layout = 64 // bit 6

	// FitSmushing allows characters to overlap according to smushing rules
	FitSmushing Layout = 128 // bit 7
)

// Smushing rules (bits 0-5) - only apply when FitSmushing is active
const (
	// RuleEqualChar: Two equal characters can be smushed into one
	RuleEqualChar Layout = 1 // bit 0

	// RuleUnderscore: Underscore can be replaced by certain characters
	RuleUnderscore Layout = 2 // bit 1

	// RuleHierarchy: Characters follow a hierarchy for smushing
	RuleHierarchy Layout = 4 // bit 2

	// RuleOppositePair: Opposite brackets/braces can combine
	RuleOppositePair Layout = 8 // bit 3

	// RuleBigX: Slashes can form big X shapes
	RuleBigX Layout = 16 // bit 4

	// RuleHardblank: Hardblank can smush with other characters
	RuleHardblank Layout = 32 // bit 5
)

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

	// ErrLayoutConflict is returned when both kerning and smushing are enabled
	ErrLayoutConflict = errors.New("layout conflict: cannot enable both kerning and smushing")
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
