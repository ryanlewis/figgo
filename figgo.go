// Package figgo provides a high-performance Go library for rendering FIGlet ASCII art.
//
// Figgo aims to be the reference Go implementation for FIGlet fonts (FLF 2.0),
// offering compatibility with the original C figlet while providing modern Go APIs
// and performance optimizations.
package figgo

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"

	"github.com/ryanlewis/figgo/internal/parser"
	"github.com/ryanlewis/figgo/internal/renderer"
)

// ParseFont reads a FIGfont from the provided reader and returns a Font instance.
// The returned Font is immutable and safe for concurrent use across goroutines.
//
// ParseFont expects a valid FIGfont v2 format file with at least the required
// ASCII characters (32-126). The font's layout settings are normalized according
// to the FIGfont specification.
//
// Example:
//
//	file, err := os.Open("standard.flf")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer file.Close()
//
//	font, err := figgo.ParseFont(file)
//	if err != nil {
//	    log.Fatal(err)
//	}
func ParseFont(r io.Reader) (*Font, error) {
	pf, err := parser.Parse(r)
	if err != nil {
		return nil, err
	}
	// Convert internal parser.Font to public Font type
	return convertParserFont(pf)
}

// cleanFSPath validates and cleans a path for use with fs.FS.
// It ensures the path is valid according to fs.ValidPath rules and
// prevents directory traversal attacks.
func cleanFSPath(p string) (string, error) {
	if p == "" {
		return "", errors.New("path cannot be empty")
	}
	// fs.FS disallows leading slash and uses '/' only
	if strings.HasPrefix(p, "/") {
		return "", errors.New("absolute paths not allowed")
	}
	if strings.ContainsRune(p, '\\') {
		return "", errors.New("backslashes not allowed in fs paths")
	}
	if !fs.ValidPath(p) {
		// rejects ".", ".." segments, empty elements, etc.
		return "", fmt.Errorf("invalid fs path: %s", p)
	}
	clean := path.Clean(p) // purely slash semantics
	if clean == "." || strings.HasPrefix(clean, "../") {
		return "", errors.New("path traversal not allowed")
	}
	return clean, nil
}

// LoadFontFS loads a FIGfont from a filesystem at the specified path.
// The returned Font is immutable and safe for concurrent use across goroutines.
//
// The path must be a valid path within the filesystem, and the file must be
// a valid FIGfont v2 format file. Path traversal (e.g., "../") is not allowed
// for security reasons.
//
// Example with embed.FS:
//
//	//go:embed fonts/*.flf
//	var fonts embed.FS
//
//	font, err := figgo.LoadFontFS(fonts, "fonts/standard.flf")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Example with os.DirFS:
//
//	fontsDir := os.DirFS("/usr/share/figlet")
//	font, err := figgo.LoadFontFS(fontsDir, "standard.flf")
//	if err != nil {
//	    log.Fatal(err)
//	}
func LoadFontFS(fsys fs.FS, fontPath string) (*Font, error) {
	// Validate inputs
	if fsys == nil {
		return nil, fmt.Errorf("filesystem cannot be nil")
	}

	// Clean and validate the path
	clean, err := cleanFSPath(fontPath)
	if err != nil {
		return nil, err
	}

	// Open the font file
	file, err := fsys.Open(clean)
	if err != nil {
		return nil, fmt.Errorf("failed to open font file: %w", err)
	}
	defer file.Close()

	// Parse the font
	font, err := ParseFont(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse font %s: %w", clean, err)
	}

	// Set font name based on filename (without extension)
	// Use path package for fs.FS paths (not filepath)
	font.Name = strings.TrimSuffix(path.Base(clean), path.Ext(clean))

	return font, nil
}

// convertParserFont converts internal parser.Font to public Font type.
// Note: The returned Font shares the glyph map with the parser font for efficiency.
// The parser and renderer must not mutate glyphs after parsing to maintain immutability.
// TODO: Consider deep-copying glyphs if mutation becomes a concern.
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
