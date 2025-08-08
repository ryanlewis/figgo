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

// ParseFont reads a FIGfont from the provided reader and returns a Font instance.
func ParseFont(r io.Reader) (*Font, error) {
	pf, err := parser.Parse(r)
	if err != nil {
		return nil, err
	}
	// Convert internal parser.Font to public Font type
	return convertParserFont(pf), nil
}

// convertParserFont converts internal parser.Font to public Font type
func convertParserFont(pf *parser.Font) *Font {
	if pf == nil {
		return nil
	}
	return &Font{
		Name:           "", // Will be set based on filename or metadata
		Hardblank:      pf.Hardblank,
		Height:         pf.Height,
		Baseline:       pf.Baseline,
		MaxLen:         pf.MaxLength,
		OldLayout:      pf.OldLayout,
		FullLayout:     Layout(pf.FullLayout),
		PrintDirection: pf.PrintDirection,
		CommentLines:   pf.CommentLines,
		Glyphs:         pf.Characters,
	}
}

// Render converts text to ASCII art using the specified font and options.
func Render(text string, f *Font, opts ...Option) (string, error) {
	if f == nil {
		return "", ErrUnknownFont
	}
	options := defaultOptions()
	for _, opt := range opts {
		opt(options)
	}
	// Validate layout options
	if err := validateLayout(options); err != nil {
		return "", err
	}
	// Convert public Font back to internal parser.Font for renderer
	pf := convertToParserFont(f)
	return renderer.Render(text, pf, options.toInternal())
}

// convertToParserFont converts public Font to internal parser.Font
func convertToParserFont(f *Font) *parser.Font {
	if f == nil {
		return nil
	}
	return &parser.Font{
		Hardblank:      f.Hardblank,
		Height:         f.Height,
		Baseline:       f.Baseline,
		MaxLength:      f.MaxLen,
		OldLayout:      f.OldLayout,
		FullLayout:     int(f.FullLayout),
		PrintDirection: f.PrintDirection,
		CommentLines:   f.CommentLines,
		Characters:     f.Glyphs,
	}
}

// validateLayout checks for layout conflicts
func validateLayout(opts *options) error {
	if opts.layout != nil {
		layout := *opts.layout
		// Check if both FitKerning and FitSmushing are set
		if (layout&FitKerning != 0) && (layout&FitSmushing != 0) {
			return ErrLayoutConflict
		}
	}
	return nil
}

// Option configures rendering behavior.
type Option func(*options)

type options struct {
	layout         *Layout
	printDirection *int
}

func defaultOptions() *options {
	return &options{}
}

func (o *options) toInternal() *renderer.Options {
	rendererOpts := &renderer.Options{}
	if o.layout != nil {
		rendererOpts.Layout = int(*o.layout)
	}
	if o.printDirection != nil {
		rendererOpts.PrintDirection = *o.printDirection
	}
	return rendererOpts
}
