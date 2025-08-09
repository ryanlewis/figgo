// Package figgo provides a high-performance Go library for rendering FIGlet ASCII art.
//
// Figgo aims to be the reference Go implementation for FIGlet fonts (FLF 2.0),
// offering compatibility with the original C figlet while providing modern Go APIs
// and performance optimizations.
package figgo

import (
	"fmt"
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
	return convertParserFont(pf)
}

// convertParserFont converts internal parser.Font to public Font type
func convertParserFont(pf *parser.Font) (*Font, error) {
	if pf == nil {
		return nil, nil
	}

	// Normalize layout from header values
	normalized, err := NormalizeLayoutFromHeader(pf.OldLayout, pf.FullLayout, pf.FullLayoutSet)
	if err != nil {
		// Return the normalization error - this indicates a malformed font file
		// with invalid OldLayout/FullLayout values that the parser didn't catch
		return nil, fmt.Errorf("failed to normalize layout from font header: %w", err)
	}

	// Convert normalized layout to Layout bitmask
	layout := normalized.ToLayout()

	return &Font{
		glyphs:         pf.Characters,
		Name:           "", // Will be set based on filename or metadata
		FullLayout:     layout,
		Hardblank:      pf.Hardblank,
		Height:         pf.Height,
		Baseline:       pf.Baseline,
		MaxLen:         pf.MaxLength,
		OldLayout:      pf.OldLayout,
		PrintDirection: pf.PrintDirection,
		CommentLines:   pf.CommentLines,
	}, nil
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

	// Default to font's layout and direction if not specified
	if options.layout == nil {
		l := f.FullLayout
		options.layout = &l
	}
	if options.printDirection == nil {
		d := f.PrintDirection
		options.printDirection = &d
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
		Hardblank: f.Hardblank,
		Height:    f.Height,
		Baseline:  f.Baseline,
		MaxLength: f.MaxLen,
		OldLayout: f.OldLayout,
		// Note: We don't set FullLayout here as f.FullLayout is the normalized
		// horizontal layout bitmask, not the original FIGfont header value.
		// The renderer currently only consumes horizontal layout settings.
		// TODO: Thread vertical mode/rules into renderer API for vertical text support.
		PrintDirection: f.PrintDirection,
		CommentLines:   f.CommentLines,
		Characters:     f.glyphs,
	}
}

// validateLayout checks for layout conflicts and normalizes the layout
func validateLayout(opts *options) error {
	if opts.layout != nil {
		// Use NormalizeLayout to check for all conflicts
		normalized, err := NormalizeLayout(*opts.layout)
		if err != nil {
			return err
		}
		// Update with normalized layout
		*opts.layout = normalized
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
