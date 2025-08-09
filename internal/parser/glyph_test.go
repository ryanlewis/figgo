package parser

import (
	"strings"
	"testing"
)

// generateFullASCIIFont generates a simple font with all ASCII printable characters
func generateFullASCIIFont() string {
	var sb strings.Builder
	// Header: height=1, baseline=1, maxlength=5, oldlayout=0, comments=0
	sb.WriteString("flf2a@ 1 1 5 0 0\n")

	// Generate glyphs for ASCII 32-126
	for r := rune(32); r <= 126; r++ {
		// Simple glyph: just the character itself
		if r == '@' {
			// Special case: @ needs to be doubled
			sb.WriteString("@@@@\n")
		} else {
			sb.WriteString(string(r))
			sb.WriteString("@@\n")
		}
	}

	return sb.String()
}

func TestParseGlyphsEdgeCases(t *testing.T) {
	tests := []struct {
		validate    func(t *testing.T, f *Font)
		name        string
		input       string
		errContains string
		wantErr     bool
	}{
		{
			name:    "crlf_line_endings",
			input:   "flf2a@ 2 2 8 0 0\r\ntest@\r\ndata@@\r\n",
			wantErr: false,
		},
		{
			name:    "mixed_line_endings",
			input:   "flf2a@ 2 2 8 0 0\ntest@\r\ndata@@\n",
			wantErr: false,
		},
		{
			name:    "very_long_glyph_line",
			input:   "flf2a@ 1 1 1000 0 0\n" + strings.Repeat("x", 1000) + "@@\n",
			wantErr: false,
		},
		{
			name: "endmark_at_line_start",
			input: `flf2a@ 2 2 8 0 0
@test@
@data@@
`,
			validate: func(t *testing.T, f *Font) {
				space := f.Characters[' ']
				if space[0] != "@test" {
					t.Errorf("Line 0 = %q, want %q", space[0], "@test")
				}
				if space[1] != "@data" {
					t.Errorf("Line 1 = %q, want %q", space[1], "@data")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			font, err := Parse(r)

			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("Parse() error = %v, want error containing %q", err, tt.errContains)
			}

			if err == nil && tt.validate != nil {
				tt.validate(t, font)
			}
		})
	}
}

func TestGlyphBoundaryConditions(t *testing.T) {
	tests := []struct {
		validate func(t *testing.T, f *Font)
		name     string
		input    string
	}{
		{
			name: "minimum_ascii_32_space",
			input: `flf2a@ 1 1 5 0 0
 @@
`,
			validate: func(t *testing.T, f *Font) {
				if len(f.Characters) != 1 {
					t.Errorf("Expected 1 character, got %d", len(f.Characters))
				}
				if _, exists := f.Characters[' ']; !exists {
					t.Error("Space character (32) not found")
				}
			},
		},
		{
			name: "maximum_ascii_126_tilde",
			input: `flf2a@ 1 1 5 0 0
` + strings.Repeat(" @@\n", 94) + `~@@
`,
			validate: func(t *testing.T, f *Font) {
				if _, exists := f.Characters['~']; !exists {
					t.Error("Tilde character (126) not found")
				}
			},
		},
		{
			name:  "exactly_95_glyphs",
			input: generateFullASCIIFont(),
			validate: func(t *testing.T, f *Font) {
				if len(f.Characters) != 95 {
					t.Errorf("Expected exactly 95 characters, got %d", len(f.Characters))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			font, err := Parse(r)
			if err != nil {
				t.Errorf("Parse() unexpected error = %v", err)
				return
			}
			if tt.validate != nil {
				tt.validate(t, font)
			}
		})
	}
}
