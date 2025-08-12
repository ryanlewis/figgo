package renderer

import (
	"errors"
	"strings"
	"testing"

	"github.com/ryanlewis/figgo/internal/parser"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		font    *parser.Font
		opts    *Options
		want    string
		wantErr error
	}{
		{
			name:    "nil font returns error",
			text:    "test",
			font:    nil,
			opts:    nil,
			want:    "",
			wantErr: errors.New("font cannot be nil"),
		},
		{
			name: "empty text returns empty lines",
			text: "",
			font: &parser.Font{
				Height:     3,
				Hardblank:  '$',
				Characters: map[rune][]string{},
			},
			opts: nil,
			want: "\n\n",
		},
		{
			name: "single character renders correctly",
			text: "A",
			font: &parser.Font{
				Height:    3,
				Hardblank: '$',
				Characters: map[rune][]string{
					'A': {"AAA", "A$A", "A$A"},
				},
			},
			opts: &Options{Layout: 0},
			want: "AAA\nA A\nA A",
		},
		{
			name: "hardblank replacement",
			text: "B",
			font: &parser.Font{
				Height:    2,
				Hardblank: '#',
				Characters: map[rune][]string{
					'B': {"B#B", "BBB"},
				},
			},
			opts: nil,
			want: "B B\nBBB",
		},
		{
			name: "multiple characters with full width",
			text: "AB",
			font: &parser.Font{
				Height:    2,
				Hardblank: '$',
				Characters: map[rune][]string{
					'A': {"AA", "AA"},
					'B': {"BB", "BB"},
				},
			},
			opts: &Options{Layout: 0}, // Full width
			want: "AABB\nAABB",
		},
		{
			name: "newline character is skipped",
			text: "A\nB",
			font: &parser.Font{
				Height:    2,
				Hardblank: '$',
				Characters: map[rune][]string{
					'A': {"A", "A"},
					'B': {"B", "B"},
				},
			},
			opts: nil,
			want: "AB\nAB",
		},
		{
			name: "control characters are skipped",
			text: "A\x01B",
			font: &parser.Font{
				Height:    2,
				Hardblank: '$',
				Characters: map[rune][]string{
					'A': {"A", "A"},
					'B': {"B", "B"},
				},
			},
			opts: nil,
			want: "AB\nAB",
		},
		{
			name: "tab is converted to space",
			text: "A\tB",
			font: &parser.Font{
				Height:    2,
				Hardblank: '$',
				Characters: map[rune][]string{
					'A': {"A", "A"},
					'B': {"B", "B"},
					' ': {" ", " "},
				},
			},
			opts: nil,
			want: "A B\nA B",
		},
		{
			name: "unknown rune with fallback",
			text: "X",
			font: &parser.Font{
				Height:    2,
				Hardblank: '$',
				Characters: map[rune][]string{
					'?': {"?", "?"},
				},
			},
			opts: &Options{UnknownRune: func() *rune { r := '?'; return &r }()},
			want: "?\n?",
		},
		{
			name: "unknown rune without fallback returns error",
			text: "X",
			font: &parser.Font{
				Height:     2,
				Hardblank:  '$',
				Characters: map[rune][]string{},
			},
			opts:    nil,
			wantErr: errors.New("unsupported rune: X"),
		},
		{
			name: "trim whitespace option",
			text: "A",
			font: &parser.Font{
				Height:    2,
				Hardblank: '$',
				Characters: map[rune][]string{
					'A': {"A  ", "A  "},
				},
			},
			opts: &Options{TrimWhitespace: true},
			want: "A\nA",
		},
		{
			name: "right to left print direction",
			text: "AB",
			font: &parser.Font{
				Height:         2,
				Hardblank:      '$',
				PrintDirection: 1,
				Characters: map[rune][]string{
					'A': {"A", "A"},
					'B': {"B", "B"},
				},
			},
			opts: nil,
			want: "BA\nBA",
		},
		{
			name: "print direction from options overrides font",
			text: "AB",
			font: &parser.Font{
				Height:         2,
				Hardblank:      '$',
				PrintDirection: 0,
				Characters: map[rune][]string{
					'A': {"A", "A"},
					'B': {"B", "B"},
				},
			},
			opts: &Options{PrintDirection: func() *int { d := 1; return &d }()},
			want: "BA\nBA",
		},
		{
			name: "invalid glyph height returns false from addChar",
			text: "A",
			font: &parser.Font{
				Height:    3,
				Hardblank: '$',
				Characters: map[rune][]string{
					'A': {"A", "A"}, // Only 2 lines instead of 3
				},
			},
			opts: nil,
			want: "\n\n",
		},
		{
			name: "full layout from font",
			text: "AB",
			font: &parser.Font{
				Height:        2,
				Hardblank:     '$',
				FullLayout:    128, // Smushing mode
				FullLayoutSet: true,
				Characters: map[rune][]string{
					'A': {"A ", "A "},
					'B': {" B", " B"},
				},
			},
			opts: nil,
			want: "AB\nAB",
		},
		{
			name: "old layout kerning mode",
			text: "AB",
			font: &parser.Font{
				Height:    2,
				Hardblank: '$',
				OldLayout: 0, // Kerning
				Characters: map[rune][]string{
					'A': {"A ", "A "},
					'B': {" B", " B"},
				},
			},
			opts: nil,
			want: "AB\nAB",
		},
		{
			name: "old layout full width mode",
			text: "AB",
			font: &parser.Font{
				Height:    2,
				Hardblank: '$',
				OldLayout: -1, // Full width
				Characters: map[rune][]string{
					'A': {"A", "A"},
					'B': {"B", "B"},
				},
			},
			opts: nil,
			want: "AB\nAB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Render(tt.text, tt.font, tt.opts)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Render() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				return
			}
			if err != nil {
				t.Errorf("Render() unexpected error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Render() got = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLayoutToSmushMode(t *testing.T) {
	tests := []struct {
		name   string
		layout int
		want   int
	}{
		{
			name:   "full width (no bits set)",
			layout: 0,
			want:   0,
		},
		{
			name:   "kerning mode",
			layout: 1 << 6, // 64
			want:   SMKern,
		},
		{
			name:   "smushing mode with no rules",
			layout: 1 << 7, // 128
			want:   SMSmush,
		},
		{
			name:   "smushing with equal char rule",
			layout: (1 << 7) | (1 << 0), // 128 + 1
			want:   SMSmush | SMEqual,
		},
		{
			name:   "smushing with underscore rule",
			layout: (1 << 7) | (1 << 1), // 128 + 2
			want:   SMSmush | SMLowline,
		},
		{
			name:   "smushing with hierarchy rule",
			layout: (1 << 7) | (1 << 2), // 128 + 4
			want:   SMSmush | SMHierarchy,
		},
		{
			name:   "smushing with opposite pair rule",
			layout: (1 << 7) | (1 << 3), // 128 + 8
			want:   SMSmush | SMPair,
		},
		{
			name:   "smushing with big X rule",
			layout: (1 << 7) | (1 << 4), // 128 + 16
			want:   SMSmush | SMBigX,
		},
		{
			name:   "smushing with hardblank rule",
			layout: (1 << 7) | (1 << 5), // 128 + 32
			want:   SMSmush | SMHardblank,
		},
		{
			name:   "smushing with all rules",
			layout: (1 << 7) | 63, // 128 + 63
			want:   SMSmush | 63,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := layoutToSmushMode(tt.layout)
			if got != tt.want {
				t.Errorf("layoutToSmushMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOldLayoutToSmushMode(t *testing.T) {
	tests := []struct {
		name      string
		oldLayout int
		want      int
	}{
		{
			name:      "full width mode (-1)",
			oldLayout: -1,
			want:      0,
		},
		{
			name:      "kerning mode (0)",
			oldLayout: 0,
			want:      SMKern,
		},
		{
			name:      "smushing with rules (1)",
			oldLayout: 1,
			want:      SMSmush | 1,
		},
		{
			name:      "smushing with all rules (63)",
			oldLayout: 63,
			want:      SMSmush | 63,
		},
		{
			name:      "invalid negative value",
			oldLayout: -5,
			want:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := oldLayoutToSmushMode(tt.oldLayout)
			if got != tt.want {
				t.Errorf("oldLayoutToSmushMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddChar(t *testing.T) {
	tests := []struct {
		name  string
		state *renderState
		glyph []string
		want  bool
	}{
		{
			name: "invalid glyph height",
			state: &renderState{
				charHeight:      3,
				outputLine:      make([][]rune, 3),
				rowLengths:      make([]int, 3),
				outlineLenLimit: 100,
			},
			glyph: []string{"A", "A"}, // Only 2 lines
			want:  false,
		},
		{
			name: "character doesn't fit",
			state: &renderState{
				charHeight:      2,
				outputLine:      make([][]rune, 2),
				rowLengths:      make([]int, 2),
				outlineLenLimit: 1,
				outlineLen:      0,
			},
			glyph: []string{"AAA", "AAA"},
			want:  false,
		},
		{
			name: "successful add with empty output",
			state: &renderState{
				charHeight: 2,
				outputLine: [][]rune{
					make([]rune, 100),
					make([]rune, 100),
				},
				rowLengths:      []int{0, 0},
				outlineLenLimit: 100,
				outlineLen:      0,
				hardblank:       '$',
			},
			glyph: []string{"AB", "AB"},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.state.addChar(tt.glyph)
			if got != tt.want {
				t.Errorf("addChar() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOutputToString(t *testing.T) {
	tests := []struct {
		name  string
		state *renderState
		want  string
	}{
		{
			name: "empty output",
			state: &renderState{
				charHeight: 0,
				outputLine: [][]rune{},
			},
			want: "",
		},
		{
			name: "single line",
			state: &renderState{
				charHeight: 1,
				outputLine: [][]rune{
					[]rune("Hello"),
				},
				rowLengths: []int{5},
				hardblank:  '$',
			},
			want: "Hello",
		},
		{
			name: "multiple lines with hardblank replacement",
			state: &renderState{
				charHeight: 3,
				outputLine: [][]rune{
					[]rune("A$B"),
					[]rune("C$D"),
					[]rune("E$F"),
				},
				rowLengths: []int{3, 3, 3},
				hardblank:  '$',
			},
			want: "A B\nC D\nE F",
		},
		{
			name: "trim whitespace enabled",
			state: &renderState{
				charHeight: 2,
				outputLine: [][]rune{
					[]rune("ABC   "),
					[]rune("DEF   "),
				},
				rowLengths:     []int{6, 6},
				hardblank:      '$',
				trimWhitespace: true,
			},
			want: "ABC\nDEF",
		},
		{
			name: "partial row lengths",
			state: &renderState{
				charHeight: 2,
				outputLine: [][]rune{
					[]rune("ABCDEFGH"),
					[]rune("12345678"),
				},
				rowLengths: []int{3, 5},
				hardblank:  '$',
			},
			want: "ABC\n12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.state.outputToString()
			if got != tt.want {
				t.Errorf("outputToString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderIntegration(t *testing.T) {
	// Create a simple test font
	testFont := &parser.Font{
		Height:    3,
		Hardblank: '$',
		OldLayout: 0, // Kerning
		Characters: map[rune][]string{
			'H': {
				"H  H",
				"HHHH",
				"H  H",
			},
			'I': {
				"III",
				" I ",
				"III",
			},
			' ': {
				"   ",
				"   ",
				"   ",
			},
		},
	}

	tests := []struct {
		name string
		text string
		opts *Options
		want string
	}{
		{
			name: "simple HI with kerning",
			text: "HI",
			opts: nil,
			want: "H  HIII\nHHHH I \nH  HIII",
		},
		{
			name: "HI with space",
			text: "H I",
			opts: nil,
			want: strings.Join([]string{
				"H  H   III",
				"HHHH    I ",
				"H  H   III",
			}, "\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Render(tt.text, testFont, tt.opts)
			if err != nil {
				t.Errorf("Render() unexpected error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Render() got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}
