package figgo

import (
	"errors"
	"testing"
)

func TestLayoutConstants(t *testing.T) {
	// Test that fitting mode constants exist and have correct values
	tests := []struct {
		name     string
		layout   Layout
		expected uint32
	}{
		{"FitFullWidth", FitFullWidth, 0x00000040},         // Bit 6
		{"FitKerning", FitKerning, 0x00000080},             // Bit 7
		{"FitSmushing", FitSmushing, 0x00000100},           // Bit 8
		{"RuleEqualChar", RuleEqualChar, 0x00000200},       // Bit 9
		{"RuleUnderscore", RuleUnderscore, 0x00000400},     // Bit 10
		{"RuleHierarchy", RuleHierarchy, 0x00000800},       // Bit 11
		{"RuleOppositePair", RuleOppositePair, 0x00001000}, // Bit 12
		{"RuleBigX", RuleBigX, 0x00002000},                 // Bit 13
		{"RuleHardblank", RuleHardblank, 0x00004000},       // Bit 14
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
		name        string
		layout      Layout
		wantErr     error
		wantFitting Layout
	}{
		{
			name:        "valid FitFullWidth only",
			layout:      FitFullWidth,
			wantErr:     nil,
			wantFitting: FitFullWidth,
		},
		{
			name:        "valid FitKerning only",
			layout:      FitKerning,
			wantErr:     nil,
			wantFitting: FitKerning,
		},
		{
			name:        "valid FitSmushing only",
			layout:      FitSmushing,
			wantErr:     nil,
			wantFitting: FitSmushing,
		},
		{
			name:        "valid FitSmushing with rules",
			layout:      FitSmushing | RuleEqualChar | RuleHierarchy,
			wantErr:     nil,
			wantFitting: FitSmushing,
		},
		{
			name:        "invalid both FitKerning and FitSmushing",
			layout:      FitKerning | FitSmushing,
			wantErr:     ErrLayoutConflict,
			wantFitting: 0,
		},
		{
			name:        "invalid both FitFullWidth and FitKerning",
			layout:      FitFullWidth | FitKerning,
			wantErr:     ErrLayoutConflict,
			wantFitting: 0,
		},
		{
			name:        "invalid all three fitting modes",
			layout:      FitFullWidth | FitKerning | FitSmushing,
			wantErr:     ErrLayoutConflict,
			wantFitting: 0,
		},
		{
			name:        "no fitting mode defaults to FitFullWidth",
			layout:      RuleEqualChar | RuleHierarchy, // Rules without fitting mode
			wantErr:     nil,
			wantFitting: FitFullWidth,
		},
		{
			name:        "zero layout defaults to FitFullWidth",
			layout:      0,
			wantErr:     nil,
			wantFitting: FitFullWidth,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized, err := NormalizeLayout(tt.layout)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("NormalizeLayout() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && normalized.FittingMode() != tt.wantFitting {
				t.Errorf("NormalizeLayout().FittingMode() = %v, want %v", normalized.FittingMode(), tt.wantFitting)
			}
		})
	}
}

func TestLayoutHelpers(t *testing.T) {
	t.Run("HasRule", func(t *testing.T) {
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
	})

	t.Run("FittingMode", func(t *testing.T) {
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
	})

	t.Run("Rules", func(t *testing.T) {
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
	})

	t.Run("String representation", func(t *testing.T) {
		tests := []struct {
			layout Layout
			want   string
		}{
			{FitFullWidth, "FitFullWidth"},
			{FitKerning, "FitKerning"},
			{FitSmushing, "FitSmushing"},
			{FitSmushing | RuleEqualChar, "FitSmushing|RuleEqualChar"},
			{FitSmushing | RuleEqualChar | RuleBigX, "FitSmushing|RuleEqualChar|RuleBigX"},
			{0, "0x00000000"},
			{Layout(0x12345678), "0x12345678"}, // Unknown bits
		}

		for _, tt := range tests {
			t.Run(tt.want, func(t *testing.T) {
				if got := tt.layout.String(); got != tt.want {
					t.Errorf("String() = %q, want %q", got, tt.want)
				}
			})
		}
	})
}

func TestLayoutNormalizationFromOldLayout(t *testing.T) {
	// Test conversion from OldLayout integer to Layout bitmask
	// Based on spec-compliance.md section 4
	tests := []struct {
		name      string
		oldLayout int32
		want      Layout
	}{
		{
			name:      "full width mode",
			oldLayout: 0,
			want:      FitFullWidth,
		},
		{
			name:      "kerning mode",
			oldLayout: 1,
			want:      FitKerning,
		},
		{
			name:      "smushing with equal char",
			oldLayout: 2, // Smushing + equal char rule (bit 0 in old format)
			want:      FitSmushing | RuleEqualChar,
		},
		{
			name:      "smushing with underscore",
			oldLayout: 4, // Smushing + underscore rule (bit 1 in old format)
			want:      FitSmushing | RuleUnderscore,
		},
		{
			name:      "smushing with hierarchy",
			oldLayout: 8, // Smushing + hierarchy rule (bit 2 in old format)
			want:      FitSmushing | RuleHierarchy,
		},
		{
			name:      "smushing with opposite pair",
			oldLayout: 16, // Smushing + opposite pair rule (bit 3 in old format)
			want:      FitSmushing | RuleOppositePair,
		},
		{
			name:      "smushing with big x",
			oldLayout: 32, // Smushing + big x rule (bit 4 in old format)
			want:      FitSmushing | RuleBigX,
		},
		{
			name:      "smushing with hardblank",
			oldLayout: 64, // Smushing + hardblank rule (bit 5 in old format)
			want:      FitSmushing | RuleHardblank,
		},
		{
			name:      "smushing with all rules",
			oldLayout: 126, // 2 + 4 + 8 + 16 + 32 + 64 = 126
			want:      FitSmushing | RuleEqualChar | RuleUnderscore | RuleHierarchy | RuleOppositePair | RuleBigX | RuleHardblank,
		},
		{
			name:      "negative full width",
			oldLayout: -1,
			want:      FitFullWidth,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeOldLayout(tt.oldLayout)
			if got != tt.want {
				t.Errorf("NormalizeOldLayout(%d) = 0x%08X, want 0x%08X", tt.oldLayout, uint32(got), uint32(tt.want))
			}
		})
	}
}
