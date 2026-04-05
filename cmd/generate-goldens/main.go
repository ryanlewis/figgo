package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// cp437ToUnicode maps CP437 byte values 0x80-0xFF to Unicode equivalents.
var cp437ToUnicode = [128]rune{
	0x00C7, 0x00FC, 0x00E9, 0x00E2, 0x00E4, 0x00E0, 0x00E5, 0x00E7,
	0x00EA, 0x00EB, 0x00E8, 0x00EF, 0x00EE, 0x00EC, 0x00C4, 0x00C5,
	0x00C9, 0x00E6, 0x00C6, 0x00F4, 0x00F6, 0x00F2, 0x00FB, 0x00F9,
	0x00FF, 0x00D6, 0x00DC, 0x00A2, 0x00A3, 0x00A5, 0x20A7, 0x0192,
	0x00E1, 0x00ED, 0x00F3, 0x00FA, 0x00F1, 0x00D1, 0x00AA, 0x00BA,
	0x00BF, 0x2310, 0x00AC, 0x00BD, 0x00BC, 0x00A1, 0x00AB, 0x00BB,
	0x2591, 0x2592, 0x2593, 0x2502, 0x2524, 0x2561, 0x2562, 0x2556,
	0x2555, 0x2563, 0x2551, 0x2557, 0x255D, 0x255C, 0x255B, 0x2510,
	0x2514, 0x2534, 0x252C, 0x251C, 0x2500, 0x253C, 0x255E, 0x255F,
	0x255A, 0x2554, 0x2569, 0x2566, 0x2560, 0x2550, 0x256C, 0x2567,
	0x2568, 0x2564, 0x2565, 0x2559, 0x2558, 0x2552, 0x2553, 0x256B,
	0x256A, 0x2518, 0x250C, 0x2588, 0x2584, 0x258C, 0x2590, 0x2580,
	0x03B1, 0x00DF, 0x0393, 0x03C0, 0x03A3, 0x03C3, 0x00B5, 0x03C4,
	0x03A6, 0x0398, 0x03A9, 0x03B4, 0x221E, 0x03C6, 0x03B5, 0x2229,
	0x2261, 0x00B1, 0x2265, 0x2264, 0x2320, 0x2321, 0x00F7, 0x2248,
	0x00B0, 0x2219, 0x00B7, 0x221A, 0x207F, 0x00B2, 0x25A0, 0x00A0,
}

// transcodeCP437Bytes converts CP437 byte output to UTF-8.
func transcodeCP437Bytes(data []byte) []byte {
	var buf bytes.Buffer
	buf.Grow(len(data) * 2)
	for _, b := range data {
		if b < 0x80 {
			buf.WriteByte(b)
		} else {
			buf.WriteRune(cp437ToUnicode[b-0x80])
		}
	}
	return buf.Bytes()
}

// GoldenMetadata represents the YAML front matter in golden files
// This should match the struct in golden_test.go
type GoldenMetadata struct {
	Font           string `yaml:"font"`
	FontDir        string `yaml:"font_dir,omitempty"` // Directory containing the font (omitted for default "fonts")
	Layout         string `yaml:"layout"`
	Sample         string `yaml:"sample"`
	Width          int    `yaml:"width"` // Explicit width for deterministic wrapping
	FigletVersion  string `yaml:"figlet_version"`
	FontInfo       string `yaml:"font_info"`
	LayoutInfo     string `yaml:"layout_info"`
	PrintDirection int    `yaml:"print_direction"`
	Generated      string `yaml:"generated"`
	Generator      string `yaml:"generator"`
	FigletArgs     string `yaml:"figlet_args"`
	ChecksumSHA256 string `yaml:"checksum_sha256"`
}

var (
	outDir  = flag.String("out", "testdata/goldens", "Output directory")
	fonts   = flag.String("fonts", "standard slant small big", "Space-separated list of fonts")
	layouts = flag.String("layouts", "default full kern smush", "Space-separated list of layouts")
	figlet  = flag.String("figlet", "figlet", "Path to figlet binary")
	fontDir = flag.String("fontdir", "", "Font directory for figlet (-d flag)")
	scanDir = flag.String("scan-dir", "", "Scan directory for all .flf fonts (overrides -fonts)")
	samples = flag.String("samples", "", "Space-separated samples (overrides defaults; use _ for space)")
	strict  = flag.Bool("strict", false, "Exit on any warning")
)

