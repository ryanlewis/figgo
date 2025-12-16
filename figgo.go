// Package figgo provides a high-performance Go library for rendering FIGlet ASCII art.
//
// Figgo aims to be the reference Go implementation for FIGlet fonts (FLF 2.0),
// offering compatibility with the original C figlet while providing modern Go APIs
// and performance optimizations.
//
// # Concurrency
//
// All Font instances are immutable after creation and safe for concurrent use
// across goroutines without additional synchronisation. Multiple goroutines may
// call [Render] or [RenderTo] with the same Font instance simultaneously.
//
// The default font cache ([LoadFontCached], [ParseFontCached]) is thread-safe and
// uses an LRU eviction policy. Cache statistics are collected using atomic counters
// to minimise lock contention.
//
// Internal memory pools ([sync.Pool]) are used for performance optimisation and
// are inherently thread-safe.
//
// # Thread Safety Summary
//
//   - Font: Immutable, safe for concurrent reads
//   - FontCache: Thread-safe with RWMutex
//   - Render/RenderTo: Safe to call concurrently with same Font
//   - Global state: Uses atomic operations or is immutable
package figgo

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ryanlewis/figgo/internal/debug"
	"github.com/ryanlewis/figgo/internal/parser"
	"github.com/ryanlewis/figgo/internal/renderer"
)

const (
	// maxZipSize is the maximum size of a ZIP archive we'll accept (5 MiB)
	maxZipSize = 5 << 20 // 5 MiB
	// maxEntrySize is the maximum size of a single font file in a ZIP (5 MiB)
	maxEntrySize = 5 << 20 // 5 MiB
)

// ParseFont reads a FIGfont from the provided reader and returns a Font instance.
// The returned Font is immutable and safe for concurrent use across goroutines.
//
// ParseFont supports both plain text and ZIP-compressed FIGfont files.
// ZIP-compressed fonts are automatically detected by checking for ZIP magic bytes.
// When a ZIP file is detected, the first file (skipping directory entries) in
// the archive is extracted and parsed.
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
	// Buffer the reader to allow peeking at magic bytes
	// This is more efficient than reading all data upfront
	buf := &bytes.Buffer{}
	tee := io.TeeReader(r, buf)

	// Peek at first 4 bytes for ZIP magic detection
	const zipMagicLen = 4
	magic := make([]byte, zipMagicLen)
	n, err := io.ReadFull(tee, magic)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, fmt.Errorf("failed to read font data: %w", err)
	}

	// Create a reader that includes the peeked bytes
	combined := io.MultiReader(buf, r)

	// Check for ZIP magic bytes (including empty archive signature)
	zipMagic := []byte("PK\x03\x04")
	emptyZipMagic := []byte("PK\x05\x06")
	if n == zipMagicLen && (bytes.Equal(magic, zipMagic) ||
		bytes.Equal(magic, emptyZipMagic)) {
		// Handle as ZIP file - limit size to prevent ZIP bombs
		limited := io.LimitReader(combined, maxZipSize+1)
		data, readErr := io.ReadAll(limited)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read ZIP data: %w", readErr)
		}
		if len(data) > maxZipSize {
			return nil, fmt.Errorf("ZIP archive exceeds maximum size of %d bytes", maxZipSize)
		}
		return parseCompressedFont(data)
	}

	// Handle as regular FLF file - can stream directly
	pf, err := parser.Parse(combined)
	if err != nil {
		return nil, err
	}
	// Convert internal parser.Font to public Font type
	return convertParserFont(pf)
}

