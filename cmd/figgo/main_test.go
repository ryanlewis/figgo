package main

import (
	"testing"
)

func TestParseUnknownRune(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    rune
		wantErr bool
	}{
		// Literal characters
		{"literal star", "*", '*', false},
		{"literal question", "?", '?', false},
		{"literal underscore", "_", '_', false},
		{"literal unicode emoji", "☺", '☺', false},

		// Escaped Unicode
		{"unicode escape \\u2588", "\\u2588", '\u2588', false},
		{"unicode escape \\u003F", "\\u003F", '?', false},
		{"unicode escape \\U00002588", "\\U00002588", '\u2588', false},

		// Unicode notation
		{"unicode U+2588", "U+2588", '\u2588', false},
		{"unicode u+2588", "u+2588", '\u2588', false},
		{"unicode U+003F", "U+003F", '?', false},

		// Decimal
		{"decimal 63", "63", '?', false},
		{"decimal 42", "42", '*', false},
		{"decimal 9608", "9608", '█', false},

		// Hexadecimal
		{"hex 0x3F", "0x3F", '?', false},
		{"hex 0x2A", "0x2A", '*', false},
		{"hex 0X3F", "0X3F", '?', false},

		// Invalid inputs
		{"empty string", "", 0, true},
		{"invalid unicode escape", "\\u", 0, true},
		{"invalid unicode notation", "U+", 0, true},
		{"invalid hex", "0x", 0, true},
		{"multi-rune literal", "abc", 0, true},
		{"invalid format", "xyz", 0, true},

		// Invalid rune values
		{"beyond max rune", "U+110000", 0, true}, // > utf8.MaxRune
		{"negative decimal", "-1", 0, true},
		{"surrogate start", "U+D800", 0, true}, // UTF-16 surrogate
		{"surrogate end", "U+DFFF", 0, true},   // UTF-16 surrogate
		{"surrogate mid", "0xDC00", 0, true},   // UTF-16 surrogate

		// Exact length validation
		{"unicode escape too short", "\\u258", 0, true},
		{"unicode escape too long", "\\u25888", 0, true},
		{"unicode U escape too short", "\\U0002588", 0, true},
		{"unicode U escape too long", "\\U000025888", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseUnknownRune(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseUnknownRune(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseUnknownRune(%q) = %v (U+%04X), want %v (U+%04X)",
					tt.input, got, got, tt.want, tt.want)
			}
		})
	}
}
