package figgo

import (
	"fmt"
	"testing"
)

func TestNormalizeLayoutFromOldLayout(t *testing.T) { //nolint:gocognit // Test function with many test cases
	tests := []struct {
		name          string
		oldLayout     int
		fullLayout    int
		fullLayoutSet bool
		want          NormalizedLayout
		wantErr       bool
	}{
		// OldLayout only tests
		{
			name:          "OldLayout -1 -> Full width",
			oldLayout:     -1,
			fullLayout:    0,
			fullLayoutSet: false,
			want: NormalizedLayout{
				HorzMode:  ModeFull,
				HorzRules: 0,
				VertMode:  ModeFull,
				VertRules: 0,
			},
			wantErr: false,
		},
		{
			name:          "OldLayout 0 -> Fitting (kerning)",
			oldLayout:     0,
			fullLayout:    0,
			fullLayoutSet: false,
			want: NormalizedLayout{
				HorzMode:  ModeFitting,
				HorzRules: 0,
				VertMode:  ModeFull,
				VertRules: 0,
			},
			wantErr: false,
		},
		{
			name:          "OldLayout 1 -> Smushing with rule 1",
			oldLayout:     1,
			fullLayout:    0,
			fullLayoutSet: false,
			want: NormalizedLayout{
				HorzMode:  ModeSmushingControlled,
				HorzRules: 0x01, // Rule 1
				VertMode:  ModeFull,
				VertRules: 0,
			},
			wantErr: false,
		},
		{
			name:          "OldLayout 3 -> Smushing with rules 1+2",
			oldLayout:     3,
			fullLayout:    0,
			fullLayoutSet: false,
			want: NormalizedLayout{
				HorzMode:  ModeSmushingControlled,
				HorzRules: 0x03, // Rules 1+2
				VertMode:  ModeFull,
				VertRules: 0,
			},
			wantErr: false,
		},
		{
			name:          "OldLayout 63 -> Smushing with all 6 rules",
			oldLayout:     63,
			fullLayout:    0,
			fullLayoutSet: false,
			want: NormalizedLayout{
				HorzMode:  ModeSmushingControlled,
				HorzRules: 0x3F, // All 6 rules
				VertMode:  ModeFull,
				VertRules: 0,
			},
			wantErr: false,
		},
		{
			name:          "OldLayout -2 -> Fitting (alias for 0)",
			oldLayout:     -2,
			fullLayout:    0,
			fullLayoutSet: false,
			want: NormalizedLayout{
				HorzMode:  ModeFitting,
				HorzRules: 0,
				VertMode:  ModeFull,
				VertRules: 0,
			},
			wantErr: false,
		},
		{
			name:          "OldLayout -3 -> Universal Smushing",
			oldLayout:     -3,
			fullLayout:    0,
			fullLayoutSet: false,
			want: NormalizedLayout{
				HorzMode:  ModeSmushingUniversal,
				HorzRules: 0,
				VertMode:  ModeFull,
				VertRules: 0,
			},
			wantErr: false,
		},
		{
			name:          "OldLayout -4 -> Invalid",
			oldLayout:     -4,
			fullLayout:    0,
			fullLayoutSet: false,
			want:          NormalizedLayout{},
			wantErr:       true,
		},
		{
			name:          "OldLayout 64 -> Invalid",
			oldLayout:     64,
			fullLayout:    0,
			fullLayoutSet: false,
			want:          NormalizedLayout{},
			wantErr:       true,
		},
		{
			name:          "OldLayout 100 -> Invalid",
			oldLayout:     100,
			fullLayout:    0,
			fullLayoutSet: false,
			want:          NormalizedLayout{},
			wantErr:       true,
		},
		// Universal smushing tests
		{
			name:          "FullLayout 128 -> Horizontal Universal Smushing (no rules)",
			oldLayout:     0,
			fullLayout:    128,
			fullLayoutSet: true,
			want: NormalizedLayout{
				HorzMode:  ModeSmushingUniversal,
				HorzRules: 0,
				VertMode:  ModeFull,
				VertRules: 0,
			},
			wantErr: false,
		},
		{
			name:          "FullLayout 16384 -> Vertical Universal Smushing (no rules)",
			oldLayout:     0,
			fullLayout:    16384,
			fullLayoutSet: true,
			want: NormalizedLayout{
				HorzMode:  ModeFull,
				HorzRules: 0,
				VertMode:  ModeSmushingUniversal,
				VertRules: 0,
			},
			wantErr: false,
		},
		// FullLayout only tests
		{
			name:          "FullLayout 0 with set flag -> Full/Full",
			oldLayout:     0,
			fullLayout:    0,
			fullLayoutSet: true,
			want: NormalizedLayout{
				HorzMode:  ModeFull,
				HorzRules: 0,
				VertMode:  ModeFull,
				VertRules: 0,
			},
			wantErr: false,
		},
		{
			name:          "FullLayout 64 -> Horizontal Fitting",
			oldLayout:     0,
			fullLayout:    64,
			fullLayoutSet: true,
			want: NormalizedLayout{
				HorzMode:  ModeFitting,
				HorzRules: 0,
				VertMode:  ModeFull,
				VertRules: 0,
			},
			wantErr: false,
		},
		{
			name:          "FullLayout 129 -> Horizontal Controlled Smushing with rule 1",
			oldLayout:     0,
			fullLayout:    129, // 128 + 1
			fullLayoutSet: true,
			want: NormalizedLayout{
				HorzMode:  ModeSmushingControlled,
				HorzRules: 0x01,
				VertMode:  ModeFull,
				VertRules: 0,
			},
			wantErr: false,
		},
		{
			name:          "FullLayout 8192 -> Vertical Fitting",
			oldLayout:     0,
			fullLayout:    8192,
			fullLayoutSet: true,
			want: NormalizedLayout{
				HorzMode:  ModeFull,
				HorzRules: 0,
				VertMode:  ModeFitting,
				VertRules: 0,
			},
			wantErr: false,
		},
		{
			name:          "FullLayout 16640 -> Vertical Controlled Smushing with rule 1",
			oldLayout:     0,
			fullLayout:    16640, // 16384 + 256
			fullLayoutSet: true,
			want: NormalizedLayout{
				HorzMode:  ModeFull,
				HorzRules: 0,
				VertMode:  ModeSmushingControlled,
				VertRules: 0x01,
			},
			wantErr: false,
		},
		{
			name:          "FullLayout with both H and V smushing",
			oldLayout:     0,
			fullLayout:    191 + 16384 + 256, // H: 128+63, V: 16384+256
			fullLayoutSet: true,
			want: NormalizedLayout{
				HorzMode:  ModeSmushingControlled,
				HorzRules: 0x3F, // All 6 horizontal rules
				VertMode:  ModeSmushingControlled,
				VertRules: 0x01, // Vertical rule 1
			},
			wantErr: false,
		},
		// Both-bits-set precedence tests
		{
			name:          "Horizontal: both fitting(64) and smushing(128) -> smushing wins",
			oldLayout:     0,
			fullLayout:    64 + 128, // Both horizontal fitting and smushing
			fullLayoutSet: true,
			want: NormalizedLayout{
				HorzMode:  ModeSmushingUniversal, // Smushing wins, no rules = universal
				HorzRules: 0,
				VertMode:  ModeFull,
				VertRules: 0,
			},
			wantErr: false,
		},
		{
			name:          "Horizontal: conflict with no rules -> universal smushing",
			oldLayout:     0,
			fullLayout:    192, // 128|64 = both bits, no rules
			fullLayoutSet: true,
			want: NormalizedLayout{
				HorzMode:  ModeSmushingUniversal, // Universal smushing (no rules)
				HorzRules: 0,
				VertMode:  ModeFull,
				VertRules: 0,
			},
			wantErr: false,
		},
		{
			name:          "Vertical: both fitting(8192) and smushing(16384) -> smushing wins",
			oldLayout:     0,
			fullLayout:    8192 + 16384, // Both vertical fitting and smushing
			fullLayoutSet: true,
			want: NormalizedLayout{
				HorzMode:  ModeFull,
				HorzRules: 0,
				VertMode:  ModeSmushingUniversal, // Smushing wins, no rules = universal
				VertRules: 0,
			},
			wantErr: false,
		},
		{
			name:          "Vertical: conflict with no rules -> universal smushing",
			oldLayout:     0,
			fullLayout:    24576, // 16384|8192 = both bits, no rules
			fullLayoutSet: true,
			want: NormalizedLayout{
				HorzMode:  ModeFull,
				HorzRules: 0,
				VertMode:  ModeSmushingUniversal, // Universal smushing (no rules)
				VertRules: 0,
			},
			wantErr: false,
		},
		{
			name:          "Both H and V: fitting+smushing -> smushing wins on both axes",
			oldLayout:     0,
			fullLayout:    64 + 128 + 8192 + 16384 + 3, // Both bits set + some H rules
			fullLayoutSet: true,
			want: NormalizedLayout{
				HorzMode:  ModeSmushingControlled, // Smushing wins, with rules
				HorzRules: 0x03,                   // Rules 1+2
				VertMode:  ModeSmushingUniversal,  // Smushing wins, no V rules
				VertRules: 0,
			},
			wantErr: false,
		},
		// Precedence tests
		{
			name:          "FullLayout overrides OldLayout when set",
			oldLayout:     0,   // Would be Fitting
			fullLayout:    128, // Universal smushing
			fullLayoutSet: true,
			want: NormalizedLayout{
				HorzMode:  ModeSmushingUniversal,
				HorzRules: 0,
				VertMode:  ModeFull,
				VertRules: 0,
			},
			wantErr: false,
		},
		{
			name:          "FullLayout without any fit bits defaults to Full/Full",
			oldLayout:     1, // Would be controlled smushing
			fullLayout:    0, // No bits set
			fullLayoutSet: true,
			want: NormalizedLayout{
				HorzMode:  ModeFull,
				HorzRules: 0,
				VertMode:  ModeFull,
				VertRules: 0,
			},
			wantErr: false,
		},
		{
			name:          "Universal smushing when smushing bit set but no rules",
			oldLayout:     0,
			fullLayout:    128, // Smushing bit only, no rule bits
			fullLayoutSet: true,
			want: NormalizedLayout{
				HorzMode:  ModeSmushingUniversal,
				HorzRules: 0,
				VertMode:  ModeFull,
				VertRules: 0,
			},
			wantErr: false,
		},
		{
			name:          "Conflicting Old=3, Full=128 -> Full wins",
			oldLayout:     3,   // Would be controlled smushing
			fullLayout:    128, // Universal smushing
			fullLayoutSet: true,
			want: NormalizedLayout{
				HorzMode:  ModeSmushingUniversal,
				HorzRules: 0,
				VertMode:  ModeFull,
				VertRules: 0,
			},
			wantErr: false,
		},
		{
			name:          "Old=-1, Full=64 -> Full wins (Fitting)",
			oldLayout:     -1, // Would be Full width
			fullLayout:    64, // Fitting
			fullLayoutSet: true,
			want: NormalizedLayout{
				HorzMode:  ModeFitting,
				HorzRules: 0,
				VertMode:  ModeFull,
				VertRules: 0,
			},
			wantErr: false,
		},
		// Invalid FullLayout tests
		{
			name:          "FullLayout negative -> Invalid",
			oldLayout:     0,
			fullLayout:    -1,
			fullLayoutSet: true,
			want:          NormalizedLayout{},
			wantErr:       true,
		},
		{
			name:          "FullLayout > 32767 -> Invalid",
			oldLayout:     0,
			fullLayout:    32768,
			fullLayoutSet: true,
			want:          NormalizedLayout{},
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeLayoutFromHeader(tt.oldLayout, tt.fullLayout, tt.fullLayoutSet)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeLayoutFromHeader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.HorzMode != tt.want.HorzMode {
					t.Errorf("HorzMode = %v, want %v", got.HorzMode, tt.want.HorzMode)
				}
				if got.HorzRules != tt.want.HorzRules {
					t.Errorf("HorzRules = 0x%02X, want 0x%02X", got.HorzRules, tt.want.HorzRules)
				}
				if got.VertMode != tt.want.VertMode {
					t.Errorf("VertMode = %v, want %v", got.VertMode, tt.want.VertMode)
				}
				if got.VertRules != tt.want.VertRules {
					t.Errorf("VertRules = 0x%02X, want 0x%02X", got.VertRules, tt.want.VertRules)
				}
			}
		})
	}
}

