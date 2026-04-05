package figgo

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const (
	externalFontsDir = "testdata/figlet-fonts"
	compatGoldensDir = "testdata/goldens-compat"
)

// TestExternalFontCompat walks an external font collection and attempts to
// parse and render every .flf file.  It reports which fonts fail at parse
// time vs render time, producing a summary at the end.
//
// The test expects the font collection to be cloned into testdata/figlet-fonts:
//
//	git clone --depth 1 https://github.com/xero/figlet-fonts testdata/figlet-fonts
func TestExternalFontCompat(t *testing.T) {
	if _, err := os.Stat(externalFontsDir); os.IsNotExist(err) {
		t.Skipf("External font collection not found at %s — clone with: git clone --depth 1 https://github.com/xero/figlet-fonts %s", externalFontsDir, externalFontsDir)
	}

	samples := []string{
		"Hello",
		"FIGlet",
		"Testing 123",
	}

	type failure struct {
		font  string
		phase string // "parse" or "render"
		err   string
	}

	var (
		fonts    []string
		failures []failure
	)

	err := filepath.WalkDir(externalFontsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".flf") {
			return nil
		}
		fonts = append(fonts, path)
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk font directory: %v", err)
	}

	sort.Strings(fonts)
	t.Logf("Found %d .flf fonts", len(fonts))

	for _, fontPath := range fonts {
		rel, _ := filepath.Rel(externalFontsDir, fontPath)
		t.Run(rel, func(t *testing.T) {
			font, err := LoadFont(fontPath)
			if err != nil {
				failures = append(failures, failure{font: rel, phase: "parse", err: err.Error()})
				t.Errorf("parse failed: %v", err)
				return
			}

			for _, sample := range samples {
				result, err := Render(sample, font)
				if err != nil {
					failures = append(failures, failure{font: rel, phase: fmt.Sprintf("render %q", sample), err: err.Error()})
					t.Errorf("render %q failed: %v", sample, err)
					return
				}
				if result == "" {
					t.Errorf("render %q produced empty output", sample)
				}
			}
		})
	}

	// Summary
	if len(failures) > 0 {
		t.Logf("\n=== EXTERNAL FONT COMPAT SUMMARY ===")
		t.Logf("Total fonts: %d", len(fonts))
		t.Logf("Failures: %d", len(failures))

		parseFailures := 0
		renderFailures := 0
		for _, f := range failures {
			if f.phase == "parse" {
				parseFailures++
			} else {
				renderFailures++
			}
		}
		t.Logf("  Parse failures:  %d", parseFailures)
		t.Logf("  Render failures: %d", renderFailures)
		t.Logf("")

		for _, f := range failures {
			t.Logf("  [%s] %s: %s", f.phase, f.font, f.err)
		}
	} else {
		t.Logf("All %d fonts parsed and rendered successfully!", len(fonts))
	}
}

// TestCompatGoldenFiles runs golden tests against the external font collection,
// comparing figgo output to the reference figlet implementation. Mismatches are
// reported but do not fail the test — this is a compatibility tracking tool.
//
// Generate golden files with:
//
//	go run ./cmd/generate-goldens -scan-dir testdata/figlet-fonts -layouts default -out testdata/goldens-compat
func TestCompatGoldenFiles(t *testing.T) {
	if _, err := os.Stat(compatGoldensDir); os.IsNotExist(err) {
		t.Skipf("Compat golden files not found at %s", compatGoldensDir)
	}
	if _, err := os.Stat(externalFontsDir); os.IsNotExist(err) {
		t.Skipf("External font collection not found at %s", externalFontsDir)
	}

	// Collect all golden files
	var goldenFiles []string
	err := filepath.WalkDir(compatGoldensDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".md") {
			goldenFiles = append(goldenFiles, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk compat goldens directory: %v", err)
	}

	if len(goldenFiles) == 0 {
		t.Skip("No compat golden test files found")
	}

	sort.Strings(goldenFiles)
	t.Logf("Found %d compat golden test files", len(goldenFiles))

	passed := 0
	failed := 0
	skipped := 0
	var mismatches []string

	for _, goldenFile := range goldenFiles {
		relPath, _ := filepath.Rel(compatGoldensDir, goldenFile)
		testName := strings.TrimSuffix(relPath, ".md")

		t.Run(testName, func(t *testing.T) {
			metadata, expectedArt, err := parseGoldenFile(goldenFile)
			if err != nil {
				t.Skipf("Failed to parse golden file: %v", err)
				skipped++
				return
			}

			// Resolve font path
			fontBaseDir := "fonts"
			if metadata.FontDir != "" {
				fontBaseDir = metadata.FontDir
			}
			fontPath := filepath.Join(fontBaseDir, metadata.Font+".flf")
			if _, statErr := os.Stat(fontPath); os.IsNotExist(statErr) {
				t.Skipf("Font file not found: %s", fontPath)
				skipped++
				return
			}

			font, err := LoadFont(fontPath)
			if err != nil {
				t.Skipf("Failed to load font: %v", err)
				skipped++
				return
			}

			// Set up render options
			var opts []Option
			if metadata.Layout != goldenLayoutDefault {
				layout := mapLayoutString(metadata.Layout)
				opts = append(opts, WithLayout(layout))
			}
			if metadata.PrintDirection == 1 {
				opts = append(opts, WithPrintDirection(1))
			}
			width := metadata.Width
			if width == 0 {
				width = 80
			}
			opts = append(opts, WithWidth(width))

			result, err := Render(metadata.Sample, font, opts...)
			if err != nil {
				t.Logf("Render failed: %v", err)
				failed++
				mismatches = append(mismatches, testName+" (render error)")
				return
			}

			// Normalize
			expectedArt = strings.ReplaceAll(expectedArt, "\r\n", "\n")
			result = strings.ReplaceAll(result, "\r\n", "\n")
			result = strings.TrimSuffix(result, "\n")
			expectedArt = strings.TrimSuffix(expectedArt, "\n")

			if result != expectedArt {
				failed++
				mismatches = append(mismatches, testName)
				t.Logf("Output mismatch (not a failure — tracking compatibility)")
			} else {
				passed++
			}
		})
	}

	total := passed + failed + skipped
	t.Logf("\n=== COMPAT GOLDEN SUMMARY ===")
	t.Logf("Total: %d | Passed: %d | Mismatched: %d | Skipped: %d",
		total, passed, failed, skipped)
	if total > 0 {
		t.Logf("Compatibility: %.1f%%", float64(passed)/float64(total)*100)
	}
	if len(mismatches) > 0 {
		t.Logf("Mismatched tests:")
		for _, m := range mismatches {
			t.Logf("  - %s", m)
		}
	}
}
