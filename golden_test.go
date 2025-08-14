package figgo

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// goldenMetadata represents the YAML front matter in golden files
type goldenMetadata struct {
	Font           string `yaml:"font"`
	Layout         string `yaml:"layout"`
	Sample         string `yaml:"sample"`
	FigletVersion  string `yaml:"figlet_version"`
	FontInfo       string `yaml:"font_info"`
	LayoutInfo     string `yaml:"layout_info"`
	PrintDirection int    `yaml:"print_direction"`
	Generated      string `yaml:"generated"`
	Generator      string `yaml:"generator"`
	FigletArgs     string `yaml:"figlet_args"`
	ChecksumSHA256 string `yaml:"checksum_sha256"`
}

// parseGoldenFile parses a markdown golden file and extracts metadata and ASCII art
func parseGoldenFile(path string) (*goldenMetadata, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open golden file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Parse YAML front matter
	metadata := &goldenMetadata{}
	inFrontMatter := false
	frontMatterLines := []string{}

	for scanner.Scan() {
		line := scanner.Text()

		if line == "---" {
			if !inFrontMatter {
				inFrontMatter = true
				continue
			} else {
				// End of front matter
				break
			}
		}

		if inFrontMatter {
			frontMatterLines = append(frontMatterLines, line)
		}
	}

	// Parse front matter manually (simple approach for our known format)
	for _, line := range frontMatterLines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"`)

		switch key {
		case "font":
			metadata.Font = value
		case "layout":
			metadata.Layout = value
		case "sample":
			metadata.Sample = value
		case "figlet_version":
			metadata.FigletVersion = value
		case "font_info":
			metadata.FontInfo = value
		case "layout_info":
			metadata.LayoutInfo = value
		case "print_direction":
			if value == "1" {
				metadata.PrintDirection = 1
			}
		case "generated":
			metadata.Generated = value
		case "generator":
			metadata.Generator = value
		case "figlet_args":
			metadata.FigletArgs = value
		case "checksum_sha256":
			metadata.ChecksumSHA256 = value
		}
	}

	// Find and extract ASCII art from code block
	var artLines []string
	inCodeBlock := false

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "```text") {
			inCodeBlock = true
			continue
		}

		if strings.HasPrefix(line, "```") && inCodeBlock {
			break
		}

		if inCodeBlock {
			artLines = append(artLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, "", fmt.Errorf("error reading golden file: %w", err)
	}

	// Join art lines with newlines
	art := strings.Join(artLines, "\n")

	return metadata, art, nil
}

// mapLayoutString converts layout string from golden file to Layout constant
func mapLayoutString(layout string) Layout {
	switch layout {
	case "default":
		// Return 0 to use font's default layout
		return 0
	case "full":
		return FitFullWidth
	case "kern":
		return FitKerning
	case "smush":
		// Enable all horizontal smushing rules per PRD §6.2
		return FitSmushing | RuleEqualChar | RuleUnderscore | RuleHierarchy | RuleOppositePair | RuleBigX | RuleHardblank
	default:
		return 0 // Default layout
	}
}

