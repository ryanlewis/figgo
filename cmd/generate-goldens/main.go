package main

import (
	"bytes"
	"crypto/sha256"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// GoldenMetadata represents the YAML front matter in golden files
// This should match the struct in golden_test.go
type GoldenMetadata struct {
	Font           string `yaml:"font"`
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
	outDir    = flag.String("out", "testdata/goldens", "Output directory")
	fonts     = flag.String("fonts", "standard slant small big", "Space-separated list of fonts")
	layouts   = flag.String("layouts", "default full kern smush", "Space-separated list of layouts")
	figlet    = flag.String("figlet", "figlet", "Path to figlet binary")
	fontDir = flag.String("fontdir", "", "Font directory for figlet")
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

func main() {
	flag.Parse()

	// Get figlet version
	figletVersion := getFigletVersion(*figlet)
	log.Printf("Using figlet version: %s", figletVersion)

	// Parse font and layout lists
	fontList := strings.Fields(*fonts)
	layoutList := strings.Fields(*layouts)

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
			for _, sample := range defaultSamples {
				if err := generateGoldenFile(font, layout, layoutName, sample, figletVersion); err != nil {
					if *strict {
						log.Fatalf("Failed to generate golden file: %v", err)
					}
					log.Printf("Warning: %v", err)
				}
			}
		}
	}

	log.Println("Golden file generation complete")
}

func generateGoldenFile(font, layout, layoutName, sample, figletVersion string) error {
	// Generate filename slug
	slug := slugify(sample)
	outFile := filepath.Join(*outDir, font, layoutName, slug+".md")

	log.Printf("Generating %s/%s/%s.md", font, layoutName, slug)

	// Get font info
	fontInfo := getFigletInfo(*figlet, font, "-I", "0")
	layoutInfo := getFigletInfo(*figlet, font, "-I", "1")

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

	// Calculate checksum
	checksum := calculateChecksum(art)

	// Create metadata
	metadata := GoldenMetadata{
		Font:           font,
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
	buf.WriteString("```text\n")
	buf.WriteString(art)
	buf.WriteString("\n```\n")

	// Write to file
	if err := os.WriteFile(outFile, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("failed to write file %s: %w", outFile, err)
	}

	return nil
}

func getFigletVersion(figletPath string) string {
	cmd := exec.Command(figletPath, "-v")
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

	cmd := exec.Command(figletPath, cmdArgs...)
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
		cmd := exec.Command(*figlet, "-s", "-f", "standard")
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

	cmd := exec.Command(figletPath, cmdArgs...)
	cmd.Stdin = strings.NewReader(sample)
	output, err := cmd.Output()
	if err != nil {
		return "", err
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