func TestNormalizeLayoutFromHeader_FuzzRandom(t *testing.T) {
	// Test a range of valid FullLayout values to ensure:
	// - no panic
	// - modes are valid
	// - rule bits are within constraints
	testValues := []int{
		0, 1, 63, 64, 127, 128, 191, 192, 255, 256,
		512, 1024, 2048, 4096, 8192, 16384, 24576, 32767,
		// Some specific combinations
		64 + 128,        // Both H fitting and smushing
		8192 + 16384,    // Both V fitting and smushing
		191 + 16384,     // All H bits + V smushing
		128 + 8192 + 63, // H smush + V fit + all H rules
	}

	for _, fullLayout := range testValues {
		t.Run(fmt.Sprintf("FullLayout_%d", fullLayout), func(t *testing.T) {
			// Should not panic
			result, err := NormalizeLayoutFromHeader(0, fullLayout, true)
			if err != nil {
				t.Fatalf("Unexpected error for valid FullLayout %d: %v", fullLayout, err)
			}

			// Validate horizontal mode
			if result.HorzMode < ModeFull || result.HorzMode > ModeSmushingUniversal {
				t.Errorf("Invalid HorzMode %v for FullLayout %d", result.HorzMode, fullLayout)
			}

			// Validate horizontal rules (max 6 bits = 0x3F)
			if result.HorzRules > 0x3F {
				t.Errorf("HorzRules 0x%02X exceeds max 0x3F for FullLayout %d", result.HorzRules, fullLayout)
			}

			// Validate vertical mode
			if result.VertMode < ModeFull || result.VertMode > ModeSmushingUniversal {
				t.Errorf("Invalid VertMode %v for FullLayout %d", result.VertMode, fullLayout)
			}

			// Validate vertical rules (max 5 bits = 0x1F)
			if result.VertRules > 0x1F {
				t.Errorf("VertRules 0x%02X exceeds max 0x1F for FullLayout %d", result.VertRules, fullLayout)
			}
		})
	}
}