// parseCompressedFont extracts and parses a FIGfont from ZIP data
func parseCompressedFont(data []byte) (*Font, error) {
	// Check total archive size first
	if len(data) > maxZipSize {
		return nil, fmt.Errorf("ZIP archive exceeds maximum size of %d bytes", maxZipSize)
	}

	// Create a reader for the ZIP data
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to open ZIP archive: %w", err)
	}

	// Check for empty archive
	if len(reader.File) == 0 {
		return nil, errors.New("ZIP archive is empty")
	}

	// Find the first non-directory entry (following FIGlet convention)
	var fontFile *zip.File
	for _, f := range reader.File {
		if !f.FileInfo().IsDir() {
			fontFile = f
			break
		}
	}

	if fontFile == nil {
		return nil, errors.New("ZIP archive contains only directories, no font files")
	}

	// Check the uncompressed size from metadata
	if fontFile.UncompressedSize64 > uint64(maxEntrySize) {
		return nil, fmt.Errorf("font file exceeds maximum size of %d bytes", maxEntrySize)
	}

	// Open the font file
	rc, err := fontFile.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open font file in ZIP: %w", err)
	}
	defer rc.Close()

	// Read with size limit to prevent bombs regardless of metadata
	limited := io.LimitReader(rc, int64(maxEntrySize)+1)
	fontData, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("failed to read font file from ZIP: %w", err)
	}
	if len(fontData) > maxEntrySize {
		return nil, fmt.Errorf("font file exceeds maximum size of %d bytes", maxEntrySize)
	}

	// Parse the extracted font data
	pf, err := parser.Parse(bytes.NewReader(fontData))
	if err != nil {
		return nil, fmt.Errorf("failed to parse font from ZIP: %w", err)
	}

	// Convert internal parser.Font to public Font type
	return convertParserFont(pf)
}

// LoadFont loads a FIGfont from a file path on the local filesystem.
// This is a convenience wrapper around os.Open and ParseFont.
// The returned Font is immutable and safe for concurrent use across goroutines.
//
// Example:
//
//	font, err := figgo.LoadFont("/usr/share/figlet/standard.flf")
//	if err != nil {
//	    log.Fatal(err)
//	}
func LoadFont(path string) (*Font, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}
	defer file.Close()

	font, err := ParseFont(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse font %s: %w", path, err)
	}

	// Set font name based on filename (without extension)
	font.Name = deriveNameFromPath(path, true)

	return font, nil
}

// ParseFontBytes parses a FIGfont from a byte slice.
// This is a convenience wrapper around bytes.NewReader and ParseFont.
// The returned Font is immutable and safe for concurrent use across goroutines.
//
// Note: This function does not set the Font.Name field (unlike LoadFont* functions).
//
// Example:
//
//	fontData := []byte("flf2a$ 4 3 10 -1 5\n...")
//	font, err := figgo.ParseFontBytes(fontData)
//	if err != nil {
//	    log.Fatal(err)
//	}
func ParseFontBytes(data []byte) (*Font, error) {
	return ParseFont(bytes.NewReader(data))
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
	// SECURITY: ValidPath check MUST come before Clean to prevent traversal attacks.
	// ValidPath rejects ".", ".." segments which Clean would normalize away.
	// This ordering ensures we reject dangerous paths before any normalization.
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

// deriveNameFromPath extracts the font name from a file path by removing the extension.
// Set useOSPath=true for OS filesystem paths (uses filepath), false for fs.FS paths (uses path).
// Handles both .flf (font) and .flc (control file) extensions.
func deriveNameFromPath(filePath string, useOSPath bool) string {
	var base, ext string
	if useOSPath {
		base = filepath.Base(filePath)
		ext = filepath.Ext(filePath)
	} else {
		base = path.Base(filePath)
		ext = path.Ext(filePath)
	}

	// Early return if no extension
	if ext == "" {
		return base
	}

	// Handle both .flf and .flc extensions (for potential future control file support)
	if ext == ".flf" || ext == ".flc" {
		return strings.TrimSuffix(base, ext)
	}

	// Fallback: remove any extension
	return strings.TrimSuffix(base, ext)
}

// LoadFontDir loads a FIGfont from a directory on the local filesystem.
// This is a convenience wrapper around os.DirFS and LoadFontFS.
// The returned Font is immutable and safe for concurrent use across goroutines.
//
// Example:
//
//	font, err := figgo.LoadFontDir("/usr/share/figlet", "standard.flf")
//	if err != nil {
//	    log.Fatal(err)
//	}
func LoadFontDir(dir, fontName string) (*Font, error) {
	fsys := os.DirFS(dir)
	return LoadFontFS(fsys, fontName)
}

// LoadFontFS loads a FIGfont from a filesystem at the specified path.
// The returned Font is immutable and safe for concurrent use across goroutines.
//
// Security Features:
// - Validates paths using fs.ValidPath to prevent directory traversal
// - Rejects absolute paths and paths containing ".." segments
// - Uses path.Clean for normalization (not filepath.Clean)
// - Checks that target is a file, not a directory
//
// Supported Filesystems:
// - embed.FS (embedded fonts at compile time)
// - os.DirFS (local filesystem directories)
// - Any custom fs.FS implementation
// - Supports both plain .flf and ZIP-compressed fonts
//
// Path Requirements:
// - Must be a valid fs.ValidPath (no leading slash, no backslashes)
// - Cannot contain ".", "..", or empty path segments
// - Must point to a file, not a directory
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

	// Check if path is a directory (some fs.FS implementations allow opening directories)
	info, err := fs.Stat(fsys, clean)
	if err != nil {
		return nil, fmt.Errorf("stat %q: %w", clean, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%q is a directory, not a font file", clean)
	}

	// Open the font file
	file, err := fsys.Open(clean)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", clean, err)
	}
	defer file.Close()

	// Parse the font
	font, err := ParseFont(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse font %s: %w", clean, err)
	}

	// Set font name based on filename (without extension)
	// Use path package for fs.FS paths (not filepath)
	font.Name = deriveNameFromPath(clean, false)

	return font, nil
}