// Default samples including edge cases
var defaultSamples = []string{
	"Hello, World!",
	"FIGgo 1.0",
	`|/\[]{}()<>`,
	"The quick brown fox jumps over the lazy dog",
	" ", // Single space
	"a",
	"   ", // Three spaces
	"$$$$",
	`!@#$%^&*()_+-=[]{}:;'",.<>?/\|`, // Problematic special characters
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ",
	"abcdefghijklmnopqrstuvwxyz",
	"0123456789",
}

// Smaller sample set for bulk font scanning
var scanSamples = []string{
	"Hello, World!",
	"Testing 123",
	"ABCDEFGHIJKLM",
}

func main() {
	flag.Parse()

	// Get figlet version
	figletVersion := getFigletVersion(*figlet)
	log.Printf("Using figlet version: %s", figletVersion)

	// Determine font list
	fontList := strings.Fields(*fonts)
	// Track the font_dir value to embed in metadata (empty = default "fonts" dir)
	metadataFontDir := ""

	if *scanDir != "" {
		// Scan directory for all .flf files
		discovered, err := discoverFonts(*scanDir)
		if err != nil {
			log.Fatalf("Failed to scan directory %s: %v", *scanDir, err)
		}
		fontList = discovered
		metadataFontDir = *scanDir
		if *fontDir == "" {
			*fontDir = *scanDir
		}
		log.Printf("Discovered %d fonts in %s", len(fontList), *scanDir)
	}

	// Determine sample list
	sampleList := defaultSamples
	if *scanDir != "" && *samples == "" {
		sampleList = scanSamples
	}
	if *samples != "" {
		sampleList = strings.Fields(*samples)
	}

	// Parse layout list
	layoutList := strings.Fields(*layouts)

	// Track stats
	generated := 0
	skipped := 0

	// Process each combination
	for _, font := range fontList {
		for _, layout := range layoutList {
			layoutName := getLayoutName(layout)

			// Create output directory
			dir := filepath.Join(*outDir, font, layoutName)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				log.Fatalf("Failed to create directory %s: %v", dir, err)
			}

			// Process each sample
			for _, sample := range sampleList {
				err := generateGoldenFile(font, layout, layoutName, sample, figletVersion, metadataFontDir)
				if err != nil {
					if *strict {
						log.Fatalf("Failed to generate golden file: %v", err)
					}
					skipped++
					log.Printf("Warning: %v", err)
				} else {
					generated++
				}
			}
		}
	}

	log.Printf("Golden file generation complete: %d generated, %d skipped", generated, skipped)
}

// discoverFonts walks a directory and returns sorted font names (without .flf extension)
func discoverFonts(dir string) ([]string, error) {
	var fonts []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".flf") {
			return nil
		}
		name := strings.TrimSuffix(filepath.Base(path), ".flf")
		fonts = append(fonts, name)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(fonts)
	return fonts, nil
}

