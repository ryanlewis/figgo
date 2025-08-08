// Package figgo provides a high-performance Go library for rendering FIGlet ASCII art.
//
// Figgo aims to be the reference Go implementation for FIGlet fonts (FLF 2.0),
// offering compatibility with the original C figlet while providing modern Go APIs
// and performance optimizations.
package figgo

import (
	"io"

	"github.com/ryanlewis/figgo/internal/parser"
	"github.com/ryanlewis/figgo/internal/renderer"
)

// Font represents an immutable FIGfont that can be safely shared across goroutines.
type Font struct {
	font *parser.Font
}

// ParseFont reads a FIGfont from the provided reader and returns a Font instance.
func ParseFont(r io.Reader) (*Font, error) {
	pf, err := parser.Parse(r)
	if err != nil {
		return nil, err
	}
	return &Font{font: pf}, nil
}

// Render converts text to ASCII art using the specified font and options.
func Render(text string, f *Font, opts ...Option) (string, error) {
	options := defaultOptions()
	for _, opt := range opts {
		opt(options)
	}
	return renderer.Render(text, f.font, options.toInternal())
}

// Option configures rendering behavior.
type Option func(*options)

type options struct {
	// Internal option fields will be added as needed
}

func defaultOptions() *options {
	return &options{}
}

func (o *options) toInternal() *renderer.Options {
	// Convert to internal renderer options
	return &renderer.Options{}
}

