package renderer

import (
	"strings"
	"testing"

	"github.com/ryanlewis/figgo/internal/common"
	"github.com/ryanlewis/figgo/internal/parser"
)

// createSmushingTestFont creates a simple font optimized for testing smushing rules
func createSmushingTestFont() *parser.Font {
	return &parser.Font{
		Hardblank:      '$',
		Height:         2,
		Baseline:       1,
		MaxLength:      10,
		OldLayout:      0,
		PrintDirection: 0,
		FullLayout: common.FitSmushing | common.RuleEqualChar | common.RuleUnderscore |
			common.RuleHierarchy | common.RuleOppositePair | common.RuleBigX | common.RuleHardblank,
		FullLayoutSet: true,
		Characters: map[rune][]string{
			' ':  {"  ", "  "},
			'#':  {"##", "##"},
			'A':  {"AA", "AA"},
			'_':  {"__", "__"},
			'|':  {"||", "||"},
			'/':  {"//", "//"},
			'\\': {"\\\\", "\\\\"},
			'[':  {"[[", "[["},
			']':  {"]]", "]]"},
			'{':  {"{{", "{{"},
			'}':  {"}}", "}}"},
			'(':  {"((", "(("},
			')':  {"))", "))"},
			'<':  {"<<", "<<"},
			'>':  {">>", ">>"},
			'+':  {"++", "++"},
			'*':  {"**", "**"},
			'$':  {"$$", "$$"}, // Hardblank character
			'?':  {"??", "??"}, // Replacement for unknown
		},
	}
}

// TestRenderSmushingIntegration tests the full smushing pipeline
func TestRenderSmushingIntegration(t *testing.T) {
	font := createSmushingTestFont()

	tests := []struct {
		name   string
		text   string
		layout int
		want   string
		desc   string
	}{
		// Rule 1: Equal Character
		{
			name:   "rule1_equal_char",
			text:   "##",
			layout: common.FitSmushing | common.RuleEqualChar,
			want:   "##\n##",
			desc:   "Equal character rule: # + # = #",
		},
		{
			name:   "rule1_equal_A",
			text:   "AA",
			layout: common.FitSmushing | common.RuleEqualChar,
			want:   "AA\nAA",
			desc:   "Equal character rule: A + A = A",
		},

		// Rule 2: Underscore
		{
			name:   "rule2_underscore_pipe",
			text:   "_|",
			layout: common.FitSmushing | common.RuleUnderscore,
			want:   "||\n||",
			desc:   "Underscore rule: _ + | = |",
		},
		{
			name:   "rule2_bracket_underscore",
			text:   "[_",
			layout: common.FitSmushing | common.RuleUnderscore,
			want:   "[[\n[[",
			desc:   "Underscore rule: [ + _ = [",
		},

		// Rule 3: Hierarchy
		{
			name:   "rule3_pipe_wins",
			text:   "/|",
			layout: common.FitSmushing | common.RuleHierarchy,
			want:   "||\n||",
			desc:   "Hierarchy: | > /",
		},
		{
			name:   "rule3_slash_wins",
			text:   "[/",
			layout: common.FitSmushing | common.RuleHierarchy,
			want:   "//\n//",
			desc:   "Hierarchy: / > [",
		},

		// Rule 4: Opposite Pairs
		{
			name:   "rule4_brackets",
			text:   "[]",
			layout: common.FitSmushing | common.RuleOppositePair,
			want:   "||\n||",
			desc:   "Opposite pair: [ + ] = |",
		},
		{
			name:   "rule4_parens_reverse",
			text:   ")(",
			layout: common.FitSmushing | common.RuleOppositePair,
			want:   "||\n||",
			desc:   "Opposite pair: ) + ( = |",
		},

		// Rule 5: Big X
		{
			name:   "rule5_slash_backslash",
			text:   "/\\",
			layout: common.FitSmushing | common.RuleBigX,
			want:   "||\n||",
			desc:   "Big X: / + \\ = |",
		},
		{
			name:   "rule5_greater_less",
			text:   "><",
			layout: common.FitSmushing | common.RuleBigX,
			want:   "XX\nXX",
			desc:   "Big X: > + < = X",
		},

		// Rule 6: Hardblank
		{
			name:   "rule6_hardblank",
			text:   "$$",
			layout: common.FitSmushing | common.RuleHardblank,
			want:   "  \n  ", // Hardblanks replaced with spaces after composition
			desc:   "Hardblank: $ + $ = $ (then replaced with space)",
		},

		// Precedence tests
		{
			name:   "precedence_equal_over_hierarchy",
			text:   "||",
			layout: common.FitSmushing | common.RuleEqualChar | common.RuleHierarchy,
			want:   "||\n||",
			desc:   "Equal char (rule 1) takes precedence over hierarchy",
		},
		{
			name:   "precedence_underscore_over_hierarchy",
			text:   "_/",
			layout: common.FitSmushing | common.RuleUnderscore | common.RuleHierarchy,
			want:   "//\n//",
			desc:   "Underscore (rule 2) takes precedence over hierarchy",
		},

		// Universal smushing (no controlled rules)
		{
			name:   "universal_space_visible",
			text:   " A",
			layout: common.FitSmushing, // No controlled rules
			want:   "AA\nAA",
			desc:   "Universal: space + A = A",
		},
		{
			name:   "universal_visible_space",
			text:   "A ",
			layout: common.FitSmushing, // No controlled rules
			want:   "AA\nAA",
			desc:   "Universal: A + space = A",
		},
		{
			name:   "universal_no_smush_visible_collision",
			text:   "+*",
			layout: common.FitSmushing, // No controlled rules
			want:   "++**\n++**",
			desc:   "Universal: + + * cannot smush (no overlap)",
		},

		// Fallback to kerning
		{
			name:   "no_smush_fallback_kerning",
			text:   "+*",
			layout: common.FitSmushing | common.RuleEqualChar, // Has rules but none match
			want:   "++**\n++**",
			desc:   "No matching rule: fallback to kerning",
		},

		// Multiple character tests
		{
			name:   "multi_char_smushing",
			text:   "[]{}",
			layout: common.FitSmushing | common.RuleOppositePair,
			want:   "||||\n||||",
			desc:   "Multiple opposite pairs",
		},
		{
			name:   "multi_char_mixed_rules",
			text:   "_|/\\",
			layout: common.FitSmushing | common.RuleUnderscore | common.RuleBigX,
			want:   "||||\n||||",
			desc:   "Mixed rules: underscore then Big X",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &Options{
				Layout: tt.layout,
			}

			got, err := Render(tt.text, font, opts)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("%s\nRender() =\n%s\nwant =\n%s", tt.desc, got, tt.want)
			}
		})
	}
}

