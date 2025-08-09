package parser

import (
	"fmt"
	"strings"
	"testing"
)

// Test Helpers for FIGfont Parser Testing
//
// This file provides a comprehensive set of validation functions for testing
// FIGfont parsing functionality. The helpers follow consistent patterns and
// provide clear error messages for debugging test failures.
//
// Common patterns:
//   - ValidateX functions perform validation and fail tests on mismatch
//   - MustX functions return values or fail tests if not available
//   - Functions starting with a verb describe their primary action
//
// Usage example:
//   font := parseTestFont(t, fontData)
//   ValidateCharCount(t, font, 102)
//   ValidateAllASCIIChars(t, font)

// Standard test content constants used across parser tests.
// These provide consistent test data for validation functions.
const (
	testContent = "test" // Standard test content for space character line 0
	dataContent = "data" // Standard test content for space character line 1
	endContent  = "end!" // Alternative test content for additional validation
)

// CharValidation represents a character glyph validation request for batch operations.
// Used with ValidateMultipleChars to validate several characters at once.
type CharValidation struct {
	Char     rune     // Unicode code point of the character to validate
	Name     string   // Human-readable name for the character (used in error messages)
	Expected []string // Expected glyph lines for this character
}

// =============================================================================
// Character Glyph Validation Functions
// =============================================================================

// ValidateChar validates that a character exists in the font and matches expected glyph lines.
// The test fails if the character is missing, has wrong number of lines, or any line content differs.
// Parameters:
//   - t: testing context
//   - f: parsed font to validate
//   - char: Unicode code point to validate
//   - expected: slice of expected glyph lines
//   - charName: human-readable character name for error messages
func ValidateChar(t *testing.T, f *Font, char rune, expected []string, charName string) {
	t.Helper()
	glyph, exists := f.Characters[char]
	if !exists {
		t.Fatalf("%s character (%d) not found", charName, char)
	}
	if len(glyph) != len(expected) {
		t.Errorf("%s should have %d lines, got %d", charName, len(expected), len(glyph))
		return
	}
	for i, expectedLine := range expected {
		if glyph[i] != expectedLine {
			t.Errorf("%s line %d = %q, want %q", charName, i, glyph[i], expectedLine)
		}
	}
}

// ValidateSpace validates the space character (ASCII 32) with expected glyph lines.
// This is a convenience wrapper around ValidateChar for the commonly-tested space character.
func ValidateSpace(t *testing.T, f *Font, expected []string) {
	t.Helper()
	ValidateChar(t, f, ' ', expected, "space")
}

// ValidateMultipleChars validates multiple character glyphs in a single call using batch validation.
// This is efficient for testing several characters with different expected content.
//
// Example usage:
//
//	ValidateMultipleChars(t, font, []CharValidation{
//	  {Char: ' ', Name: "space", Expected: []string{"  ", "  "}},
//	  {Char: '!', Name: "exclamation", Expected: []string{" |", " |"}},
//	})
func ValidateMultipleChars(t *testing.T, f *Font, validations []CharValidation) {
	t.Helper()
	for _, v := range validations {
		ValidateChar(t, f, v.Char, v.Expected, v.Name)
	}
}

// MustGetChar returns a character's glyph lines or fails the test if the character doesn't exist.
// Use this when you need the glyph data for further processing and the character must exist.
// Prefer ValidateChar when you want to validate the glyph content.
func MustGetChar(t *testing.T, f *Font, char rune) []string {
	t.Helper()
	glyph, exists := f.Characters[char]
	if !exists {
		t.Fatalf("Character %c (%d) not found in font", char, char)
	}
	return glyph
}

// ValidateCharExists checks that a character exists in the font and returns its glyph lines.
// Similar to MustGetChar but with a custom character name for better error messages.
// The test fails with a descriptive message if the character is missing.
func ValidateCharExists(t *testing.T, f *Font, char rune, charName string) []string {
	t.Helper()
	glyph, exists := f.Characters[char]
	if !exists {
		t.Fatalf("%s character (%d) not found", charName, char)
	}
	return glyph
}

// ValidateEmptyChar validates that all lines in a character's glyph are empty strings.
// If the character doesn't exist, logs a message rather than failing (useful for optional characters).
// Use this for testing empty FIGcharacters or characters that should have zero-width glyphs.
func ValidateEmptyChar(t *testing.T, f *Font, char rune, charName string) {
	t.Helper()
	if glyph, exists := f.Characters[char]; exists {
		for i, line := range glyph {
			if line != "" {
				t.Errorf("%s line %d should be empty, got %q", charName, i, line)
			}
		}
	} else {
		t.Logf("%s character (%d) not found (may be expected)", charName, char)
	}
}

// =============================================================================
// Specialized Validation Functions
// =============================================================================

// ValidateTestDataSpace validates the space character contains standard test content.
// Expects the space glyph to have exactly 2 lines: "test" and "data".
// This is a common pattern in parser tests for validating basic font parsing.
func ValidateTestDataSpace(t *testing.T, f *Font) {
	t.Helper()
	ValidateSpace(t, f, []string{testContent, dataContent})
}

// ValidateEndmarkStripping validates that endmark characters are correctly removed from glyph lines.
// This function specifically tests the space character (which is often used in endmark tests)
// and expects it to have exactly 2 lines with the provided content.
// Use this to verify that trailing endmark characters are properly stripped during parsing.
func ValidateEndmarkStripping(t *testing.T, f *Font, expectedLine0, expectedLine1 string) {
	t.Helper()
	space := MustGetChar(t, f, ' ')
	if space[0] != expectedLine0 {
		t.Errorf("Space line 0 = %q, want %q", space[0], expectedLine0)
	}
	if space[1] != expectedLine1 {
		t.Errorf("Space line 1 = %q, want %q", space[1], expectedLine1)
	}
}

