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

// ErrInvalidOldLayout is returned when OldLayout is outside the valid range (-1..63)
var ErrInvalidOldLayout = errors.New("invalid OldLayout: must be in range -1..63")

// ErrInvalidFullLayout is returned when FullLayout is outside the valid range (0..32767)
var ErrInvalidFullLayout = errors.New("invalid FullLayout: must be in range 0..32767")

// AxisMode represents the fitting mode for horizontal or vertical axis
type AxisMode int

const (
	// ModeFull displays with full width/height spacing
	ModeFull AxisMode = iota
	// ModeFitting displays with minimal spacing (kerning)
	ModeFitting
	// ModeSmushingControlled allows overlap using specific smushing rules
	ModeSmushingControlled
	// ModeSmushingUniversal allows overlap using universal smushing
	ModeSmushingUniversal
)

// NormalizedLayout represents the fully normalized layout for both axes
type NormalizedLayout struct {
	HorzMode  AxisMode
	HorzRules uint16 // rules 1–6 encoded as bits 0..5
	VertMode  AxisMode
	VertRules uint16 // rules 1–5 encoded as bits 0..4
}

// Constants for FullLayout interpretation (from FIGfont v2 spec)
const (
	// Horizontal layout bits
	fullLayoutHorzRule1    = 1   // Equal character rule
	fullLayoutHorzRule2    = 2   // Underscore rule
	fullLayoutHorzRule3    = 4   // Hierarchy rule
	fullLayoutHorzRule4    = 8   // Opposite pair rule
	fullLayoutHorzRule5    = 16  // Big X rule
	fullLayoutHorzRule6    = 32  // Hardblank rule
	fullLayoutHorzFitting  = 64  // Horizontal fitting (kerning)
	fullLayoutHorzSmushing = 128 // Horizontal smushing

	// Vertical layout bits
	fullLayoutVertRule1    = 256   // Vertical equal character
	fullLayoutVertRule2    = 512   // Vertical underscore
	fullLayoutVertRule3    = 1024  // Vertical hierarchy
	fullLayoutVertRule4    = 2048  // Vertical horizontal line
	fullLayoutVertRule5    = 4096  // Vertical vertical line
	fullLayoutVertFitting  = 8192  // Vertical fitting
	fullLayoutVertSmushing = 16384 // Vertical smushing

	// Bit masks for extracting rule bits
	horzRuleMask = 0x3F // Bits 0-5: horizontal rules (6 bits)
	vertRuleMask = 0x1F // Bits 0-4: vertical rules (5 bits)
	
	// Bit shift for vertical rules
	vertRuleShift = 8 // Shift right 8 bits to get vertical rules
)

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

// NormalizeLayoutFromHeader normalizes layout from FIGfont header values.
// It handles both OldLayout and FullLayout with proper precedence:
//   - If fullLayoutSet is true, FullLayout takes precedence (ignores OldLayout)
//   - If fullLayoutSet is false, derives layout from OldLayout only
//
// OldLayout valid range: -1..63
//   - -1: Horizontal full width
//   - 0: Horizontal fitting (kerning)
//   - >0: Horizontal smushing with rules from bits 1-6
//
// FullLayout valid range: 0..32767
//   - Supports both horizontal and vertical layout modes
//   - Universal smushing = smushing bit set with no rule bits
func NormalizeLayoutFromHeader(oldLayout, fullLayout int, fullLayoutSet bool) (NormalizedLayout, error) {
	// Validate ranges
	if oldLayout < -1 || oldLayout > 63 {
		return NormalizedLayout{}, ErrInvalidOldLayout
	}
	if fullLayout < 0 || fullLayout > 32767 {
		return NormalizedLayout{}, ErrInvalidFullLayout
	}

	var result NormalizedLayout

	if fullLayoutSet {
		// FullLayout takes precedence when present
		result = parseFullLayout(fullLayout)
	} else {
		// Use OldLayout only
		result = parseOldLayout(oldLayout)
	}

	return result, nil
}

