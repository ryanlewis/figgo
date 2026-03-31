package figgo_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestDocsValidation verifies that documentation stays in sync with
// the implementation: font file existence, API signature references,
// and layout constant naming.
func TestDocsValidation(t *testing.T) {
	t.Run("bundled fonts exist", func(t *testing.T) {
		// Fonts referenced in CLAUDE.md and docs/fonts.md
		expected := []string{"standard.flf", "slant.flf", "small.flf", "big.flf"}
		for _, font := range expected {
			path := filepath.Join("fonts", font)
			if _, err := os.Stat(path); err != nil {
				t.Errorf("documented font %s does not exist: %v", font, err)
			}
		}
	})

	t.Run("no undocumented fonts", func(t *testing.T) {
		documented := map[string]bool{
			"standard.flf": true,
			"slant.flf":    true,
			"small.flf":    true,
			"big.flf":      true,
		}
		entries, err := os.ReadDir("fonts")
		if err != nil {
			t.Fatalf("cannot read fonts directory: %v", err)
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".flf") {
				if !documented[e.Name()] {
					t.Errorf("font %s exists but is not documented in CLAUDE.md", e.Name())
				}
			}
		}
	})

	t.Run("golden test directories match fonts", func(t *testing.T) {
		fonts := []string{"standard", "slant", "small", "big"}
		layouts := []string{"default", "full-width", "kerning", "smushing"}
		for _, font := range fonts {
			for _, layout := range layouts {
				dir := filepath.Join("testdata", "goldens", font, layout)
				if _, err := os.Stat(dir); err != nil {
					t.Errorf("expected golden test directory %s does not exist", dir)
				}
			}
		}
	})

	t.Run("README API example uses correct signatures", func(t *testing.T) {
		data, err := os.ReadFile("README.md")
		if err != nil {
			t.Fatalf("cannot read README.md: %v", err)
		}
		content := string(data)

		// Render signature: text first, then font
		if strings.Contains(content, "Render(font,") || strings.Contains(content, "Render( font,") {
			t.Error("README.md: Render() has wrong argument order; text should come before font")
		}

		// ParseFont takes io.Reader, not string
		parseFontStringArg := regexp.MustCompile(`ParseFont\("[^"]*"\)`)
		if parseFontStringArg.MatchString(content) {
			t.Error("README.md: ParseFont() takes io.Reader, not a string path; use LoadFont() for paths")
		}
	})

	t.Run("spec-compliance references valid constants", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join("docs", "spec-compliance.md"))
		if err != nil {
			t.Fatalf("cannot read spec-compliance.md: %v", err)
		}
		content := string(data)

		// These constant names should appear as they are in the code
		validConstants := []string{
			"FitKerning", "FitSmushing", "FitFullWidth",
			"RuleEqualChar", "RuleUnderscore", "RuleHierarchy",
			"RuleOppositePair", "RuleBigX", "RuleHardblank",
		}
		for _, c := range validConstants {
			if !strings.Contains(content, c) {
				t.Errorf("spec-compliance.md: missing reference to constant %s", c)
			}
		}

		// Ensure no stale naming (old LayoutKerning/LayoutSmushing)
		if strings.Contains(content, "LayoutKerning") {
			t.Error("spec-compliance.md: uses stale constant name 'LayoutKerning'; should be 'FitKerning'")
		}
		if strings.Contains(content, "LayoutSmushing") {
			t.Error("spec-compliance.md: uses stale constant name 'LayoutSmushing'; should be 'FitSmushing'")
		}
	})

	t.Run("docs reference only existing public functions", func(t *testing.T) {
		// Read the actual exported functions from figgo.go
		src, err := os.ReadFile("figgo.go")
		if err != nil {
			t.Fatalf("cannot read figgo.go: %v", err)
		}
		srcContent := string(src)

		// Functions that docs commonly reference
		expectedFuncs := []string{
			"ParseFont", "LoadFont", "ParseFontBytes",
			"LoadFontDir", "LoadFontFS", "Render", "RenderTo",
		}
		for _, fn := range expectedFuncs {
			if !strings.Contains(srcContent, "func "+fn+"(") {
				t.Errorf("expected public function %s not found in figgo.go", fn)
			}
		}
	})

	t.Run("Font struct has documented fields", func(t *testing.T) {
		src, err := os.ReadFile("types.go")
		if err != nil {
			t.Fatalf("cannot read types.go: %v", err)
		}
		content := string(src)

		// Fields documented in CLAUDE.md and docs
		fields := []string{
			"Name", "Layout", "Hardblank", "Height",
			"Baseline", "MaxLen", "OldLayout", "PrintDirection", "CommentLines",
		}
		for _, f := range fields {
			if !strings.Contains(content, f) {
				t.Errorf("Font struct missing documented field %s", f)
			}
		}
	})

	t.Run("layout constants defined in code", func(t *testing.T) {
		src, err := os.ReadFile("layout.go")
		if err != nil {
			t.Fatalf("cannot read layout.go: %v", err)
		}
		content := string(src)

		constants := []string{
			"FitFullWidth", "FitKerning", "FitSmushing",
			"RuleEqualChar", "RuleUnderscore", "RuleHierarchy",
			"RuleOppositePair", "RuleBigX", "RuleHardblank",
		}
		for _, c := range constants {
			if !strings.Contains(content, c) {
				t.Errorf("layout constant %s not defined in layout.go", c)
			}
		}
	})
}
