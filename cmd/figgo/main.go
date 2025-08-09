// Command figgo renders ASCII art text using FIGlet fonts.
package main

import (
	"fmt"
	"os"
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
		showVersion bool
		showHelp    bool
	)

	pflag.StringVarP(&fontPath, "font", "f", "standard", "Path to FIGfont file or font name")
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

func printHelp() {
	fmt.Println("figgo - FIGlet ASCII art generator")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  figgo [flags] <text>")
	fmt.Println()
	fmt.Println("Flags:")
	pflag.PrintDefaults()
}
