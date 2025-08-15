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

// WithLayout sets the layout mode for rendering, overriding the font's default.
//
// Layout Override Behavior:
// - Completely replaces the font's built-in layout settings
// - Can change fitting mode (full-width, kerning, smushing)
// - Can enable/disable specific smushing rules
// - Layout validation occurs during rendering (will return error if invalid)
//
// Common Usage:
//   - WithLayout(FitFullWidth): Force full-width spacing
//   - WithLayout(FitKerning): Use minimal spacing without overlap
//   - WithLayout(FitSmushing | RuleEqualChar): Enable equal character smushing only
//
// The layout will be normalized and validated when rendering begins.
func WithLayout(layout Layout) Option {
	return func(opts *options) {
		opts.layout = &layout
	}
}

// WithPrintDirection sets the print direction, overriding the font's default.
//
// Direction Values:
//   - 0: Left-to-right (LTR) - normal reading direction
//   - 1: Right-to-left (RTL) - characters added in reverse order
//
// RTL Behavior:
// - Characters are processed in input order but assembled right-to-left
// - Affects smushing calculations and character positioning
// - Useful for Arabic-style layouts or special visual effects
// - Does not reverse the input string, only the rendering direction
//
// Note: Most fonts are designed for LTR rendering. RTL may produce
// unexpected results with some font designs.
func WithPrintDirection(direction int) Option {
	return func(opts *options) {
		opts.printDirection = &direction
	}
}

// WithUnknownRune sets the rune used to replace unknown/unsupported runes during rendering.
// Default is '?' when not set.
//
// Error Handling Strategy:
// - Without this option: rendering fails with ErrUnsupportedRune
// - With this option: unknown runes are replaced with the specified rune
// - The replacement rune must exist in the font, or rendering will still fail
//
// Common Usage:
//   - WithUnknownRune('?'): Replace with question mark (if font supports it)
//   - WithUnknownRune(' '): Replace with space (creates gaps)
//   - WithUnknownRune('*'): Replace with asterisk (visible placeholder)
//
// This is particularly useful when rendering user input that may contain
// characters not supported by the chosen font.
func WithUnknownRune(r rune) Option {
	return func(opts *options) {
		opts.unknownRune = &r
	}
}

// WithTrimWhitespace enables trimming of trailing whitespace from each line.
// By default, figgo preserves trailing spaces to match figlet's behavior.
//
// Whitespace Handling:
//   - true: Removes trailing spaces from each output line
//   - false: Preserves all trailing spaces (maintains rectangular output)
//
// Considerations:
// - FIGlet traditionally preserves trailing spaces for consistent formatting
// - Trimming can reduce output size and eliminate unwanted trailing spaces
// - May affect visual alignment in multi-line layouts
// - Applies to each row independently during output generation
//
// Use Cases:
// - Enable for web output where trailing spaces cause display issues
// - Disable for terminal output where consistent width is important
// - Enable when concatenating with other text to avoid spacing issues
func WithTrimWhitespace(trim bool) Option {
	return func(opts *options) {
		opts.trimWhitespace = trim
	}
}

// WithWidth sets the maximum output width in characters.
// Lines longer than this width will be wrapped at word boundaries when possible.
// The default width is 80 characters (standard terminal width).
// Valid range is 1-1000. Values outside this range will be clamped.
//
// Width Behavior:
//   - 0 or negative: Uses default of 80 characters
//   - 1-1000: Sets maximum line width
//   - >1000: Clamped to 1000 (prevents excessive memory usage)
//
// Line Breaking:
// - Attempts to break at word boundaries (spaces) when possible
// - Forces character breaks when words exceed the width limit
// - Maintains proper FIGlet character alignment across wrapped lines
//
// Compatibility:
// - Matches FIGlet's -w flag behavior
// - Default of 80 matches traditional terminal width
// - Maximum of 1000 supports 4K displays with small fonts
//
// Example:
//
//	// Wide output for modern terminals
//	figgo.Render("Long text", font, figgo.WithWidth(120))
//
//	// Narrow output for constrained displays
//	figgo.Render("Text", font, figgo.WithWidth(40))
func WithWidth(width int) Option {
	return func(opts *options) {
		// Clamp to valid range
		if width <= 0 {
			width = 80 // Default
		} else if width > 1000 {
			width = 1000 // Maximum
		}
		opts.width = &width
	}
}