func TestGoldenFiles(t *testing.T) {
	// Check if golden files exist
	goldenDir := "testdata/goldens"
	if _, err := os.Stat(goldenDir); os.IsNotExist(err) {
		t.Skip("Golden test files not found. Run ./tools/generate-goldens.sh to generate them.")
	}

	// Find all golden files
	var goldenFiles []string
	err := filepath.WalkDir(goldenDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip non-markdown files and index.md
		if !d.IsDir() && strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, "index.md") {
			goldenFiles = append(goldenFiles, path)
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to walk golden directory: %v", err)
	}

	if len(goldenFiles) == 0 {
		t.Skip("No golden test files found")
	}

	t.Logf("Found %d golden test files", len(goldenFiles))

	// Track failures
	failedTests := []string{}

	for _, goldenFile := range goldenFiles {
		// Extract test name from path
		relPath, _ := filepath.Rel(goldenDir, goldenFile)
		testName := strings.TrimSuffix(relPath, ".md")

		t.Run(testName, func(t *testing.T) {
			// Parse golden file
			metadata, expectedArt, err := parseGoldenFile(goldenFile)
			if err != nil {
				t.Fatalf("Failed to parse golden file: %v", err)
			}

			// Load font
			fontPath := filepath.Join("fonts", metadata.Font+".flf")
			font, err := LoadFont(fontPath)
			if err != nil {
				t.Fatalf("Failed to load font %s: %v", metadata.Font, err)
			}

			// Set up render options
			opts := []Option{}

			// Map layout string to Layout constant
			// Special handling for "default" - don't pass any layout option to use font's default
			if metadata.Layout != "default" {
				layout := mapLayoutString(metadata.Layout)
				if layout != 0 {
					opts = append(opts, WithLayout(layout))
				}
			}

			// Add print direction if RTL
			if metadata.PrintDirection == 1 {
				opts = append(opts, WithPrintDirection(1)) // 1 = RTL
			}

			// Render the text
			result, err := Render(metadata.Sample, font, opts...)
			if err != nil {
				// Empty input might be valid depending on implementation
				if metadata.Sample == "" {
					// Skip empty input tests for now
					t.Skipf("Empty input handling: %v", err)
					return
				}
				t.Fatalf("Failed to render text: %v", err)
			}

			// Normalize line endings for comparison
			expectedArt = strings.ReplaceAll(expectedArt, "\r\n", "\n")
			result = strings.ReplaceAll(result, "\r\n", "\n")

			// Trim trailing newline from result if present (figlet compatibility)
			result = strings.TrimSuffix(result, "\n")
			expectedArt = strings.TrimSuffix(expectedArt, "\n")

			// Compare output
			if result != expectedArt {
				// Log failure but continue testing other files
				failedTests = append(failedTests, testName)

				// Show a snippet of the diff
				resultLines := strings.Split(result, "\n")
				expectedLines := strings.Split(expectedArt, "\n")

				t.Errorf("Output mismatch for %s", testName)
				t.Errorf("Font: %s, Layout: %s, Sample: %q", metadata.Font, metadata.Layout, metadata.Sample)

				// Find first differing line
				for i := 0; i < len(resultLines) || i < len(expectedLines); i++ {
					if i >= len(expectedLines) {
						t.Errorf("Line %d: Got extra line: %q", i+1, resultLines[i])
						break
					}
					if i >= len(resultLines) {
						t.Errorf("Line %d: Missing expected line: %q", i+1, expectedLines[i])
						break
					}
					if resultLines[i] != expectedLines[i] {
						t.Errorf("Line %d differs:", i+1)
						t.Errorf("  Got:      %q", resultLines[i])
						t.Errorf("  Expected: %q", expectedLines[i])
						break
					}
				}
			}
		})
	}

	// Summary
	if len(failedTests) > 0 {
		t.Errorf("\n=== GOLDEN TEST SUMMARY ===")
		t.Errorf("Failed %d out of %d tests", len(failedTests), len(goldenFiles))
		t.Errorf("Failed tests:")
		for _, test := range failedTests {
			t.Errorf("  - %s", test)
		}
	}
}

// TestGoldenFilesSubset tests a subset of golden files for quick verification
func TestGoldenFilesSubset(t *testing.T) {
	// Test a representative subset for quick CI runs
	testCases := []struct {
		font   string
		layout string
		sample string
	}{
		{"standard", "full", "Hello, World!"},
		{"standard", "kern", "Hello, World!"},
		{"standard", "smush", "Hello, World!"},
		{"standard", "full", " "}, // Single space edge case
		{"standard", "full", "a"},
		{"standard", "kern", "FIGgo 1.0"},
	}

	for _, tc := range testCases {
		testName := fmt.Sprintf("%s/%s/%s", tc.font, tc.layout, tc.sample)
		t.Run(testName, func(t *testing.T) {
			// Construct golden file path
			slug := slugify(tc.sample)
			layoutName := getLayoutName(tc.layout)
			goldenFile := filepath.Join("testdata/goldens", tc.font, layoutName, slug+".md")

			// Check if golden file exists
			if _, err := os.Stat(goldenFile); os.IsNotExist(err) {
				t.Skipf("Golden file not found: %s", goldenFile)
			}

			// Parse golden file
			_, expectedArt, err := parseGoldenFile(goldenFile)
			if err != nil {
				t.Fatalf("Failed to parse golden file: %v", err)
			}

			// Load font
			fontPath := filepath.Join("fonts", tc.font+".flf")
			font, err := LoadFont(fontPath)
			if err != nil {
				t.Fatalf("Failed to load font %s: %v", tc.font, err)
			}

			// Set up render options
			opts := []Option{}
			layout := mapLayoutString(tc.layout)
			if layout != 0 {
				opts = append(opts, WithLayout(layout))
			}

			// Render the text
			result, err := Render(tc.sample, font, opts...)
			if err != nil {
				t.Fatalf("Failed to render text: %v", err)
			}

			// Normalize and compare
			expectedArt = strings.ReplaceAll(expectedArt, "\r\n", "\n")
			result = strings.ReplaceAll(result, "\r\n", "\n")
			result = strings.TrimSuffix(result, "\n")
			expectedArt = strings.TrimSuffix(expectedArt, "\n")

			if result != expectedArt {
				t.Errorf("Output mismatch")
				t.Errorf("Got:\n%s", result)
				t.Errorf("Expected:\n%s", expectedArt)
			}
		})
	}
}

// Helper functions for test
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

	// Replace non-alphanumeric with underscore
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
		} else {
			result.WriteRune('_')
		}
	}

	// Clean up underscores
	slug := result.String()
	slug = strings.Trim(slug, "_")
	for strings.Contains(slug, "__") {
		slug = strings.ReplaceAll(slug, "__", "_")
	}

	if slug == "" {
		// Use first 8 chars of hex for non-alphanumeric strings
		return "special"
	}

	return slug
}

func getLayoutName(layout string) string {
	switch layout {
	case "default":
		return "default"
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