func TestNormalizeLayoutFromHeader_FullLayoutPrecedence(t *testing.T) {
	tests := []struct {
		name          string
		oldLayout     int
		fullLayout    int
		wantHorzMode  AxisMode
		wantHorzRules uint8
		wantVertMode  AxisMode
		wantVertRules uint8
	}{
		{
			name:          "FullLayout precedence: smush no rules -> universal",
			oldLayout:     23, // Would be controlled if used
			fullLayout:    128,
			wantHorzMode:  ModeSmushingUniversal,
			wantHorzRules: 0,
			wantVertMode:  ModeFull,
			wantVertRules: 0,
		},
		{
			name:          "FullLayout precedence: conflict + rule -> smushing wins",
			oldLayout:     0, // Would be fitting if used
			fullLayout:    128 | 64 | 1,
			wantHorzMode:  ModeSmushingControlled,
			wantHorzRules: 1, // Rule 1 set
			wantVertMode:  ModeFull,
			wantVertRules: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeLayoutFromHeader(tt.oldLayout, tt.fullLayout, true)
			if err != nil {
				t.Fatalf("NormalizeLayoutFromHeader() error = %v", err)
			}
			if got.HorzMode != tt.wantHorzMode {
				t.Errorf("HorzMode = %v, want %v", got.HorzMode, tt.wantHorzMode)
			}
			if got.HorzRules != tt.wantHorzRules {
				t.Errorf("HorzRules = 0x%02X, want 0x%02X", got.HorzRules, tt.wantHorzRules)
			}
			if got.VertMode != tt.wantVertMode {
				t.Errorf("VertMode = %v, want %v", got.VertMode, tt.wantVertMode)
			}
			if got.VertRules != tt.wantVertRules {
				t.Errorf("VertRules = 0x%02X, want 0x%02X", got.VertRules, tt.wantVertRules)
			}
		})
	}
}

