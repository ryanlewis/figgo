package parser

import (
	"strings"
	"testing"
)

const testSignature = "flf2a"

func TestParseHeader(t *testing.T) { //nolint:gocognit // Test function with many test cases
	tests := []struct {
		validate    func(t *testing.T, f *Font)
		name        string
		input       string
		errContains string
		wantErr     bool
	}{
		{
			name: "valid_standard_header",
			input: `flf2a$ 8 6 14 15 2 1 24463 229
Standard FIGfont
More comments here
`,
			validate: func(t *testing.T, f *Font) {
				if f.Signature != testSignature {
					t.Errorf("Signature = %q, want %q", f.Signature, testSignature)
				}
				if f.Hardblank != '$' {
					t.Errorf("Hardblank = %q, want %q", f.Hardblank, '$')
				}
				if f.Height != 8 {
					t.Errorf("Height = %d, want %d", f.Height, 8)
				}
				if f.Baseline != 6 {
					t.Errorf("Baseline = %d, want %d", f.Baseline, 6)
				}
				if f.MaxLength != 14 {
					t.Errorf("MaxLength = %d, want %d", f.MaxLength, 14)
				}
				if f.OldLayout != 15 {
					t.Errorf("OldLayout = %d, want %d", f.OldLayout, 15)
				}
				if f.CommentLines != 2 {
					t.Errorf("CommentLines = %d, want %d", f.CommentLines, 2)
				}
				if f.PrintDirection != 1 {
					t.Errorf("PrintDirection = %d, want %d", f.PrintDirection, 1)
				}
				if f.FullLayout != 24463 {
					t.Errorf("FullLayout = %d, want %d", f.FullLayout, 24463)
				}
				if f.CodetagCount != 229 {
					t.Errorf("CodetagCount = %d, want %d", f.CodetagCount, 229)
				}
			},
		},
		{
			name: "minimal_header_old_layout",
			input: `flf2a# 6 5 10 -1 2
Comment 1
Comment 2
`,
			validate: func(t *testing.T, f *Font) {
				if f.Hardblank != '#' {
					t.Errorf("Hardblank = %q, want %q", f.Hardblank, '#')
				}
				if f.Height != 6 {
					t.Errorf("Height = %d, want %d", f.Height, 6)
				}
				if f.Baseline != 5 {
					t.Errorf("Baseline = %d, want %d", f.Baseline, 5)
				}
				if f.MaxLength != 10 {
					t.Errorf("MaxLength = %d, want %d", f.MaxLength, 10)
				}
				if f.OldLayout != -1 {
					t.Errorf("OldLayout = %d, want %d", f.OldLayout, -1)
				}
				if f.CommentLines != 2 {
					t.Errorf("CommentLines = %d, want %d", f.CommentLines, 2)
				}
				// Optional fields should be zero
				if f.PrintDirection != 0 {
					t.Errorf("PrintDirection = %d, want %d", f.PrintDirection, 0)
				}
				if f.FullLayout != 0 {
					t.Errorf("FullLayout = %d, want %d", f.FullLayout, 0)
				}
			},
		},
		{
			name:  "header_with_crlf",
			input: "flf2a$ 8 6 14 15 1 1 24463 229\r\nComment line\r\n",
			validate: func(t *testing.T, f *Font) {
				if f.Signature != testSignature {
					t.Errorf("Signature = %q, want %q", f.Signature, testSignature)
				}
				if f.Height != 8 {
					t.Errorf("Height = %d, want %d", f.Height, 8)
				}
			},
		},
		{
			name:  "header_with_trailing_spaces",
			input: "flf2a$ 8 6 14 15 1 1 24463 229   \nComment line\n",
			validate: func(t *testing.T, f *Font) {
				if f.Signature != testSignature {
					t.Errorf("Signature = %q, want %q", f.Signature, testSignature)
				}
				if f.Height != 8 {
					t.Errorf("Height = %d, want %d", f.Height, 8)
				}
			},
		},
		{
			name: "header_with_extra_fields",
			input: `flf2a$ 8 6 14 15 1 1 24463 229 extra fields ignored
Comment line
`,
			validate: func(t *testing.T, f *Font) {
				if f.Signature != testSignature {
					t.Errorf("Signature = %q, want %q", f.Signature, testSignature)
				}
				if f.CodetagCount != 229 {
					t.Errorf("CodetagCount = %d, want %d", f.CodetagCount, 229)
				}
			},
		},
		{
			name: "invalid_signature",
			input: `badheader$ 8 6 14 15 16
Comment
`,
			wantErr:     true,
			errContains: "invalid signature",
		},
		{
			name: "missing_hardblank",
			input: `flf2a 8 6 14 15 16
Comment
`,
			wantErr:     true,
			errContains: "invalid signature",
		},
		{
			name: "insufficient_fields",
			input: `flf2a$ 8 6 14
Comment
`,
			wantErr:     true,
			errContains: "insufficient header fields",
		},
		{
			name: "non_numeric_height",
			input: `flf2a$ abc 6 14 15 16
Comment
`,
			wantErr:     true,
			errContains: "invalid height",
		},
		{
			name: "non_numeric_baseline",
			input: `flf2a$ 8 xyz 14 15 16
Comment
`,
			wantErr:     true,
			errContains: "invalid baseline",
		},
		{
			name: "negative_height",
			input: `flf2a$ -8 6 14 15 16
Comment
`,
			wantErr:     true,
			errContains: "height must be positive",
		},
		{
			name: "baseline_exceeds_height",
			input: `flf2a$ 8 10 14 15 16
Comment
`,
			wantErr:     true,
			errContains: "baseline exceeds height",
		},
		{
			name: "negative_maxlength",
			input: `flf2a$ 8 6 -14 15 16
Comment
`,
			wantErr:     true,
			errContains: "maxlength must be positive",
		},
		{
			name: "negative_comment_lines",
			input: `flf2a$ 8 6 14 15 -2
`,
			wantErr:     true,
			errContains: "comment lines must be non-negative",
		},
		{
			name:        "empty_input",
			input:       "",
			wantErr:     true,
			errContains: "empty font data",
		},
		{
			name:        "only_whitespace",
			input:       "   \n   \t   \n",
			wantErr:     true,
			errContains: "empty font data",
		},
		{
			name: "oldlayout_minus_1",
			input: `flf2a$ 8 6 14 -1 2
Comment 1
Comment 2
`,
			validate: func(t *testing.T, f *Font) {
				if f.OldLayout != -1 {
					t.Errorf("OldLayout = %d, want %d", f.OldLayout, -1)
				}
			},
		},
		{
			name: "oldlayout_minus_2",
			input: `flf2a$ 8 6 14 -2 1
Comment
`,
			validate: func(t *testing.T, f *Font) {
				if f.OldLayout != -2 {
					t.Errorf("OldLayout = %d, want %d", f.OldLayout, -2)
				}
			},
		},
		{
			name: "oldlayout_minus_3",
			input: `flf2a$ 8 6 14 -3 1
Comment
`,
			validate: func(t *testing.T, f *Font) {
				if f.OldLayout != -3 {
					t.Errorf("OldLayout = %d, want %d", f.OldLayout, -3)
				}
			},
		},
		{
			name: "print_direction_rtl",
			input: `flf2a$ 8 6 14 15 1 1
Comment
`,
			validate: func(t *testing.T, f *Font) {
				if f.PrintDirection != 1 {
					t.Errorf("PrintDirection = %d, want %d", f.PrintDirection, 1)
				}
			},
		},
		{
			name: "full_layout_with_bits",
			input: `flf2a$ 8 6 14 15 1 0 191
Comment
`,
			validate: func(t *testing.T, f *Font) {
				if f.FullLayout != 191 { // All horizontal smushing rules + smushing mode
					t.Errorf("FullLayout = %d, want %d", f.FullLayout, 191)
				}
			},
		},
		{
			name: "zero_comment_lines",
			input: `flf2a$ 8 6 14 15 0
`,
			validate: func(t *testing.T, f *Font) {
				if f.CommentLines != 0 {
					t.Errorf("CommentLines = %d, want %d", f.CommentLines, 0)
				}
				if len(f.Comments) != 0 {
					t.Errorf("len(Comments) = %d, want %d", len(f.Comments), 0)
				}
			},
		},
		{
			name: "multiple_comment_lines",
			input: `flf2a$ 8 6 14 15 3
Line 1
Line 2
Line 3
`,
			validate: func(t *testing.T, f *Font) {
				if f.CommentLines != 3 {
					t.Errorf("CommentLines = %d, want %d", f.CommentLines, 3)
				}
				if len(f.Comments) != 3 {
					t.Errorf("len(Comments) = %d, want %d", len(f.Comments), 3)
				}
				expectedComments := []string{"Line 1", "Line 2", "Line 3"}
				for i, want := range expectedComments {
					if i >= len(f.Comments) {
						t.Errorf("Missing comment %d", i)
						continue
					}
					if f.Comments[i] != want {
						t.Errorf("Comment[%d] = %q, want %q", i, f.Comments[i], want)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			font, err := ParseHeader(r)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseHeader() error = nil, want error containing %q", tt.errContains)
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ParseHeader() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseHeader() unexpected error = %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, font)
			}
		})
	}
}

// TestParseHeaderEdgeCases tests additional edge cases
func TestParseHeaderEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "header_line_too_long",
			input:   "flf2a$ " + strings.Repeat("9", 10000) + " 6 14 15 1\nComment\n",
			wantErr: true, // Integer overflow
		},
		{
			name:    "unicode_in_comment",
			input:   "flf2a$ 8 6 14 15 1\n日本語コメント\n",
			wantErr: false, // Should accept unicode in comments
		},
		{
			name:    "tabs_as_separators",
			input:   "flf2a$\t8\t6\t14\t15\t1\nComment\n",
			wantErr: false, // Should handle tabs as field separators
		},
		{
			name:    "multiple_spaces_between_fields",
			input:   "flf2a$    8     6    14    15    1\nComment\n",
			wantErr: false, // Should handle multiple spaces
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			_, err := ParseHeader(r)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseHeader() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
