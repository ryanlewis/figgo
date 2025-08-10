// Command figgo renders ASCII art text using FIGlet fonts.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var (
		fontPath    string
		unknownRune string
		showVersion bool
		showHelp    bool
	)

	pflag.StringVarP(&fontPath, "font", "f", "standard", "Path to FIGfont file or font name")
	pflag.StringVarP(&unknownRune, "unknown-rune", "u", "?", "Rune to replace unknown/unsupported characters")
	pflag.BoolVarP(&showVersion, "version", "v", false, "Show version information")
	pflag.BoolVarP(&showHelp, "help", "h", false, "Show help message")
	pflag.Parse()

	if showHelp {
		printHelp()
		os.Exit(0)
	}

	if showVersion {
		fmt.Printf("figgo version %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	args := pflag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no text provided")
		printHelp()
		os.Exit(1)
	}

	// TODO: Implement font loading and rendering
	text := strings.Join(args, " ")
	fmt.Printf("TODO: Render '%s' with font '%s'\n", text, fontPath)
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

func parseEscapedUnicode(s string) (rune, bool) {
	// \uXXXX format
	if strings.HasPrefix(s, "\\u") && len(s) >= 6 {
		code, err := strconv.ParseInt(s[2:6], 16, 32)
		if err == nil {
			return rune(code), true
		}
	}
	// \UXXXXXXXX format
	if strings.HasPrefix(s, "\\U") && len(s) >= 10 {
		code, err := strconv.ParseInt(s[2:10], 16, 32)
		if err == nil {
			return rune(code), true
		}
	}
	return 0, false
}

func parseUnicodeNotation(s string) (rune, bool) {
	if strings.HasPrefix(s, "U+") || strings.HasPrefix(s, "u+") {
		code, err := strconv.ParseInt(s[2:], 16, 32)
		if err == nil {
			return rune(code), true
		}
	}
	return 0, false
}

func parseHexadecimal(s string) (rune, bool) {
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		code, err := strconv.ParseInt(s[2:], 16, 32)
		if err == nil {
			return rune(code), true
		}
	}
	return 0, false
}

func parseDecimal(s string) (rune, bool) {
	code, err := strconv.ParseInt(s, 10, 32)
	if err == nil {
		return rune(code), true
	}
	return 0, false
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
