package renderer

import (
	"testing"
)

func TestConstants(t *testing.T) {
	// Verify that constants have expected values
	tests := []struct {
		name string
		got  int
		want int
	}{
		{"SM_SMUSH", SM_SMUSH, 128},
		{"SM_KERN", SM_KERN, 64},
		{"SM_EQUAL", SM_EQUAL, 1},
		{"SM_LOWLINE", SM_LOWLINE, 2},
		{"SM_HIERARCHY", SM_HIERARCHY, 4},
		{"SM_PAIR", SM_PAIR, 8},
		{"SM_BIGX", SM_BIGX, 16},
		{"SM_HARDBLANK", SM_HARDBLANK, 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestOptionsStruct(t *testing.T) {
	// Test Options struct initialization and field access
	t.Run("zero value options", func(t *testing.T) {
		opts := Options{}
		if opts.Layout != 0 {
			t.Errorf("Layout = %v, want 0", opts.Layout)
		}
		if opts.PrintDirection != nil {
			t.Errorf("PrintDirection = %v, want nil", opts.PrintDirection)
		}
		if opts.UnknownRune != nil {
			t.Errorf("UnknownRune = %v, want nil", opts.UnknownRune)
		}
		if opts.TrimWhitespace != false {
			t.Errorf("TrimWhitespace = %v, want false", opts.TrimWhitespace)
		}
	})

	t.Run("options with values", func(t *testing.T) {
		dir := 1
		unknownRune := '?'
		opts := Options{
			Layout:         128,
			PrintDirection: &dir,
			UnknownRune:    &unknownRune,
			TrimWhitespace: true,
		}

		if opts.Layout != 128 {
			t.Errorf("Layout = %v, want 128", opts.Layout)
		}
		if opts.PrintDirection == nil || *opts.PrintDirection != 1 {
			t.Errorf("PrintDirection = %v, want 1", opts.PrintDirection)
		}
		if opts.UnknownRune == nil || *opts.UnknownRune != '?' {
			t.Errorf("UnknownRune = %v, want '?'", opts.UnknownRune)
		}
		if opts.TrimWhitespace != true {
			t.Errorf("TrimWhitespace = %v, want true", opts.TrimWhitespace)
		}
	})
}

func TestRenderStateStruct(t *testing.T) {
	// Test renderState struct initialization
	t.Run("zero value renderState", func(t *testing.T) {
		state := renderState{}
		if state.outputLine != nil {
			t.Errorf("outputLine = %v, want nil", state.outputLine)
		}
		if state.rowLengths != nil {
			t.Errorf("rowLengths = %v, want nil", state.rowLengths)
		}
		if state.outlineLen != 0 {
			t.Errorf("outlineLen = %v, want 0", state.outlineLen)
		}
		if state.outlineLenLimit != 0 {
			t.Errorf("outlineLenLimit = %v, want 0", state.outlineLenLimit)
		}
		if state.currChar != nil {
			t.Errorf("currChar = %v, want nil", state.currChar)
		}
		if state.currCharWidth != 0 {
			t.Errorf("currCharWidth = %v, want 0", state.currCharWidth)
		}
		if state.previousCharWidth != 0 {
			t.Errorf("previousCharWidth = %v, want 0", state.previousCharWidth)
		}
		if state.charHeight != 0 {
			t.Errorf("charHeight = %v, want 0", state.charHeight)
		}
		if state.right2left != 0 {
			t.Errorf("right2left = %v, want 0", state.right2left)
		}
		if state.smushMode != 0 {
			t.Errorf("smushMode = %v, want 0", state.smushMode)
		}
		if state.hardblank != 0 {
			t.Errorf("hardblank = %v, want 0", state.hardblank)
		}
		if state.trimWhitespace != false {
			t.Errorf("trimWhitespace = %v, want false", state.trimWhitespace)
		}
	})

	t.Run("renderState with values", func(t *testing.T) {
		state := renderState{
			outputLine: [][]rune{
				[]rune("test"),
				[]rune("line"),
			},
			rowLengths:        []int{4, 4},
			outlineLen:        10,
			outlineLenLimit:   100,
			currChar:          []string{"A", "B"},
			currCharWidth:     1,
			previousCharWidth: 2,
			charHeight:        2,
			right2left:        1,
			smushMode:         SM_SMUSH | SM_EQUAL,
			hardblank:         '$',
			trimWhitespace:    true,
		}

		if len(state.outputLine) != 2 {
			t.Errorf("len(outputLine) = %v, want 2", len(state.outputLine))
		}
		if len(state.rowLengths) != 2 {
			t.Errorf("len(rowLengths) = %v, want 2", len(state.rowLengths))
		}
		if state.outlineLen != 10 {
			t.Errorf("outlineLen = %v, want 10", state.outlineLen)
		}
		if state.outlineLenLimit != 100 {
			t.Errorf("outlineLenLimit = %v, want 100", state.outlineLenLimit)
		}
		if len(state.currChar) != 2 {
			t.Errorf("len(currChar) = %v, want 2", len(state.currChar))
		}
		if state.currCharWidth != 1 {
			t.Errorf("currCharWidth = %v, want 1", state.currCharWidth)
		}
		if state.previousCharWidth != 2 {
			t.Errorf("previousCharWidth = %v, want 2", state.previousCharWidth)
		}
		if state.charHeight != 2 {
			t.Errorf("charHeight = %v, want 2", state.charHeight)
		}
		if state.right2left != 1 {
			t.Errorf("right2left = %v, want 1", state.right2left)
		}
		if state.smushMode != (SM_SMUSH | SM_EQUAL) {
			t.Errorf("smushMode = %v, want %v", state.smushMode, SM_SMUSH|SM_EQUAL)
		}
		if state.hardblank != '$' {
			t.Errorf("hardblank = %v, want '$'", state.hardblank)
		}
		if state.trimWhitespace != true {
			t.Errorf("trimWhitespace = %v, want true", state.trimWhitespace)
		}
	})
}

func TestSmushModeCombinations(t *testing.T) {
	// Test that smush mode combinations work correctly
	tests := []struct {
		name string
		mode int
		hasSmush bool
		hasKern bool
		rules []int
	}{
		{
			name: "no mode",
			mode: 0,
			hasSmush: false,
			hasKern: false,
			rules: []int{},
		},
		{
			name: "kern only",
			mode: SM_KERN,
			hasSmush: false,
			hasKern: true,
			rules: []int{},
		},
		{
			name: "smush only",
			mode: SM_SMUSH,
			hasSmush: true,
			hasKern: false,
			rules: []int{},
		},
		{
			name: "smush with one rule",
			mode: SM_SMUSH | SM_EQUAL,
			hasSmush: true,
			hasKern: false,
			rules: []int{SM_EQUAL},
		},
		{
			name: "smush with multiple rules",
			mode: SM_SMUSH | SM_EQUAL | SM_LOWLINE | SM_HIERARCHY,
			hasSmush: true,
			hasKern: false,
			rules: []int{SM_EQUAL, SM_LOWLINE, SM_HIERARCHY},
		},
		{
			name: "smush with all rules",
			mode: SM_SMUSH | SM_EQUAL | SM_LOWLINE | SM_HIERARCHY | SM_PAIR | SM_BIGX | SM_HARDBLANK,
			hasSmush: true,
			hasKern: false,
			rules: []int{SM_EQUAL, SM_LOWLINE, SM_HIERARCHY, SM_PAIR, SM_BIGX, SM_HARDBLANK},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check if smush mode is set
			hasSmush := (tt.mode & SM_SMUSH) != 0
			if hasSmush != tt.hasSmush {
				t.Errorf("hasSmush = %v, want %v", hasSmush, tt.hasSmush)
			}

			// Check if kern mode is set
			hasKern := (tt.mode & SM_KERN) != 0
			if hasKern != tt.hasKern {
				t.Errorf("hasKern = %v, want %v", hasKern, tt.hasKern)
			}

			// Check individual rules
			for _, rule := range tt.rules {
				if (tt.mode & rule) == 0 {
					t.Errorf("rule %v not set in mode %v", rule, tt.mode)
				}
			}
		})
	}
}

func TestLayoutBitmaskConversion(t *testing.T) {
	// Test that layout bitmasks convert correctly
	tests := []struct {
		name string
		layout int
		expectedMode int
	}{
		{
			name: "full width",
			layout: 0,
			expectedMode: 0,
		},
		{
			name: "kerning",
			layout: 1 << 6,
			expectedMode: SM_KERN,
		},
		{
			name: "smushing no rules",
			layout: 1 << 7,
			expectedMode: SM_SMUSH,
		},
		{
			name: "smushing with equal rule",
			layout: (1 << 7) | 1,
			expectedMode: SM_SMUSH | SM_EQUAL,
		},
		{
			name: "smushing with all rules",
			layout: (1 << 7) | 63,
			expectedMode: SM_SMUSH | 63,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := layoutToSmushMode(tt.layout)
			if got != tt.expectedMode {
				t.Errorf("layoutToSmushMode(%v) = %v, want %v", tt.layout, got, tt.expectedMode)
			}
		})
	}
}