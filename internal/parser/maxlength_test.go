package parser

import (
	"strings"
	"testing"
)

// TestParseGlyph_MaxLengthValidation tests that MaxLength generates warnings (not errors)
func TestParseGlyph_MaxLengthValidation(t *testing.T) { //nolint:gocognit // Test function with many test cases
	tests := []struct {
		expectWarning bool
		name          string
		input         string
		warnContains  string
	}{
		{
			name: "within_maxlength",
			input: `flf2a@ 2 2 10 0 0
hello@@
world@@
`,
			expectWarning: false,
		},
		{
			name: "exactly_maxlength",
			input: `flf2a@ 2 2 7 0 0
hello@@
world@@
`,
			expectWarning: false,
		},
		{
			name: "exceeds_maxlength_after_stripping",
			input: `flf2a@ 2 2 3 0 0
hello@@
world@@
`,
			expectWarning: true,
			warnContains:  "exceeds MaxLength",
		},
		{
			name: "maxlength_with_long_endmark_run",
			input: `flf2a@ 2 2 15 0 0
test@@@@@@@@@@
data@@@@@@@@@@
`,
			expectWarning: false,
		},
		{
			name: "maxlength_exceeded_with_long_content",
			input: `flf2a@ 2 2 10 0 0
` + strings.Repeat("x", 20) + `@@
` + strings.Repeat("y", 20) + `@@
`,
			expectWarning: true,
			warnContains:  "exceeds MaxLength",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			font, err := Parse(r)

			// MaxLength violations are now warnings, not errors
			if err != nil {
				t.Fatalf("Parse() unexpected error = %v", err)
			}

			if tt.expectWarning {
				if font == nil || len(font.Warnings) == 0 {
					t.Fatalf("Parse() expected warning containing %q, got no warnings", tt.warnContains)
				}
				found := false
				for _, w := range font.Warnings {
					if strings.Contains(w, tt.warnContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Parse() warnings = %v, want warning containing %q", font.Warnings, tt.warnContains)
				}
			} else if font != nil && len(font.Warnings) > 0 {
				// Check for unexpected MaxLength warnings
				for _, w := range font.Warnings {
					if strings.Contains(w, "exceeds MaxLength") {
						t.Errorf("Parse() unexpected MaxLength warning: %v", w)
					}
				}
			}
		})
	}
}

// TestParseGlyph_MaxLengthWithComments tests MaxLength validation doesn't affect comments
func TestParseGlyph_MaxLengthWithComments(t *testing.T) {
	// Comments can be longer than MaxLength - only glyph lines are validated
	input := `flf2a@ 2 2 10 0 1
This is a very long comment line that exceeds the MaxLength of 10 characters
hello@@
world@@
`
	r := strings.NewReader(input)
	font, err := Parse(r)
	if err != nil {
		t.Fatalf("Parse() unexpected error = %v", err)
	}

	// Verify comment was preserved
	if len(font.Comments) != 1 {
		t.Errorf("Expected 1 comment, got %d", len(font.Comments))
	}
	if !strings.Contains(font.Comments[0], "very long comment") {
		t.Errorf("Comment not preserved correctly: %q", font.Comments[0])
	}
}
