package figgo

import (
	"errors"
	"fmt"
	"strings"
)

// Layout represents a bitmask for fitting modes and smushing rules.
// The layout determines how glyphs are combined when rendering text.
//
// Bits 6-8 control fitting modes (exactly one must be set):
//   - Bit 6: FitFullWidth - Full width spacing, no overlap
//   - Bit 7: FitKerning - Minimal spacing, no character overlap
//   - Bit 8: FitSmushing - Characters can overlap using smushing rules
//
// Bits 9-14 control smushing rules (only apply when FitSmushing is active):
//   - Bit 9: RuleEqualChar - Equal characters merge
//   - Bit 10: RuleUnderscore - Underscores merge with certain characters
//   - Bit 11: RuleHierarchy - Character hierarchy determines which survives
//   - Bit 12: RuleOppositePair - Opposite pairs merge (brackets, parens)
//   - Bit 13: RuleBigX - Diagonal pairs form X patterns
//   - Bit 14: RuleHardblank - Hardblanks merge into one
type Layout uint32

// Fitting mode constants (bits 6-8)
const (
	// FitFullWidth displays characters at full width with no overlap (bit 6)
	FitFullWidth Layout = 0x00000040

	// FitKerning displays characters with minimal spacing, no overlap (bit 7)
	FitKerning Layout = 0x00000080

	// FitSmushing allows characters to overlap using smushing rules (bit 8)
	FitSmushing Layout = 0x00000100
)

// Smushing rule constants (bits 9-14)
const (
	// RuleEqualChar merges equal characters into one (bit 9)
	RuleEqualChar Layout = 0x00000200

	// RuleUnderscore allows underscores to merge with certain characters (bit 10)
	RuleUnderscore Layout = 0x00000400

	// RuleHierarchy uses character hierarchy to determine which survives (bit 11)
	RuleHierarchy Layout = 0x00000800

	// RuleOppositePair merges opposite bracket pairs into | (bit 12)
	RuleOppositePair Layout = 0x00001000

	// RuleBigX merges diagonal pairs to form X patterns (bit 13)
	RuleBigX Layout = 0x00002000

	// RuleHardblank merges two hardblanks into one (bit 14)
	RuleHardblank Layout = 0x00004000
)

// ErrLayoutConflict is returned when multiple fitting modes are set simultaneously
var ErrLayoutConflict = errors.New("layout conflict: multiple fitting modes set")

// NormalizeLayout validates and normalizes a Layout value.
// It ensures exactly one fitting mode is set:
//   - If both FitKerning and FitSmushing are set, returns ErrLayoutConflict
//   - If both FitFullWidth and FitKerning are set, returns ErrLayoutConflict
//   - If both FitFullWidth and FitSmushing are set, returns ErrLayoutConflict
//   - If no fitting mode is set, defaults to FitFullWidth
//
// Rule bits are preserved but only have effect when FitSmushing is active.
func NormalizeLayout(layout Layout) (Layout, error) {
	fittingModes := layout & (FitFullWidth | FitKerning | FitSmushing)

	// Count how many fitting modes are set
	count := 0
	if fittingModes&FitFullWidth != 0 {
		count++
	}
	if fittingModes&FitKerning != 0 {
		count++
	}
	if fittingModes&FitSmushing != 0 {
		count++
	}

	// Check for conflicts
	if count > 1 {
		return 0, ErrLayoutConflict
	}

	// Default to FitFullWidth if no fitting mode is set
	if count == 0 {
		layout |= FitFullWidth
	}

	return layout, nil
}

// NormalizeOldLayout converts an OldLayout integer to a Layout bitmask.
// This handles the legacy FIGfont format where:
//   - -1: Full width
//   - 0: Full width
//   - 1: Kerning
//   - Even numbers >= 2: Smushing with rules encoded in bits
func NormalizeOldLayout(oldLayout int32) Layout {
	if oldLayout <= 0 {
		return FitFullWidth
	}

	if oldLayout == 1 {
		return FitKerning
	}

	// oldLayout >= 2 means smushing with rules
	layout := FitSmushing

	// Extract rule bits from the old layout
	// In old format, bit 0 = equal char, bit 1 = underscore, etc.
	if oldLayout&2 != 0 {
		layout |= RuleEqualChar
	}
	if oldLayout&4 != 0 {
		layout |= RuleUnderscore
	}
	if oldLayout&8 != 0 {
		layout |= RuleHierarchy
	}
	if oldLayout&16 != 0 {
		layout |= RuleOppositePair
	}
	if oldLayout&32 != 0 {
		layout |= RuleBigX
	}
	if oldLayout&64 != 0 {
		layout |= RuleHardblank
	}

	return layout
}

// HasRule checks if a specific smushing rule is enabled in the layout.
// Note that rules only have effect when FitSmushing is the active fitting mode.
func (l Layout) HasRule(rule Layout) bool {
	// Only check rule bits (9-14)
	ruleMask := RuleEqualChar | RuleUnderscore | RuleHierarchy |
		RuleOppositePair | RuleBigX | RuleHardblank

	// Ensure we're only checking valid rule bits
	if rule&ruleMask == 0 {
		return false
	}

	return l&rule != 0
}

// FittingMode returns only the fitting mode bits from the layout.
// This will return one or more of: FitFullWidth, FitKerning, FitSmushing.
// Note that a valid normalized layout should have exactly one fitting mode.
func (l Layout) FittingMode() Layout {
	return l & (FitFullWidth | FitKerning | FitSmushing)
}

// Rules returns only the smushing rule bits from the layout.
// This excludes the fitting mode bits and returns the combination of
// all active smushing rules.
func (l Layout) Rules() Layout {
	return l & (RuleEqualChar | RuleUnderscore | RuleHierarchy |
		RuleOppositePair | RuleBigX | RuleHardblank)
}

// String returns a human-readable representation of the layout.
// It shows the fitting mode and any active smushing rules.
func (l Layout) String() string {
	if l == 0 {
		return "0x00000000"
	}

	var parts []string

	// Add fitting modes
	if l&FitFullWidth != 0 {
		parts = append(parts, "FitFullWidth")
	}
	if l&FitKerning != 0 {
		parts = append(parts, "FitKerning")
	}
	if l&FitSmushing != 0 {
		parts = append(parts, "FitSmushing")
	}

	// Add smushing rules
	if l&RuleEqualChar != 0 {
		parts = append(parts, "RuleEqualChar")
	}
	if l&RuleUnderscore != 0 {
		parts = append(parts, "RuleUnderscore")
	}
	if l&RuleHierarchy != 0 {
		parts = append(parts, "RuleHierarchy")
	}
	if l&RuleOppositePair != 0 {
		parts = append(parts, "RuleOppositePair")
	}
	if l&RuleBigX != 0 {
		parts = append(parts, "RuleBigX")
	}
	if l&RuleHardblank != 0 {
		parts = append(parts, "RuleHardblank")
	}

	// Check for unknown bits
	knownBits := FitFullWidth | FitKerning | FitSmushing |
		RuleEqualChar | RuleUnderscore | RuleHierarchy |
		RuleOppositePair | RuleBigX | RuleHardblank

	if l&^knownBits != 0 {
		// Has unknown bits, return hex representation
		return fmt.Sprintf("0x%08X", uint32(l))
	}

	if len(parts) == 0 {
		return fmt.Sprintf("0x%08X", uint32(l))
	}

	return strings.Join(parts, "|")
}

