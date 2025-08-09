package parser

import (
	"strings"
	"testing"
)

// TestParseGlyphs_CRLF tests that CRLF line endings are handled correctly
func TestParseGlyphs_CRLF(t *testing.T) {
	input := "flf2a@ 2 2 10 0 0\r\ntest@@\r\ndata@@\r\n"
	r := strings.NewReader(input)
	font, err := Parse(r)
	if err != nil {
		t.Fatalf("Parse() unexpected error = %v", err)
	}

	space := MustGetChar(t, font, ' ')
	if space[0] != testContent {
		t.Errorf("Line 0 = %q, want %q", space[0], testContent)
	}
	if space[1] != dataContent {
		t.Errorf("Line 1 = %q, want %q", space[1], dataContent)
	}
}

// TestParseGlyphs_OnlyEndmarks tests lines that are only endmarks (valid empty rows)
func TestParseGlyphs_OnlyEndmarks(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name: "single_endmark_only",
			input: `flf2a@ 3 3 5 0 0
@@
@@
@@@
`,
			expected: []string{"", "", ""},
		},
		{
			name: "multiple_endmarks_only",
			input: `flf2a@ 3 3 7 0 0
@@@@@
######
$$$$$$$
`,
			expected: []string{"", "", ""},
		},
		{
			name: "mixed_content_and_endmark_only",
			input: `flf2a@ 3 3 10 0 0
test@@
    @@@@
data##
`,
			expected: []string{testContent, "    ", dataContent},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			font, err := Parse(r)
			if err != nil {
				t.Fatalf("Parse() unexpected error = %v", err)
			}

			space := MustGetChar(t, font, ' ')
			for i, expected := range tt.expected {
				if i >= len(space) {
					t.Errorf("Missing line %d", i)
					continue
				}
				if space[i] != expected {
					t.Errorf("Line %d = %q, want %q", i, space[i], expected)
				}
			}
		})
	}
}

// TestParseGlyphs_VeryLongLine tests handling of very long lines
func TestParseGlyphs_VeryLongLine(t *testing.T) {
	// Create a very long line (100KB)
	longContent := strings.Repeat("X", 100*1024)
	input := "flf2a@ 2 2 200000 0 0\n" + longContent + "@@\n" + longContent + "@@\n"

	r := strings.NewReader(input)
	font, err := Parse(r)
	if err != nil {
		t.Fatalf("Parse() unexpected error = %v", err)
	}

	space := MustGetChar(t, font, ' ')
	if !strings.HasPrefix(space[0], strings.Repeat("X", 1000)) {
		t.Error("Long line not parsed correctly")
	}
}

// TestParseHeader_HardblankValidation tests that invalid hardblank characters are rejected
func TestParseHeader_HardblankValidation(t *testing.T) {
	tests := []struct {
		wantErr     bool
		name        string
		input       string
		errContains string
	}{
		{
			name:        "hardblank_space",
			input:       "flf2a  8 6 14 0 0\n",
			wantErr:     true,
			errContains: "invalid hardblank",
		},
		{
			name:        "hardblank_cr",
			input:       "flf2a\r 8 6 14 0 0\n",
			wantErr:     true,
			errContains: "invalid hardblank",
		},
		{
			name:        "hardblank_lf",
			input:       "flf2a\n 8 6 14 0 0\n",
			wantErr:     true,
			errContains: "invalid",
		},
		{
			name:        "hardblank_nul",
			input:       "flf2a\x00 8 6 14 0 0\n",
			wantErr:     true,
			errContains: "invalid hardblank",
		},
		{
			name:    "hardblank_valid_dollar",
			input:   "flf2a$ 8 6 14 0 0\n",
			wantErr: false,
		},
		{
			name:    "hardblank_valid_unicode",
			input:   "flf2a£ 8 6 14 0 0\n",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			r := strings.NewReader(tt.input)
			_, err := ParseHeader(r)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseHeader() error = nil, want error containing %q", tt.errContains)
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ParseHeader() error = %v, want error containing %q", err, tt.errContains)
				}
			} else if err != nil {
				t.Errorf("ParseHeader() unexpected error = %v", err)
			}
		})
	}
}

