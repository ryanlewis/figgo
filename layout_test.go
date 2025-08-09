package figgo

import (
	"strings"
	"testing"
)

func TestLayoutConstants(t *testing.T) {
	// Test that fitting mode constants exist and have correct values
	tests := []struct {
		name     string
		layout   Layout
		expected uint32
	}{
		{"FitFullWidth", FitFullWidth, 0x00000000},         // No bits set
		{"FitKerning", FitKerning, 0x00000040},             // Bit 6
		{"FitSmushing", FitSmushing, 0x00000080},           // Bit 7
		{"RuleEqualChar", RuleEqualChar, 0x00000001},       // Bit 0
		{"RuleUnderscore", RuleUnderscore, 0x00000002},     // Bit 1
		{"RuleHierarchy", RuleHierarchy, 0x00000004},       // Bit 2
		{"RuleOppositePair", RuleOppositePair, 0x00000008}, // Bit 3
		{"RuleBigX", RuleBigX, 0x00000010},                 // Bit 4
		{"RuleHardblank", RuleHardblank, 0x00000020},       // Bit 5
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if uint32(tt.layout) != tt.expected {
				t.Errorf("%s = 0x%08X, want 0x%08X", tt.name, uint32(tt.layout), tt.expected)
			}
		})
	}
}

func TestLayoutValidation(t *testing.T) {
	tests := []struct {
		wantErrContains string
		name            string
		layout          Layout
		wantFitting     Layout
	}{
		{
			name:            "valid FitFullWidth only",
			layout:          FitFullWidth,
			wantErrContains: "",
			wantFitting:     FitFullWidth,
		},
		{
			name:            "valid FitKerning only",
			layout:          FitKerning,
			wantErrContains: "",
			wantFitting:     FitKerning,
		},
		{
			name:            "valid FitSmushing only",
			layout:          FitSmushing,
			wantErrContains: "",
			wantFitting:     FitSmushing,
		},
		{
			name:            "valid FitSmushing with rules",
			layout:          FitSmushing | RuleEqualChar | RuleHierarchy,
			wantErrContains: "",
			wantFitting:     FitSmushing,
		},
		{
			name:            "invalid both FitKerning and FitSmushing",
			layout:          FitKerning | FitSmushing,
			wantErrContains: "layout conflict",
			wantFitting:     0,
		},
		{
			name:            "no fitting mode defaults to FitFullWidth",
			layout:          RuleEqualChar | RuleHierarchy, // Rules without fitting mode
			wantErrContains: "",
			wantFitting:     FitFullWidth,
		},
		{
			name:            "zero layout defaults to FitFullWidth",
			layout:          0,
			wantErrContains: "",
			wantFitting:     FitFullWidth,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized, err := NormalizeLayout(tt.layout)
			if tt.wantErrContains != "" {
				if err == nil {
					t.Errorf("NormalizeLayout() error = nil, want error containing %q", tt.wantErrContains)
					return
				}
				if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("NormalizeLayout() error = %v, want error containing %q", err, tt.wantErrContains)
					return
				}
			} else if err != nil {
				t.Errorf("NormalizeLayout() unexpected error = %v", err)
				return
			}
			if err == nil && normalized.FittingMode() != tt.wantFitting {
				t.Errorf("NormalizeLayout().FittingMode() = %v, want %v", normalized.FittingMode(), tt.wantFitting)
			}
		})
	}
}

func TestLayoutHasRule(t *testing.T) {
	layout := FitSmushing | RuleEqualChar | RuleBigX

	if !layout.HasRule(RuleEqualChar) {
		t.Error("HasRule(RuleEqualChar) = false, want true")
	}
	if !layout.HasRule(RuleBigX) {
		t.Error("HasRule(RuleBigX) = false, want true")
	}
	if layout.HasRule(RuleHierarchy) {
		t.Error("HasRule(RuleHierarchy) = true, want false")
	}
	if layout.HasRule(RuleUnderscore) {
		t.Error("HasRule(RuleUnderscore) = true, want false")
	}
}

func TestLayoutFittingMode(t *testing.T) {
	tests := []struct {
		name   string
		layout Layout
		want   Layout
	}{
		{"FitFullWidth only", FitFullWidth, FitFullWidth},
		{"FitKerning only", FitKerning, FitKerning},
		{"FitSmushing only", FitSmushing, FitSmushing},
		{"FitSmushing with rules", FitSmushing | RuleEqualChar | RuleHardblank, FitSmushing},
		{"Rules only (no fitting)", RuleEqualChar | RuleHierarchy, 0},
		{"Multiple fitting modes", FitKerning | FitSmushing, FitKerning | FitSmushing}, // Returns all fitting bits
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.layout.FittingMode(); got != tt.want {
				t.Errorf("FittingMode() = 0x%08X, want 0x%08X", uint32(got), uint32(tt.want))
			}
		})
	}
}