func TestNormalizeLayoutFromHeader_OldLayoutFallback(t *testing.T) {
	got, err := NormalizeLayoutFromHeader(23, 0, false)
	if err != nil {
		t.Fatalf("NormalizeLayoutFromHeader() error = %v", err)
	}
	if got.HorzMode != ModeSmushingControlled {
		t.Errorf("HorzMode = %v, want %v", got.HorzMode, ModeSmushingControlled)
	}
	if got.HorzRules != 0x17 {
		t.Errorf("HorzRules = 0x%02X, want 0x%02X", got.HorzRules, 0x17)
	}
	if got.VertMode != ModeFull {
		t.Errorf("VertMode = %v, want %v", got.VertMode, ModeFull)
	}
	if got.VertRules != 0 {
		t.Errorf("VertRules = 0x%02X, want 0x%02X", got.VertRules, 0)
	}
}

func TestNormalizeLayoutFromHeader_VerticalParsing(t *testing.T) {
	tests := []struct {
		name          string
		fullLayout    int
		wantVertMode  AxisMode
		wantVertRules uint8
	}{
		{
			name:          "Vertical controlled smushing with rules",
			fullLayout:    16384 | 256 | 512, // V smush + rules 1,2
			wantVertMode:  ModeSmushingControlled,
			wantVertRules: 0x03, // Rules 1,2
		},
		{
			name:          "Vertical fitting only",
			fullLayout:    8192,
			wantVertMode:  ModeFitting,
			wantVertRules: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeLayoutFromHeader(0, tt.fullLayout, true)
			if err != nil {
				t.Fatalf("NormalizeLayoutFromHeader() error = %v", err)
			}
			if got.VertMode != tt.wantVertMode {
				t.Errorf("VertMode = %v, want %v", got.VertMode, tt.wantVertMode)
			}
			if got.VertRules != tt.wantVertRules {
				t.Errorf("VertRules = 0x%02X, want 0x%02X", got.VertRules, tt.wantVertRules)
			}
		})
	}
}