// cloneGlyphs creates a deep copy of the glyph map to ensure immutability.
func cloneGlyphs(src map[rune][]string) map[rune][]string {
	if src == nil {
		return nil
	}
	dst := make(map[rune][]string, len(src))
	for r, lines := range src {
		cp := make([]string, len(lines))
		copy(cp, lines)
		dst[r] = cp
	}
	return dst
}

// convertParserFont converts internal parser.Font to public Font type.
// The returned Font has its own copy of glyphs to ensure true immutability.
func convertParserFont(pf *parser.Font) (*Font, error) {
	if pf == nil {
		return nil, fmt.Errorf("nil parser font from Parse()")
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
		glyphs:         cloneGlyphs(pf.Characters),
		Name:           "", // Will be set based on filename or metadata
		Layout:         layout,
		Hardblank:      pf.Hardblank,
		Height:         pf.Height,
		Baseline:       pf.Baseline,
		MaxLen:         pf.MaxLength,
		OldLayout:      pf.OldLayout,
		PrintDirection: pf.PrintDirection,
		CommentLines:   pf.CommentLines,
	}, nil
}

// RenderTo writes ASCII art directly to the provided writer using the specified font and options.
// This is more efficient than Render as it avoids allocating a string for the result.
//
// Performance Benefits:
// - Streams output directly to writer (no intermediate string allocation)
// - Uses pooled buffers internally for UTF-8 encoding
// - Ideal for writing to files, HTTP responses, or other streaming destinations
// - Significantly reduces memory usage for large rendered output
//
// Default Behavior:
// - Uses font's built-in layout and print direction if not overridden
// - Replaces unknown runes with '?' (unless WithUnknownRune is used)
// - Preserves trailing whitespace (unless WithTrimWhitespace is used)
//
// Error Conditions:
// - ErrUnknownFont: if font is nil
// - ErrUnsupportedRune: if text contains runes not in the font
// - Layout conflicts: if conflicting layout options are specified
//
// Example:
//
//	var buf bytes.Buffer
//	err := figgo.RenderTo(&buf, "Hello", font, figgo.WithLayout(figgo.LayoutKerning))
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Print(buf.String())
func RenderTo(w io.Writer, text string, f *Font, opts ...Option) error {
	if f == nil {
		return ErrUnknownFont
	}
	options := defaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	// Default to font's layout and direction if not specified
	if options.layout == nil {
		l := f.Layout
		options.layout = &l
	}
	if options.printDirection == nil {
		d := f.PrintDirection
		options.printDirection = &d
	}

	// Validate layout options
	if err := validateLayout(options); err != nil {
		return err
	}

	// Emit layout merge event if debug is enabled
	if options.debug != nil {
		requestedLayout := 0
		if options.layout != nil {
			requestedLayout = int(*options.layout)
		}

		// Check for FitSmushing rule injection
		injectedRules := 0
		rationale := ""
		finalLayout := requestedLayout

		// Check if FitSmushing is requested without specific rules
		if options.layout != nil && (*options.layout&FitSmushing) != 0 {
			// Extract rule bits (0-5)
			ruleMask := RuleEqualChar | RuleUnderscore | RuleHierarchy | RuleOppositePair | RuleBigX | RuleHardblank
			requestedRules := *options.layout & ruleMask

			if requestedRules == 0 {
				// No rules specified, use font defaults
				fontRules := f.Layout & ruleMask
				if fontRules == 0 {
					// Font has no default rules, inject all
					injectedRules = int(ruleMask)
					rationale = "FitSmushing requested without rules; font has no defaults; injecting all rules"
				} else {
					// Use font's default rules
					injectedRules = int(fontRules)
					rationale = "FitSmushing requested without rules; using font's default rules"
				}
				finalLayout = int(*options.layout | Layout(injectedRules))
			} else {
				rationale = "FitSmushing with explicit rules specified"
			}
		} else if options.layout == nil {
			rationale = "Using font's default layout"
			finalLayout = int(f.Layout)
		}

		// Calculate final smush mode
		finalSmushMode := 0
		if finalLayout&int(FitSmushing) != 0 {
			finalSmushMode = 128 | (finalLayout & 63)
		} else if finalLayout&int(FitKerning) != 0 {
			finalSmushMode = 64
		}

		options.debug.Emit("api", "LayoutMerge", debug.LayoutMergeData{
			RequestedLayout: requestedLayout,
			FontDefaults:    int(f.Layout),
			InjectedRules:   injectedRules,
			FinalLayout:     finalLayout,
			FinalSmushMode:  finalSmushMode,
			Rationale:       rationale,
		})
	}

	// Convert public Font back to internal parser.Font for renderer
	pf := convertToParserFont(f)
	return renderer.RenderTo(w, text, pf, options.toInternal())
}