// TestRenderSmushingWithReplacement tests unknown rune replacement before composition
func TestRenderSmushingWithReplacement(t *testing.T) {
	font := createSmushingTestFont()

	// Test with non-ASCII character that should be replaced with '?'
	text := "A\x00" // A followed by null character (not in ASCII 32-126)
	opts := &Options{
		Layout: common.FitSmushing | common.RuleEqualChar,
	}

	got, err := Render(text, font, opts)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	// Should render as A followed by ? (replacement)
	want := "AA??\nAA??"
	if got != want {
		t.Errorf("Render with replacement =\n%s\nwant =\n%s", got, want)
	}
}

// TestRenderSmushingPrintDirection tests RTL rendering with smushing
func TestRenderSmushingPrintDirection(t *testing.T) {
	font := createSmushingTestFont()

	tests := []struct {
		name string
		text string
		dir  int
		want string
	}{
		{
			name: "RTL with smushing",
			text: "[]",
			dir:  1,        // RTL
			want: "||\n||", // Same result due to how smushing works
		},
		{
			name: "RTL no smushing",
			text: "+*",
			dir:  1,            // RTL
			want: "**++\n**++", // Reversed order
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &Options{
				Layout:         common.FitSmushing | common.RuleOppositePair,
				PrintDirection: &tt.dir,
			}

			got, err := Render(tt.text, font, opts)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("Render() =\n%s\nwant =\n%s", got, tt.want)
			}
		})
	}
}

// TestPickLayoutWithSmushing tests layout selection for smushing mode
func TestPickLayoutWithSmushing(t *testing.T) {
	tests := []struct {
		name       string
		font       *parser.Font
		opts       *Options
		wantLayout int
		wantErr    bool
	}{
		{
			name: "font with FullLayout smushing",
			font: &parser.Font{
				FullLayout:    common.FitSmushing | common.RuleEqualChar,
				FullLayoutSet: true,
			},
			opts:       nil,
			wantLayout: common.FitSmushing | common.RuleEqualChar,
		},
		{
			name: "options override font",
			font: &parser.Font{
				FullLayout:    common.FitKerning,
				FullLayoutSet: true,
			},
			opts: &Options{
				Layout: common.FitSmushing | common.RuleBigX,
			},
			wantLayout: common.FitSmushing | common.RuleBigX,
		},
		{
			name: "oldlayout positive means smushing",
			font: &parser.Font{
				OldLayout:     1,
				FullLayoutSet: false,
			},
			opts:       nil,
			wantLayout: common.FitSmushing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pickLayout(tt.font, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("pickLayout() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantLayout {
				t.Errorf("pickLayout() = %d, want %d", got, tt.wantLayout)
			}
		})
	}
}

// TestRenderSmushingComplex tests complex multi-column overlap scenarios
func TestRenderSmushingComplex(t *testing.T) {
	// Create a font with glyphs designed to test multi-column overlaps
	// Using simple two-char wide glyphs where smushing rules will apply
	font := &parser.Font{
		Hardblank:      '$',
		Height:         2,
		Baseline:       1,
		MaxLength:      10,
		OldLayout:      0,
		PrintDirection: 0,
		FullLayout:     common.FitSmushing | common.RuleOppositePair | common.RuleBigX,
		FullLayoutSet:  true,
		Characters: map[rune][]string{
			'A': {"/]", "]/"},   // Ends with /] pattern
			'B': {"\\[", "[\\"}, // Starts with \[ pattern
		},
	}

	tests := []struct {
		name string
		text string
		want string
		desc string
	}{
		{
			name: "multi_column_overlap",
			text: "AB",
			want: "||", // After Big X and Opposite pairs rules
			desc: "Two columns overlap: /\\ → | (Big X), ][ → | (Opposites)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &Options{
				Layout: font.FullLayout,
			}

			got, err := Render(tt.text, font, opts)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			// Split into lines for easier comparison
			gotLines := strings.Split(strings.TrimSpace(got), "\n")

			// Both rows should have the same result after smushing
			if len(gotLines) != 2 || gotLines[0] != "||" || gotLines[1] != "||" {
				t.Errorf("%s\nRender() =\n%s\nwant =\n|| (both rows)", tt.desc, got)
			}
		})
	}
}
