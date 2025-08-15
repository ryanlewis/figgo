package renderer

import "errors"

// Error definitions for the renderer package
var (
	// ErrNilFont is returned when a nil font is provided to Render
	ErrNilFont = errors.New("font cannot be nil")
	// ErrUnsupportedRune is returned when a character is not found in the font
	ErrUnsupportedRune = errors.New("unsupported rune")
	// ErrInvalidGlyphHeight is returned when a glyph has incorrect height
	ErrInvalidGlyphHeight = errors.New("invalid glyph height")
)

// Smushing mode constants
const (
	// SMSmush indicates smushing mode is active
	SMSmush = 128
	// SMKern indicates kerning mode is active
	SMKern = 64

	// Smushing rules (bits 0-5)
	SMEqual     = 1  // Equal character rule
	SMLowline   = 2  // Underscore rule
	SMHierarchy = 4  // Hierarchy rule
	SMPair      = 8  // Opposite pair rule
	SMBigX      = 16 // Big X rule
	SMHardblank = 32 // Hardblank rule
)

// Options contains rendering options passed from the main package
type Options struct {
	// Layout is the layout bitmask from figgo.Layout
	Layout int
	// PrintDirection specifies direction (0=LTR, 1=RTL)
	PrintDirection *int
	// UnknownRune is the rune to use for unknown characters
	UnknownRune *rune
	// TrimWhitespace removes trailing spaces from each line
	TrimWhitespace bool
	// Width is the maximum output width in characters (default 80)
	Width *int
}

// renderState holds the current rendering state.
// Fields are ordered for optimal memory alignment (largest to smallest)
type renderState struct {
	// Slice fields (24 bytes each on 64-bit)
	outputLine  [][]rune // Current output line being built (one per font height)
	rowLengths  []int    // Length of each row
	currentChar []string // Current character being processed
	inputBuffer []rune   // Buffer holding input characters for current line

	// String builder for accumulated output
	outputBuffer []byte // Accumulated output from completed lines

	// int fields (8 bytes each on 64-bit)
	outlineLen        int // Length of current output line
	outlineLenLimit   int // Maximum line length allowed
	currentCharWidth  int // Width of current character
	previousCharWidth int // Width of previous character
	charHeight        int // Character height from font
	right2left        int // Print direction (0=LTR, 1=RTL)
	smushMode         int // Smushing mode calculated from layout
	inputCount        int // Count of characters in input buffer
	lastWordBreak     int // Position of last space/word boundary in inputBuffer
	wordbreakmode     int // State machine for line breaking

	// rune field (4 bytes)
	hardblank rune // Hardblank character from font

	// bool field (1 byte)
	trimWhitespace bool // Whether to trim trailing whitespace
}