func TestLayoutRules(t *testing.T) {
	layout := FitSmushing | RuleEqualChar | RuleBigX | RuleHardblank
	rules := layout.Rules()

	expectedRules := RuleEqualChar | RuleBigX | RuleHardblank
	if rules != expectedRules {
		t.Errorf("Rules() = 0x%08X, want 0x%08X", uint32(rules), uint32(expectedRules))
	}

	// Test that fitting bits are not included
	if rules&FitSmushing != 0 {
		t.Error("Rules() should not include fitting bits")
	}
}

func TestLayoutString(t *testing.T) {
	tests := []struct {
		want   string
		layout Layout
	}{
		{"FitFullWidth", FitFullWidth},
		{"FitKerning", FitKerning},
		{"FitSmushing", FitSmushing},
		{"FitSmushing|RuleEqualChar", FitSmushing | RuleEqualChar},
		{"FitSmushing|RuleEqualChar|RuleBigX", FitSmushing | RuleEqualChar | RuleBigX},
		{"INVALID:FitKerning|FitSmushing", FitKerning | FitSmushing}, // Multiple fitting modes
		{"FitSmushing|RuleEqualChar|RuleUnderscore|RuleHierarchy|RuleOppositePair|RuleBigX|RuleHardblank",
			FitSmushing | RuleEqualChar | RuleUnderscore | RuleHierarchy | RuleOppositePair | RuleBigX | RuleHardblank}, // All rules
		{"FitFullWidth", 0},                // Zero value represents FitFullWidth
		{"0x12345678", Layout(0x12345678)}, // Unknown bits
		{"0x80000000", Layout(0x80000000)}, // High bit set
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.layout.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLayoutNormalizationFromOldLayout(t *testing.T) {
	// Test conversion from OldLayout integer to Layout bitmask
	// Based on spec-compliance.md section 4
	tests := []struct {
		name      string
		oldLayout int
		want      Layout
	}{
		{
			name:      "negative full width",
			oldLayout: -1,
			want:      FitFullWidth,
		},
		{
			name:      "fitting (kerning) mode",
			oldLayout: 0,
			want:      FitKerning,
		},
		{
			name:      "smushing with equal char",
			oldLayout: 1, // Smushing + equal char rule (bit 0)
			want:      FitSmushing | RuleEqualChar,
		},
		{
			name:      "smushing with underscore",
			oldLayout: 2, // Smushing + underscore rule (bit 1)
			want:      FitSmushing | RuleUnderscore,
		},
		{
			name:      "smushing with hierarchy",
			oldLayout: 4, // Smushing + hierarchy rule (bit 2)
			want:      FitSmushing | RuleHierarchy,
		},
		{
			name:      "smushing with opposite pair",
			oldLayout: 8, // Smushing + opposite pair rule (bit 3)
			want:      FitSmushing | RuleOppositePair,
		},
		{
			name:      "smushing with big x",
			oldLayout: 16, // Smushing + big x rule (bit 4)
			want:      FitSmushing | RuleBigX,
		},
		{
			name:      "smushing with hardblank",
			oldLayout: 32, // Smushing + hardblank rule (bit 5)
			want:      FitSmushing | RuleHardblank,
		},
		{
			name:      "smushing with multiple rules",
			oldLayout: 3, // bits 0+1 = equal char + underscore
			want:      FitSmushing | RuleEqualChar | RuleUnderscore,
		},
		{
			name:      "smushing with all rules",
			oldLayout: 63, // All 6 rule bits
			want:      FitSmushing | RuleEqualChar | RuleUnderscore | RuleHierarchy | RuleOppositePair | RuleBigX | RuleHardblank,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeOldLayout(tt.oldLayout)
			if err != nil {
				t.Fatalf("NormalizeOldLayout(%d) unexpected error: %v", tt.oldLayout, err)
			}
			if got != tt.want {
				t.Errorf("NormalizeOldLayout(%d) = 0x%08X, want 0x%08X", tt.oldLayout, uint32(got), uint32(tt.want))
			}
		})
	}
}