// =============================================================================
// Font Structure Validation Functions
// =============================================================================

// ValidateCharCount validates that the font contains exactly the expected number of characters.
// This is useful for testing that fonts have the correct total character count,
// such as 95 ASCII characters or 102 characters including German extensions.
func ValidateCharCount(t *testing.T, f *Font, expected int) {
	t.Helper()
	actual := len(f.Characters)
	if actual != expected {
		t.Errorf("Font has %d characters, want %d", actual, expected)
	}
}

// ValidateAllASCIIChars validates that all printable ASCII characters (33-126) exist in the font.
// This covers all required ASCII characters from '!' to '~' but excludes the space character (32).
// FIGfont specification requires these characters to be present in compliant fonts.
func ValidateAllASCIIChars(t *testing.T, f *Font) {
	t.Helper()
	for i := rune(33); i <= 126; i++ {
		if _, exists := f.Characters[i]; !exists {
			t.Errorf("Required ASCII character %d (%c) not found", i, i)
		}
	}
}

// ValidateASCIICharsHaveContent validates that ASCII characters contain expected content.
// This function checks that ASCII characters 33-126 have glyphs with at least 2 lines
// and that both lines match the expected content string. Used for testing font generation
// where all ASCII characters have identical content.
//
// Parameters:
//   - expectedContent: the string that should appear in both line 0 and line 1 of each ASCII character
func ValidateASCIICharsHaveContent(t *testing.T, f *Font, expectedContent string) {
	t.Helper()
	for i := rune(33); i <= 126; i++ {
		if glyph, exists := f.Characters[i]; exists {
			if len(glyph) >= 2 && (glyph[0] != expectedContent || glyph[1] != expectedContent) {
				t.Errorf("ASCII char %d should have content %q, got %v", i, expectedContent, glyph)
			}
		}
	}
}

// =============================================================================
// German Character Validation Functions
// =============================================================================

// germanChars contains the 7 required German characters as specified in FIGfont v2.
// These are the additional characters beyond ASCII 32-126 that complete the 102-character set:
// Ä(196), Ö(214), Ü(220), ä(228), ö(246), ü(252), ß(223)
var germanChars = []rune{196, 214, 220, 228, 246, 252, 223}

// ValidateGermanChar validates a specific German character with expected glyph content.
// This is a convenience wrapper around ValidateChar that automatically prefixes the character
// name with "German" for clearer error messages.
func ValidateGermanChar(t *testing.T, f *Font, char rune, expectedLines []string, charName string) {
	t.Helper()
	ValidateChar(t, f, char, expectedLines, fmt.Sprintf("German %s", charName))
}

// ValidateAllGermanChars validates that all 7 required German characters exist in the font.
// These characters (Ä, Ö, Ü, ä, ö, ü, ß) are required by the FIGfont v2 specification
// for fonts that support the complete 102-character set.
func ValidateAllGermanChars(t *testing.T, f *Font) {
	t.Helper()
	for _, char := range germanChars {
		if _, exists := f.Characters[char]; !exists {
			t.Errorf("Required German character %d not found", char)
		}
	}
}

// ValidateGermanCharsEmpty validates that all German characters exist but have empty glyph content.
// This is useful for testing fonts where German characters are present but intentionally empty.
// Uses ValidateEmptyChar which logs missing characters rather than failing the test.
func ValidateGermanCharsEmpty(t *testing.T, f *Font) {
	t.Helper()
	for _, char := range germanChars {
		ValidateEmptyChar(t, f, char, fmt.Sprintf("German char %d", char))
	}
}

// ValidateGermanCharsAbsent validates that no German characters are present in the font.
// This is useful for testing partial fonts that only support ASCII characters 32-126
// and should not contain the extended German character set.
func ValidateGermanCharsAbsent(t *testing.T, f *Font) {
	t.Helper()
	for _, char := range germanChars {
		if _, exists := f.Characters[char]; exists {
			t.Errorf("German character %d should not be present in ASCII-only font", char)
		}
	}
}

// =============================================================================
// Test Data Generation Functions
// =============================================================================

// GenerateFontWithDeutschChars generates a complete test font containing all 102 FIGfont v2 required characters.
// This includes ASCII characters 32-126 (95 characters) plus German characters 196,214,220,228,246,252,223 (7 characters).
//
// The generated font has the following structure:
//   - Header: "flf2a@ 2 2 10 0 0" (height=2, baseline=2, maxlength=10, oldlayout=0, comments=0)
//   - Space character (32): Two lines of "  " (2 spaces each)
//   - ASCII characters (33-126): Two lines of "X" each
//   - German characters (196,214,220,228,246,252,223): Two lines of "G" each
//   - All characters use "@" as endmark
//
// Returns a complete FIGfont file as a string suitable for parsing with Parse().
//
// Example usage:
//
//	fontData := GenerateFontWithDeutschChars()
//	font, err := Parse(strings.NewReader(fontData))
//	// font now contains all 102 characters
func GenerateFontWithDeutschChars() string {
	var sb strings.Builder
	sb.WriteString("flf2a@ 2 2 10 0 0\n")

	// ASCII characters 32-126 (95 characters)
	for i := 32; i <= 126; i++ {
		if i == 32 {
			sb.WriteString("  @@\n  @@\n") // Space character: 2 spaces per line
		} else {
			sb.WriteString("X@@\nX@@\n") // Other ASCII: "X" content per line
		}
	}

	// German characters (7 characters): Ä(196), Ö(214), Ü(220), ä(228), ö(246), ü(252), ß(223)
	for range 7 {
		sb.WriteString("G@@\nG@@\n") // German chars: "G" content per line
	}

	return sb.String()
}