func generateGoldenFile(font, layout, layoutName, sample, figletVersion, metadataFontDir string) error {
	// Generate filename slug
	slug := slugify(sample)
	outFile := filepath.Join(*outDir, font, layoutName, slug+".md")

	// Get layout arguments
	layoutArgs := getLayoutArgs(layout)

	// Use explicit width for deterministic output
	width := 80
	layoutArgsWithWidth := layoutArgs
	if layoutArgsWithWidth != "" {
		layoutArgsWithWidth += " "
	}
	layoutArgsWithWidth += fmt.Sprintf("-w %d", width)

	// Generate ASCII art
	art, err := generateArt(*figlet, font, sample, layoutArgsWithWidth)
	if err != nil {
		return fmt.Errorf("failed to generate art for %s/%s/%s: %w", font, layoutName, slug, err)
	}

	log.Printf("Generating %s/%s/%s.md", font, layoutName, slug)

	// Get font info
	fontInfo := getFigletInfo(*figlet, font, "-I", "0")
	layoutInfo := getFigletInfo(*figlet, font, "-I", "1")

	// Calculate checksum
	checksum := calculateChecksum(art)

	// Create metadata
	metadata := GoldenMetadata{
		Font:           font,
		FontDir:        metadataFontDir,
		Layout:         layout,
		Sample:         sample, // YAML marshaling will handle escaping
		Width:          width,  // Explicit width for deterministic wrapping
		FigletVersion:  figletVersion,
		FontInfo:       fontInfo,
		LayoutInfo:     layoutInfo,
		PrintDirection: 0,
		Generated:      time.Now().UTC().Format("2006-01-02"),
		Generator:      "generate-goldens",
		FigletArgs:     layoutArgsWithWidth,
		ChecksumSHA256: checksum,
	}

	// Marshal metadata to YAML
	yamlData, err := yaml.Marshal(&metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Create the markdown file
	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(yamlData)
	buf.WriteString("---\n\n")
	buf.WriteString("````text\n")
	buf.WriteString(art)
	buf.WriteString("\n````\n")

	// Write to file
	if err := os.WriteFile(outFile, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("failed to write file %s: %w", outFile, err)
	}

	return nil
}

func getFigletVersion(figletPath string) string {
	cmd := exec.CommandContext(context.Background(), figletPath, "-v")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "unknown"
	}
	// Extract version from output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "FIGlet") || strings.Contains(line, "flf2") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[0] + " " + parts[1]
			}
			return strings.TrimSpace(line)
		}
	}
	return "unknown"
}

func getFigletInfo(figletPath, font string, args ...string) string {
	cmdArgs := []string{}
	if *fontDir != "" {
		cmdArgs = append(cmdArgs, "-d", *fontDir)
	}
	cmdArgs = append(cmdArgs, "-f", font)
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.CommandContext(context.Background(), figletPath, cmdArgs...)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return ""
}

func getLayoutArgs(layout string) string {
	switch layout {
	case "default":
		return ""
	case "full":
		return "-W"
	case "kern":
		return "-k"
	case "smush":
		// Check if figlet supports -s
		//nolint:gosec // figlet path is from trusted flag, not user input
		cmd := exec.CommandContext(context.Background(), *figlet, "-s", "-f", "standard")
		cmd.Stdin = strings.NewReader("test")
		if err := cmd.Run(); err == nil {
			return "-s"
		}
		// Fall back to -S if -s not supported
		return "-S"
	default:
		return ""
	}
}

func getLayoutName(layout string) string {
	switch layout {
	case "full":
		return "full-width"
	case "kern":
		return "kerning"
	case "smush":
		return "smushing"
	default:
		return layout
	}
}

func generateArt(figletPath, font, sample, layoutArgs string) (string, error) {
	cmdArgs := []string{}
	if *fontDir != "" {
		cmdArgs = append(cmdArgs, "-d", *fontDir)
	}
	cmdArgs = append(cmdArgs, "-f", font)
	if layoutArgs != "" {
		cmdArgs = append(cmdArgs, strings.Fields(layoutArgs)...)
	}

	cmd := exec.CommandContext(context.Background(), figletPath, cmdArgs...)
	cmd.Stdin = strings.NewReader(sample)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Transcode non-UTF-8 output (CP437 fonts) to Unicode
	if !utf8.Valid(output) {
		output = transcodeCP437Bytes(output)
	}

	// Remove trailing newline if present
	result := string(output)
	result = strings.TrimSuffix(result, "\n")
	return result, nil
}

func calculateChecksum(data string) string {
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}

func slugify(s string) string {
	if s == "" {
		return "empty"
	}
	if s == " " {
		return "space"
	}
	if s == "  " {
		return "two_spaces"
	}
	if s == "   " {
		return "three_spaces"
	}

	// For other strings, replace non-alphanumeric with underscore
	var result []rune
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result = append(result, r)
		} else if len(result) == 0 || result[len(result)-1] != '_' {
			result = append(result, '_')
		}
	}

	// Trim leading/trailing underscores
	slug := strings.Trim(string(result), "_")

	// If empty after processing, use hash
	if slug == "" {
		hash := sha256.Sum256([]byte(s))
		return fmt.Sprintf("%x", hash)[:8]
	}

	return slug
}
