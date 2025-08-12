package renderer

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/ryanlewis/figgo/internal/parser"
)

func TestRenderTo(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		font    *parser.Font
		opts    *Options
		want    string
		wantErr error
	}{
		{
			name:    "nil font returns error",
			text:    "test",
			font:    nil,
			opts:    nil,
			want:    "",
			wantErr: ErrNilFont,
		},
		{
			name: "simple text renders correctly",
			text: "AB",
			font: &parser.Font{
				Height:    2,
				Hardblank: '$',
				Characters: map[rune][]string{
					'A': {"AA", "AA"},
					'B': {"BB", "BB"},
				},
			},
			opts: &Options{Layout: 0}, // Full width
			want: "AABB\nAABB",
		},
		{
			name: "hardblank replacement works",
			text: "A",
			font: &parser.Font{
				Height:    2,
				Hardblank: '#',
				Characters: map[rune][]string{
					'A': {"A#A", "AAA"},
				},
			},
			opts: nil,
			want: "A A\nAAA",
		},
		{
			name: "trim whitespace works",
			text: "A",
			font: &parser.Font{
				Height:    2,
				Hardblank: '$',
				Characters: map[rune][]string{
					'A': {"A  ", "A  "},
				},
			},
			opts: &Options{TrimWhitespace: true},
			want: "A\nA",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := RenderTo(&buf, tt.text, tt.font, tt.opts)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("RenderTo() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("RenderTo() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("RenderTo() unexpected error = %v", err)
				return
			}

			got := buf.String()
			if got != tt.want {
				t.Errorf("RenderTo() got = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderToWriterTypes(t *testing.T) {
	font := &parser.Font{
		Height:    2,
		Hardblank: '$',
		Characters: map[rune][]string{
			'H': {"HH", "HH"},
			'I': {"I", "I"},
		},
	}
	text := "HI"
	want := "HHI\nHHI"

	t.Run("bytes.Buffer", func(t *testing.T) {
		var buf bytes.Buffer
		if err := RenderTo(&buf, text, font, nil); err != nil {
			t.Fatalf("RenderTo() error = %v", err)
		}
		if got := buf.String(); got != want {
			t.Errorf("RenderTo() = %q, want %q", got, want)
		}
	})

	t.Run("strings.Builder", func(t *testing.T) {
		var sb strings.Builder
		if err := RenderTo(&sb, text, font, nil); err != nil {
			t.Fatalf("RenderTo() error = %v", err)
		}
		if got := sb.String(); got != want {
			t.Errorf("RenderTo() = %q, want %q", got, want)
		}
	})

	t.Run("io.Discard", func(t *testing.T) {
		// Should not error even though output is discarded
		if err := RenderTo(io.Discard, text, font, nil); err != nil {
			t.Errorf("RenderTo(io.Discard) error = %v", err)
		}
	})
}

func TestRenderUsesRenderTo(t *testing.T) {
	// Test that Render internally uses RenderTo
	font := &parser.Font{
		Height:    3,
		Hardblank: '$',
		Characters: map[rune][]string{
			'T': {"TTT", " T ", " T "},
			'E': {"EEE", "E$ ", "EEE"},
			'S': {"SSS", "S$ ", "SSS"},
		},
	}

	text := "TEST"
	opts := &Options{Layout: 0}

	// Render using both methods
	got1, err1 := Render(text, font, opts)
	if err1 != nil {
		t.Fatalf("Render() error = %v", err1)
	}

	var buf bytes.Buffer
	err2 := RenderTo(&buf, text, font, opts)
	if err2 != nil {
		t.Fatalf("RenderTo() error = %v", err2)
	}
	got2 := buf.String()

	// They should produce identical output
	if got1 != got2 {
		t.Errorf("Render and RenderTo produce different output:\nRender: %q\nRenderTo: %q", got1, got2)
	}
}

// mockFailWriter simulates write failures for error testing
type mockFailWriter struct {
	failAfter int
	written   int
}

func (m *mockFailWriter) Write(p []byte) (n int, err error) {
	if m.written+len(p) > m.failAfter {
		n = m.failAfter - m.written
		m.written = m.failAfter
		return n, errors.New("write failed")
	}
	m.written += len(p)
	return len(p), nil
}

func TestRenderToWriteError(t *testing.T) {
	font := &parser.Font{
		Height:    2,
		Hardblank: '$',
		Characters: map[rune][]string{
			'A': {"AAAA", "AAAA"},
			'B': {"BBBB", "BBBB"},
		},
	}

	// Create a writer that fails after writing 5 bytes
	w := &mockFailWriter{failAfter: 5}

	err := RenderTo(w, "AB", font, nil)
	if err == nil {
		t.Error("RenderTo() should have returned an error for failed write")
	}
	if err.Error() != "write failed" {
		t.Errorf("RenderTo() error = %v, want 'write failed'", err)
	}
}

func TestRenderToLargeOutput(t *testing.T) {
	// Test that the buffer flushing works correctly for large outputs
	font := &parser.Font{
		Height:     1,
		Hardblank:  '$',
		Characters: map[rune][]string{},
	}

	// Create a character that's 100 runes wide
	wideChar := strings.Repeat("X", 100)
	font.Characters['X'] = []string{wideChar}

	// Render 10 of them (1000 chars total)
	text := strings.Repeat("X", 10)

	var buf bytes.Buffer
	err := RenderTo(&buf, text, font, nil)
	if err != nil {
		t.Fatalf("RenderTo() error = %v", err)
	}

	got := buf.String()
	want := strings.Repeat(wideChar, 10)

	if got != want {
		t.Errorf("RenderTo() length = %d, want %d", len(got), len(want))
	}
}

func TestRenderToUTF8(t *testing.T) {
	// Test that UTF-8 characters are handled correctly
	font := &parser.Font{
		Height:    1,
		Hardblank: '$',
		Characters: map[rune][]string{
			'H': {"H€llo"}, // Contains Euro sign (3 bytes in UTF-8)
			'W': {"Wörld"}, // Contains umlaut (2 bytes in UTF-8)
		},
	}

	var buf bytes.Buffer
	err := RenderTo(&buf, "HW", font, nil)
	if err != nil {
		t.Fatalf("RenderTo() error = %v", err)
	}

	got := buf.String()
	want := "H€lloWörld"

	if got != want {
		t.Errorf("RenderTo() = %q, want %q", got, want)
	}

	// Verify the byte count is correct for UTF-8
	if gotLen := len(got); gotLen != len(want) {
		t.Errorf("RenderTo() byte length = %d, want %d", gotLen, len(want))
	}
}
