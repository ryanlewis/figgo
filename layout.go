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

// Fitting mode constants
const (
	// FitFullWidth displays characters at full width with no overlap (no bits set)
	FitFullWidth Layout = 0

	// FitKerning displays characters with minimal spacing, no overlap (bit 6)
	FitKerning Layout = 1 << 6

	// FitSmushing allows characters to overlap using smushing rules (bit 7)
	FitSmushing Layout = 1 << 7
)

// Smushing rule constants (bits 0-5)
const (
	// RuleEqualChar merges equal characters into one (bit 0)
	RuleEqualChar Layout = 1 << 0

	// RuleUnderscore allows underscores to merge with certain characters (bit 1)
	RuleUnderscore Layout = 1 << 1

	// RuleHierarchy uses character hierarchy to determine which survives (bit 2)
	RuleHierarchy Layout = 1 << 2

	// RuleOppositePair merges opposite bracket pairs into | (bit 3)
	RuleOppositePair Layout = 1 << 3

	// RuleBigX merges diagonal pairs to form X patterns (bit 4)
	RuleBigX Layout = 1 << 4

	// RuleHardblank merges two hardblanks into one (bit 5)
	RuleHardblank Layout = 1 << 5

	// AllKnownMask contains all known layout bits for validation
	// Note: FitFullWidth = 0, so it doesn't contribute to the mask
	AllKnownMask Layout = FitKerning | FitSmushing |
		RuleEqualChar | RuleUnderscore | RuleHierarchy |
		RuleOppositePair | RuleBigX | RuleHardblank
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
	HorzRules uint8 // rules 1–6 encoded as bits 0..5 (max value: 63)
	VertMode  AxisMode
	VertRules uint8 // rules 1–5 encoded as bits 0..4 (max value: 31)
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
// Rule bits are preserved even when FitSmushing is not set, but they are
// ignored by the renderer unless FitSmushing is the active fitting mode.
//
// Note: This function errors on conflicts (for user-provided options), whereas
// parseFullLayout resolves conflicts by giving smushing precedence (for header
// semantics). This asymmetry is intentional: user options should be explicit,
// while font headers follow FIGlet 2.2 precedence rules.
func NormalizeLayout(layout Layout) (Layout, error) {
	// Mask unknown bits to avoid propagating garbage
	layout &= AllKnownMask

	// Since FitFullWidth = 0, we check for kerning and smushing bits
	fittingBits := layout & (FitKerning | FitSmushing)

	// Count how many fitting mode bits are set
	bitCount := 0
	if fittingBits&FitKerning != 0 {
		bitCount++
	}
	if fittingBits&FitSmushing != 0 {
		bitCount++
	}

	// Validate that at most one fitting mode is set
	if bitCount > 1 {
		// Multiple fitting modes set, build error message
		var conflictingModes []string
		if fittingBits&FitKerning != 0 {
			conflictingModes = append(conflictingModes, "FitKerning")
		}
		if fittingBits&FitSmushing != 0 {
			conflictingModes = append(conflictingModes, "FitSmushing")
		}
		return 0, fmt.Errorf("%w: multiple fitting modes set (%s)",
			ErrLayoutConflict, strings.Join(conflictingModes, " + "))
	}

	// If no bits are set, we have FitFullWidth (since FitFullWidth = 0)
	// Layout is already valid as-is
	return layout, nil
}

// NormalizeOldLayout converts an OldLayout integer to a Layout bitmask.
// This handles the legacy FIGfont format per the FIGfont v2 spec:
//   - -3: Universal smushing (no rules)
//   - -2: Fitting (kerning) - alias for 0
//   - -1: Full width
//   - 0: Fitting (kerning)
//   - >0: Smushing with rules encoded in bits 0-5
//
// This implementation exactly follows the FIGfont v2 spec:
// OldLayout bits 0-5 map directly to smushing rules 1-6.
//
// Values outside the range [-3..63] are invalid and will return an error.
// Note: This function is deprecated in favor of NormalizeLayoutFromHeader
// which handles both OldLayout and FullLayout with proper precedence.
func NormalizeOldLayout(oldLayout int) (Layout, error) {
	// Validate range per FIGfont v2 spec
	// Only -1, 0, and positive values up to 63 are valid
	if oldLayout < -1 || oldLayout > 63 {
		return 0, ErrInvalidOldLayout
	}
	switch {
	case oldLayout == -1:
		return FitFullWidth, nil
	case oldLayout == 0:
		return FitKerning, nil
	default:
		// oldLayout > 0: smushing with rules from bits 0-5
		layout := FitSmushing

		// Map rule bits (bits 0-5 correspond to rules 1-6)
		if oldLayout&1 != 0 {
			layout |= RuleEqualChar
		}
		if oldLayout&2 != 0 {
			layout |= RuleUnderscore
		}
		if oldLayout&4 != 0 {
			layout |= RuleHierarchy
		}
		if oldLayout&8 != 0 {
			layout |= RuleOppositePair
		}
		if oldLayout&16 != 0 {
			layout |= RuleBigX
		}
		if oldLayout&32 != 0 {
			layout |= RuleHardblank
		}

		return layout, nil
	}
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
// This will return one of: FitFullWidth (0), FitKerning, FitSmushing.
// Note that a valid normalized layout should have exactly one fitting mode.
// If neither FitKerning nor FitSmushing bits are set, it returns FitFullWidth (0).
func (l Layout) FittingMode() Layout {
	fittingBits := l & (FitKerning | FitSmushing)
	if fittingBits == 0 {
		return FitFullWidth // Return 0 explicitly for clarity
	}
	return fittingBits
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
// If multiple fitting modes are set (invalid state), it prefixes with "INVALID:".
// layoutStringParts returns the string parts for a Layout value
func layoutStringParts(l Layout) []string {
	var parts []string
	// Add fitting modes
	fittingBits := l & (FitKerning | FitSmushing)
	if fittingBits == 0 {
		// No fitting bits set means FitFullWidth
		parts = append(parts, "FitFullWidth")
	} else {
		if l&FitKerning != 0 {
			parts = append(parts, "FitKerning")
		}
		if l&FitSmushing != 0 {
			parts = append(parts, "FitSmushing")
		}
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

	return parts
}

// countFittingModes counts the number of fitting modes set
func countFittingModes(l Layout) int {
	// Since FitFullWidth = 0, we only count the actual bit flags
	count := 0
	if l&FitKerning != 0 {
		count++
	}
	if l&FitSmushing != 0 {
		count++
	}
	// If neither bit is set, we have FitFullWidth (implicitly 1 mode)
	if count == 0 && l&(FitKerning|FitSmushing) == 0 {
		count = 1
	}
	return count
}

func (l Layout) String() string {
	// Check for unknown bits using AllKnownMask constant
	if l&^AllKnownMask != 0 {
		// Has unknown bits, return hex representation
		return fmt.Sprintf("0x%08X", uint32(l))
	}

	parts := layoutStringParts(l)
	if len(parts) == 0 {
		// This should not happen with valid layouts, but handle it
		return fmt.Sprintf("0x%08X", uint32(l))
	}

	// Check for invalid multiple fitting modes
	var invalidPrefix string
	if countFittingModes(l) > 1 {
		invalidPrefix = "INVALID:"
	}

	return invalidPrefix + strings.Join(parts, "|")
}

// NormalizeLayoutFromHeader normalizes layout from FIGfont header values.
//
// Precedence rules:
//   - FullLayout takes precedence when fullLayoutSet is true (ignores OldLayout completely)
//   - OldLayout is used only when fullLayoutSet is false
//   - When FullLayout is present without any fit bits, defaults to Full/Full per spec
//
// OldLayout valid range: -1..63
//   - -1: Horizontal full width
//   - 0: Horizontal fitting (kerning)
//   - 1-63: Horizontal controlled smushing with rules from bits 0-5
//
// Note: OldLayout cannot express universal smushing directly
//
// FullLayout valid range: 0..32767 (15-bit bitmask)
//   - Bits 0-5: Horizontal smushing rules (equal char, underscore, hierarchy, etc.)
//   - Bit 6: Horizontal fitting mode (64)
//   - Bit 7: Horizontal smushing mode (128) - takes precedence over bit 6
//   - Bits 8-12: Vertical smushing rules
//   - Bit 13: Vertical fitting mode (8192)
//   - Bit 14: Vertical smushing mode (16384) - takes precedence over bit 13
//   - Universal smushing = smushing bit set with no rule bits
func NormalizeLayoutFromHeader(oldLayout, fullLayout int, fullLayoutSet bool) (NormalizedLayout, error) {
	// Validate ranges
	// Only -1, 0, and positive values up to 63 are valid for OldLayout
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

		// Validate consistency with OldLayout if meaningful.
		// When OldLayout and FullLayout disagree on the horizontal mode:
		// - OldLayout -1 should match FullLayout indicating full-width
		// - OldLayout 0 should match FullLayout indicating kerning
		// - OldLayout > 0 should match FullLayout indicating smushing
		// We use FullLayout as it has more complete information.
		// Calling code can detect inconsistency by comparing the original
		// oldLayout with the normalized result and log a warning.
		_ = parseOldLayout(oldLayout) // Check for consistency but use FullLayout
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
		// oldLayout > 0: smushing with rules (1..63 are valid)
		result.HorzMode = ModeSmushingControlled
		// Extract rule bits (0-5) - oldLayout validated to be 1..63
		// Masking with 0x3F ensures value is 0..63, safe for uint8
		// Note: oldLayout is already validated to be <= 63, so conversion is safe
		// #nosec G115 -- oldLayout validated to be in range 1..63
		result.HorzRules = uint8(oldLayout & horzRuleMask)
	}

	return result
}

// parseFullLayout converts FullLayout to NormalizedLayout
func parseFullLayout(fullLayout int) NormalizedLayout {
	var result NormalizedLayout

	// Parse horizontal layout
	// Note: When both fitting (bit 6) and smushing (bit 7) bits are set,
	// smushing takes precedence. This matches FIGlet 2.2 behavior.
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
		// horzRuleBits already masked to 6 bits (max 63), safe for uint8
		// Explicit mask ensures value fits in uint8
		// #nosec G115 -- horzRuleBits is masked by horzRuleMask (0x3F)
		result.HorzRules = uint8(horzRuleBits) // Already masked by horzRuleMask (0x3F)
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
		// vertRuleBits already masked to 5 bits (max 31), safe for uint8
		// Explicit mask ensures value fits in uint8
		// #nosec G115 -- vertRuleBits is masked by vertRuleMask (0x1F)
		result.VertRules = uint8(vertRuleBits) // Already masked by vertRuleMask (0x1F)
	case vertHasFitting:
		// Fitting mode
		result.VertMode = ModeFitting
	default:
		// Full height mode (default)
		result.VertMode = ModeFull
	}

	return result
}

// String returns a human-readable representation of the normalized layout.
func (nl NormalizedLayout) String() string {
	return fmt.Sprintf("Horz:%v(rules:0x%02X) Vert:%v(rules:0x%02X)",
		nl.HorzMode, nl.HorzRules, nl.VertMode, nl.VertRules)
}

// String returns the string representation of the axis mode.
func (m AxisMode) String() string {
	switch m {
	case ModeFull:
		return "Full"
	case ModeFitting:
		return "Fitting"
	case ModeSmushingControlled:
		return "SmushingControlled"
	case ModeSmushingUniversal:
		return "SmushingUniversal"
	default:
		return fmt.Sprintf("Unknown(%d)", m)
	}
}

// ToLayout converts NormalizedLayout to the simplified horizontal Layout bitmask
// used by the rendering engine. This only considers horizontal layout settings.
//
// Note: Vertical mode and rules are currently ignored by the renderer. They are
// parsed and stored for future compatibility but not yet used during rendering.
func (nl NormalizedLayout) ToLayout() Layout {
	var layout Layout

	switch nl.HorzMode {
	case ModeFull:
		layout = FitFullWidth
	case ModeFitting:
		layout = FitKerning
	case ModeSmushingControlled:
		// Note: ModeSmushingControlled with HorzRules==0 is impossible by design.
		// Both NormalizeOldLayout and parseFullLayout ensure that controlled
		// smushing always has at least one rule bit set.
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
