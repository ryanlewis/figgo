// Command figgo renders ASCII art text using FIGlet fonts.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ryanlewis/figgo"
	"github.com/ryanlewis/figgo/internal/debug"
	"github.com/spf13/pflag"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	os.Exit(run())
}

func run() int {
	var (
		fontPath       string
		unknownRune    string
		showVersion    bool
		showHelp       bool
		trimWhitespace bool
		width          int
		fullWidth      bool
		smushMode      bool
		kernMode       bool
		debugMode      bool
		debugFile      string
		debugPretty    bool
	)

	pflag.StringVarP(&fontPath, "font", "f", "standard", "Path to FIGfont file or font name")
	pflag.StringVarP(&unknownRune, "unknown-rune", "u", "?", "Rune to replace unknown/unsupported characters")
	pflag.BoolVarP(&showVersion, "version", "v", false, "Show version information")
	pflag.BoolVarP(&showHelp, "help", "h", false, "Show help message")
	pflag.BoolVar(&trimWhitespace, "trim-whitespace", false, "Trim trailing whitespace from each line")
	pflag.IntVarP(&width, "width", "w", 80, "Maximum output width in characters (1-1000, 0=default)")
	pflag.BoolVarP(&fullWidth, "full-width", "W", false, "Use full-width mode (no kerning or smushing)")
	pflag.BoolVarP(&smushMode, "smush", "s", false, "Use smushing mode (characters overlap)")
	pflag.BoolVarP(&kernMode, "kern", "k", false, "Use kerning mode (characters touch but don't overlap)")
	pflag.BoolVar(&debugMode, "debug", false, "Enable debug mode (outputs to stderr)")
	pflag.StringVar(&debugFile, "debug-file", "", "Write debug output to file instead of stderr")
	pflag.BoolVar(&debugPretty, "debug-pretty", false, "Use pretty format for debug output (default: JSON)")
	pflag.Parse()

	if showHelp {
		printHelp()
		return 0
	}

	if showVersion {
		fmt.Printf("figgo version %s (commit: %s, built: %s)\n", version, commit, date)
		return 0
	}

	args := pflag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no text provided")
		printHelp()
		return 1
	}

	// Parse unknown rune option
	var unknownRuneValue = '?'
	if unknownRune != "" {
		parsed, err := parseUnknownRune(unknownRune)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing unknown rune: %v\n", err)
			return 1
		}
		unknownRuneValue = parsed
	}

	// Resolve font path
	resolvedPath := resolveFontPath(fontPath)

	// Load font
	fontFile, err := os.Open(resolvedPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening font file: %v\n", err)
		return 1
	}
	defer fontFile.Close()

	font, err := figgo.ParseFont(fontFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing font: %v\n", err)
		return 1
	}

	// Prepare text for rendering
	text := strings.Join(args, " ")
	
	// Setup debug if enabled
	var debugSession interface{}
	if debugMode || debugFile != "" || os.Getenv("FIGGO_DEBUG") == "1" {
		// Enable debug mode
		debug.SetEnabled(true)
		// Initialize debug from environment
		debug.InitFromEnv()
		
		// Create output sink
		var output io.Writer = os.Stderr
		if debugFile != "" {
			file, err := os.Create(debugFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating debug file: %v\n", err)
				return 1
			}
			defer file.Close()
			output = file
		}
		
		// Create sink based on format preference
		var sink debug.Sink
		if debugPretty || os.Getenv("FIGGO_DEBUG_PRETTY") == "1" {
			sink = debug.NewPrettySink(output)
		} else {
			sink = debug.NewJSONSink(output)
		}
		
		// Create session
		session := debug.NewSession(sink)
		if session != nil {
			defer session.Close()
			debugSession = session
		}
	}

	// Build render options
	renderOpts := []figgo.Option{
		figgo.WithUnknownRune(unknownRuneValue),
		figgo.WithWidth(width),
	}
	
	// Add debug if enabled
	if debugSession != nil {
		renderOpts = append(renderOpts, figgo.WithDebug(debugSession))
	}
	if trimWhitespace {
		renderOpts = append(renderOpts, figgo.WithTrimWhitespace(true))
	}
	// Layout mode flags are mutually exclusive
	if fullWidth {
		renderOpts = append(renderOpts, figgo.WithLayout(figgo.FitFullWidth))
	} else if kernMode {
		renderOpts = append(renderOpts, figgo.WithLayout(figgo.FitKerning))
	} else if smushMode {
		// Use smushing mode - this tells figgo to use the font's default smushing rules
		// Don't specify any rules - let the font's layout be used
		// This matches figlet's behavior when -s is specified (no override)
	}

	output, err := figgo.Render(text, font, renderOpts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering text: %v\n", err)
		return 1
	}

	fmt.Println(output)
	return 0
}

