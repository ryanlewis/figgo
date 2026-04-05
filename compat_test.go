package figgo

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const externalFontsDir = "testdata/figlet-fonts"

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
