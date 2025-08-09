// Package common provides shared constants and types for internal packages.
// These constants must match the public API in the figgo package.
package common

import "errors"

// Layout bit constants (must match public API in figgo package)
const (
	// FitFullWidth displays characters at full width with no overlap
	FitFullWidth = 0
	// FitKerning displays characters with minimal spacing, no overlap (bit 6)
	FitKerning = 1 << 6
	// FitSmushing allows characters to overlap using smushing rules (bit 7)
	FitSmushing = 1 << 7
)

// Smushing rule constants (bits 0-5, must match public API)
const (
	// RuleEqualChar merges equal characters into one (bit 0)
	RuleEqualChar = 1 << 0
	// RuleUnderscore allows underscores to merge with certain characters (bit 1)
	RuleUnderscore = 1 << 1
	// RuleHierarchy uses character hierarchy to determine which survives (bit 2)
	RuleHierarchy = 1 << 2
	// RuleOppositePair merges opposite bracket pairs into | (bit 3)
	RuleOppositePair = 1 << 3
	// RuleBigX merges diagonal pairs to form X patterns (bit 4)
	RuleBigX = 1 << 4
	// RuleHardblank merges two hardblanks into one (bit 5)
	RuleHardblank = 1 << 5
)

// Common errors (must match public API in figgo package)
var (
	// ErrUnknownFont is returned when font is nil
	ErrUnknownFont = errors.New("unknown font")
	// ErrUnsupportedRune is returned when a rune is not supported by the font
	ErrUnsupportedRune = errors.New("unsupported rune")
	// ErrBadFontFormat is returned when font has invalid structure
	ErrBadFontFormat = errors.New("bad font format")
	// ErrLayoutConflict is returned when multiple fitting modes are set
	ErrLayoutConflict = errors.New("layout conflict")
)
