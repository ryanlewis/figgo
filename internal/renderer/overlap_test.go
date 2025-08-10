package renderer

import (
	"testing"

	"github.com/ryanlewis/figgo/internal/common"
	"github.com/ryanlewis/figgo/internal/parser"
)

// TestOverlapSelection tests the overlap selection algorithm per issue #14
func TestOverlapSelection(t *testing.T) {
	const hardblank = '$'

	tests := []struct {
		name        string
		lines       [][]byte           // Current composed lines
		glyph       []string           // New glyph to add
		layout      int                // Layout bitmask
		trims       []parser.GlyphTrim // Precomputed trims (optional)
		wantOverlap int                // Expected overlap
		description string             // Test case explanation
	}{
		// Basic cases from issue #14 compliance matrix
		{
			name:        "rule2_one_column_pass",
			lines:       [][]byte{{'|'}},
			glyph:       []string{"_"},
			layout:      common.FitSmushing | common.RuleUnderscore,
			wantOverlap: 1,
			description: "Rule 2 (underscore) should allow overlap of 1",
		},
		{
			name:        "hardblank_collision_fail",
			lines:       [][]byte{{'#', '#'}},
			glyph:       []string{"$#"},
			layout:      common.FitSmushing | common.RuleEqualChar,
			wantOverlap: 0,
			description: "Hardblank vs visible should fail, fall back to kerning",
		},
		{
			name:        "universal_space_override",
			lines:       [][]byte{{' ', 'A'}},
			glyph:       []string{"B "},
			layout:      common.FitSmushing, // No controlled rules
			wantOverlap: 2,
			description: "Universal: both columns can smush (space+B, A+space)",
		},
		{
			name:        "universal_no_visible_collision",
			lines:       [][]byte{{'A', 'B'}},
			glyph:       []string{"CD"},
			layout:      common.FitSmushing, // No controlled rules
			wantOverlap: 0,
			description: "Universal: visible vs visible not allowed without rules",
		},
		{
			name:        "rule1_equal_char_multi_column",
			lines:       [][]byte{{'#', '#'}},
			glyph:       []string{"##"},
			layout:      common.FitSmushing | common.RuleEqualChar,
			wantOverlap: 2,
			description: "Rule 1: Both columns match, overlap 2",
		},
		{
			name:        "rule4_opposite_pair",
			lines:       [][]byte{{'('}},
			glyph:       []string{")"},
			layout:      common.FitSmushing | common.RuleOppositePair,
			wantOverlap: 1,
			description: "Rule 4: Opposite pairs collapse to |",
		},
		{
			name:        "rule5_big_x",
			lines:       [][]byte{{'/'}},
			glyph:       []string{"\\"},
			layout:      common.FitSmushing | common.RuleBigX,
			wantOverlap: 1,
			description: "Rule 5: /\\ becomes X",
		},
		{
			name:        "rule6_hardblank_smush",
			lines:       [][]byte{{'$'}},
			glyph:       []string{"$"},
			layout:      common.FitSmushing | common.RuleHardblank,
			wantOverlap: 1,
			description: "Rule 6: Hardblank + hardblank allowed",
		},
		// Multi-line tests
		{
			name: "multi_line_minimum_overlap",
			lines: [][]byte{
				{'|', '|', '|'}, // Row 1: all positions work with hierarchy
				{'#', '#', ' '}, // Row 2: all positions work (equal char + universal)
			},
			glyph: []string{
				"|||",
				" ##",
			},
			layout:      common.FitSmushing | common.RuleEqualChar | common.RuleHierarchy,
			wantOverlap: 3,
			description: "Multi-line: all 3 positions valid for both rows",
		},
		{
			name: "multi_line_hardblank_blocks",
			lines: [][]byte{
				{' ', ' ', '$'}, // Row 1 has hardblank
				{' ', ' ', ' '}, // Row 2 all spaces
			},
			glyph: []string{
				" A ",
				" B ",
			},
			layout:      common.FitSmushing,
			wantOverlap: 3,
			description: "Multi-line: overlap 3 works (hardblank vs space allowed)",
		},
		// Precedence tests
		{
			name:        "precedence_rule2_over_rule3",
			lines:       [][]byte{{'_'}},
			glyph:       []string{"|"},
			layout:      common.FitSmushing | common.RuleUnderscore | common.RuleHierarchy,
			wantOverlap: 1,
			description: "Rule 2 takes precedence over Rule 3",
		},
		{
			name:        "precedence_rule4_over_hierarchy",
			lines:       [][]byte{{'('}},
			glyph:       []string{")"},
			layout:      common.FitSmushing | common.RuleOppositePair | common.RuleHierarchy,
			wantOverlap: 1,
			description: "Rule 4 takes precedence over hierarchy",
		},
		// Edge cases
		{
			name:        "empty_left_side",
			lines:       [][]byte{{}},
			glyph:       []string{"ABC"},
			layout:      common.FitSmushing,
			wantOverlap: 0,
			description: "Empty left side should not overlap",
		},
		{
			name:        "empty_right_side",
			lines:       [][]byte{{'A', 'B', 'C'}},
			glyph:       []string{""},
			layout:      common.FitSmushing,
			wantOverlap: 0,
			description: "Empty right side should not overlap",
		},
		{
			name:        "trailing_spaces_left",
			lines:       [][]byte{{'A', ' ', ' '}},
			glyph:       []string{"  B"},
			layout:      common.FitSmushing,
			wantOverlap: 3,
			description: "All positions work with universal (A+space, space+space, space+B)",
		},
		{
			name:        "leading_spaces_right",
			lines:       [][]byte{{'A', ' '}},
			glyph:       []string{" B"},
			layout:      common.FitSmushing,
			wantOverlap: 2,
			description: "Both positions work (A+space, space+B)",
		},
		// Test with precomputed trims
		{
			name:   "with_precomputed_trims",
			lines:  [][]byte{{' ', 'A', ' ', ' '}},
			glyph:  []string{" B "},
			layout: common.FitSmushing,
			trims: []parser.GlyphTrim{
				{LeftmostVisible: 1, RightmostVisible: 1},
			},
			wantOverlap: 3,
			description: "All 3 positions work with universal",
		},
		// Complex multi-column overlap validation
		{
			name:        "partial_overlap_failure",
			lines:       [][]byte{{'/', '\\', 'A'}},
			glyph:       []string{"\\/#"},
			layout:      common.FitSmushing | common.RuleBigX | common.RuleEqualChar,
			wantOverlap: 0,
			description: "No overlap works (A vs # fails, A vs / fails, A vs \\ fails)",
		},
		{
			name:        "decreasing_overlap_search",
			lines:       [][]byte{{'A', 'B', 'C', 'D'}},
			glyph:       []string{"EFGH"},
			layout:      common.FitSmushing,
			wantOverlap: 0,
			description: "No valid overlap when all are visible chars",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOverlap := calculateOptimalOverlap(tt.lines, tt.glyph, tt.layout, hardblank, tt.trims, len(tt.lines))
			if gotOverlap != tt.wantOverlap {
				t.Errorf("%s: got overlap %d, want %d\nDescription: %s",
					tt.name, gotOverlap, tt.wantOverlap, tt.description)
			}
		})
	}
}