// TestParseHeader_PrintDirectionValidation tests PrintDirection validation
func TestParseHeader_PrintDirectionValidation(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    int
		expectError bool
	}{
		{
			name:        "print_direction_0",
			input:       "flf2a$ 5 3 10 0 0 0\n",
			expected:    0,
			expectError: false,
		},
		{
			name:        "print_direction_1",
			input:       "flf2a$ 5 3 10 0 0 1\n",
			expected:    1,
			expectError: false,
		},
		{
			name:        "print_direction_negative_error",
			input:       "flf2a$ 5 3 10 0 0 -5\n",
			expected:    0,
			expectError: true, // Should return error
		},
		{
			name:        "print_direction_too_large_error",
			input:       "flf2a$ 5 3 10 0 0 99\n",
			expected:    0,
			expectError: true, // Should return error
		},
		{
			name:        "print_direction_2_error",
			input:       "flf2a$ 5 3 10 0 0 2\n",
			expected:    0,
			expectError: true, // Should return error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			r := strings.NewReader(tt.input)
			font, err := ParseHeader(r)

			if tt.expectError {
				if err == nil {
					t.Fatalf("ParseHeader() expected error for invalid print direction, got nil")
				}
				if !strings.Contains(err.Error(), "invalid print direction") {
					t.Errorf("ParseHeader() error = %v, want error containing 'invalid print direction'", err)
				}
			} else {
				if err != nil {
					t.Fatalf("ParseHeader() unexpected error = %v", err)
				}
				if font.PrintDirection != tt.expected {
					t.Errorf("PrintDirection = %d, want %d", font.PrintDirection, tt.expected)
				}
			}
		})
	}
}

// TestStripTrailingRun_InvalidUTF8 tests the fallback behavior for invalid UTF-8
func TestStripTrailingRun_InvalidUTF8(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantBody    string
		wantEndmark rune
		wantRunLen  int
	}{
		{
			name:        "valid_utf8",
			input:       "test@@",
			wantBody:    testContent,
			wantEndmark: '@',
			wantRunLen:  2,
		},
		{
			name:        "invalid_utf8_at_end",
			input:       "test\xff\xff\xff",
			wantBody:    testContent,
			wantEndmark: rune(0xff),
			wantRunLen:  3,
		},
		{
			name:        "mixed_invalid_utf8",
			input:       "test\xfe\xff\xff",
			wantBody:    "test\xfe",
			wantEndmark: rune(0xff),
			wantRunLen:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, endmark, runLen := stripTrailingRun(tt.input)
			if body != tt.wantBody {
				t.Errorf("body = %q, want %q", body, tt.wantBody)
			}
			if endmark != tt.wantEndmark {
				t.Errorf("endmark = %v, want %v", endmark, tt.wantEndmark)
			}
			if runLen != tt.wantRunLen {
				t.Errorf("runLen = %d, want %d", runLen, tt.wantRunLen)
			}
		})
	}
}

// TestParseHeader_AllValidations tests all header validations together
func TestParseHeader_AllValidations(t *testing.T) {
	tests := []struct {
		wantErr     bool
		name        string
		input       string
		errContains string
	}{
		{
			name:    "all_valid",
			input:   "flf2a$ 5 3 10 0 0\n",
			wantErr: false,
		},
		{
			name:        "height_zero",
			input:       "flf2a$ 0 3 10 0 0\n",
			wantErr:     true,
			errContains: "height must be positive",
		},
		{
			name:        "height_negative",
			input:       "flf2a$ -5 3 10 0 0\n",
			wantErr:     true,
			errContains: "height must be positive",
		},
		{
			name:        "baseline_zero",
			input:       "flf2a$ 5 0 10 0 0\n",
			wantErr:     true,
			errContains: "baseline must be at least 1",
		},
		{
			name:        "baseline_exceeds_height",
			input:       "flf2a$ 5 6 10 0 0\n",
			wantErr:     true,
			errContains: "baseline exceeds height",
		},
		{
			name:        "maxlength_zero",
			input:       "flf2a$ 5 3 0 0 0\n",
			wantErr:     true,
			errContains: "maxlength must be positive",
		},
		{
			name:        "maxlength_negative",
			input:       "flf2a$ 5 3 -10 0 0\n",
			wantErr:     true,
			errContains: "maxlength must be positive",
		},
		{
			name:        "comment_lines_negative",
			input:       "flf2a$ 5 3 10 0 -1\n",
			wantErr:     true,
			errContains: "comment lines must be non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			r := strings.NewReader(tt.input)
			_, err := ParseHeader(r)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseHeader() error = nil, want error containing %q", tt.errContains)
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ParseHeader() error = %v, want error containing %q", err, tt.errContains)
				}
			} else if err != nil {
				t.Errorf("ParseHeader() unexpected error = %v", err)
			}
		})
	}
}
