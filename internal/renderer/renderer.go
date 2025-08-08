// Package renderer implements ASCII art rendering from parsed FIGfonts.
package renderer

import (
	"fmt"

	"github.com/ryanlewis/figgo/internal/parser"
)

// Options configures rendering behavior.
type Options struct {
	// Layout controls the fitting/smushing rules
	Layout int

	// PrintDirection overrides the font's default print direction
	PrintDirection int
}

// Render converts text to ASCII art using the specified font and options.
func Render(text string, font *parser.Font, opts *Options) (string, error) {
	_ = text // TODO: Implement rendering logic
	_ = font
	_ = opts
	return "", fmt.Errorf("renderer not yet implemented")
}
