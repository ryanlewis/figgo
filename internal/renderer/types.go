package renderer

// Smushing mode constants from figlet.c
const (
	// SM_SMUSH indicates smushing mode is active
	SM_SMUSH = 128
	// SM_KERN indicates kerning mode is active  
	SM_KERN = 64
	
	// Smushing rules (bits 0-5)
	SM_EQUAL     = 1  // Equal character rule
	SM_LOWLINE   = 2  // Underscore rule
	SM_HIERARCHY = 4  // Hierarchy rule
	SM_PAIR      = 8  // Opposite pair rule  
	SM_BIGX      = 16 // Big X rule
	SM_HARDBLANK = 32 // Hardblank rule
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
}

// renderState holds the current rendering state, similar to figlet.c globals
type renderState struct {
	// Current output line being built (one per font height)
	outputLine [][]rune
	// Length of each row (emulates C's strlen per row)
	rowLengths []int
	// Length of current output line  
	outlineLen int
	// Maximum line length allowed
	outlineLenLimit int
	// Current character being processed
	currChar []string
	// Width of current character
	currCharWidth int
	// Width of previous character
	previousCharWidth int
	// Character height from font
	charHeight int
	// Print direction (0=LTR, 1=RTL)
	right2left int
	// Smushing mode calculated from layout
	smushMode int
	// Hardblank character from font
	hardblank rune
	// Whether to trim trailing whitespace
	trimWhitespace bool
}