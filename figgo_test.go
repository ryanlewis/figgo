package figgo

import (
	"strings"
	"testing"
)

// createTestFont creates a minimal font for testing
func createTestFont(t *testing.T) *Font {
	fontData := `flf2a$ 4 3 10 -1 1
Test font
$@
$@
$@
$@@
H@
H@
H@
H@@
e@
e@
e@
e@@
l@
l@
l@
l@@
o@
o@
o@
o@@
`
	font, err := ParseFontBytes([]byte(fontData))
	if err != nil {
		t.Fatalf("ParseFontBytes() error = %v", err)
	}
	return font
}

// skipIfRendererNotImplemented skips the test if renderer is not yet implemented
func skipIfRendererNotImplemented(t *testing.T, err error) {
	if err != nil && strings.Contains(err.Error(), "not yet implemented") {
		t.Skip("Renderer not yet implemented")
	}
}

// verifyRenderOutput checks that render produced valid output
func verifyRenderOutput(t *testing.T, output string, err error, context string) {
	skipIfRendererNotImplemented(t, err)
	if err != nil {
		t.Errorf("Render() %s error = %v", context, err)
	}
	if output == "" {
		t.Errorf("Render() %s returned empty output", context)
	}
}

// TestRenderDefaulting verifies that Render uses the font's default layout and direction
// when no options are provided.
func TestRenderDefaulting(t *testing.T) {
	font := createTestFont(t)

	// Test that Render works with no options (should use font's defaults)
	t.Run("render with font defaults", func(t *testing.T) {
		output, err := Render("Hello", font)
		verifyRenderOutput(t, output, err, "with defaults")
		// Verify we got some ASCII art back
		if err == nil && (!strings.Contains(output, "H") || !strings.Contains(output, "e")) {
			t.Errorf("Render() output doesn't seem to contain expected characters: %q", output)
		}
	})

	// Test nil font handling
	t.Run("render with nil font", func(t *testing.T) {
		output, err := Render("Hello", nil)
		if err == nil {
			t.Error("Render() should return error for nil font")
		}
		if err != ErrUnknownFont {
			t.Errorf("Render() error = %v, want ErrUnknownFont", err)
		}
		if output != "" {
			t.Errorf("Render() should return empty string on error, got %q", output)
		}
	})

	// Test with explicit layout option to verify overrides work
	t.Run("render with layout override", func(t *testing.T) {
		output, err := Render("Hello", font, WithLayout(FitKerning))
		verifyRenderOutput(t, output, err, "with layout override")
	})

	// Test with print direction override
	t.Run("render with direction override", func(t *testing.T) {
		output, err := Render("Hello", font, WithPrintDirection(1)) // RTL
		verifyRenderOutput(t, output, err, "with direction override")
	})
}

// TestRenderValidation verifies that Render properly validates layout options
func TestRenderValidation(t *testing.T) {
	// Create a minimal font for testing
	fontData := `flf2a$ 4 3 10 -1 1
Test font
$@
$@
$@
$@@
x@
x@
x@
x@@
`
	font, err := ParseFontBytes([]byte(fontData))
	if err != nil {
		t.Fatalf("ParseFontBytes() error = %v", err)
	}

	// Test invalid layout combinations
	t.Run("conflicting layout modes", func(t *testing.T) {
		// Try to set both FitKerning and FitSmushing (invalid)
		invalidLayout := FitKerning | FitSmushing
		_, err := Render("x", font, WithLayout(invalidLayout))
		if err == nil {
			t.Error("Render() should return error for conflicting layout modes")
		}
		// Check if the error contains the conflict message
		if !strings.Contains(err.Error(), "layout conflict") || !strings.Contains(err.Error(), "multiple fitting modes") {
			t.Errorf("Render() error = %v, should indicate layout conflict", err)
		}
	})

	// Test valid layout with rules
	t.Run("valid smushing with rules", func(t *testing.T) {
		validLayout := FitSmushing | RuleEqualChar | RuleBigX
		output, err := Render("x", font, WithLayout(validLayout))
		// Skip if renderer not implemented yet
		if err != nil && strings.Contains(err.Error(), "not yet implemented") {
			t.Skip("Renderer not yet implemented")
		}
		if err != nil {
			t.Errorf("Render() with valid smushing layout error = %v", err)
		}
		if output == "" {
			t.Error("Render() with valid smushing layout returned empty output")
		}
	})
}