func TestNormalizeLayoutFromHeader_InvalidRanges(t *testing.T) {
	tests := []struct {
		name          string
		oldLayout     int
		fullLayout    int
		fullLayoutSet bool
	}{
		{
			name:          "OldLayout -4 invalid",
			oldLayout:     -4,
			fullLayout:    0,
			fullLayoutSet: false,
		},
		{
			name:          "FullLayout 32768 invalid",
			oldLayout:     0,
			fullLayout:    32768,
			fullLayoutSet: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeLayoutFromHeader(tt.oldLayout, tt.fullLayout, tt.fullLayoutSet)
			if err == nil {
				t.Errorf("NormalizeLayoutFromHeader() expected error but got none")
			}
		})
	}
}

func TestNormalizeLayout_OptionValidation(t *testing.T) {
	tests := []struct {
		name    string
		input   Layout
		want    Layout
		wantErr bool
	}{
		{
			name:    "Conflicting FitKerning|FitSmushing errors",
			input:   FitKerning | FitSmushing,
			wantErr: true,
		},
		{
			name:    "Zero defaults to FitFullWidth",
			input:   0,
			want:    FitFullWidth,
			wantErr: false,
		},
		{
			name:    "Valid FitSmushing with rules preserved",
			input:   FitSmushing | RuleEqualChar | RuleBigX,
			want:    FitSmushing | RuleEqualChar | RuleBigX,
			wantErr: false,
		},
		{
			name:    "All three fitting modes error",
			input:   FitFullWidth | FitKerning | FitSmushing,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeLayout(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeLayout() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("NormalizeLayout() = 0x%08X, want 0x%08X", uint32(got), uint32(tt.want))
			}
		})
	}
}

