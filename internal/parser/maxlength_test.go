package parser

import (
	"strings"
	"testing"
)

// TestParseGlyph_MaxLengthValidation tests that MaxLength is properly enforced
func TestParseGlyph_MaxLengthValidation(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errContains string
	}{
		{
			name: "within_maxlength",
			input: `flf2a@ 2 2 10 0 0
hello@@
world@@
`,
			expectError: false,
		},
		{
			name: "exactly_maxlength",
			input: `flf2a@ 2 2 7 0 0
hello@@
world@@
`,
			expectError: false,
		},
		{
			name: "exceeds_maxlength",
			input: `flf2a@ 2 2 5 0 0
hello@@
world@@
`,
			expectError: true,
			errContains: "exceeds MaxLength",
		},
		{
			name: "maxlength_with_long_endmark_run",
			input: `flf2a@ 2 2 15 0 0
test@@@@@@@@@@
data@@@@@@@@@@
`,
			expectError: false,
		},
		{
			name: "maxlength_exceeded_with_long_content",
			input: `flf2a@ 2 2 10 0 0
` + strings.Repeat("x", 20) + `@@
` + strings.Repeat("y", 20) + `@@
`,
			expectError: true,
			errContains: "exceeds MaxLength",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			_, err := Parse(r)

			if tt.expectError {
				if err == nil {
					t.Fatalf("Parse() expected error containing %q, got nil", tt.errContains)
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Parse() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Fatalf("Parse() unexpected error = %v", err)
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