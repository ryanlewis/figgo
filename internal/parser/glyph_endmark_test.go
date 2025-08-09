package parser

import (
	"strings"
	"testing"
)

// TestParseGlyphs_EnhancedEndmarkDetection tests advanced endmark handling
// as per FIGfont spec (lines 943-948): "The FIGdriver will eliminate the
// last block of consecutive equal characters from each line"
func TestParseGlyphs_EnhancedEndmarkDetection(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		validate    func(t *testing.T, f *Font)
		errContains string
		wantErr     bool
	}{
		{
			name: "triple_endmarks",
			input: `flf2a@ 3 3 10 0 0
test@@@
data@@@
end!@@@
`,
			validate: func(t *testing.T, f *Font) {
				ValidateSpace(t, f, []string{"test", "data", "end!"})
			},
		},
		{
			name: "unusual_endmark_hash",
			input: `flf2a@ 2 2 10 0 0
test##
data##
more##
stuf##
`,
			validate: func(t *testing.T, f *Font) {
				ValidateSpace(t, f, []string{"test", "data"})
				ValidateChar(t, f, '!', []string{"more", "stuf"}, "!")
			},
		},
		{
			name: "unusual_endmark_number",
			input: `flf2a@ 2 2 10 0 0
test1
data11
`,
			validate: ValidateTestDataSpace,
		},
		{
			name: "unusual_endmark_letter",
			input: `flf2a@ 2 2 10 0 0
testZ
dataZZ
`,
			validate: ValidateTestDataSpace,
		},
		{
			name:     "unicode_endmark_emoji",
			input:    "flf2a@ 2 2 12 0 0\ntest😀\ndata😀😀\n",
			validate: ValidateTestDataSpace,
		},
		{
			name: "five_consecutive_endmarks",
			input: `flf2a@ 2 2 10 0 0
test@@@@@
data@@@@@@
`,
			validate: ValidateTestDataSpace,
		},
		{
			name: "endmark_in_middle_of_line",
			input: `flf2a@ 2 2 10 0 0
te@st@
da@ta@@
`,
			validate: func(t *testing.T, f *Font) {
				ValidateEndmarkStripping(t, f, "te@st", "da@ta")
			},
		},
		{
			name: "inconsistent_endmark_error",
			input: `flf2a@ 3 3 10 0 0
test@
data#
end@@
`,
			wantErr:     true,
			errContains: "inconsistent", // Width inconsistency now detected
		},
		{
			name: "no_endmark_becomes_wrong_parse",
			input: `flf2a@ 2 2 10 0 0
test
data
`,
			// With the new approach, this will strip 't' and 'a' as single-char runs,
			// resulting in "tes" and "dat". This is wrong but the font is malformed.
			// The parser can't know there's no endmark without prior knowledge.
			validate: func(t *testing.T, f *Font) {
				ValidateEndmarkStripping(t, f, "tes", "dat")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			font, err := Parse(r)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Parse() error = nil, want error containing %q", tt.errContains)
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Parse() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("Parse() unexpected error = %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, font)
			}
		})
	}
}

// TestParseGlyphs_EndmarkVariation verifies that different glyphs can use different endmarks
func TestParseGlyphs_EndmarkVariation(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		validate    func(t *testing.T, f *Font)
		errContains string
		wantErr     bool
	}{
		{
			name: "changing_endmark_between_glyphs_is_allowed",
			input: `flf2a@ 2 2 10 0 0
test@
data@@
next#
line##
`,
			wantErr: false, // This is actually allowed per spec
			validate: func(t *testing.T, f *Font) {
				// First glyph strips @
				space := f.Characters[' ']
				if space[0] != "test" || space[1] != "data" {
					t.Errorf("Space = %v, want [test, data]", space)
				}
				// Second glyph strips #
				excl := f.Characters['!']
				if excl[0] != "next" || excl[1] != "line" {
					t.Errorf("! = %v, want [next, line]", excl)
				}
			},
		},
		{
			name: "consistent_endmark_across_glyphs",
			input: `flf2a@ 2 2 10 0 0
test@
data@@
next@
line@@
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			font, err := Parse(r)

			switch {
			case tt.wantErr && err == nil:
				t.Errorf("Parse() error = nil, want error containing %q", tt.errContains)
				return
			case tt.wantErr && !strings.Contains(err.Error(), tt.errContains):
				t.Errorf("Parse() error = %v, want error containing %q", err, tt.errContains)
			case !tt.wantErr && err != nil:
				t.Errorf("Parse() unexpected error = %v", err)
			case !tt.wantErr && tt.validate != nil:
				tt.validate(t, font)
			}
		})
	}
}