// Render converts text to ASCII art using the specified font and options.
// It returns the rendered text as a string.
//
// This is the most convenient function for simple use cases where you need
// the result as a string. However, for better performance when writing to
// files, HTTP responses, or other io.Writer destinations, use RenderTo instead.
//
// Memory Usage:
// - Pre-sizes the internal string builder based on estimated output size
// - For large text or tall fonts, consider using RenderTo to avoid large allocations
// - Uses the same rendering engine as RenderTo (delegates to RenderTo internally)
//
// Example:
//
//	result, err := figgo.Render("Hello", font)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(result)
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
		l := f.Layout
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
		// Note: We don't set FullLayout here as f.Layout is the normalized
		// horizontal layout bitmask, not the original FIGfont header value.
		// The renderer currently only consumes horizontal layout settings.
		// TODO: Thread vertical bits into renderer API when vertical rendering is implemented.
		PrintDirection: f.PrintDirection,
		CommentLines:   f.CommentLines,
		// IMPORTANT: We pass glyphs directly without cloning. The renderer MUST NOT
		// mutate this map or its contents to maintain Font immutability guarantees.
		// If mutation becomes necessary, add defensive cloning here (with performance note).
		Characters: f.glyphs,
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
	unknownRune    *rune
	trimWhitespace bool
	width          *int
	debug          *debug.Session // Debug session for tracing
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
		rendererOpts.PrintDirection = o.printDirection
	}
	if o.unknownRune != nil {
		rendererOpts.UnknownRune = o.unknownRune
	}
	rendererOpts.TrimWhitespace = o.trimWhitespace
	if o.width != nil {
		rendererOpts.Width = o.width
	}
	rendererOpts.Debug = o.debug
	return rendererOpts
}
