package renderer

import (
	"testing"
)

// Test controlled smushing rules 1-6 per FIGfont v2 spec
func TestSmushPair(t *testing.T) {
	// Test data structure for table-driven tests
	type testCase struct {
		name      string
		left      rune
		right     rune
		layout    int // Layout bits to enable specific rules
		hardblank rune
		want      rune
		wantOK    bool
	}

	// Define layout bit combinations for testing (matching internal/common/constants.go)
	const (
		layoutKerning  = 0x40 // FitKerning (bit 6)
		layoutSmushing = 0x80 // FitSmushing (bit 7)
		layoutRule1    = 0x01 // RuleEqualChar (bit 0)
		layoutRule2    = 0x02 // RuleUnderscore (bit 1)
		layoutRule3    = 0x04 // RuleHierarchy (bit 2)
		layoutRule4    = 0x08 // RuleOppositePair (bit 3)
		layoutRule5    = 0x10 // RuleBigX (bit 4)
		layoutRule6    = 0x20 // RuleHardblank (bit 5)
		layoutAllRules = layoutSmushing | layoutRule1 | layoutRule2 | layoutRule3 | layoutRule4 | layoutRule5 | layoutRule6
	)

	tests := []testCase{
		// Rule 1: Equal Character (non-space identical chars smush to that char)
		{
			name:      "rule1_equal_hash",
			left:      '#',
			right:     '#',
			layout:    layoutSmushing | layoutRule1,
			hardblank: '$',
			want:      '#',
			wantOK:    true,
		},
		{
			name:      "rule1_equal_dash",
			left:      '-',
			right:     '-',
			layout:    layoutSmushing | layoutRule1,
			hardblank: '$',
			want:      '-',
			wantOK:    true,
		},
		{
			name:      "rule1_not_equal",
			left:      '#',
			right:     '*',
			layout:    layoutSmushing | layoutRule1,
			hardblank: '$',
			want:      0,
			wantOK:    false,
		},
		{
			name:      "rule1_space_not_applied",
			left:      ' ',
			right:     ' ',
			layout:    layoutSmushing | layoutRule1,
			hardblank: '$',
			want:      ' ',
			wantOK:    true, // Rule 1 doesn't match spaces, but universal fallback allows space + space
		},
		{
			name:      "rule1_hardblank_not_applied",
			left:      '$',
			right:     '$',
			layout:    layoutSmushing | layoutRule1,
			hardblank: '$',
			want:      0,
			wantOK:    false, // Rule 1 should not match hardblanks (Rule 6 handles them)
		},

		// Rule 2: Underscore (underscore with border char becomes border)
		{
			name:      "rule2_underscore_pipe",
			left:      '_',
			right:     '|',
			layout:    layoutSmushing | layoutRule2,
			hardblank: '$',
			want:      '|',
			wantOK:    true,
		},
		{
			name:      "rule2_pipe_underscore",
			left:      '|',
			right:     '_',
			layout:    layoutSmushing | layoutRule2,
			hardblank: '$',
			want:      '|',
			wantOK:    true,
		},
		{
			name:      "rule2_underscore_slash",
			left:      '_',
			right:     '/',
			layout:    layoutSmushing | layoutRule2,
			hardblank: '$',
			want:      '/',
			wantOK:    true,
		},
		{
			name:      "rule2_backslash_underscore",
			left:      '\\',
			right:     '_',
			layout:    layoutSmushing | layoutRule2,
			hardblank: '$',
			want:      '\\',
			wantOK:    true,
		},
		{
			name:      "rule2_underscore_bracket",
			left:      '_',
			right:     '[',
			layout:    layoutSmushing | layoutRule2,
			hardblank: '$',
			want:      '[',
			wantOK:    true,
		},
		{
			name:      "rule2_underscore_brace",
			left:      '_',
			right:     '{',
			layout:    layoutSmushing | layoutRule2,
			hardblank: '$',
			want:      '{',
			wantOK:    true,
		},
		{
			name:      "rule2_underscore_paren",
			left:      '_',
			right:     '(',
			layout:    layoutSmushing | layoutRule2,
			hardblank: '$',
			want:      '(',
			wantOK:    true,
		},
		{
			name:      "rule2_underscore_angle",
			left:      '_',
			right:     '<',
			layout:    layoutSmushing | layoutRule2,
			hardblank: '$',
			want:      '<',
			wantOK:    true,
		},
		{
			name:      "rule2_underscore_non_border",
			left:      '_',
			right:     '+',
			layout:    layoutSmushing | layoutRule2,
			hardblank: '$',
			want:      0,
			wantOK:    false, // '+' is not a border char
		},

		// Rule 3: Hierarchy (| > /\ > [] > {} > ())
		{
			name:      "rule3_slash_pipe",
			left:      '/',
			right:     '|',
			layout:    layoutSmushing | layoutRule3,
			hardblank: '$',
			want:      '|',
			wantOK:    true,
		},
		{
			name:      "rule3_pipe_slash",
			left:      '|',
			right:     '/',
			layout:    layoutSmushing | layoutRule3,
			hardblank: '$',
			want:      '|',
			wantOK:    true,
		},
		{
			name:      "rule3_bracket_slash",
			left:      '[',
			right:     '/',
			layout:    layoutSmushing | layoutRule3,
			hardblank: '$',
			want:      '/',
			wantOK:    true,
		},
		{
			name:      "rule3_backslash_bracket",
			left:      '\\',
			right:     ']',
			layout:    layoutSmushing | layoutRule3,
			hardblank: '$',
			want:      '\\',
			wantOK:    true,
		},
		{
			name:      "rule3_brace_bracket",
			left:      '{',
			right:     '[',
			layout:    layoutSmushing | layoutRule3,
			hardblank: '$',
			want:      '[',
			wantOK:    true,
		},
		{
			name:      "rule3_bracket_brace",
			left:      ']',
			right:     '}',
			layout:    layoutSmushing | layoutRule3,
			hardblank: '$',
			want:      ']',
			wantOK:    true,
		},
		{
			name:      "rule3_paren_brace",
			left:      '(',
			right:     '{',
			layout:    layoutSmushing | layoutRule3,
			hardblank: '$',
			want:      '{',
			wantOK:    true,
		},
		{
			name:      "rule3_same_class_brackets",
			left:      '[',
			right:     ']',
			layout:    layoutSmushing | layoutRule3,
			hardblank: '$',
			want:      0,
			wantOK:    false, // Same class - Rule 3 doesn't apply (Rule 4 would)
		},
		{
			name:      "rule3_same_class_parens",
			left:      '(',
			right:     ')',
			layout:    layoutSmushing | layoutRule3,
			hardblank: '$',
			want:      0,
			wantOK:    false, // Same class - Rule 3 doesn't apply (Rule 4 would)
		},

		// Rule 4: Opposite pairs become '|'
		{
			name:      "rule4_paren_close",
			left:      '(',
			right:     ')',
			layout:    layoutSmushing | layoutRule4,
			hardblank: '$',
			want:      '|',
			wantOK:    true,
		},
		{
			name:      "rule4_paren_reverse",
			left:      ')',
			right:     '(',
			layout:    layoutSmushing | layoutRule4,
			hardblank: '$',
			want:      '|',
			wantOK:    true,
		},
		{
			name:      "rule4_bracket_close",
			left:      '[',
			right:     ']',
			layout:    layoutSmushing | layoutRule4,
			hardblank: '$',
			want:      '|',
			wantOK:    true,
		},
		{
			name:      "rule4_bracket_reverse",
			left:      ']',
			right:     '[',
			layout:    layoutSmushing | layoutRule4,
			hardblank: '$',
			want:      '|',
			wantOK:    true,
		},
		{
			name:      "rule4_brace_close",
			left:      '{',
			right:     '}',
			layout:    layoutSmushing | layoutRule4,
			hardblank: '$',
			want:      '|',
			wantOK:    true,
		},
		{
			name:      "rule4_brace_reverse",
			left:      '}',
			right:     '{',
			layout:    layoutSmushing | layoutRule4,
			hardblank: '$',
			want:      '|',
			wantOK:    true,
		},

		// Rule 5: Big X (per spec: /\ → '|', \/ → 'Y', >< → 'X')
		{
			name:      "rule5_slash_backslash",
			left:      '/',
			right:     '\\',
			layout:    layoutSmushing | layoutRule5,
			hardblank: '$',
			want:      '|',
			wantOK:    true,
		},
		{
			name:      "rule5_backslash_slash",
			left:      '\\',
			right:     '/',
			layout:    layoutSmushing | layoutRule5,
			hardblank: '$',
			want:      'Y',
			wantOK:    true,
		},
		{
			name:      "rule5_greater_less",
			left:      '>',
			right:     '<',
			layout:    layoutSmushing | layoutRule5,
			hardblank: '$',
			want:      'X',
			wantOK:    true,
		},
		{
			name:      "rule5_less_greater_no_match",
			left:      '<',
			right:     '>',
			layout:    layoutSmushing | layoutRule5,
			hardblank: '$',
			want:      0,
			wantOK:    false, // Only >< makes X, not <>
		},

		// Rule 6: Hardblank
		{
			name:      "rule6_hardblank_smush",
			left:      '$',
			right:     '$',
			layout:    layoutSmushing | layoutRule6,
			hardblank: '$',
			want:      '$',
			wantOK:    true,
		},
		{
			name:      "rule6_different_hardblank",
			left:      '@',
			right:     '@',
			layout:    layoutSmushing | layoutRule6,
			hardblank: '@',
			want:      '@',
			wantOK:    true,
		},
		{
			name:      "rule6_hardblank_with_visible",
			left:      '$',
			right:     'A',
			layout:    layoutSmushing | layoutRule6,
			hardblank: '$',
			want:      0,
			wantOK:    false, // Rule 6 doesn't match, universal forbids hardblank vs visible
		},

		// Rule 6 precedence tests - ensure Rule 6 takes priority over universal HB-HB block
		{
			name:      "rule6_hardblank_smush_applies",
			left:      ',',
			right:     ',',
			layout:    layoutSmushing | layoutRule6, // Rule 6 enabled
			hardblank: ',',
			want:      ',', // Rule 6 smushes to a single hardblank
			wantOK:    true,
		},
		{
			name:      "no_rule6_hardblank_pair_blocks",
			left:      ',',
			right:     ',',
			layout:    layoutSmushing, // universal only
			hardblank: ',',
			want:      0,
			wantOK:    false, // Blocked by universal path
		},

		// Precedence tests (multiple rules enabled)
		{
			name:      "precedence_rule1_over_rule3",
			left:      '|',
			right:     '|',
			layout:    layoutAllRules,
			hardblank: '$',
			want:      '|',
			wantOK:    true, // Rule 1 (equal) takes precedence
		},
		{
			name:      "precedence_underscore_underscore",
			left:      '_',
			right:     '_',
			layout:    layoutAllRules,
			hardblank: '$',
			want:      '_',
			wantOK:    true, // Rule 1 (equal) wins over Rule 2
		},
		{
			name:      "precedence_rule3_over_rule5",
			left:      '/',
			right:     '|',
			layout:    layoutAllRules,
			hardblank: '$',
			want:      '|',
			wantOK:    true, // Rule 3 (hierarchy) wins even though / is part of Rule 5
		},
		{
			name:      "precedence_rule4_over_rule3",
			left:      '(',
			right:     ')',
			layout:    layoutAllRules,
			hardblank: '$',
			want:      '|',
			wantOK:    true, // Rule 4 wins (opposites) even though Rule 3 could theoretically apply
		},

		// Universal smushing (when no controlled rule applies)
		{
			name:      "universal_space_left",
			left:      ' ',
			right:     'A',
			layout:    layoutSmushing, // No specific rules enabled
			hardblank: '$',
			want:      'A',
			wantOK:    true,
		},
		{
			name:      "universal_space_right",
			left:      'A',
			right:     ' ',
			layout:    layoutSmushing, // No specific rules enabled
			hardblank: '$',
			want:      'A',
			wantOK:    true,
		},
		{
			name:      "universal_no_match",
			left:      'A',
			right:     'B',
			layout:    layoutSmushing, // No specific rules enabled
			hardblank: '$',
			want:      0,
			wantOK:    false, // Universal only allows space replacement, not visible collision
		},
		{
			name:      "universal_hardblank_override",
			left:      '$',
			right:     'A',
			layout:    layoutSmushing, // No Rule 6 enabled
			hardblank: '$',
			want:      0,
			wantOK:    false, // Universal never allows hardblank vs visible
		},
		{
			name:      "universal_hardblank_override_reverse",
			left:      'A',
			right:     '$',
			layout:    layoutSmushing, // No Rule 6 enabled
			hardblank: '$',
			want:      0,
			wantOK:    false, // Universal never allows visible vs hardblank
		},
		{
			name:      "universal_hardblank_collision",
			left:      '$',
			right:     '$',
			layout:    layoutSmushing, // No Rule 6 enabled
			hardblank: '$',
			want:      0,
			wantOK:    false, // HB-HB blocked in universal (only allowed via Rule 6)
		},

		// No smushing mode (kerning only)
		{
			name:      "kerning_mode_no_smush",
			left:      'A',
			right:     'B',
			layout:    layoutKerning, // Kerning mode, not smushing
			hardblank: '$',
			want:      0,
			wantOK:    false,
		},

		// Universal smushing fallback when controlled rules fail
		{
			name:      "controlled_rules_fallback_space_left",
			left:      ' ',
			right:     'A',
			layout:    layoutSmushing | layoutRule1, // Rule 1 enabled but won't match
			hardblank: '$',
			want:      'A',
			wantOK:    true, // Should fall back to universal smushing
		},
		{
			name:      "controlled_rules_fallback_space_right",
			left:      'A',
			right:     ' ',
			layout:    layoutSmushing | layoutRule1, // Rule 1 enabled but won't match
			hardblank: '$',
			want:      'A',
			wantOK:    true, // Should fall back to universal smushing
		},
		{
			name:      "controlled_rules_fallback_visible_collision",
			left:      'A',
			right:     'B',
			layout:    layoutSmushing | layoutRule1, // Rule 1 enabled but won't match (A != B)
			hardblank: '$',
			want:      0,
			wantOK:    false, // Universal fallback: visible vs visible = no smush
		},
		{
			name:      "controlled_rules_fallback_hardblank_left",
			left:      '$',
			right:     'A',
			layout:    layoutSmushing | layoutRule1, // Rule 1 enabled but won't match
			hardblank: '$',
			want:      0,
			wantOK:    false, // Universal fallback: hardblank vs visible forbidden
		},
		{
			name:      "controlled_rules_fallback_hardblank_right",
			left:      'A',
			right:     '$',
			layout:    layoutSmushing | layoutRule1, // Rule 1 enabled but won't match
			hardblank: '$',
			want:      0,
			wantOK:    false, // Universal fallback: visible vs hardblank forbidden
		},
		{
			name:      "controlled_rules_fallback_both_hardblanks",
			left:      '$',
			right:     '$',
			layout:    layoutSmushing | layoutRule1, // Rule 1 enabled but won't match (hardblanks excluded from Rule 1)
			hardblank: '$',
			want:      0,
			wantOK:    false, // Universal fallback: hardblank collision blocks universal smushing
		},
		{
			name:      "controlled_rules_fallback_with_multiple_rules",
			left:      ' ',
			right:     'X',
			layout:    layoutSmushing | layoutRule2 | layoutRule3 | layoutRule4, // Multiple rules, none match space
			hardblank: '$',
			want:      'X',
			wantOK:    true, // Should fall back to universal smushing
		},
		{
			name:      "controlled_rules_one_matches_no_fallback",
			left:      '#',
			right:     '#',
			layout:    layoutSmushing | layoutRule1 | layoutRule2, // Rule 1 will match
			hardblank: '$',
			want:      '#',
			wantOK:    true, // Rule 1 matches, no need for fallback
		},

		// Fallback universal HB vs space tests (later wins special case)
		{
			name:      "fallback_universal_hardblank_vs_space_right_later_wins",
			left:      ',',
			right:     ' ',
			layout:    layoutSmushing | layoutRule1, // rules exist but don't match
			hardblank: ',',
			want:      ' ',
			wantOK:    true,
		},
		{
			name:      "fallback_universal_space_vs_hardblank_right_later_wins",
			left:      ' ',
			right:     ',',
			layout:    layoutSmushing | layoutRule1,
			hardblank: ',',
			want:      ',',
			wantOK:    true,
		},

		// Pure universal mode tests (no controlled rules enabled)
		{
			name:      "pure_universal_space_space",
			left:      ' ',
			right:     ' ',
			layout:    layoutSmushing, // no rules
			hardblank: ',',
			want:      ' ',
			wantOK:    true,
		},
		{
			name:      "pure_universal_visible_vs_visible_later_wins",
			left:      'A',
			right:     'B',
			layout:    layoutSmushing, // no rules
			hardblank: ',',
			want:      0, // per issue #14: universal only for space replacement
			wantOK:    false,
		},
		{
			name:      "pure_universal_visible_vs_hardblank_left",
			left:      '$',
			right:     'X',
			layout:    layoutSmushing, // Pure universal - no rules
			hardblank: '$',
			want:      0,
			wantOK:    false, // Universal never allows hardblank vs visible
		},
		{
			name:      "pure_universal_visible_vs_hardblank_right",
			left:      'Y',
			right:     '$',
			layout:    layoutSmushing, // Pure universal - no rules
			hardblank: '$',
			want:      0,
			wantOK:    false, // Universal never allows visible vs hardblank
		},
		{
			name:      "pure_universal_hardblank_vs_space_left",
			left:      '$',
			right:     ' ',
			layout:    layoutSmushing, // Pure universal - no rules
			hardblank: '$',
			want:      ' ',
			wantOK:    true, // Later char (right) overrides in pure universal
		},
		{
			name:      "pure_universal_hardblank_vs_space_right",
			left:      ' ',
			right:     '$',
			layout:    layoutSmushing, // Pure universal - no rules
			hardblank: '$',
			want:      '$',
			wantOK:    true, // Later char (right) overrides in pure universal
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := smushPair(tt.left, tt.right, tt.layout, tt.hardblank)
			if ok != tt.wantOK {
				t.Errorf("smushPair() ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("smushPair() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Test that border character set is correctly defined
func TestBorderCharSet(t *testing.T) {
	borderChars := []rune{'|', '/', '\\', '[', ']', '{', '}', '(', ')', '<', '>'}
	for _, ch := range borderChars {
		if !isBorderChar(ch) {
			t.Errorf("isBorderChar(%q) = false, want true", ch)
		}
	}

	// Test non-border chars
	nonBorderChars := []rune{'_', '+', '-', '*', '#', '@', ' ', 'A', '1'}
	for _, ch := range nonBorderChars {
		if isBorderChar(ch) {
			t.Errorf("isBorderChar(%q) = true, want false", ch)
		}
	}
}

// Test hierarchy class mapping
func TestHierarchyClass(t *testing.T) {
	tests := []struct {
		char  rune
		class int
	}{
		{'|', 6},
		{'/', 5},
		{'\\', 5},
		{'[', 4},
		{']', 4},
		{'{', 3},
		{'}', 3},
		{'(', 2},
		{')', 2},
		{'<', 1},
		{'>', 1},
		// Non-hierarchy chars
		{'_', 0},
		{'A', 0},
		{' ', 0},
		{'#', 0},
	}

	for _, tt := range tests {
		got := getHierarchyClass(tt.char)
		if got != tt.class {
			t.Errorf("getHierarchyClass(%q) = %d, want %d", tt.char, got, tt.class)
		}
	}
}

// TestUniversalSmushingMultiColumn tests universal smushing fallback in multi-column overlaps
//
//nolint:gocognit // Test requires complex simulation of multi-column overlap scenarios
func TestUniversalSmushingMultiColumn(t *testing.T) {
	// This test simulates multi-column overlap scenarios where some columns
	// match controlled rules and others fall back to universal smushing

	const (
		layoutSmushing = 0x80
		layoutRule1    = 0x01
		layoutRule5    = 0x10
	)

	hardblank := '$'
	layout := layoutSmushing | layoutRule1 | layoutRule5

	tests := []struct {
		name         string
		leftLine     string
		rightLine    string
		overlap      int
		wantCanSmush bool
		wantResult   string // Expected result after smushing (if canSmush is true)
	}{
		{
			name:         "mixed_controlled_and_universal",
			leftLine:     "##  ",
			rightLine:    "## A",
			overlap:      2,
			wantCanSmush: true,
			// Column 1: ' ' + '#' → '#' (universal fallback: space + visible = visible)
			// Column 2: ' ' + '#' → '#' (universal fallback: space + visible = visible)
			// Non-overlapped: " A"
			wantResult: "#### A",
		},
		{
			name:         "universal_with_spaces",
			leftLine:     "A  ",
			rightLine:    " B",
			overlap:      1,
			wantCanSmush: true,
			// Column 1: ' ' + ' ' → ' ' (universal: space + space fallback)
			// Non-overlapped: "B"
			wantResult: "A  B",
		},
		{
			name:         "hardblank_blocks_hardblank",
			leftLine:     "A$",
			rightLine:    "$B",
			overlap:      1,
			wantCanSmush: false,
			// Column 1: '$' + '$' → blocked (hardblank vs hardblank collision, no Rule 6)
			wantResult: "",
		},
		{
			name:         "visible_collision_blocks",
			leftLine:     "AB",
			rightLine:    "CD",
			overlap:      1,
			wantCanSmush: false,
			// Column 1: 'B' + 'C' → blocked (visible vs visible, no rule matches)
			wantResult: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the overlap check logic
			canSmush := true
			var result []byte

			// Start with the non-overlapped portion of the left line
			result = []byte(tt.leftLine[:len(tt.leftLine)-tt.overlap])

			for col := 0; col < tt.overlap; col++ {
				leftPos := len(tt.leftLine) - tt.overlap + col
				rightPos := col

				leftChar := rune(tt.leftLine[leftPos])
				rightChar := rune(tt.rightLine[rightPos])

				smushed, ok := smushPair(leftChar, rightChar, layout, hardblank)
				if !ok {
					canSmush = false
					break
				}

				if canSmush {
					result = append(result, byte(smushed))
				}
			}

			if canSmush {
				// Append non-overlapped portion of the right line
				result = append(result, []byte(tt.rightLine[tt.overlap:])...)
			}

			if canSmush != tt.wantCanSmush {
				t.Errorf("canSmush = %v, want %v", canSmush, tt.wantCanSmush)
			}

			if tt.wantCanSmush && canSmush {
				if string(result) != tt.wantResult {
					t.Errorf("result = %q, want %q", string(result), tt.wantResult)
				}
			}
		})
	}
}

// Test opposite pair detection
func TestIsOppositePair(t *testing.T) {
	tests := []struct {
		left  rune
		right rune
		want  bool
	}{
		// Valid opposite pairs
		{'(', ')', true},
		{')', '(', true},
		{'[', ']', true},
		{']', '[', true},
		{'{', '}', true},
		{'}', '{', true},
		// Not opposite pairs
		{'(', '(', false},
		{'[', '[', false},
		{'(', ']', false},
		{'[', '}', false},
		{'{', ')', false},
		// Non-pair chars
		{'A', 'B', false},
		{'|', '/', false},
	}

	for _, tt := range tests {
		got := isOppositePair(tt.left, tt.right)
		if got != tt.want {
			t.Errorf("isOppositePair(%q, %q) = %v, want %v", tt.left, tt.right, got, tt.want)
		}
	}
}
