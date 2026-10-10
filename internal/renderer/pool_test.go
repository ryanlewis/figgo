package renderer

import (
	"reflect"
	"testing"

	"github.com/ryanlewis/figgo/internal/debug"
)

// A pooled renderState must not carry options from its previous render:
// a leaked outputWidth, for example, right-justifies an RTL render that
// set no width.
func TestRenderStateResetClearsPerRenderFields(t *testing.T) {
	state := &renderState{
		currentChar:          []string{"x"},
		outlineLen:           7,
		outlineLenLimit:      79,
		outputWidth:          80,
		currentCharWidth:     3,
		previousCharWidth:    2,
		charHeight:           9,
		right2left:           1,
		smushMode:            SMSmush,
		inputCount:           5,
		lastWordBreak:        4,
		wordbreakmode:        1,
		hardblank:            '#',
		trimWhitespace:       true,
		processingSpaceGlyph: true,
		debug:                &debug.Session{},
	}

	state.reset(4, '$', 2)

	if len(state.outputLine) != 4 || len(state.rowLengths) != 4 {
		t.Fatalf("buffers sized %d/%d, want 4/4", len(state.outputLine), len(state.rowLengths))
	}

	// Compare everything except the reusable buffers.
	got := *state
	got.outputLine, got.rowLengths, got.inputBuffer, got.outputBuffer = nil, nil, nil, nil
	want := renderState{
		charHeight:      4,
		hardblank:       '$',
		outlineLenLimit: defaultOutlineLimit,
		lastWordBreak:   -1,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("after reset:\n got  %+v\n want %+v", got, want)
	}
}