func TestNormalizedLayoutToLayout(t *testing.T) {
	tests := []struct {
		name  string
		input NormalizedLayout
		want  Layout
	}{
		{
			name: "Horizontal Full width",
			input: NormalizedLayout{
				HorzMode: ModeFull,
				VertMode: ModeFull,
			},
			want: FitFullWidth,
		},
		{
			name: "Horizontal Fitting (kerning)",
			input: NormalizedLayout{
				HorzMode: ModeFitting,
				VertMode: ModeFull,
			},
			want: FitKerning,
		},
		{
			name: "Horizontal Controlled Smushing with rules",
			input: NormalizedLayout{
				HorzMode:  ModeSmushingControlled,
				HorzRules: 0x15, // Rules 1, 3, 5
				VertMode:  ModeFull,
			},
			want: FitSmushing | RuleEqualChar | RuleHierarchy | RuleBigX,
		},
		{
			name: "Horizontal Universal Smushing",
			input: NormalizedLayout{
				HorzMode: ModeSmushingUniversal,
				VertMode: ModeFull,
			},
			want: FitSmushing, // Universal smushing has no rule bits
		},
		{
			name: "All horizontal rules",
			input: NormalizedLayout{
				HorzMode:  ModeSmushingControlled,
				HorzRules: 0x3F, // All 6 rules
				VertMode:  ModeFull,
			},
			want: FitSmushing | RuleEqualChar | RuleUnderscore | RuleHierarchy | RuleOppositePair | RuleBigX | RuleHardblank,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.ToLayout()
			if got != tt.want {
				t.Errorf("ToLayout() = 0x%08X (%s), want 0x%08X (%s)",
					uint32(got), got.String(), uint32(tt.want), tt.want.String())
			}
		})
	}
}

// TestToLayoutMappingCompleteness verifies that each individual rule bit (0-5)
// maps correctly to the corresponding Layout rule constant.
func TestToLayoutMappingCompleteness(t *testing.T) {
	tests := []struct {
		name     string
		ruleBit  uint8
		wantRule Layout
	}{
		{"Rule 1 (bit 0) -> RuleEqualChar", 1 << 0, RuleEqualChar},
		{"Rule 2 (bit 1) -> RuleUnderscore", 1 << 1, RuleUnderscore},
		{"Rule 3 (bit 2) -> RuleHierarchy", 1 << 2, RuleHierarchy},
		{"Rule 4 (bit 3) -> RuleOppositePair", 1 << 3, RuleOppositePair},
		{"Rule 5 (bit 4) -> RuleBigX", 1 << 4, RuleBigX},
		{"Rule 6 (bit 5) -> RuleHardblank", 1 << 5, RuleHardblank},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nl := NormalizedLayout{
				HorzMode:  ModeSmushingControlled,
				HorzRules: tt.ruleBit,
				VertMode:  ModeFull,
			}
			got := nl.ToLayout()

			// Should have FitSmushing plus the specific rule
			want := FitSmushing | tt.wantRule
			if got != want {
				t.Errorf("ToLayout() with rule bit %d = 0x%08X, want 0x%08X",
					tt.ruleBit, uint32(got), uint32(want))
			}

			// Verify the specific rule bit is set
			if got&tt.wantRule == 0 {
				t.Errorf("ToLayout() missing expected rule %s", tt.wantRule.String())
			}
		})
	}
}

// TestNormalizeOldLayoutRange tests that NormalizeOldLayout properly validates its input range
func TestNormalizeOldLayoutRange(t *testing.T) {
	tests := []struct {
		name      string
		oldLayout int
		wantErr   bool
	}{
		{"Valid: -3 (universal smushing)", -3, false},
		{"Valid: -2 (fitting alias)", -2, false},
		{"Valid: -1 (full width)", -1, false},
		{"Valid: 0 (fitting)", 0, false},
		{"Valid: 1 (smushing rule 1)", 1, false},
		{"Valid: 63 (all rules)", 63, false},
		{"Invalid: -4", -4, true},
		{"Invalid: -10", -10, true},
		{"Invalid: -100", -100, true},
		{"Invalid: 64", 64, true},
		{"Invalid: 100", 100, true},
		{"Invalid: 1000", 1000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeOldLayout(tt.oldLayout)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeOldLayout(%d) error = %v, wantErr %v",
					tt.oldLayout, err, tt.wantErr)
			}
		})
	}
}