// TestOverlapWithRTL tests overlap selection with RTL print direction
func TestOverlapWithRTL(t *testing.T) {
	const hardblank = '$'

	// In RTL, glyphs are composed right-to-left but the overlap logic remains the same
	tests := []struct {
		name        string
		lines       [][]byte
		glyph       []string
		layout      int
		wantOverlap int
		description string
	}{
		{
			name:        "rtl_opposite_pairs",
			lines:       [][]byte{{')'}}, // In RTL, this would have been composed right-to-left
			glyph:       []string{"("},
			layout:      common.FitSmushing | common.RuleOppositePair,
			wantOverlap: 1,
			description: "RTL: opposite pairs still work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOverlap := calculateOptimalOverlap(tt.lines, tt.glyph, tt.layout, hardblank, nil, len(tt.lines))
			if gotOverlap != tt.wantOverlap {
				t.Errorf("%s: got overlap %d, want %d", tt.name, gotOverlap, tt.wantOverlap)
			}
		})
	}
}

// TestMaxCandidateOverlap tests the calculation of maximum candidate overlap
// After issue #14 changes, this now just returns the minimum glyph width
func TestMaxCandidateOverlap(t *testing.T) {
	tests := []struct {
		name    string
		lines   [][]byte
		glyph   []string
		trims   []parser.GlyphTrim
		wantMax int
	}{
		{
			name: "single_row_glyph",
			lines: [][]byte{
				{' ', 'A', ' ', ' '}, // doesn't matter for new algorithm
			},
			glyph: []string{
				"  B ", // width 4
			},
			trims: []parser.GlyphTrim{
				{LeftmostVisible: 2, RightmostVisible: 2},
			},
			wantMax: 4, // just returns glyph width
		},
		{
			name: "multi_row_minimum_width",
			lines: [][]byte{
				{'A', ' ', ' ', ' '}, // doesn't matter
			},
			glyph: []string{
				" BB", // width 3
			},
			trims:   nil,
			wantMax: 3, // glyph width
		},
		{
			name: "multi_row_different_widths",
			lines: [][]byte{
				{'A', ' ', ' '}, // doesn't matter
				{'B', ' '},      // doesn't matter
			},
			glyph: []string{
				" C",   // width 2
				"  DD", // width 4
			},
			trims:   nil,
			wantMax: 2, // minimum glyph width across rows
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMax := calculateMaxCandidateOverlap(tt.lines, tt.glyph, tt.trims, len(tt.lines))
			if gotMax != tt.wantMax {
				t.Errorf("%s: got max candidate %d, want %d", tt.name, gotMax, tt.wantMax)
			}
		})
	}
}

// Benchmarks for performance verification
func BenchmarkOverlapSelection(b *testing.B) {
	// Setup test data
	lines := [][]byte{
		{'#', '#', '#', ' ', ' '},
		{'#', ' ', '#', ' ', ' '},
		{'#', '#', '#', ' ', ' '},
	}
	glyph := []string{
		"  ###",
		"  # #",
		"  ###",
	}
	layout := common.FitSmushing | common.RuleEqualChar
	hardblank := '$'

	b.Run("without_trims", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = calculateOptimalOverlap(lines, glyph, layout, hardblank, nil, len(lines))
		}
	})

	b.Run("with_trims", func(b *testing.B) {
		trims := []parser.GlyphTrim{
			{LeftmostVisible: 2, RightmostVisible: 4},
			{LeftmostVisible: 2, RightmostVisible: 4},
			{LeftmostVisible: 2, RightmostVisible: 4},
		}
		for i := 0; i < b.N; i++ {
			_ = calculateOptimalOverlap(lines, glyph, layout, hardblank, trims, len(lines))
		}
	})
}