// parseOldLayout converts OldLayout to NormalizedLayout
func parseOldLayout(oldLayout int) NormalizedLayout {
	result := NormalizedLayout{
		VertMode: ModeFull, // OldLayout doesn't specify vertical, default to full
	}

	switch {
	case oldLayout == -1:
		result.HorzMode = ModeFull
	case oldLayout == 0:
		result.HorzMode = ModeFitting
	default:
		// oldLayout > 0: smushing with rules
		result.HorzMode = ModeSmushingControlled
		// Extract rule bits safely
		maskedBits := oldLayout & horzRuleMask
		if maskedBits < 0 || maskedBits > 63 {
			// This shouldn't happen due to validation and masking, but check anyway
			maskedBits = 0
		}
		result.HorzRules = uint16(maskedBits)
	}

	return result
}

// parseFullLayout converts FullLayout to NormalizedLayout
func parseFullLayout(fullLayout int) NormalizedLayout {
	var result NormalizedLayout

	// Parse horizontal layout
	horzRuleBits := fullLayout & horzRuleMask // Bits 0-5: horizontal rules
	horzHasFitting := (fullLayout & fullLayoutHorzFitting) != 0
	horzHasSmushing := (fullLayout & fullLayoutHorzSmushing) != 0

	switch {
	case horzHasSmushing && horzRuleBits == 0:
		// Universal smushing: smushing bit set, no rule bits
		result.HorzMode = ModeSmushingUniversal
	case horzHasSmushing && horzRuleBits != 0:
		// Controlled smushing: smushing bit set with rule bits
		result.HorzMode = ModeSmushingControlled
		// horzRuleBits is already masked to 6 bits (max 63), safe to convert
		if horzRuleBits < 0 || horzRuleBits > 63 {
			horzRuleBits = 0
		}
		result.HorzRules = uint16(horzRuleBits)
	case horzHasFitting:
		// Fitting (kerning) mode
		result.HorzMode = ModeFitting
	default:
		// Full width mode (default)
		result.HorzMode = ModeFull
	}

	// Parse vertical layout
	vertRuleBits := (fullLayout >> vertRuleShift) & vertRuleMask // Bits 8-12 mapped to 0-4: vertical rules
	vertHasFitting := (fullLayout & fullLayoutVertFitting) != 0
	vertHasSmushing := (fullLayout & fullLayoutVertSmushing) != 0

	switch {
	case vertHasSmushing && vertRuleBits == 0:
		// Universal smushing: smushing bit set, no rule bits
		result.VertMode = ModeSmushingUniversal
	case vertHasSmushing && vertRuleBits != 0:
		// Controlled smushing: smushing bit set with rule bits
		result.VertMode = ModeSmushingControlled
		// vertRuleBits is already masked to 5 bits (max 31), safe to convert
		if vertRuleBits < 0 || vertRuleBits > 31 {
			vertRuleBits = 0
		}
		result.VertRules = uint16(vertRuleBits)
	case vertHasFitting:
		// Fitting mode
		result.VertMode = ModeFitting
	default:
		// Full height mode (default)
		result.VertMode = ModeFull
	}

	return result
}

// ToLayout converts NormalizedLayout to the simplified horizontal Layout bitmask
// used by the rendering engine. This only considers horizontal layout settings.
func (nl NormalizedLayout) ToLayout() Layout {
	var layout Layout

	switch nl.HorzMode {
	case ModeFull:
		layout = FitFullWidth
	case ModeFitting:
		layout = FitKerning
	case ModeSmushingControlled:
		layout = FitSmushing
		// Map horizontal rules to Layout rule bits
		if nl.HorzRules&0x01 != 0 {
			layout |= RuleEqualChar
		}
		if nl.HorzRules&0x02 != 0 {
			layout |= RuleUnderscore
		}
		if nl.HorzRules&0x04 != 0 {
			layout |= RuleHierarchy
		}
		if nl.HorzRules&0x08 != 0 {
			layout |= RuleOppositePair
		}
		if nl.HorzRules&0x10 != 0 {
			layout |= RuleBigX
		}
		if nl.HorzRules&0x20 != 0 {
			layout |= RuleHardblank
		}
	case ModeSmushingUniversal:
		// Universal smushing: just the smushing mode, no rule bits
		layout = FitSmushing
	}

	return layout
}