// TestFullLayoutPrecedenceMatrix tests that FullLayout always wins when fullLayoutSet=true
func TestFullLayoutPrecedenceMatrix(t *testing.T) {
	tests := []struct {
		name       string
		oldLayout  int
		fullLayout int
		wantHorz   AxisMode
		wantVert   AxisMode
	}{
		// FullLayout wins regardless of OldLayout
		{"Old=-1(full) vs Full=64(fitting)", -1, 64, ModeFitting, ModeFull},
		{"Old=0(fitting) vs Full=0(default->full)", 0, 0, ModeFull, ModeFull},
		{"Old=1(smush) vs Full=64(fitting)", 1, 64, ModeFitting, ModeFull},
		{"Old=63(all rules) vs Full=128(universal)", 63, 128, ModeSmushingUniversal, ModeFull},
		// Only vertical bits set (no horizontal fit bits)
		{"Only vertical fitting", -1, 8192, ModeFull, ModeFitting},
		{"Only vertical smushing", 0, 16384, ModeFull, ModeSmushingUniversal},
		{"Only vertical controlled", 1, 16384 | 256, ModeFull, ModeSmushingControlled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeLayoutFromHeader(tt.oldLayout, tt.fullLayout, true)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result.HorzMode != tt.wantHorz {
				t.Errorf("HorzMode = %v, want %v", result.HorzMode, tt.wantHorz)
			}
			if result.VertMode != tt.wantVert {
				t.Errorf("VertMode = %v, want %v", result.VertMode, tt.wantVert)
			}
		})
	}
}

// TestBothFitBitsInFullLayout tests that smushing wins when both fitting and smushing bits are set
func TestBothFitBitsInFullLayout(t *testing.T) {
	tests := []struct {
		name       string
		fullLayout int
		wantHorz   AxisMode
		wantVert   AxisMode
	}{
		// Horizontal: both 64(fitting) and 128(smushing) set
		{"H: both bits, no rules -> universal", 64 | 128, ModeSmushingUniversal, ModeFull},
		{"H: both bits, with rules -> controlled", 64 | 128 | 3, ModeSmushingControlled, ModeFull},
		// Vertical: both 8192(fitting) and 16384(smushing) set
		{"V: both bits, no rules -> universal", 8192 | 16384, ModeFull, ModeSmushingUniversal},
		{"V: both bits, with rules -> controlled", 8192 | 16384 | 256, ModeFull, ModeSmushingControlled},
		// Both axes with conflicts
		{"Both axes conflicts", 64 | 128 | 8192 | 16384, ModeSmushingUniversal, ModeSmushingUniversal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeLayoutFromHeader(0, tt.fullLayout, true)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result.HorzMode != tt.wantHorz {
				t.Errorf("HorzMode = %v, want %v", result.HorzMode, tt.wantHorz)
			}
			if result.VertMode != tt.wantVert {
				t.Errorf("VertMode = %v, want %v", result.VertMode, tt.wantVert)
			}
		})
	}
}

// TestUniversalVsControlledSmushing tests the distinction between universal and controlled smushing
func TestUniversalVsControlledSmushing(t *testing.T) {
	tests := []struct {
		name          string
		fullLayout    int
		wantHorzMode  AxisMode
		wantHorzRules uint8
		wantVertMode  AxisMode
		wantVertRules uint8
	}{
		// Horizontal
		{"H: smushing bit only -> universal", 128, ModeSmushingUniversal, 0, ModeFull, 0},
		{"H: smushing + rule 1 -> controlled", 128 | 1, ModeSmushingControlled, 1, ModeFull, 0},
		{"H: smushing + rules 1,3,5 -> controlled", 128 | 1 | 4 | 16, ModeSmushingControlled, 0x15, ModeFull, 0},
		{"H: smushing + all rules -> controlled", 128 | 63, ModeSmushingControlled, 0x3F, ModeFull, 0},
		// Vertical
		{"V: smushing bit only -> universal", 16384, ModeFull, 0, ModeSmushingUniversal, 0},
		{"V: smushing + rule 1 -> controlled", 16384 | 256, ModeFull, 0, ModeSmushingControlled, 1},
		{"V: smushing + rules 1,2,3 -> controlled", 16384 | 256 | 512 | 1024, ModeFull, 0, ModeSmushingControlled, 7},
		{"V: smushing + all 5 rules -> controlled", 16384 | (31 << 8), ModeFull, 0, ModeSmushingControlled, 0x1F},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeLayoutFromHeader(0, tt.fullLayout, true)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result.HorzMode != tt.wantHorzMode {
				t.Errorf("HorzMode = %v, want %v", result.HorzMode, tt.wantHorzMode)
			}
			if result.HorzRules != tt.wantHorzRules {
				t.Errorf("HorzRules = 0x%02X, want 0x%02X", result.HorzRules, tt.wantHorzRules)
			}
			if result.VertMode != tt.wantVertMode {
				t.Errorf("VertMode = %v, want %v", result.VertMode, tt.wantVertMode)
			}
			if result.VertRules != tt.wantVertRules {
				t.Errorf("VertRules = 0x%02X, want 0x%02X", result.VertRules, tt.wantVertRules)
			}
		})
	}
}

