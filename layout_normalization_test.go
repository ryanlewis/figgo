package figgo

import (
	"testing"
)

func TestNormalizeLayoutFromOldLayout(t *testing.T) {
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
			name:          "OldLayout -2 -> Invalid",
			oldLayout:     -2,
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
			name:          "FullLayout 128 -> Horizontal Universal Smushing",
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
			name:          "FullLayout 16384 -> Vertical Universal Smushing",
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
