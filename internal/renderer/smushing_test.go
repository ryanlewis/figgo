package renderer

import (
	"testing"
)

func TestSmushem(t *testing.T) {
	tests := []struct {
		name      string
		state     *renderState
		lch       rune
		rch       rune
		want      rune
	}{
		// Space handling
		{
			name:  "left space returns right",
			state: &renderState{smushMode: SM_SMUSH},
			lch:   ' ',
			rch:   'A',
			want:  'A',
		},
		{
			name:  "right space returns left",
			state: &renderState{smushMode: SM_SMUSH},
			lch:   'B',
			rch:   ' ',
			want:  'B',
		},
		{
			name:  "both spaces returns space",
			state: &renderState{smushMode: SM_SMUSH},
			lch:   ' ',
			rch:   ' ',
			want:  ' ',
		},
		// Width constraints
		{
			name:  "previous char width < 2 prevents smushing",
			state: &renderState{smushMode: SM_SMUSH, previousCharWidth: 1, currCharWidth: 3},
			lch:   'A',
			rch:   'B',
			want:  0,
		},
		{
			name:  "current char width < 2 prevents smushing",
			state: &renderState{smushMode: SM_SMUSH, previousCharWidth: 3, currCharWidth: 1},
			lch:   'A',
			rch:   'B',
			want:  0,
		},
		// Kerning mode (no smushing)
		{
			name:  "kerning mode returns 0",
			state: &renderState{smushMode: SM_KERN, previousCharWidth: 3, currCharWidth: 3},
			lch:   'A',
			rch:   'B',
			want:  0,
		},
		// Universal smushing (no specific rules)
		{
			name:  "universal smushing LTR prefers right",
			state: &renderState{smushMode: SM_SMUSH, previousCharWidth: 3, currCharWidth: 3, right2left: 0, hardblank: '$'},
			lch:   'A',
			rch:   'B',
			want:  'B',
		},
		{
			name:  "universal smushing RTL prefers left",
			state: &renderState{smushMode: SM_SMUSH, previousCharWidth: 3, currCharWidth: 3, right2left: 1, hardblank: '$'},
			lch:   'A',
			rch:   'B',
			want:  'A',
		},
		{
			name:  "universal smushing left hardblank returns right",
			state: &renderState{smushMode: SM_SMUSH, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   '$',
			rch:   'B',
			want:  'B',
		},
		{
			name:  "universal smushing right hardblank returns left",
			state: &renderState{smushMode: SM_SMUSH, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   'A',
			rch:   '$',
			want:  'A',
		},
		// Rule 1: Equal character
		{
			name:  "equal character rule matches",
			state: &renderState{smushMode: SM_SMUSH | SM_EQUAL, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   'A',
			rch:   'A',
			want:  'A',
		},
		{
			name:  "equal character rule no match",
			state: &renderState{smushMode: SM_SMUSH | SM_EQUAL, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   'A',
			rch:   'B',
			want:  0,
		},
		// Rule 2: Underscore
		{
			name:  "underscore rule left underscore",
			state: &renderState{smushMode: SM_SMUSH | SM_LOWLINE, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   '_',
			rch:   '|',
			want:  '|',
		},
		{
			name:  "underscore rule right underscore",
			state: &renderState{smushMode: SM_SMUSH | SM_LOWLINE, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   '/',
			rch:   '_',
			want:  '/',
		},
		{
			name:  "underscore rule with bracket",
			state: &renderState{smushMode: SM_SMUSH | SM_LOWLINE, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   '_',
			rch:   '[',
			want:  '[',
		},
		{
			name:  "underscore rule no match",
			state: &renderState{smushMode: SM_SMUSH | SM_LOWLINE, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   '_',
			rch:   'A',
			want:  0,
		},
		// Rule 3: Hierarchy
		{
			name:  "hierarchy rule | over /",
			state: &renderState{smushMode: SM_SMUSH | SM_HIERARCHY, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   '|',
			rch:   '/',
			want:  '/',
		},
		{
			name:  "hierarchy rule | over [",
			state: &renderState{smushMode: SM_SMUSH | SM_HIERARCHY, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   '|',
			rch:   '[',
			want:  '[',
		},
		{
			name:  "hierarchy rule / over {",
			state: &renderState{smushMode: SM_SMUSH | SM_HIERARCHY, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   '/',
			rch:   '{',
			want:  '{',
		},
		{
			name:  "hierarchy rule ] over (",
			state: &renderState{smushMode: SM_SMUSH | SM_HIERARCHY, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   ']',
			rch:   '(',
			want:  '(',
		},
		{
			name:  "hierarchy rule } over <",
			state: &renderState{smushMode: SM_SMUSH | SM_HIERARCHY, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   '}',
			rch:   '<',
			want:  '<',
		},
		{
			name:  "hierarchy rule ( over >",
			state: &renderState{smushMode: SM_SMUSH | SM_HIERARCHY, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   '(',
			rch:   '>',
			want:  '>',
		},
		// Rule 4: Opposite pairs
		{
			name:  "opposite pair [] -> |",
			state: &renderState{smushMode: SM_SMUSH | SM_PAIR, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   '[',
			rch:   ']',
			want:  '|',
		},
		{
			name:  "opposite pair ][ -> |",
			state: &renderState{smushMode: SM_SMUSH | SM_PAIR, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   ']',
			rch:   '[',
			want:  '|',
		},
		{
			name:  "opposite pair {} -> |",
			state: &renderState{smushMode: SM_SMUSH | SM_PAIR, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   '{',
			rch:   '}',
			want:  '|',
		},
		{
			name:  "opposite pair }{ -> |",
			state: &renderState{smushMode: SM_SMUSH | SM_PAIR, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   '}',
			rch:   '{',
			want:  '|',
		},
		{
			name:  "opposite pair () -> |",
			state: &renderState{smushMode: SM_SMUSH | SM_PAIR, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   '(',
			rch:   ')',
			want:  '|',
		},
		{
			name:  "opposite pair )( -> |",
			state: &renderState{smushMode: SM_SMUSH | SM_PAIR, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   ')',
			rch:   '(',
			want:  '|',
		},
		// Rule 5: Big X
		{
			name:  "big X /\\ -> |",
			state: &renderState{smushMode: SM_SMUSH | SM_BIGX, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   '/',
			rch:   '\\',
			want:  '|',
		},
		{
			name:  "big X \\/ -> Y",
			state: &renderState{smushMode: SM_SMUSH | SM_BIGX, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   '\\',
			rch:   '/',
			want:  'Y',
		},
		{
			name:  "big X >< -> X",
			state: &renderState{smushMode: SM_SMUSH | SM_BIGX, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   '>',
			rch:   '<',
			want:  'X',
		},
		{
			name:  "big X <> does not give X",
			state: &renderState{smushMode: SM_SMUSH | SM_BIGX, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   '<',
			rch:   '>',
			want:  0,
		},
		// Rule 6: Hardblank
		{
			name:  "hardblank rule both hardblanks",
			state: &renderState{smushMode: SM_SMUSH | SM_HARDBLANK, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   '$',
			rch:   '$',
			want:  '$',
		},
		{
			name:  "hardblank rule only left",
			state: &renderState{smushMode: SM_SMUSH | SM_HARDBLANK, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   '$',
			rch:   'A',
			want:  0,
		},
		{
			name:  "hardblank rule only right",
			state: &renderState{smushMode: SM_SMUSH | SM_HARDBLANK, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   'A',
			rch:   '$',
			want:  0,
		},
		// Hardblank without hardblank rule
		{
			name:  "hardblank without rule prevents smushing",
			state: &renderState{smushMode: SM_SMUSH | SM_EQUAL, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   '$',
			rch:   'A',
			want:  0,
		},
		// Multiple rules - should use first matching
		{
			name:  "multiple rules equal takes precedence",
			state: &renderState{smushMode: SM_SMUSH | SM_EQUAL | SM_LOWLINE, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   '_',
			rch:   '_',
			want:  '_',
		},
		{
			name:  "multiple rules hierarchy over pair",
			state: &renderState{smushMode: SM_SMUSH | SM_HIERARCHY | SM_PAIR, previousCharWidth: 3, currCharWidth: 3, hardblank: '$'},
			lch:   '|',
			rch:   '[',
			want:  '[',
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.state.smushem(tt.lch, tt.rch)
			if got != tt.want {
				t.Errorf("smushem(%c, %c) = %c, want %c", tt.lch, tt.rch, got, tt.want)
			}
		})
	}
}

func TestSmushAmt(t *testing.T) {
	tests := []struct {
		name  string
		state *renderState
		want  int
	}{
		{
			name: "no kerning or smushing returns 0",
			state: &renderState{
				smushMode:     0,
				currCharWidth: 5,
				charHeight:    2,
			},
			want: 0,
		},
		{
			name: "empty output line LTR",
			state: &renderState{
				smushMode:     SM_KERN,
				currCharWidth: 3,
				charHeight:    2,
				right2left:    0,
				outputLine: [][]rune{
					make([]rune, 10),
					make([]rune, 10),
				},
				rowLengths: []int{0, 0},
				currChar:   []string{"ABC", "DEF"},
			},
			want: 3,
		},
		{
			name: "basic overlap LTR all spaces",
			state: &renderState{
				smushMode:     SM_KERN,
				currCharWidth: 5,
				charHeight:    2,
				right2left:    0,
				outlineLen:    5,
				outputLine: [][]rune{
					[]rune("ABC  "),
					[]rune("DEF  "),
				},
				rowLengths: []int{5, 5},
				currChar:   []string{"  XYZ", "  123"},
			},
			want: 4,
		},
		{
			name: "overlap with non-space characters LTR",
			state: &renderState{
				smushMode:     SM_KERN,
				currCharWidth: 3,
				charHeight:    2,
				right2left:    0,
				outlineLen:    3,
				outputLine: [][]rune{
					[]rune("ABC"),
					[]rune("DEF"),
				},
				rowLengths: []int{3, 3},
				currChar:   []string{" XY", " 12"},
			},
			want: 1,
		},
		{
			name: "smushing mode with matching rule",
			state: &renderState{
				smushMode:         SM_SMUSH | SM_EQUAL,
				currCharWidth:     3,
				previousCharWidth: 3,
				charHeight:        2,
				right2left:        0,
				outlineLen:        3,
				hardblank:         '$',
				outputLine: [][]rune{
					[]rune("AAA"),
					[]rune("BBB"),
				},
				rowLengths: []int{3, 3},
				currChar:   []string{"AAA", "BBB"},
			},
			want: 1,
		},
		{
			name: "RTL processing",
			state: &renderState{
				smushMode:     SM_KERN,
				currCharWidth: 3,
				charHeight:    2,
				right2left:    1,
				outputLine: [][]rune{
					[]rune("  ABC"),
					[]rune("  DEF"),
				},
				rowLengths: []int{5, 5},
				currChar:   []string{"XY ", "12 "},
			},
			want: 2,
		},
		{
			name: "minimum across rows",
			state: &renderState{
				smushMode:     SM_KERN,
				currCharWidth: 5,
				charHeight:    3,
				right2left:    0,
				outlineLen:    5,
				outputLine: [][]rune{
					[]rune("ABC  "), // Can overlap 4
					[]rune("DEF  "), // Can overlap 4
					[]rune("GHIJK"), // Can overlap 0
				},
				rowLengths: []int{5, 5, 5},
				currChar:   []string{"  XYZ", "  123", "MNOPQ"},
			},
			want: 0,
		},
		{
			name: "all spaces in current char",
			state: &renderState{
				smushMode:     SM_KERN,
				currCharWidth: 3,
				charHeight:    2,
				right2left:    0,
				outlineLen:    3,
				outputLine: [][]rune{
					[]rune("ABC"),
					[]rune("DEF"),
				},
				rowLengths: []int{3, 3},
				currChar:   []string{"   ", "   "},
			},
			want: 3,
		},
		{
			name: "edge case empty current char",
			state: &renderState{
				smushMode:     SM_KERN,
				currCharWidth: 0,
				charHeight:    2,
				right2left:    0,
				outlineLen:    3,
				outputLine: [][]rune{
					[]rune("ABC"),
					[]rune("DEF"),
				},
				rowLengths: []int{3, 3},
				currChar:   []string{"", ""},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize outputLine with sufficient size if needed
			for i := range tt.state.outputLine {
				if len(tt.state.outputLine[i]) < 100 {
					extended := make([]rune, 100)
					copy(extended, tt.state.outputLine[i])
					tt.state.outputLine[i] = extended
				}
			}
			
			got := tt.state.smushAmt()
			if got != tt.want {
				t.Errorf("smushAmt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkSmushem(b *testing.B) {
	state := &renderState{
		smushMode:         SM_SMUSH | SM_EQUAL | SM_HIERARCHY,
		previousCharWidth: 3,
		currCharWidth:     3,
		hardblank:         '$',
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = state.smushem('A', 'B')
	}
}

func BenchmarkSmushAmt(b *testing.B) {
	state := &renderState{
		smushMode:     SM_KERN,
		currCharWidth: 5,
		charHeight:    3,
		right2left:    0,
		outlineLen:    10,
		outputLine: [][]rune{
			[]rune("Hello World"),
			[]rune("Test String"),
			[]rune("Benchmark!!"),
		},
		rowLengths: []int{11, 11, 11},
		currChar:   []string{"  ABC", "  DEF", "  GHI"},
	}
	
	// Ensure outputLine has sufficient capacity
	for i := range state.outputLine {
		extended := make([]rune, 100)
		copy(extended, state.outputLine[i])
		state.outputLine[i] = extended
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = state.smushAmt()
	}
}