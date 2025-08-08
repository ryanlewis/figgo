package figgo

import (
	"errors"
	"testing"
)

func TestFontStruct(t *testing.T) {
	// Test that Font struct has the required fields
	font := &Font{
		Name:           "standard",
		Hardblank:      '$',
		Height:         8,
		Baseline:       6,
		MaxLen:         16,
		OldLayout:      -1,
		FullLayout:     FitFullWidth,
		PrintDirection: 0,
		CommentLines:   25,
		Glyphs:         make(map[rune][]string),
	}

	if font.Name != "standard" {
		t.Errorf("expected Name to be 'standard', got %s", font.Name)
	}
	if font.Hardblank != '$' {
		t.Errorf("expected Hardblank to be '$', got %c", font.Hardblank)
	}
	if font.Height != 8 {
		t.Errorf("expected Height to be 8, got %d", font.Height)
	}
	if font.Baseline != 6 {
		t.Errorf("expected Baseline to be 6, got %d", font.Baseline)
	}
	if font.MaxLen != 16 {
		t.Errorf("expected MaxLen to be 16, got %d", font.MaxLen)
	}
	if font.OldLayout != -1 {
		t.Errorf("expected OldLayout to be -1, got %d", font.OldLayout)
	}
	if font.FullLayout != FitFullWidth {
		t.Errorf("expected FullLayout to be FitFullWidth, got %d", font.FullLayout)
	}
	if font.PrintDirection != 0 {
		t.Errorf("expected PrintDirection to be 0, got %d", font.PrintDirection)
	}
	if font.CommentLines != 25 {
		t.Errorf("expected CommentLines to be 25, got %d", font.CommentLines)
	}
	if font.Glyphs == nil {
		t.Error("expected Glyphs to be initialized")
	}
}

func TestErrorTypes(t *testing.T) {
	// Test that error variables are defined
	if !errors.Is(ErrUnknownFont, ErrUnknownFont) {
		t.Error("ErrUnknownFont not defined properly")
	}
	if !errors.Is(ErrUnsupportedRune, ErrUnsupportedRune) {
		t.Error("ErrUnsupportedRune not defined properly")
	}
	if !errors.Is(ErrBadFontFormat, ErrBadFontFormat) {
		t.Error("ErrBadFontFormat not defined properly")
	}
	if !errors.Is(ErrLayoutConflict, ErrLayoutConflict) {
		t.Error("ErrLayoutConflict not defined properly")
	}
}

func TestOptionPattern(t *testing.T) {
	// Test that Option type is defined
	var opt Option
	if opt == nil {
		// Option should be a function type
	}

	// Test WithLayout option
	opt = WithLayout(FitSmushing | RuleEqualChar)
	if opt == nil {
		t.Error("WithLayout should return an Option")
	}

	// Test WithPrintDirection option
	opt = WithPrintDirection(1)
	if opt == nil {
		t.Error("WithPrintDirection should return an Option")
	}
}