// parseUnknownRune parses the unknown rune flag value which can be in various formats:
// - Literal character (e.g., "*", "?")
// - Escaped Unicode: "\uXXXX", "\UXXXXXXXX"
// - Unicode notation: "U+XXXX"
// - Decimal: "63"
// - Hexadecimal: "0x3F"
func parseUnknownRune(s string) (rune, error) {
	if s == "" {
		return 0, fmt.Errorf("unknown rune cannot be empty")
	}

	// Try literal character first (single rune)
	runes := []rune(s)
	if len(runes) == 1 {
		return runes[0], nil
	}

	// Try each format parser
	if r, ok := parseEscapedUnicode(s); ok {
		return r, nil
	}
	if r, ok := parseUnicodeNotation(s); ok {
		return r, nil
	}
	if r, ok := parseHexadecimal(s); ok {
		return r, nil
	}
	if r, ok := parseDecimal(s); ok {
		return r, nil
	}

	return 0, fmt.Errorf("invalid rune format: %s", s)
}

// validateRune checks if a rune is valid UTF-8 and not a surrogate
func validateRune(r rune) (rune, bool) {
	if r < 0 || r > utf8.MaxRune {
		return 0, false
	}
	// Reject UTF-16 surrogates
	if r >= 0xD800 && r <= 0xDFFF {
		return 0, false
	}
	return r, true
}

func parseEscapedUnicode(s string) (rune, bool) {
	// \uXXXX format - must be exactly 6 characters
	if strings.HasPrefix(s, "\\u") && len(s) == 6 {
		code, err := strconv.ParseInt(s[2:], 16, 32)
		if err == nil {
			return validateRune(rune(code))
		}
	}
	// \UXXXXXXXX format - must be exactly 10 characters
	if strings.HasPrefix(s, "\\U") && len(s) == 10 {
		code, err := strconv.ParseInt(s[2:], 16, 32)
		if err == nil {
			return validateRune(rune(code))
		}
	}
	return 0, false
}

func parseUnicodeNotation(s string) (rune, bool) {
	if strings.HasPrefix(s, "U+") || strings.HasPrefix(s, "u+") {
		code, err := strconv.ParseInt(s[2:], 16, 32)
		if err == nil {
			return validateRune(rune(code))
		}
	}
	return 0, false
}

func parseHexadecimal(s string) (rune, bool) {
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		code, err := strconv.ParseInt(s[2:], 16, 32)
		if err == nil {
			return validateRune(rune(code))
		}
	}
	return 0, false
}

func parseDecimal(s string) (rune, bool) {
	code, err := strconv.ParseInt(s, 10, 32)
	if err == nil {
		return validateRune(rune(code))
	}
	return 0, false
}

// resolveFontPath resolves a font path from either a full path or just a font name
func resolveFontPath(fontPath string) string {
	// If it's already a full path to a .flf file, use it directly
	if filepath.Ext(fontPath) == ".flf" {
		return fontPath
	}

	// Check if it exists as is
	if _, err := os.Stat(fontPath); err == nil {
		return fontPath
	}

	// Try adding .flf extension
	withExt := fontPath + ".flf"
	if _, err := os.Stat(withExt); err == nil {
		return withExt
	}

	// Try in fonts/ directory
	inFonts := filepath.Join("fonts", fontPath+".flf")
	if _, err := os.Stat(inFonts); err == nil {
		return inFonts
	}

	// Default to original path (will fail with better error later)
	return fontPath
}

func printHelp() {
	fmt.Println("figgo - FIGlet ASCII art generator")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  figgo [flags] <text>")
	fmt.Println()
	fmt.Println("Flags:")
	pflag.PrintDefaults()
	fmt.Println()
	fmt.Println("Unknown rune formats:")
	fmt.Println("  Literal: -u '*'")
	fmt.Println("  Unicode escape: -u '\\u2588'")
	fmt.Println("  Unicode notation: -u 'U+2588'")
	fmt.Println("  Decimal: -u '63'")
	fmt.Println("  Hexadecimal: -u '0x3F'")
}