// TestNormalizeLayoutDefaulting tests that NormalizeLayout properly defaults when only rule bits are present
func TestNormalizeLayoutDefaulting(t *testing.T) {
	tests := []struct {
		name  string
		input Layout
		want  Layout
	}{
		{"No bits -> FitFullWidth", 0, FitFullWidth},
		{"Only RuleEqualChar -> FitFullWidth preserved", RuleEqualChar, FitFullWidth | RuleEqualChar},
		{"Only rules 1,3,5 -> FitFullWidth preserved", RuleEqualChar | RuleHierarchy | RuleBigX,
			FitFullWidth | RuleEqualChar | RuleHierarchy | RuleBigX},
		{"All rules, no fit -> FitFullWidth preserved",
			RuleEqualChar | RuleUnderscore | RuleHierarchy | RuleOppositePair | RuleBigX | RuleHardblank,
			FitFullWidth | RuleEqualChar | RuleUnderscore | RuleHierarchy | RuleOppositePair | RuleBigX | RuleHardblank},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeLayout(tt.input)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("NormalizeLayout(0x%08X) = 0x%08X, want 0x%08X",
					uint32(tt.input), uint32(got), uint32(tt.want))
			}
			// Verify rules are preserved
			if got.Rules() != tt.input.Rules() {
				t.Errorf("Rules not preserved: got 0x%08X, want 0x%08X",
					uint32(got.Rules()), uint32(tt.input.Rules()))
			}
		})
	}
}

// TestLayoutStringWithInvalid tests that Layout.String() properly handles invalid states
func TestLayoutStringWithInvalid(t *testing.T) {
	tests := []struct {
		name   string
		layout Layout
		want   string
	}{
		{"Single mode", FitFullWidth, "FitFullWidth"},
		{"Conflict 2 modes", FitKerning | FitSmushing, "INVALID:FitKerning|FitSmushing"},
		{"Conflict 2 modes (FitKerning|FitSmushing)", FitKerning | FitSmushing, "INVALID:FitKerning|FitSmushing"},
		{"Valid with rules", FitSmushing | RuleEqualChar | RuleBigX, "FitSmushing|RuleEqualChar|RuleBigX"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.layout.String()
			if got != tt.want {
				t.Errorf("Layout.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestAxisModeString tests the AxisMode String method
func TestAxisModeString(t *testing.T) {
	tests := []struct {
		mode AxisMode
		want string
	}{
		{ModeFull, "Full"},
		{ModeFitting, "Fitting"},
		{ModeSmushingControlled, "SmushingControlled"},
		{ModeSmushingUniversal, "SmushingUniversal"},
		{AxisMode(99), "Unknown(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.mode.String()
			if got != tt.want {
				t.Errorf("AxisMode.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestNormalizedLayoutString tests the NormalizedLayout String method
func TestNormalizedLayoutString(t *testing.T) {
	nl := NormalizedLayout{
		HorzMode:  ModeSmushingControlled,
		HorzRules: 0x15,
		VertMode:  ModeFitting,
		VertRules: 0,
	}

	want := "Horz:SmushingControlled(rules:0x15) Vert:Fitting(rules:0x00)"
	got := nl.String()
	if got != want {
		t.Errorf("NormalizedLayout.String() = %q, want %q", got, want)
	}
}
